package hash

import (
	"bytes"
	"encoding/gob"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type testBlock struct {
	Customer     string
	Timestamp    time.Time
	Event        string
	Hash         string
	PreviousHash string
}

type extendedBlock struct {
	testBlock
	Field1 int
	Field2 string
}

// testBlock is private so need to tell gob to explicitly encode/decode it
func (eb *extendedBlock) GobEncode() ([]byte, error) {
	w := new(bytes.Buffer)
	encoder := gob.NewEncoder(w)
	err := encoder.Encode(eb.testBlock)
	if err != nil {
		return nil, err
	}
	err = encoder.Encode(eb.Field1)
	if err != nil {
		return nil, err
	}
	err = encoder.Encode(eb.Field2)
	if err != nil {
		return nil, err
	}
	return w.Bytes(), nil
}

func (eb *extendedBlock) GobDecode(buf []byte) error {
	r := bytes.NewBuffer(buf)
	decoder := gob.NewDecoder(r)
	if err := decoder.Decode(&eb.testBlock); err != nil {
		return err
	}
	if err := decoder.Decode(&eb.Field1); err != nil {
		return err
	}
	return decoder.Decode(&eb.Field2)
}

func newExampleBlock(customer string, timestamp time.Time, event string, field1 int, field2 string, previousBlock *extendedBlock) (*extendedBlock, error) {
	var previousHash string
	if previousBlock != nil {
		previousHash = previousBlock.Hash
	}
	block := testBlock{Customer: customer, Timestamp: timestamp, Event: event, PreviousHash: previousHash}

	newExampleBlock := &extendedBlock{testBlock: block, Field1: field1, Field2: field2}
	hash, err := ComputeHash(newExampleBlock)
	if err != nil {
		return nil, err
	}
	newExampleBlock.Hash = hash
	return newExampleBlock, nil
}

func TestComputeHash(t *testing.T) {
	timestamp := time.Now()
	exampleBlock1, err := newExampleBlock("abc", timestamp, "record updated", 1234, "456", nil)
	assert.Nil(t, err)
	assert.Len(t, exampleBlock1.Hash, 64)

	// same values as exampleBlock1 but with previousBlock provided
	exampleBlock2, err := newExampleBlock("abc", timestamp, "record updated", 1234, "456", exampleBlock1)
	assert.Nil(t, err)
	assert.Len(t, exampleBlock2.Hash, 64)
	assert.NotEqual(t, exampleBlock1.Hash, exampleBlock2.Hash)

	// same values as exampleBlock1 but with Field2 set to: "4567"
	exampleBlock3, err := newExampleBlock("abc", timestamp, "record updated", 1234, "4567", nil)
	assert.Nil(t, err)
	assert.Len(t, exampleBlock3.Hash, 64)
	assert.NotEqual(t, exampleBlock1.Hash, exampleBlock3.Hash)

	// same values as exampleBlock1 but Event set to: "record deleted"
	exampleBlock4, err := newExampleBlock("abc", timestamp, "record delete", 1234, "456", nil)
	assert.Nil(t, err)
	assert.Len(t, exampleBlock4.Hash, 64)
	assert.NotEqual(t, exampleBlock1.Hash, exampleBlock4.Hash)
}
