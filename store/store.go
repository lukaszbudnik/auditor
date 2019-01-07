package store

// Store represents store operations for audit database
type Store interface {
	Save(block interface{}) error
	Read(result interface{}, limit int64, last interface{}) error
	Close()
}
