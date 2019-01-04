package model

import (
	"time"

	"github.com/lukaszbudnik/auditor/hash"
)

// Block is a base struct which should be embedded by implementation-specific ones
type Block struct {
	Customer     string
	Timestamp    *time.Time `validate:"nonzero"`
	Category     string
	Subcategory  string
	Event        string `validate:"nonzero"`
	Hash         string
	PreviousHash string
}

// NewBlockWithSerialize creates new Block, sets PreviousHash based on previous Block.Hash
// and computes new Block.Hash using serialize function passed as last parameter
func NewBlockWithSerialize(customer string, timestamp *time.Time, category, subcategory, event string, previousBlock *Block, serialize func(object interface{}) ([]byte, error)) (*Block, error) {
	newBlock := &Block{Customer: customer, Timestamp: timestamp, Category: category, Subcategory: subcategory, Event: event}
	if previousBlock != nil {
		newBlock.PreviousHash = previousBlock.Hash
	}
	hash, err := hash.ComputeHashWithSerialize(newBlock, serialize)
	if err != nil {
		return nil, err
	}
	newBlock.Hash = hash
	return newBlock, nil
}

// NewBlock creates new Block, sets index and PreviousHash based on previous Block's values
func NewBlock(customer string, timestamp *time.Time, category, subcategory, event string, previousBlock *Block) (*Block, error) {
	return NewBlockWithSerialize(customer, timestamp, category, subcategory, event, previousBlock, hash.Serialize)
}
