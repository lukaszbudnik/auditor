#!/bin/bash

# I use relative paths so make sure we are inside tests dir
cd $(dirname "$0")

mongodb=$(docker-compose -f docker-compose-distributed-performance-tests.yml ps -q mongodb)

docker cp mongodb-verify-integrity.js $mongodb:/

docker exec -it $mongodb mongo 127.0.0.1/audit mongodb-verify-integrity.js
