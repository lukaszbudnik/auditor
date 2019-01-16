package model

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testBlock struct {
	Category     string     `auditor:"mongodb_index"`
	Timestamp    *time.Time `auditor:"sort,mongodb_index"`
	Hash         string     `auditor:"hash"`
	PreviousHash string     `auditor:"previoushash"`
}

func TestValidateBlockTypeNilError(t *testing.T) {
	assert.Panics(t, func() {
		ValidateBlockType(nil)
	})
}

func TestValidateBlockTypeInvalidPointerError(t *testing.T) {
	assert.Panics(t, func() {
		ValidateBlockType(TestValidateBlockTypeInvalidTypeError)
	})
}

func TestValidateBlockTypeInvalidTypeError(t *testing.T) {
	assert.Panics(t, func() {
		test := "string"
		ValidateBlockType(&test)
	})
}

func TestValidateBlockTypeError1(t *testing.T) {
	os.Setenv("AUDITOR_STORE", "")
	// missing hash field
	s := struct {
	}{}
	assert.Panics(t, func() {
		ValidateBlockType(&s)
	})
}

func TestValidateBlockTypeError2(t *testing.T) {
	os.Setenv("AUDITOR_STORE", "")
	// missing previous hash field
	s := struct {
		Hash string `auditor:"hash"`
	}{}
	assert.Panics(t, func() {
		ValidateBlockType(&s)
	})
}

func TestValidateBlockTypeError3(t *testing.T) {
	os.Setenv("AUDITOR_STORE", "")
	// missing sort field
	s := struct {
		Hash         string `auditor:"hash"`
		PreviousHash string `auditor:"previoushash"`
	}{}
	assert.Panics(t, func() {
		ValidateBlockType(&s)
	})
}

func TestValidateBlockTypeDynamoDBError(t *testing.T) {
	os.Setenv("AUDITOR_STORE", "dynamodb")
	// missing dynamodb_partition field
	s := struct {
		Hash         string     `auditor:"hash"`
		PreviousHash string     `auditor:"previoushash"`
		Timestamp    *time.Time `auditor:"sort,mongodb_index"`
	}{}
	assert.Panics(t, func() {
		ValidateBlockType(&s)
	})
}

func TestGetFieldsWithTag(t *testing.T) {
	block := &testBlock{}
	fields := GetFieldsTaggedWith(block, "mongodb_index")
	assert.Len(t, fields, 2)
}

func TestGetFieldValue(t *testing.T) {
	category := "category goes here"
	block := &testBlock{Category: category}
	fields := GetFieldsTaggedWith(block, "mongodb_index")
	fieldValue := GetFieldValue(block, fields[0])
	assert.Equal(t, category, fieldValue)
}

func TestGetFieldStringValue(t *testing.T) {
	hash := "123"
	block := &testBlock{Hash: hash}
	fields := GetFieldsTaggedWith(block, "hash")
	fieldString := GetFieldStringValue(block, fields[0])
	assert.Equal(t, hash, fieldString)
}

func TestGetFieldValuePtr(t *testing.T) {
	time := time.Now()
	block := &testBlock{Timestamp: &time}
	fields := GetFieldsTaggedWith(block, "mongodb_index")
	fieldValue := GetFieldValue(block, fields[1])
	assert.Equal(t, &time, fieldValue)
}

func TestSetFieldValue(t *testing.T) {
	block := &testBlock{}
	fields := GetFieldsTaggedWith(block, "mongodb_index")
	expected := "This is new value"
	SetFieldValue(block, fields[0], expected)
	assert.Equal(t, expected, block.Category)
}

func TestComputeAndSetHash(t *testing.T) {
	block := &testBlock{}
	hash, err := ComputeAndSetHash(block)
	assert.Nil(t, err)
	assert.NotEmpty(t, block.Hash)
	assert.Equal(t, hash, block.Hash)
}

func TestSetPreviousHash(t *testing.T) {
	block := &testBlock{}
	previousBlock := &testBlock{Hash: "abcdef123"}
	SetPreviousHash(block, previousBlock)
	assert.Equal(t, previousBlock.Hash, block.PreviousHash)
}
