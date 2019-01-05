package mongodb

import (
	"crypto/tls"
	"fmt"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/lukaszbudnik/auditor/store"
)

type mongoDB struct {
	session *mgo.Session
}

func (c *mongoDB) Save(block interface{}) error {
	collection := c.session.DB("audit").C("audit")

	// add indexes
	t := reflect.ValueOf(block).Elem().Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		name := field.Name
		tag := field.Tag.Get("auditor")
		if strings.Contains(tag, "mongodb_index") {
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
	if err := collection.Insert(block); err != nil {
		return err
	}

	return nil
}

func (c *mongoDB) Read(result interface{}, limit int64, last interface{}) error {

	resultv := reflect.ValueOf(result)
	if resultv.Kind() != reflect.Ptr {
		panic("result argument must be a pointer to slice of struct")
	}
	slicev := resultv.Elem()
	if slicev.Kind() != reflect.Slice {
		panic("result argument must be a pointer to slice of struct")
	}
	if slicev.Type().Elem().Kind() != reflect.Struct {
		panic("result argument must be a pointer to slice of struct")
	}

	var timestampFieldName string
	// dynamic here
	for i := 0; i < slicev.Type().Elem().NumField(); i++ {
		field := slicev.Type().Elem().Field(i)
		tag := field.Tag.Get("auditor")
		if strings.Contains(tag, "mongodb_range") && field.Type == reflect.TypeOf(&time.Time{}) {
			timestampFieldName = field.Name
		}
	}

	query := bson.M{}

	if last != nil {
		lastv := reflect.ValueOf(last)
		if lastv.Kind() != reflect.Ptr {
			panic("last argument must be a pointer to struct")
		}
		if lastv.Type().Elem().Kind() != reflect.Struct {
			panic("last argument must be a pointer to struct")
		}

		if lastv.Type().Elem() != slicev.Type().Elem() {
			panic("result and last arguments must be of the same type")
		}

		// dynamic here
		field, _ := slicev.Type().Elem().FieldByName(timestampFieldName)
		tag := field.Tag.Get("auditor")
		if strings.Contains(tag, "mongodb_range") && field.Type == reflect.TypeOf(&time.Time{}) {
			timestamp := lastv.Elem().FieldByName(timestampFieldName).Interface()
			query = bson.M{strings.ToLower(timestampFieldName): bson.M{"$lt": timestamp}}
		}

	}

	collection := c.session.DB("audit").C("audit")
	return collection.Find(query).Sort(fmt.Sprintf("-%v", strings.ToLower(timestampFieldName))).Limit(int(limit)).All(result)
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

	var mongoDB store.Store = &mongoDB{session: session}
	return mongoDB, nil
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
