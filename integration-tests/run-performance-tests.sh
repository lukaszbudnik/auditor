#!/bin/bash

# I use relative paths so make sure we are inside tests dir
cd $(dirname "$0")

docker-compose -f docker-compose-distributed-performance-tests.yml up --detach --scale auditor=3 --scale tester=5

testers=$(docker-compose -f docker-compose-distributed-performance-tests.yml ps -q tester)

no_of_testers=$(echo $testers | wc -w)

while true; do
  finished=0
  for tester in $testers; do
    running=$(docker inspect -f {{.State.Running}} $tester)
    if [[ "false" == "$running" ]]; then
      echo "Tests finished"
      finished=$(($finished+1))
      continue
    fi
    echo "Tests running..."
    sleep 5
  done
  if [ $finished -eq $no_of_testers ]; then
    echo "All done"
    break
  fi
done

auditors=$(docker-compose -f docker-compose-distributed-performance-tests.yml ps -q auditor)

all=0

for auditor in $auditors; do
  count=$(docker logs $auditor 2>&1 | grep "POST /audit" | wc -l)
  echo "auditor $auditor: $count"
  all=$((all+count))
done

echo "All requests: $all"
