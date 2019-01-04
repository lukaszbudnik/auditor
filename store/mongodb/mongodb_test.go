package mongodb

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/joho/godotenv"
	"github.com/lukaszbudnik/auditor/model"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	if err := godotenv.Load("../../.env.test.mongodb"); err != nil {
		log.Fatalf("Could not read env variables: %v", err.Error())
	}
	if err := tearDown(); err != nil {
		log.Fatalf("Error cleaning old records: %v", err.Error())
	}
	result := m.Run()
	tearDown()
	os.Exit(result)
}

func TestMongoDB(t *testing.T) {
	store, err := New()
	assert.Nil(t, err)
	defer store.Close()

	time1 := time.Now().Truncate(time.Millisecond)
	time2 := time1.Add(1 * time.Second).Truncate(time.Millisecond)
	store.Save(&model.Block{Customer: "abc", Timestamp: &time1, Category: "restapi", Subcategory: "db", Event: "record updated"})
	store.Save(&model.Block{Customer: "abc", Timestamp: &time2, Category: "restapi", Subcategory: "cache", Event: "record updated"})

	audit, err := store.Read(1, nil)
	assert.Nil(t, err)
	assert.Equal(t, time2.UTC().String(), audit[0].Timestamp.UTC().String())

	audit, err = store.Read(1, &audit[0])
	assert.Nil(t, err)
	assert.Equal(t, time1.UTC().String(), audit[0].Timestamp.UTC().String())

	session, err := newSession()
	assert.Nil(t, err)
	indexes, err := session.DB("audit").C("audit").Indexes()
	assert.Nil(t, err)
	// there are at minimum 5 indexes (there is a default _id_ index in addition to 4 defined in model.Block)
	// when using CosmosDB there are additional indexes prefixed DocumentDBDefaultIndex thus using greater than
	assert.True(t, len(indexes) > 5)
}

func tearDown() error {
	session, err := newSession()
	if err != nil {
		return err
	}

	collection := session.DB("audit").C("audit")

	iter := collection.Find(nil).Iter()

	var entry entry
	for iter.Next(&entry) {
		deleteQuery := bson.M{"_id": entry.ID}
		err = collection.Remove(deleteQuery)
		if err != nil {
			return err
		}
	}

	if err = iter.Close(); err != nil {
		return err
	}

	return nil
}
