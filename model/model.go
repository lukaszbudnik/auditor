package model

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/lukaszbudnik/auditor/hash"
)

func validateBlock(block interface{}) {
	if block == nil {
		panic("block must not be nil")
	}
	if reflect.TypeOf(block).Kind() != reflect.Ptr {
		panic(fmt.Sprintf("block argument must be a pointer to struct, but got: %v", reflect.TypeOf(block).Kind()))
	}
	if reflect.TypeOf(block).Elem().Kind() != reflect.Struct {
		panic(fmt.Sprintf("block argument must be a pointer to struct, but got: %v", reflect.TypeOf(block).Elem().Kind()))
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

// SetField sets a new value of a given field on a passed pointer to struct
func SetField(block interface{}, field reflect.StructField, value interface{}) bool {
	// we expect block to be a pointer to a struct
	validateBlock(block)
	fieldValue := reflect.ValueOf(block).Elem().FieldByName(field.Name)
	if fieldValue.IsValid() && fieldValue.CanSet() {
		fieldValue.Set(reflect.ValueOf(value))
		return true
	}
	return false
}

// ComputeAndSetHash computes and sets hash on given block
func ComputeAndSetHash(block interface{}) (err error) {
	validateBlock(block)
	hash, err := hash.ComputeHash(block)
	if err != nil {
		return
	}
	hashField := GetFieldsTaggedWith(block, "hash")
	SetField(block, hashField[0], hash)
	return
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
	SetField(block, previousHashField[0], previousHash)
}

// NewBlockWithSerialize creates new Block, sets PreviousHash based on previous Block.Hash
// and computes new Block.Hash using serialize function passed as last parameter
// func NewBlockWithSerialize(customer string, timestamp *time.Time, category, subcategory, event string, previousBlock *Block, serialize func(object interface{}) ([]byte, error)) (*Block, error) {
// 	newBlock := &Block{Customer: customer, Timestamp: timestamp, Category: category, Subcategory: subcategory, Event: event}
// 	if previousBlock != nil {
// 		newBlock.PreviousHash = previousBlock.Hash
// 	}
// 	hash, err := hash.ComputeHashWithSerialize(newBlock, serialize)
// 	if err != nil {
// 		return nil, err
// 	}
// 	newBlock.Hash = hash
// 	return newBlock, nil
// }

// NewBlock creates new Block, sets index and PreviousHash based on previous Block's values
// func NewBlock(customer string, timestamp *time.Time, category, subcategory, event string, previousBlock *Block) (*Block, error) {
// 	return NewBlockWithSerialize(customer, timestamp, category, subcategory, event, previousBlock, hash.Serialize)
// }
