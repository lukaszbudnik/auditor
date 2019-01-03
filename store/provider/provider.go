package provider

import (
	"fmt"
	"os"

	"github.com/lukaszbudnik/auditor/store"
	"github.com/lukaszbudnik/auditor/store/cosmosdb"
	"github.com/lukaszbudnik/auditor/store/dynamodb"
)

func NewStore() (store.Store, error) {
	storeName := os.Getenv("AUDITOR_STORE")
	switch storeName {
	case "cosmosdb":
		return cosmosdb.New()
	case "dynamodb":
		return dynamodb.New()
	default:
		return nil, fmt.Errorf("Unknown store impl: %v", storeName)
	}
}
