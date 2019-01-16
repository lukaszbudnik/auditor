package model

import (
	"log"
	"os"
	"reflect"
	"strings"

	"github.com/lukaszbudnik/auditor/hash"
)

// ValidateBlockType validates if passed pointer to struct is a valid auditor block
func ValidateBlockType(block interface{}) {
	validateBlock(block)
	hashField := GetTypeFieldsTaggedWith(reflect.TypeOf(block).Elem(), "hash")
	if len(hashField) != 1 {
		log.Panicf("block type must have one field tagged with 'hash', found: %v", len(hashField))
	}
	previousHashField := GetTypeFieldsTaggedWith(reflect.TypeOf(block).Elem(), "previoushash")
	if len(previousHashField) != 1 {
		log.Panicf("block type must have one field tagged with 'previoushash', found: %v", len(previousHashField))
	}
	sortField := GetTypeFieldsTaggedWith(reflect.TypeOf(block).Elem(), "sort")
	if len(sortField) != 1 {
		log.Panicf("block type must have one field tagged with 'sort', found: %v", len(sortField))
	}
	if os.Getenv("AUDITOR_STORE") == "dynamodb" {
		partitionField := GetTypeFieldsTaggedWith(reflect.TypeOf(block).Elem(), "dynamodb_partition")
		if len(partitionField) != 1 {
			log.Panicf("when using DynamoDB block type must have one field tagged with 'dynamodb_partition', found: %v", len(partitionField))
		}
	}
}

func validateBlock(block interface{}) {
	if block == nil {
		log.Panic("block must not be nil")
	}
	if reflect.TypeOf(block).Kind() != reflect.Ptr {
		log.Panicf("block argument must be a pointer to struct, but got: %v", reflect.TypeOf(block).Kind())
	}
	if reflect.TypeOf(block).Elem().Kind() != reflect.Struct {
		log.Panicf("block argument must be a pointer to struct, but got: %v", reflect.TypeOf(block).Elem().Kind())
	}
}

// GetTypeFieldsTaggedWith gets a StructField tagged with a specific auditor value
func GetTypeFieldsTaggedWith(t reflect.Type, tagValue string) []reflect.StructField {
	fields := []reflect.StructField{}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		tags := strings.Split(field.Tag.Get("auditor"), ",")
		for _, tag := range tags {
			if tag == tagValue {
				fields = append(fields, field)
			}
		}
	}

	return fields
}

// GetFieldsTaggedWith gets a StructField tagged with a specific auditor value
func GetFieldsTaggedWith(block interface{}, tagValue string) []reflect.StructField {
	// we expect block to be a pointer to a struct
	validateBlock(block)

	t := reflect.ValueOf(block).Elem().Type()
	return GetTypeFieldsTaggedWith(t, tagValue)
}

// GetFieldValue gets a StructField tagged with a specific auditor value
func GetFieldValue(block interface{}, field reflect.StructField) interface{} {
	// we expect block to be a pointer to a struct
	validateBlock(block)

	fieldValue := reflect.ValueOf(block).Elem().FieldByName(field.Name)

	return fieldValue.Interface()
}

// SetFieldValue sets a new value of a given field on a passed pointer to struct
func SetFieldValue(block interface{}, field reflect.StructField, value interface{}) bool {
	// we expect block to be a pointer to a struct
	validateBlock(block)
	fieldValue := reflect.ValueOf(block).Elem().FieldByName(field.Name)
	if fieldValue.IsValid() && fieldValue.CanSet() {
		fieldValue.Set(reflect.ValueOf(value))
		return true
	}
	return false
}

// ComputeAndSetHash computes and sets hash on given block, returns new hash or error
func ComputeAndSetHash(block interface{}) (string, error) {
	validateBlock(block)
	hash, err := hash.ComputeHash(block)
	if err != nil {
		return "", err
	}
	hashField := GetFieldsTaggedWith(block, "hash")
	SetFieldValue(block, hashField[0], hash)
	return hash, nil
}

// SetPreviousHash sets a PreviousHash field on a block from Hash field of previous one
func SetPreviousHash(block, previousBlock interface{}) {
	if previousBlock == nil {
		return
	}
	validateBlock(block)
	validateBlock(previousBlock)

	hashField := GetFieldsTaggedWith(previousBlock, "hash")
	previousHashField := GetFieldsTaggedWith(block, "previoushash")
	previousHash := GetFieldValue(previousBlock, hashField[0])
	SetFieldValue(block, previousHashField[0], previousHash)
}
