# Auditor [![Build Status](https://www.travis-ci.org/lukaszbudnik/auditor.svg?branch=master)](https://www.travis-ci.org/lukaszbudnik/auditor)

Auditor records audit entries in a blockchain backed by DynamoDB and CosmosDB (using MongoDB API).

This is a work in progress.

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
