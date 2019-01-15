package mongodb

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/bsm/redis-lock"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/go-redis/redis"
	"github.com/lukaszbudnik/auditor/model"
	"github.com/lukaszbudnik/auditor/store"
)

type mongoDB struct {
	session *mgo.Session
	redis   *redis.Client
	lock    *sync.Mutex
	lock1   *lock.Locker
}

func (m *mongoDB) Save(block interface{}) error {
	m.lock.Lock()
	defer m.lock.Unlock()

	collection := m.session.DB("audit").C("audit")

	indexFields := model.GetFieldsTaggedWith(block, "mongodb_index")
	for _, field := range indexFields {
		name := field.Name
		index := mgo.Index{
			Key:        []string{name},
			Background: true,
		}
		if err := collection.EnsureIndex(index); err != nil {
			return err
		}
	}

	_, err := m.lock1.Lock()
	if err != nil {
		log.Printf("ERROR Could not acquire distributed lock1: %v", err.Error())
		return err
	}
	defer m.lock1.Unlock()

	previousHash, err := m.redis.Get("auditor.previoushash").Result()
	if err != nil && err != redis.Nil {
		log.Printf("ERROR Could not get previoushash key from Redis: %v", err.Error())
		return err
	}

	if len(previousHash) > 0 {
		previousHashField := model.GetFieldsTaggedWith(block, "previoushash")
		model.SetFieldValue(block, previousHashField[0], previousHash)
	} else {
		// get type
		t := reflect.ValueOf(block).Elem().Type()
		// create *[]type
		ts := reflect.SliceOf(t)
		ptr := reflect.New(ts)
		ptr.Elem().Set(reflect.MakeSlice(ts, 0, 1))

		m.Read(ptr.Interface(), 1, nil)
		if ptr.Elem().Len() > 0 {
			model.SetPreviousHash(block, ptr.Elem().Index(0).Addr().Interface())
		}
	}
	currentHash, err := model.ComputeAndSetHash(block)
	if err != nil {
		return err
	}

	if err := collection.Insert(block); err != nil {
		return err
	}

	m.redis.Set("auditor.previoushash", currentHash, time.Second)

	return nil
}

func (m *mongoDB) Read(result interface{}, limit int64, last interface{}) error {

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

	query := bson.M{}

	sortFields := model.GetTypeFieldsTaggedWith(slicev.Type().Elem(), "sort")
	sortField := sortFields[0]

	lastv := reflect.ValueOf(last)
	if last != nil && !lastv.IsNil() {
		if lastv.Kind() != reflect.Ptr {
			panic("last argument must be a pointer to struct")
		}
		if lastv.Type().Elem().Kind() != reflect.Struct {
			panic("last argument must be a pointer to struct")
		}

		if lastv.Type().Elem() != slicev.Type().Elem() {
			panic("result and last arguments must be of the same type")
		}

		timestamp := lastv.Elem().FieldByName(sortField.Name).Interface()
		timestampv := reflect.ValueOf(timestamp)
		if !timestampv.IsNil() {
			query = bson.M{strings.ToLower(sortField.Name): bson.M{"$lt": timestamp}}
		}
	}

	collection := m.session.DB("audit").C("audit")
	return collection.Find(query).Sort(fmt.Sprintf("-%v", strings.ToLower(sortField.Name))).Limit(int(limit)).All(result)
}

func (m *mongoDB) Close() {
	if m.session != nil {
		m.session.Close()
	}
}

// New creates Store implementation for MongoDB
func New() (store.Store, error) {
	session, err := newSession()
	if err != nil {
		return nil, err
	}

	redisEndpoint := os.Getenv("AUDITOR_REDIS")
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "localhost"
	}
	token := fmt.Sprintf("%v-%v", hostname, os.Getpid())

	redis := redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    redisEndpoint,
	})

	lock1 := lock.New(redis, "auditor.lock1", &lock.Options{
		RetryCount:  10,
		TokenPrefix: token,
	})

	var mongoDB store.Store = &mongoDB{session: session, redis: redis, lock: &sync.Mutex{}, lock1: lock1}
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

	session, err := mgo.DialWithInfo(dialInfo)
	if err != nil {
		return nil, err
	}

	session.SetSafe(&mgo.Safe{J: true})
	session.SetMode(mgo.Primary, true)

	return session, nil
}
