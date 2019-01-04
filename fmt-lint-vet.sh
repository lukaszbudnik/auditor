#!/bin/bash

# when called with no arguments runs checks for all packages
if [[ -z "$1" ]]; then
  packages='...'
else
  packages="$1"
fi

echo "1. fmt..."
gofmt -s -w .

echo "2. lint..."
golint ./$packages

echo "3. vet..."
go vet ./$packages
