package dynamodb

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/joho/godotenv"
	"github.com/lukaszbudnik/auditor/model"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	if err := godotenv.Load("../../.env.test.dynamodb"); err != nil {
		log.Fatalf("Could not read env variables: %v", err.Error())
	}
	if err := tearDown(); err != nil {
		log.Fatalf("Could not delete old table: %v", err.Error())
	}
	if err := setup(); err != nil {
		log.Fatalf("Could not create table: %v", err.Error())
	}
	result := m.Run()
	tearDown()
	os.Exit(result)
}

func newTestCreds() *credentials.Credentials {
	creds := credentials.NewChainCredentials(
		[]credentials.Provider{
			&credentials.EnvProvider{},
			&credentials.SharedCredentialsProvider{},
			&ec2rolecreds.EC2RoleProvider{},
		})
	return creds
}

func setup() error {
	client, err := newClient()
	if err != nil {
		return err
	}

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

	_, err = client.CreateTable(createTableInput)
	if err != nil {
		log.Fatalf("Got error calling CreateTable: %v", err.Error())
	}

	return err
}

func tearDown() error {

	client, err := newClient()
	if err != nil {
		return err
	}

	listTableInput := &dynamodb.ListTablesInput{}
	listTableOutput, err := client.ListTables(listTableInput)
	if err != nil {
		return err
	}

	found := false
	for _, tableName := range listTableOutput.TableNames {
		if *tableName == "audit" {
			found = true
			break
		}
	}

	if !found {
		return nil
	}

	deleteTableInput := &dynamodb.DeleteTableInput{
		TableName: aws.String("audit"),
	}

	_, err = client.DeleteTable(deleteTableInput)

	return err
}

func TestDynamoDB(t *testing.T) {
	store, err := New()
	assert.Nil(t, err)
	defer store.Close()

	// need to truncate to nonoseconds as golang adds mono which is truncated by dynamodb
	// and nanoseconds are just fine...
	time1 := time.Now().Truncate(time.Nanosecond)
	time2 := time1.Add(1 * time.Second).Truncate(time.Nanosecond)
	store.Save(&model.Block{Customer: "abc", Timestamp: &time1, Category: "restapi", Subcategory: "db", Event: "record updated"})
	store.Save(&model.Block{Customer: "abc", Timestamp: &time2, Category: "restapi", Subcategory: "cache", Event: "record updated"})

	audit, err := store.Read(1, nil)
	assert.Nil(t, err)
	assert.Equal(t, time2.UTC().String(), audit[0].Timestamp.UTC().String())

	audit, err = store.Read(1, &audit[0])
	assert.Nil(t, err)
	assert.Equal(t, time1.UTC().String(), audit[0].Timestamp.UTC().String())

}
