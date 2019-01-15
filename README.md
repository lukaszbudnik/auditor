# Auditor [![Build Status](https://www.travis-ci.org/lukaszbudnik/auditor.svg?branch=master)](https://www.travis-ci.org/lukaszbudnik/auditor)

Auditor records audit entries in a blockchain backed by AWS DynamoDB and Azure CosmosDB (MongoDB API).

This is an experiment I conducted to see if these backed stores (despite of their consistency models) can be used to store blockchains. Yes, I am fully aware that both AWS and Azure have managed blockchain services available. These will be my next research subject.

Anyway, this is a project I developed for fun in just a couple of nights. There are a few places that could be done a little bit better. If you would like to contribute take a look at the open issues: https://github.com/lukaszbudnik/auditor/issues.
If you see other things that could be done better don't hesitate and send me a pull requests with your changes.

# Blockchain

Auditor uses a simple blockchain implementation on top of AWS DynamoDB and Azure CosmosDB (MongoDB API). The `store.Store` interface looks like this:

```
type Store interface {
	Save(block interface{}) error
	Read(result interface{}, limit int64, last interface{}) error
	Close()
}
```

## CosmosDB/MongoDB

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
* `Read(result interface{}, limit int64, last interface{})` - reads blocks from DynamoDB and copies them to `result` which is a pointer to a slice of structs, `limit` specifies how many records to read, `last` in DynamoDB implementation is a required argument, must be a pointer to a struct of the same type as `result`, values from `last`'s fields tagged with `auditor: "dynamodb_partition"` and `auditor: "sort"` are used in DynamoDB query's _KeyConditionExpression_ and _ExclusiveStartKey_ parameters, results are sorted in descending order by setting _ScanIndexForward_ parameter to false

For usage see test: `store/dynamodb/dynamodb_test.go`.

# Configuration

auditor uses a well-known concept of `.env` files. By default auditor will look for `.env` file in the current directory. If you use a custom location/filename you need to provide it as `-configFile` command line argument.

## MongoDB

If you would like to use CosmosDB/MongoDB use this:

```
AUDITOR_STORE=mongodb
AUDITOR_REDIS=redis.endpoint.here:6379
MONGODB_USERNAME=XXX
MONGODB_PASSWORD=XXX
MONGODB_HOST=XXX.documents.azure.com:10255
MONGODB_TLS=true
```

Note:

auditor will create `audit` database and `audit` collection automatically.

## DynamoDB

If you would like to use DynamoDB use this:

```
AUDITOR_STORE=dynamodb
AUDITOR_REDIS=redis.endpoint.here:6379
AWS_REGION=us-west-2
```

By default auditor uses a credentials provider chain of: env variable provider, shared profile provider, and roles provider. Should you need it, you can explicitly set AWS API keys in configuration file too:

```
AWS_ACCESS_KEY_ID=abc
AWS_SECRET_ACCESS_KEY=def
```

Finally, you can also override the default DynamoDB endpoint:

```
AWS_DYNAMODB_ENDPOINT=http://localhost:8000
```

Note:

Creating DynamoDB tables usually requires a little bit more configuration (read/write capacity units, secondary indexes, global tables, autoscaling, etc.) and/or additional permissions (full/custom permissions). That is why auditor will not create `audit` table automatically and instead expects that this table already exists. If you would like to see a sample `audit` table definition please take a look at the `store/dynamodb/dynamodb_test.go` and the `setup()` method. You can also use AWS DynamoDB web console to create `audit` table in less than a minute.

# REST API

There is a simple HTTP server implementation provided which exposes `stores.Store` operations as REST API.

The operations are:

* POST /audit - creates new audit entry, entry is passed as JSON input, auditor will validate the JSON before processing it, for request tracing you may use optional `X-Request-Id` header
* GET /audit - reads audit entries, for request tracing you may use optional `X-Request-Id` header

The server package comes with a sample struct which looks like this (yes, a single struct can be used for both DynamoDB and MongoDB):

```
type Block struct {
	Customer     string     `auditor:"dynamodb_partition,mongodb_index"`
	Timestamp    *time.Time `auditor:"sort,mongodb_index" validate:"nonzero"`
	Category     string     `auditor:"mongodb_index"`
	Subcategory  string     `auditor:"mongodb_index"`
	Event        string     `validate:"nonzero"`
	Hash         string     `auditor:"hash"`
	PreviousHash string     `auditor:"previoushash"`
}
```

Feel free to modify it to match your requirements.

And a couple of MongoDB examples to get you started:

