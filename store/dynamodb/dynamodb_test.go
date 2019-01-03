package dynamodb

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/lukaszbudnik/auditor/model"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	tearDown()
	setup()
	os.Exit(m.Run())
	tearDown()
}

func newTestCreds() *credentials.Credentials {
	creds := credentials.NewChainCredentials(
		[]credentials.Provider{
			&credentials.StaticProvider{Value: credentials.Value{
				AccessKeyID:     "abc",
				SecretAccessKey: "def",
				SessionToken:    "xyz",
			}},
			&credentials.EnvProvider{},
			&ec2rolecreds.EC2RoleProvider{},
		})
	return creds
}

func setup() {

	creds := newTestCreds()

	sess, err := session.NewSession(&aws.Config{
		Credentials: creds,
		Region:      aws.String("us-west-2"),
		Endpoint:    aws.String("http://localhost:8000")},
	)

	if err != nil {
		fmt.Println("Got error connecting:")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Create DynamoDB client
	svc := dynamodb.New(sess)

	createTableInput := &dynamodb.CreateTableInput{
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("Customer"),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String("Timestamp"),
				AttributeType: aws.String("S"),
			},
		},
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("Customer"),
				KeyType:       aws.String("HASH"),
			},
			{
				AttributeName: aws.String("Timestamp"),
				KeyType:       aws.String("RANGE"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(10),
			WriteCapacityUnits: aws.Int64(10),
		},
		TableName: aws.String("audit"),
	}

	_, err = svc.CreateTable(createTableInput)

	if err != nil {
		fmt.Println("Got error calling CreateTable:")
		fmt.Println(err.Error())
		os.Exit(1)
	}

}

func tearDown() {

	creds := newTestCreds()

	sess, err := session.NewSession(&aws.Config{
		Credentials: creds,
		Region:      aws.String("us-west-2"),
		Endpoint:    aws.String("http://localhost:8000")},
	)
	if err != nil {
		fmt.Println("Got error connecting:")
		fmt.Println(err.Error())
		os.Exit(1)
	}

	// Create DynamoDB client
	svc := dynamodb.New(sess)

	deleteTableInput := &dynamodb.DeleteTableInput{
		TableName: aws.String("audit"),
	}

	svc.DeleteTable(deleteTableInput)
}

func TestDynamoDB(t *testing.T) {
	store, err := NewDynamoDB(newTestCreds(), "http://localhost:8000")
	assert.Nil(t, err)
	defer store.Close()

	// need to truncate to nonoseconds as golang adds mono which is truncated by dynamodb
	// and nanoseconds are just fine...
	time1 := time.Now().Truncate(time.Nanosecond)
	time2 := time1.Add(1 * time.Second).Truncate(time.Nanosecond)
	store.Save(&model.Block{Customer: "abc", Timestamp: time1, Category: "restapi", Subcategory: "db", Event: "record updated"})
	store.Save(&model.Block{Customer: "abc", Timestamp: time2, Category: "restapi", Subcategory: "cache", Event: "record updated"})

	audit, err := store.Read(1, nil)
	assert.Nil(t, err)
	assert.Equal(t, time2.UTC().String(), audit[0].Timestamp.UTC().String())

	audit, err = store.Read(1, &audit[0])
	assert.Nil(t, err)
	assert.Equal(t, time1.UTC().String(), audit[0].Timestamp.UTC().String())

}
