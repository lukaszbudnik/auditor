package provider

import (
	"fmt"
	"os"

	"github.com/lukaszbudnik/auditor/store"
	"github.com/lukaszbudnik/auditor/store/dynamodb"
	"github.com/lukaszbudnik/auditor/store/mongodb"
)

// NewStore creates new Store implementation based on AUDITOR_STORE or returns error
func NewStore() (store.Store, error) {
	storeName := os.Getenv("AUDITOR_STORE")
	switch storeName {
	case "mongodb":
		return mongodb.New()
	case "dynamodb":
		return dynamodb.New()
	default:
		return nil, fmt.Errorf("Unknown store: %v", storeName)
	}
}
