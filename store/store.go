package store

import (
	"github.com/lukaszbudnik/auditor/model"
)

// Store represents store operations for audit database
type Store interface {
	Save(block *model.Block) error
	Read(limit int64, lastBlock *model.Block) ([]model.Block, error)
	Close()
}
