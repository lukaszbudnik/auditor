#!/bin/sh

# docker-compose uses depends and continues when container is running
# but we also need to give DynamoDB and MongoDB some time to init properly
sleep 5

# 10K
DEFAULT_NO_OF_TESTS=10000

# if auditor config file is not provided explicitly fallback to default one
if [ -z "$NO_OF_TESTS" ]; then
  NO_OF_TESTS=$DEFAULT_NO_OF_TESTS
fi

pids=""
count=0
for i in $(seq 1 1 $NO_OF_TESTS)
do
  t=$(date --rfc-3339=ns | tr ' ' 'T')
  curl -X POST -H "Content-Type: application/json" -d "{\"Customer\": \"abc\", \"Timestamp\": \"$t\", \"Event\": \"something new - $count\"}" http://auditor:8080/audit
  pids="$pids $!"
  count=$((count+1))
done

for pid in $pids; do
    wait $pid
done
