package mongodb

import (
	"crypto/tls"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/lukaszbudnik/auditor/model"
	"github.com/lukaszbudnik/auditor/store"
)

// entry represents an audit entry in MongoDB
type entry struct {
	ID bson.ObjectId `bson:"_id,omitempty"`
	model.Block
}

type mongoDB struct {
	session *mgo.Session
}

func (c *mongoDB) Save(block *model.Block) error {
	collection := c.session.DB("audit").C("audit")

	// add indexes
	t := reflect.TypeOf(*block)
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name := field.Name
		tag := field.Tag.Get("auditor")
		if tag == "index" {
			index := mgo.Index{
				Key:        []string{name},
				Background: true,
			}
			if err := collection.EnsureIndex(index); err != nil {
				return err
			}
		}
	}

	// insert Document in collection
	if err := collection.Insert(&entry{Block: *block}); err != nil {
		return err
	}

	return nil
}

func (c *mongoDB) Read(limit int64, lastBlock *model.Block) ([]model.Block, error) {
	collection := c.session.DB("audit").C("audit")

	query := bson.M{}
	if lastBlock != nil {
		query = bson.M{"block.timestamp": bson.M{"$lt": lastBlock.Timestamp}}
	}
	iter := collection.Find(query).Sort("-block.timestamp").Limit(int(limit)).Iter()

	var audit []model.Block
	var entry entry

	for iter.Next(&entry) {
		audit = append(audit, entry.Block)
	}

	if err := iter.Close(); err != nil {
		return nil, err
	}

	return audit, nil
}

func (c *mongoDB) Close() {
	if c.session != nil {
		c.session.Close()
	}
}

// New creates Store implementation for MongoDB
func New() (store.Store, error) {
	session, err := newSession()
	if err != nil {
		return nil, err
	}

	var mongoDBPersister store.Store = &mongoDB{session: session}
	return mongoDBPersister, nil
}

func newSession() (*mgo.Session, error) {
	username := os.Getenv("MONGODB_USERNAME")
	password := os.Getenv("MONGODB_PASSWORD")
	addrs := strings.Split(os.Getenv("MONGODB_HOST"), ",")
	tlsEncryption, err := strconv.ParseBool(os.Getenv("MONGODB_TLS"))
	// by default we are secure set tls
	if err != nil {
		tlsEncryption = true
	}
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

	return session, nil
}
