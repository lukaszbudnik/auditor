package dynamodb

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

type testBlock struct {
	Customer     string     `auditor:"dynamodb_partition"`
	Timestamp    *time.Time `auditor:"sort"`
	Category     string
	Subcategory  string
	Event        string
	Hash         string `auditor:"hash"`
	PreviousHash string `auditor:"previoushash"`
}

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
	store.Save(&testBlock{Customer: "abc", Timestamp: &time1, Category: "restapi", Subcategory: "db", Event: "record updated"})
	store.Save(&testBlock{Customer: "abc", Timestamp: &time2, Category: "restapi", Subcategory: "cache", Event: "record updated"})

	last := testBlock{Customer: "abc"}
	page1 := []testBlock{}
	err = store.Read(&page1, 1, &last)
	assert.Nil(t, err)
	assert.Equal(t, time2.UTC().String(), page1[0].Timestamp.UTC().String())

	page2 := []testBlock{}
	err = store.Read(&page2, 1, &page1[0])
	assert.Nil(t, err)
	assert.Equal(t, time1.UTC().String(), page2[0].Timestamp.UTC().String())

	all := []testBlock{}
	store.Read(&all, 2, &last)
	assert.Nil(t, err)
	assert.Equal(t, len(all), len(page1)+len(page2))
	assert.Subset(t, all, page1)
	assert.Subset(t, all, page2)
}
