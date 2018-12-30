package cosmosdb

import (
	"crypto/tls"
	"net"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/lukaszbudnik/auditor/hash"
	"github.com/lukaszbudnik/auditor/store"
)

// entry represents an audit entry in CosmosDB
type entry struct {
	ID bson.ObjectId `bson:"_id,omitempty"`
	hash.Block
}

type cosmosDB struct {
	session *mgo.Session
}

func (c *cosmosDB) Save(block *hash.Block) error {
	collection := c.session.DB("audit").C("audit")

	// insert Document in collection
	if err := collection.Insert(&entry{Block: *block}); err != nil {
		return err
	}

	return nil
}

func (c *cosmosDB) Read(limit int64, lastBlock *hash.Block) ([]hash.Block, error) {
	collection := c.session.DB("audit").C("audit")

	query := bson.M{}
	if lastBlock != nil {
		query = bson.M{"block.timestamp": bson.M{"$lt": lastBlock.Timestamp}}
	}
	iter := collection.Find(query).Sort("-block.timestamp").Limit(int(limit)).Iter()

	var audit []hash.Block
	var entry entry

	for iter.Next(&entry) {
		audit = append(audit, entry.Block)
	}

	if err := iter.Close(); err != nil {
		return nil, err
	}

	return audit, nil
}

func (c *cosmosDB) Close() {
	if c.session != nil {
		c.session.Close()
	}
}

func NewCosmosDB(username, password string, addrs []string, tlsEncryption bool) (store.Store, error) {
	// DialInfo holds options for establishing a session with Azure Cosmos DB for MongoDB API account.
	dialInfo := &mgo.DialInfo{
		Database: username, // can be anything
		Username: username,
		Password: password,
		Addrs:    addrs,
		Timeout:  1 * time.Second,
	}

	if tlsEncryption {
		dialInfo.DialServer = func(addr *mgo.ServerAddr) (net.Conn, error) {
			return tls.Dial("tcp", addr.String(), &tls.Config{})
		}
	}

	// Create a session which maintains a pool of socket connections
	session, err := mgo.DialWithInfo(dialInfo)
	if err != nil {
		return nil, err
	}

	// SetSafe changes the session safety mode.
	// If the safe parameter is nil, the session is put in unsafe mode, and writes become fire-and-forget,
	// without error checking. The unsafe mode is faster since operations won't hold on waiting for a confirmation.
	// http://godoc.org/labix.org/v2/mgo#Session.SetMode.
	session.SetSafe(&mgo.Safe{})

	var cosmosDBPersister store.Store = &cosmosDB{session: session}

	return cosmosDBPersister, nil
}
