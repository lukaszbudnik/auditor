package store

import "github.com/lukaszbudnik/auditor/hash"

// Store represents store operations for audit database
type Store interface {
	Save(block *hash.Block) error
	Read(limit int64, lastBlock *hash.Block) ([]hash.Block, error)
	Close()
}
