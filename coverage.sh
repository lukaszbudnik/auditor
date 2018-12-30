#!/bin/bash

# when called with no arguments calls tests for all packages
if [[ -z "$1" ]]; then
  packages='...'
else
  packages="$1"
fi

go test -race -covermode=atomic -coverprofile=coverage.txt ./$packages
if [[ $? -ne 0 ]]; then
  fail=1
fi

exit $fail
