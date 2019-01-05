package dynamodb

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/lukaszbudnik/auditor/store"
)

type dynamoDB struct {
	client *dynamodb.DynamoDB
}

func (d *dynamoDB) Save(block interface{}) error {
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

	queryInput := &dynamodb.QueryInput{
		TableName:        aws.String("audit"),
		Limit:            aws.Int64(limit),
		ScanIndexForward: aws.Bool(false),
	}

	var exclusiveStartKey map[string]*dynamodb.AttributeValue
	for i := 0; i < lastv.Elem().NumField(); i++ {
		field := slicev.Type().Elem().Field(i)
		tag := field.Tag.Get("auditor")
		if strings.Contains(tag, "dynamodb_range") && field.Type == reflect.TypeOf(&time.Time{}) && !lastv.Elem().Field(i).IsNil() {
			in := []reflect.Value{reflect.ValueOf(time.RFC3339Nano)}
			timestamp := lastv.Elem().Field(i).MethodByName("Format").Call(in)[0]
			exclusiveStartKey = make(map[string]*dynamodb.AttributeValue)
			exclusiveStartKey[field.Name] = &dynamodb.AttributeValue{
				S: aws.String(fmt.Sprintf("%v", timestamp)),
			}
		}
	}

	for i := 0; i < lastv.Elem().NumField(); i++ {
		field := slicev.Type().Elem().Field(i)
		tag := field.Tag.Get("auditor")
		if strings.Contains(tag, "dynamodb_hash") {
			value := lastv.Elem().Field(i).Interface()
			queryInput.SetKeyConditionExpression(fmt.Sprintf("%v = :hash", field.Name))
			queryInput.SetExpressionAttributeValues(map[string]*dynamodb.AttributeValue{":hash": {
				S: aws.String(fmt.Sprintf("%v", value)),
			}})
			if exclusiveStartKey != nil {
				exclusiveStartKey[field.Name] = &dynamodb.AttributeValue{
					S: aws.String(fmt.Sprintf("%v", value)),
				}
			}
			break
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

	dynamoDB := &dynamoDB{client: client}
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
