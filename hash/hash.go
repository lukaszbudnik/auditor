package hash

import (
	"bytes"
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"time"
)

// Block is a base struct which should be embedded by implementation-specific ones
type Block struct {
	Customer     string
	Timestamp    time.Time
	Category     string
	Subcategory  string
	Event        string
	Hash         string
	PreviousHash string
}

// NewBlockWithSerialize creates new Block, sets index and PreviousHash based on previous Block's values
// and using serialize function passed as last parameter
func NewBlockWithSerialize(customer string, timestamp time.Time, category, subcategory, event string, previousBlock *Block, serialize func(object interface{}) ([]byte, error)) (*Block, error) {
	newBlock := &Block{Customer: customer, Timestamp: timestamp, Category: category, Subcategory: subcategory, Event: event}
	if previousBlock != nil {
		newBlock.PreviousHash = previousBlock.Hash
	}
	hash, err := ComputeHash(newBlock)
	if err != nil {
		return nil, err
	}
	newBlock.Hash = hash
	return newBlock, nil
}

// NewBlock creates new Block, sets index and PreviousHash based on previous Block's values
func NewBlock(customer string, timestamp time.Time, category, subcategory, event string, previousBlock *Block) (*Block, error) {
	return NewBlockWithSerialize(customer, timestamp, category, subcategory, event, previousBlock, Serialize)
}

// Serialize serializes passed struct to bytes using GOB
func Serialize(object interface{}) ([]byte, error) {
	var buffer bytes.Buffer
	enc := gob.NewEncoder(&buffer)
	if err := enc.Encode(object); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

// ComputeHash computes hash for passed struct using default Serialize function
func ComputeHash(object interface{}) (string, error) {
	return ComputeHashWithSerialize(object, Serialize)
}

// ComputeHashWithSerialize computes hash based on passed struct
// before computing hash, serializes object using provided function
func ComputeHashWithSerialize(object interface{}, serialize func(object interface{}) ([]byte, error)) (string, error) {
	bytes, err := serialize(object)
	if err != nil {
		return "", err
	}
	h := sha256.New()
	h.Write(bytes)
	hashed := h.Sum(nil)
	return hex.EncodeToString(hashed), nil
}
