package mongodb

import (
	"log"
	"os"
	"testing"
	"time"

	"github.com/globalsign/mgo/bson"
	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
)

type testBlock struct {
	Category     string     `auditor:"mongodb_index"`
	Timestamp    *time.Time `auditor:"range,mongodb_index"`
	Event        string
	Hash         string `auditor:"hash"`
	PreviousHash string `auditor:"previoushash"`
}

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
	store.Save(&testBlock{Timestamp: &time1, Category: "restapi", Event: "first record updated"})
	store.Save(&testBlock{Timestamp: &time2, Category: "restapi", Event: "second record updated"})

	page1 := []testBlock{}
	err = store.Read(&page1, 1, nil)
	assert.Nil(t, err)
	assert.Equal(t, time2.UTC().String(), page1[0].Timestamp.UTC().String())
	assert.Equal(t, "second record updated", page1[0].Event)

	page2 := []testBlock{}
	err = store.Read(&page2, 1, &page1[0])
	assert.Nil(t, err)
	assert.Equal(t, time1.UTC().String(), page2[0].Timestamp.UTC().String())
	assert.Equal(t, "first record updated", page2[0].Event)

	all := []testBlock{}
	err = store.Read(&all, 2, nil)
	assert.Nil(t, err)
	assert.Equal(t, len(all), len(page1)+len(page2))
	assert.Subset(t, all, page1)
	assert.Subset(t, all, page2)

	session, err := newSession()
	assert.Nil(t, err)
	indexes, err := session.DB("audit").C("audit").Indexes()
	assert.Nil(t, err)
	// there are at minimum 3 indexes (there is a default _id_ index in addition to 2 defined in testBlock)
	// when using CosmosDB there are additional indexes prefixed DocumentDBDefaultIndex thus using greater than assertion
	assert.True(t, len(indexes) >= 3)
}

func tearDown() error {
	session, err := newSession()
	if err != nil {
		return err
	}

	collection := session.DB("audit").C("audit")

	iter := collection.Find(nil).Iter()

	var entry testBlock
	for iter.Next(&entry) {
		deleteQuery := bson.M{"hash": entry.Hash}
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
