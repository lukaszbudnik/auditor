package dynamodb

import (
	"fmt"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/lukaszbudnik/auditor/model"
	"github.com/lukaszbudnik/auditor/store"
)

type dynamoDB struct {
	client *dynamodb.DynamoDB
	lock   *sync.Mutex
}

func (d *dynamoDB) Save(block interface{}) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	time.Sleep(50 * time.Millisecond)
	// get type
	t := reflect.ValueOf(block).Elem().Type()
	// create *[]type
	ts := reflect.SliceOf(t)
	ptr := reflect.New(ts)
	ptr.Elem().Set(reflect.MakeSlice(ts, 0, 1))

	// for dynamodb last block must not be empty
	// and most field tagged with dynamodb_partiion populated
	// below we are copying it from the block
	lastv := reflect.New(t)
	fields := model.GetTypeFieldsTaggedWith(t, "dynamodb_partition")
	value := model.GetFieldValue(block, fields[0])
	model.SetField(lastv.Interface(), fields[0], value)

	d.Read(ptr.Interface(), 1, lastv.Interface())
	if ptr.Elem().Len() > 0 {
		model.SetPreviousHash(block, ptr.Elem().Index(0).Addr().Interface())
	}
	model.ComputeAndSetHash(block)

	av, err := dynamodbattribute.MarshalMap(block)
	if err != nil {
		return err
	}

	putInput := &dynamodb.PutItemInput{
		Item:      av,
		TableName: aws.String("audit"),
	}

	_, err = d.client.PutItem(putInput)

	return err
}

func (d *dynamoDB) Read(result interface{}, limit int64, last interface{}) error {

	if last == nil {
		panic("last argument must not be nil as it is used for DynamoDB hash key")
	}

	resultv := reflect.ValueOf(result)
	if resultv.Kind() != reflect.Ptr {
		panic("result argument must be a pointer to slice of struct")
	}
	slicev := resultv.Elem()
	if slicev.Kind() != reflect.Slice {
		panic("result argument must be a pointer to slice of struct")
	}
	if slicev.Type().Elem().Kind() != reflect.Struct {
		panic("result argument must be a pointer to slice of struct")
	}

	var exclusiveStartKey map[string]*dynamodb.AttributeValue

	queryInput := &dynamodb.QueryInput{
		TableName:        aws.String("audit"),
		Limit:            aws.Int64(limit),
		ScanIndexForward: aws.Bool(false),
		ConsistentRead:   aws.Bool(true),
	}

	lastv := reflect.ValueOf(last)
	if lastv.Kind() != reflect.Ptr {
		panic("last argument must be a pointer to struct")
	}
	if lastv.Type().Elem().Kind() != reflect.Struct {
		panic("last argument must be a pointer to struct")
	}

	if lastv.Type().Elem() != slicev.Type().Elem() {
		panic("result and last arguments must be of the same type")
	}

	fields := model.GetTypeFieldsTaggedWith(lastv.Type().Elem(), "sort")

	field := fields[0]
	fieldi := model.GetFieldValue(last, field)
	fieldv := reflect.ValueOf(fieldi)
	if field.Type == reflect.TypeOf(&time.Time{}) && !fieldv.IsNil() {
		in := []reflect.Value{reflect.ValueOf(time.RFC3339Nano)}
		timestamp := fieldv.MethodByName("Format").Call(in)[0]
		exclusiveStartKey = make(map[string]*dynamodb.AttributeValue)
		exclusiveStartKey[field.Name] = &dynamodb.AttributeValue{
			S: aws.String(fmt.Sprintf("%v", timestamp)),
		}
	}

	fields = model.GetTypeFieldsTaggedWith(lastv.Type().Elem(), "dynamodb_partition")
	field = fields[0]
	value := model.GetFieldValue(last, field)

	queryInput.SetKeyConditionExpression(fmt.Sprintf("%v = :partition", field.Name))
	queryInput.SetExpressionAttributeValues(map[string]*dynamodb.AttributeValue{":partition": {
		S: aws.String(fmt.Sprintf("%v", value)),
	}})
	if exclusiveStartKey != nil {
		exclusiveStartKey[field.Name] = &dynamodb.AttributeValue{
			S: aws.String(fmt.Sprintf("%v", value)),
		}
	}

	if exclusiveStartKey != nil {
		queryInput.SetExclusiveStartKey(exclusiveStartKey)
	}

	output, err := d.client.Query(queryInput)
	if err != nil {
		return err
	}

	return dynamodbattribute.UnmarshalListOfMaps(output.Items, &result)
}

func (d *dynamoDB) Close() {
	if d.client != nil {
		d.client.Config.Credentials.Expire()
	}
}

// New creates Store implementation for DynamoDB
func New() (store.Store, error) {
	client, err := newClient()
	if err != nil {
		return nil, err
	}

	dynamoDB := &dynamoDB{client: client, lock: &sync.Mutex{}}
	return dynamoDB, nil
}

func newClient() (*dynamodb.DynamoDB, error) {
	endpoint := os.Getenv("AWS_DYNAMODB_ENDPOINT")
	region := os.Getenv("AWS_REGION")

	creds := credentials.NewChainCredentials(
		[]credentials.Provider{
			&credentials.EnvProvider{},
			&credentials.SharedCredentialsProvider{},
			&ec2rolecreds.EC2RoleProvider{},
		})

	sess, err := session.NewSession(&aws.Config{
		Credentials: creds,
		Region:      aws.String(region),
		Endpoint:    aws.String(endpoint)})

	if err != nil {
		return nil, err
	}

	client := dynamodb.New(sess)

	return client, nil
}