```
# add some audit entries, with explicit X-Request-Id headers
t1=$(date --rfc-3339=ns | tr ' ' 'T')
t2=$(date --rfc-3339=ns | tr ' ' 'T')
t3=$(date --rfc-3339=ns | tr ' ' 'T')
curl -v -X POST -H "X-Request-Id: id1" -H "Content-Type: application/json" -d "{\"Timestamp\": \"$t1\", \"Event\": \"something new - 01.01.2019\"}" http://localhost:8080/audit
curl -v -X POST -H "X-Request-Id: id2" -H "Content-Type: application/json" -d "{\"Timestamp\": \"$t2\", \"Event\": \"something new - 02.01.2019\"}" http://localhost:8080/audit
curl -v -X POST -H "X-Request-Id: id3" -H "Content-Type: application/json" -d "{\"Timestamp\": \"$t3\", \"Event\": \"something new - 03.01.2019\"}" http://localhost:8080/audit

# get audit entries, if no X-Request-Id present dynamic id is generated
curl -v http://localhost:8080/audit
# fetch all older than 2019-01-03T00:00:00.000000000+00:00 - returns 2 entries
curl -v http://localhost:8080/audit?sort=2019-01-02T00:00:00.000000000%2B00:00
# finally you may provide an optional limit parameter to limit number of returned results
curl -v http://localhost:8080/audit?limit=1
# or combined together
curl -v "http://localhost:8080/audit?sort=2019-01-02T00:00:00.000000000%2B00:00&limit=1"
```

When running AWS DynamoDB as a backend store you must provide values for the partition key of the DynamoDB table. In the sample struct there is a field called `Customer` tagged with `auditor:"dynamodb_partition"`. This means that POST JSON input must include a value for this field. Also, GET method must have a query parameter `Customer` set.

Here are some examples to get you started:

```
# add some audit entries, with explicit X-Request-Id headers
t1=$(date --rfc-3339=ns | tr ' ' 'T')
t2=$(date --rfc-3339=ns | tr ' ' 'T')
t3=$(date --rfc-3339=ns | tr ' ' 'T')
curl -v -X POST -H "X-Request-Id: id1" -H "Content-Type: application/json" -d "{\"Customer\": \"abc\", \"Timestamp\": \"$t1\", \"Event\": \"something new - 01.01.2019\"}" http://localhost:8080/audit
curl -v -X POST -H "X-Request-Id: id2" -H "Content-Type: application/json" -d "{\"Customer\": \"abc\", \"Timestamp\": \"$t2\", \"Event\": \"something new - 02.01.2019\"}" http://localhost:8080/audit
curl -v -X POST -H "X-Request-Id: id3" -H "Content-Type: application/json" -d "{\"Customer\": \"abc\", \"Timestamp\": \"$t3\", \"Event\": \"something new - 03.01.2019\"}" http://localhost:8080/audit

# get audit entries, if no X-Request-Id present dynamic id is generated
curl -v http://localhost:8080/audit?Customer=abc
# fetch all older than 2019-01-03T00:00:00.000000000+00:00 - returns 2 entries
curl -v "http://localhost:8080/audit?sort=2019-01-03T00:00:00.000000000%2B00:00&Customer=abc"
# finally you may provide an optional limit parameter to limit number of returned results
curl -v "http://localhost:8080/audit?limit=1&Customer=abc"
# or combined together
curl -v "http://localhost:8080/audit?sort=2019-01-02T00:00:00.000000000%2B00:00&limit=1&Customer=abc"
```

# Unit and integration tests

In order to execute unit and integration tests you need to setup local MongoDB, DynamoDB, and Redis containers.
There is a `docker-compose.yml` available for your convenience:

```
$ docker-compose up -d
$ ./coverage.sh
$ docker-compose down
```

# Performance tests

My implementation is based on AWS DynamoDB and MongoDB (Azure CosmosDB). Both are known for their single digit latencies. This comes at a cost of being eventually consistent.

Everything is fine when you make a reasonable number of HTTP requests. I decided to do some performance testing to see how my blockchain implementation would behave under a high load.

auditor is using a combination of local and distributed locks. Local lock is used to throttle `store.Save()` method calls, followed by distributed Redis lock. For MongoDB single distributed lock is used. For AWS DynamoDB I'm using two. Why two Redis locks? Because with only one lock I was still getting (not always) invalid blockchains (run several tests to verify that). Other solution would probably involve introducing some `time.Sleep()` to allow DynamoDB to propagate changes correctly. Anyway proper distributed synchronisation is out of the scope here. If you have a better idea how to solve it, contributions are most welcomed!

