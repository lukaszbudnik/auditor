package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testBlock struct {
	Category     string     `auditor:"mongodb_index"`
	Timestamp    *time.Time `auditor:"dynamodb_range,mongodb_range,mongodb_index"`
	Hash         string     `auditor:"hash"`
	PreviousHash string     `auditor:"previoushash"`
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

func TestGetFieldValuePtr(t *testing.T) {
	time := time.Now()
	block := &testBlock{Timestamp: &time}
	fields := GetFieldsTaggedWith(block, "mongodb_index")
	fieldValue := GetFieldValue(block, fields[1])
	assert.Equal(t, &time, fieldValue)
}

func TestSetField(t *testing.T) {
	block := &testBlock{}
	fields := GetFieldsTaggedWith(block, "mongodb_index")
	expected := "This is new value"
	SetField(block, fields[0], expected)
	assert.Equal(t, expected, block.Category)
}

func TestSetPreviousHash(t *testing.T) {
	block := &testBlock{}
	previousBlock := &testBlock{Hash: "abcdef123"}
	SetPreviousHash(block, previousBlock)
	assert.Equal(t, previousBlock.Hash, block.PreviousHash)
}
