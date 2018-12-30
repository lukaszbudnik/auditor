package hash

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type exampleBlock struct {
	Block
	Field1 int
	Field2 string
}

func newExampleBlock(customer string, timestamp time.Time, category, subcategory, event string, field1 int, field2 string, previousExampleBlock *exampleBlock) (*exampleBlock, error) {
	var previousBlock *Block
	if previousExampleBlock != nil {
		previousBlock = &previousExampleBlock.Block
	}
	block, err := NewBlock(customer, timestamp, category, subcategory, event, previousBlock)
	if err != nil {
		return nil, err
	}
	newExampleBlock := &exampleBlock{Block: *block, Field1: field1, Field2: field2}
	hash, err := ComputeHash(newExampleBlock)
	if err != nil {
		return nil, err
	}
	newExampleBlock.Hash = hash
	return newExampleBlock, nil
}

func TestComputeHash(t *testing.T) {
	timestamp := time.Now()
	exampleBlock1, err := newExampleBlock("abc", timestamp, "restapi", "db", "record updated", 1234, "456", nil)
	assert.Nil(t, err)
	assert.Len(t, exampleBlock1.Hash, 64)

	// same values as exampleBlock1 but with previousBlock provided
	exampleBlock2, err := newExampleBlock("abc", timestamp, "restapi", "db", "record updated", 1234, "456", exampleBlock1)
	assert.Nil(t, err)
	assert.Len(t, exampleBlock2.Hash, 64)
	assert.NotEqual(t, exampleBlock1.Hash, exampleBlock2.Hash)

	// same values as exampleBlock1 but exampleBlock.Field2: "4567"
	exampleBlock3, err := newExampleBlock("abc", timestamp, "restapi", "db", "record updated", 1234, "4567", nil)
	assert.Nil(t, err)
	assert.Len(t, exampleBlock3.Hash, 64)
	assert.NotEqual(t, exampleBlock1.Hash, exampleBlock3.Hash)

	// same values as exampleBlock1 but Block.Subcategory: "cache"
	exampleBlock4, err := newExampleBlock("abc", timestamp, "restapi", "cache", "record updated", 1234, "456", nil)
	assert.Nil(t, err)
	assert.Len(t, exampleBlock4.Hash, 64)
	assert.NotEqual(t, exampleBlock1.Hash, exampleBlock4.Hash)
}

// test different serialize
