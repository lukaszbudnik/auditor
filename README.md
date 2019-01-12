# Auditor [![Build Status](https://www.travis-ci.org/lukaszbudnik/auditor.svg?branch=master)](https://www.travis-ci.org/lukaszbudnik/auditor)

Auditor records audit entries in a blockchain backed by AWS DynamoDB and Azure CosmosDB (MongoDB API).

This is an experiment I conducted to see if these backed stores (despite of their consistency models) can be used to store blockchains. Yes, I am fully aware that both AWS and Azure have managed blockchain services available. These will be my next research subject.

Please also see known issues section at the bottom of this document.

Anyway, this is a project I developed for fun in just a couple of nights. There are a few places that could be done a little bit better. If you would like to contribute take a look at the open issues: https://github.com/lukaszbudnik/auditor/issues.

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

auditor uses a well-known concept of `.env` files. By default auditor will look for `.env` file in the current directory. If you use a custom location/filename please provide a path in `-configFile` command line argument.

## MongoDB

If you would like to use CosmosDB/MongoDB use this:

```
AUDITOR_STORE=mongodb
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

When running DynamoDB as a backend store you must provide values for the partition key of the DynamoDB table. In the sample struct there is a field called `Customer` tagged with `auditor:"dynamodb_partition"`. This means that POST JSON input must include a value for this field. Also, GET method must have a query parameter `Customer` set.

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

# Executing tests

In order to execute tests you need to setup local MongoDB and DynamoDB containers.
There is a `docker-compose.yml` available for your convenience:

```
$ docker-compose up -d
$ ./coverage.sh
$ docker-compose down
```

# Known issues

Currently (and as per created issues) auditor is using local locks which in distributed environment do not guarantee any integrity at all. Further, it even turned out that under a high load blockchain integrity is not guaranteed when running on a single auditor instance. This can be due to consistency limitations of backend DBs, using local containers for tests rather than fine-tuned production-grade servers (local DynamoDB has a limit of 5 for both read and write capacities - which is a very low value), or... a bogus implementation of auditor (but I will get to the bottom of it!).

For running distributed simulations/performance tests there is an integration test suite in `integration-tests` directory. It comprises of the following 4 key files:

* `run-performance-tests.sh` - main script which creates the whole setup and runs the tests
* `docker-compose-distributed-performance-tests.yml` - contains the test infrastructure definition
* `dynamodb-verify-integrity.sh` - verifies if blockchain is correct in DynamoDB
* `mongodb-verify-integrity.sh` - verifies if blockchain is correct in MongoDB

By default `run-performance-tests.sh` launches 1 auditor and 1 tester container. When launched, tester container starts making HTTP requests.

MongoDB example:

```
$ ./integration-tests/run-performance-tests.sh
Creating network "integration-tests_default" with the default driver
Creating integration-tests_coordinator_1 ... done
Creating integration-tests_dynamodb_1    ... done
Creating integration-tests_mongodb_1     ... done
Creating integration-tests_auditor_1     ... done
Creating integration-tests_tester_1      ... done
Tests running...
Tests running...
Tests finshed
All done
auditor f47f8ddf25a0781941658fde7eec46ca4203b35f0f72c8a2a6ef9729d2fa3569: 100
All requests: 100
$ ./integration-tests/mongodb-verify-integrity.sh
MongoDB shell version v4.0.4
connecting to: mongodb://127.0.0.1:27017/audit
Implicit session: session { "id" : UUID("663cfa26-7cf8-47fe-a275-049bb04d743f") }
MongoDB server version: 4.0.4
Checked 100 records and everything is fine!
```

In the above setup everything is fine.

To simulate distributed environments modify `--scale` parameters inside `run-performance-tests.sh`. When running multiple auditor containers Docker's DNS service kicks in and provides a simple load balancing so these all instances receive more or less same number of requests.

For example here is an output for 2 auditors and 3 testers and a failed verification:

```
$ ./integration-tests/run-performance-tests.sh
Creating network "integration-tests_default" with the default driver
Creating integration-tests_coordinator_1 ... done
Creating integration-tests_dynamodb_1    ... done
Creating integration-tests_mongodb_1     ... done
Creating integration-tests_auditor_1     ... done
Creating integration-tests_auditor_2     ... done
Creating integration-tests_tester_1      ... done
Creating integration-tests_tester_2      ... done
Creating integration-tests_tester_3      ... done
Tests running...
Tests running...
Tests finshed
Tests finshed
Tests finshed
Tests finshed
All done
auditor 32c18d988b283122c8bb68dd2e874933be673bef56b240cf787679b5b9d8abf9: 150
auditor 574057729ab06ef2bb177fd0389dc98ea513edfadfc00db9905961edb44ab6ba: 150
All requests: 300
$ ./integration-tests/mongodb-verify-integrity.sh
MongoDB shell version v4.0.4
connecting to: mongodb://127.0.0.1:27017/audit
Implicit session: session { "id" : UUID("02e5d606-6e9b-47b0-901a-5d1502c5f704") }
MongoDB server version: 4.0.4
Error in iteration 7: there are 3 records pointing to hash 171c61acf289d8cb7ae269ee0d3352ee2d125ae651a0589bfbe5ff5d53416ea6
```

As you can see in distributed environment blockchain is invalid.

And same tests but for DynamoDB. 1 instance of auditor and tester is fine:

```
$ ./integration-tests/run-performance-tests.sh
Creating network "integration-tests_default" with the default driver
Creating integration-tests_coordinator_1 ... done
Creating integration-tests_mongodb_1     ... done
Creating integration-tests_dynamodb_1    ... done
Creating integration-tests_auditor_1     ... done
Creating integration-tests_tester_1      ... done
Tests running...
Tests running...
Tests finshed
All done
auditor 0504c423dd35418cd96c6e88d2e8042658767667f244fd32ad7f1630dd255f06: 100
All requests: 100
$ ./integration-tests/dynamodb-verify-integrity.sh
Checked 100 records and everything is fine!
```

And 2 auditors and 3 testers result in an invalid blockchain:

```
$ ./integration-tests/run-performance-tests.sh
Creating network "integration-tests_default" with the default driver
Creating integration-tests_coordinator_1 ... done
Creating integration-tests_mongodb_1     ... done
Creating integration-tests_dynamodb_1    ... done
Creating integration-tests_auditor_1     ... done
Creating integration-tests_auditor_2     ... done
Creating integration-tests_tester_1      ... done
Creating integration-tests_tester_2      ... done
Creating integration-tests_tester_3      ... done
Tests running...
Tests running...
Tests running...
Tests finshed
Tests finshed
Tests finshed
All done
auditor a1d06f2f3a0d08678de2d52f71cf82d4eccbcd5c3a3648f2de36bec06cf29d5e: 148
auditor ae4758bdc7fc61f6bf86600917681bcbf5bb226274ae37d36d3e0de90a2a1a92: 152
All requests: 300
$ ./integration-tests/dynamodb-verify-integrity.sh
Error in iteration 3, there are 0 records pointing to hash ed6805b4da5248d02f9b01e4e7608051e29b50cd7dbd1ba9ce0a1ae9351bf78e
```

# License

Copyright 2018-2019 Łukasz Budnik

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
