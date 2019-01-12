#!/bin/bash

# I use relative paths so make sure we are inside tests dir
cd $(dirname "$0")

coordinator=$(docker-compose -f docker-compose-distributed-performance-tests.yml ps -q coordinator)

docker cp dynamodb-verify-integrity.js $coordinator:/

docker exec -it $coordinator node dynamodb-verify-integrity.js
