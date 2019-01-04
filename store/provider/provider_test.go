package provider

import (
	"os"
	"reflect"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

func TestNewDynamoDB(t *testing.T) {
	err := godotenv.Load("../../.env.test.dynamodb")
	assert.Nil(t, err)
	os.Setenv("AUDITOR_STORE", "dynamodb")

	store, err := NewStore()
	assert.Nil(t, err)

	// dynamoDB is private struct thus using reflection
	assert.Equal(t, "*dynamodb.dynamoDB", reflect.TypeOf(store).String())
}

func TestNewMongoDB(t *testing.T) {
	err := godotenv.Load("../../.env.test.mongodb")
	assert.Nil(t, err)
	os.Setenv("AUDITOR_STORE", "mongodb")

	store, err := NewStore()
	assert.Nil(t, err)

	// dynamoDB is private struct thus using reflection
	assert.Equal(t, "*mongodb.mongoDB", reflect.TypeOf(store).String())
}

func TestUnknownStore(t *testing.T) {
	os.Setenv("AUDITOR_STORE", "X")

	_, err := NewStore()
	assert.Equal(t, "Unknown store: X", err.Error())
}
