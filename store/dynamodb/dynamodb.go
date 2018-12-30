package dynamodb

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/lukaszbudnik/auditor/hash"
	"github.com/lukaszbudnik/auditor/store"
)

type dynamoDB struct {
	client *dynamodb.DynamoDB
}

func (d *dynamoDB) Save(block *hash.Block) error {
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

func (d *dynamoDB) Read(limit int64, lastBlock *hash.Block) ([]hash.Block, error) {
	queryInput := &dynamodb.QueryInput{
		TableName:              aws.String("audit"),
		Limit:                  aws.Int64(limit),
		ScanIndexForward:       aws.Bool(false),
		KeyConditionExpression: aws.String("Customer = :customer"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{":customer": {
			S: aws.String("abc"),
		}},
	}

	if lastBlock != nil {
		queryInput.ExclusiveStartKey = map[string]*dynamodb.AttributeValue{"Customer": {
			S: aws.String(lastBlock.Customer),
		}, "Timestamp": {
			S: aws.String(lastBlock.Timestamp.Format(time.RFC3339Nano)),
		}}
	}

	output, err := d.client.Query(queryInput)
	if err != nil {
		return nil, err
	}

	audit := []hash.Block{}
	err = dynamodbattribute.UnmarshalListOfMaps(output.Items, &audit)
	return audit, err
}

func (d *dynamoDB) Close() {
	if d.client != nil {
		d.client.Config.Credentials.Expire()
	}
}

func NewDynamoDB(creds *credentials.Credentials, endpoint string) (store.Store, error) {
	dynamoDBPersister := &dynamoDB{}

	sess, err := session.NewSession(&aws.Config{
		Credentials: creds,
		Region:      aws.String("us-west-2"),
		Endpoint:    aws.String(endpoint)})

	if err != nil {
		return nil, err
	}

	client := dynamodb.New(sess)

	dynamoDBPersister.client = client

	return dynamoDBPersister, nil
}
