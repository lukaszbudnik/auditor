# Auditor

Auditor records audit entries in a blockchain backed by DynamoDB and CosmosDB.

This is a work in progress.

# Executing tests

In order to execute tests you need to setup local MongoDB and DynamoDB containers.
There is a `docker-compose.yml` available for your convenience:

```
$ docker-compose up -d
$ ./coverage.sh
$ docker-compose down
```
