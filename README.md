# Auditor [![Build Status](https://www.travis-ci.org/lukaszbudnik/auditor.svg?branch=master)](https://www.travis-ci.org/lukaszbudnik/auditor)

Auditor records audit entries in a blockchain backed by AWS DynamoDB and Azure CosmosDB (MongoDB API).

This is a work in progress.

# Blockchain

Auditor uses simple blockchain implementation on top of AWS DynamoDB and Azure CosmosDB (MongoDB API). The `store.Store` interface looks like this:

```
type Store interface {
	Save(block interface{}) error
	Read(result interface{}, limit int64, last interface{}) error
	Close()
}
```

The interface is generic and operates on well known `interface{}` constructs.

## MongoDB

For MongoDB a simple block struct could look like this:

```
type Block struct {
	Timestamp    *time.Time `auditor:"sort,mongodb_index" validate:"nonzero"`
	Category     string     `auditor:"mongodb_index"`
	Event        string     `validate:"nonzero"`
	Hash         string     `auditor:"hash"`
	PreviousHash string     `auditor:"previoushash"`
}
```

Such struct has:

* [required] string field tagged with `auditor:"hash"` - used for storing block hash
* [required] string field tagged with `auditor:"previoushash"` - used for storing previous block hash
* [required] time field tagged with `auditor:"sort"` - used for viewing/paging blocks
* [optional] any field can have `mongodb_index` added to auditor tag for example `auditor:"sort,mongodb_index"` - used for ensuring collection indexes
* [optional] if you want to have access to native `_id` column add field: `` ID bson.ObjectId bson:"_id,omitempty"` ``

MongoDB implementation works like this:

* `Save(block interface{})` - accepts a pointer to struct and saves it in MongoDB, before saving computes hash and sets previous hash values, also ensures that all relevant indexes are created
* `Read(result interface{}, limit int64, last interface{})` - reads blocks from MongoDB and copies them to `result` which is a pointer to a slice of structs, `limit` specifies how many records to read, `last` is an optional argument, must be a pointer to a struct of the same type as `result`, `last` is used for paging, the field tagged with `auditor: "sort"` is used in MongoDB's less than query: `{field: {$lt: value} }`, results are sorted by the same field in descending order `{$sort: {field: -1}}`

For usage see test: `store/mongodb/mongodb_test.go`.

## DynamoDB

For DynamoDB a simple block struct could look like this:

```
type Block struct {
	Customer     string     `auditor:"dynamodb_partition"`
	Timestamp    *time.Time `auditor:"sort" validate:"nonzero"`
	Category     string
	Event        string     `validate:"nonzero"`
	Hash         string     `auditor:"hash"`
	PreviousHash string     `auditor:"previoushash"`
}
```

Such struct has:

* [required] string field tagged with `auditor:"hash"` - used for storing block hash
* [required] string field tagged with `auditor:"previoushash"` - used for storing previous block hash
* [required] string field tagged with `auditor:"dynamodb_partition"` - used as partition key of DynamoDB primary key, used for viewing/paging blocks
* [required] time field tagged with `auditor:"sort"` - used as a sort key of DynamoDB primary key, used for viewing/paging blocks

DynamoDB implementation works like this:

* `Save(block interface{})` - accepts a pointer to struct and saves it in DynamoDB, before saving computes hash and sets previous hash values
* `Read(result interface{}, limit int64, last interface{})` - reads blocks from DynamoDB and copies them to `result` which is a pointer to a slice of structs, `limit` specifies how many records to read, `last` in DynamoDB implementation is a required argument, must be a pointer to a struct of the same type as `result`, values from `last` fields tagged with `auditor: "dynamodb_partition"` and `auditor: "sort"` are used in DynamoDB query's _KeyConditionExpression_ and _ExclusiveStartKey_ parameters, results are sorted in descending order by setting _ScanIndexForward_ parameter to false

For usage see test: `store/dynamodb/dynamodb_test.go`.

# REST API

```
curl -v -X POST -H "Content-Type: application/json" -d "{\"Timestamp\": \"2019-01-01T12:39:01.999999999+01:00\", \"Event\": \"something new\"}" http://localhost:8080/audit
```

# Executing tests

In order to execute tests you need to setup local MongoDB and DynamoDB containers.
There is a `docker-compose.yml` available for your convenience:

```
$ docker-compose up -d
$ ./coverage.sh
$ docker-compose down
```

# License

Copyright 2018-2019 Łukasz Budnik

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