auditor's distributed locks and caching at work:

1. on the entry of `Store.Save()` method - local lock is acquired, this lock is used to throttle method calls
2. inside `Store.Save()` distributed Redis lock1 is acquired
3. inside `Store.Save()` distributed Redis lock2 is acquired [AWS DynamoDB only]
4. audit.previoushash is read from Redis, and:
    1. if empty auditor reads previous block from the backend store and uses its hash as a previous hash
    2. if found in Redis, this is the value used
5. auditor sets previous hash on current block
6. auditor computes hash and persists current block in the backend store (this takes time to propagate)
7. auditor stores current hash in Redis as key audit.previoushash
8. Redis distributed lock2 is released [AWS DynamoDB only]
9. Redis distributed lock1 is released
10. local lock is released

For running distributed simulations/performance tests there is an integration test suite in `integration-tests` directory. It comprises of the following 4 key files:

* `run-performance-tests.sh` - main script which creates the whole setup and runs the tests
* `docker-compose-distributed-performance-tests.yml` - contains the test infrastructure definition
* `dynamodb-verify-integrity.sh` - verifies if blockchain is correct in DynamoDB
* `mongodb-verify-integrity.sh` - verifies if blockchain is correct in MongoDB

By default `run-performance-tests.sh` launches 3 auditor and 5 tester containers. When launched, tester containers start making HTTP requests.

MongoDB example:

```
$ ./integration-tests/run-performance-tests.sh
Creating network "integration-tests_default" with the default driver
Creating integration-tests_coordinator_1 ... done
Creating integration-tests_dynamodb_1    ... done
Creating integration-tests_redis_1       ... done
Creating integration-tests_mongodb_1     ... done
Creating integration-tests_auditor_1     ... done
Creating integration-tests_auditor_2     ... done
Creating integration-tests_auditor_3     ... done
Creating integration-tests_tester_1      ... done
Creating integration-tests_tester_2      ... done
Creating integration-tests_tester_3      ... done
Creating integration-tests_tester_4      ... done
Creating integration-tests_tester_5      ... done
Tests running...
Tests running...
Tests running...
Tests finished
Tests finished
Tests finished
Tests finished
Tests finished
Tests finished
Tests finished
All done
auditor b7e607f208390d936da85fa0ed8d91c40c328a52d83ef9d52eecb7d445e3e39a: 164
auditor 216595c2e2f200aa79a34da225a3e436e2f550835a20469b33f44308c0a6d65b: 174
auditor a7dcd7f1888ba1d15183c0d0f152a778168024a5e17b8e183505218a17517087: 162
All requests: 500
$ ./integration-tests/mongodb-verify-integrity.sh
MongoDB shell version v4.0.4
connecting to: mongodb://127.0.0.1:27017/audit
Implicit session: session { "id" : UUID("00bc0e81-925c-4723-9f2a-7715cb8ae9ec") }
MongoDB server version: 4.0.4
Checked 500 records and everything is fine!
```

And same tests but for DynamoDB:

```
$ ./integration-tests/run-performance-tests.sh
Creating network "integration-tests_default" with the default driver
Creating integration-tests_coordinator_1 ... done
Creating integration-tests_redis_1       ... done
Creating integration-tests_dynamodb_1    ... done
Creating integration-tests_mongodb_1     ... done
Creating integration-tests_auditor_1     ... done
Creating integration-tests_auditor_2     ... done
Creating integration-tests_auditor_3     ... done
Creating integration-tests_tester_1      ... done
Creating integration-tests_tester_2      ... done
Creating integration-tests_tester_3      ... done
Creating integration-tests_tester_4      ... done
Creating integration-tests_tester_5      ... done
Tests running...
Tests running...
Tests running...
Tests running...
Tests running...
Tests running...
Tests running...
Tests running...
Tests running...
Tests finished
Tests finished
Tests finished
Tests finished
Tests finished
Tests finished
All done
auditor abbb88b8301b54bcf3760dff605239cc1d2530f19e358b9fffa31506438b3c78: 156
auditor 4fdd5fd1313f63bd840e8cbb1c86c298f8ecebd6ec0bfb86653a43117737e8ce: 175
auditor 93cf9c63d909ecbb16d0a9d981ac05459ff2e66e2cb9e481822c1407b209714d: 169
All requests: 500
$ ./integration-tests/dynamodb-verify-integrity.sh
Checked 500 records and everything is fine!
```

# License

Copyright 2018-2019 Łukasz Budnik

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
