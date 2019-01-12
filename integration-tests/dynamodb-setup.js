var AWS = require('aws-sdk');
AWS.config.update({region: 'us-west-2'});

var creds = new AWS.Credentials('abc', 'def')
var ddb = new AWS.DynamoDB({apiVersion: '2012-08-10', endpoint: 'http://dynamodb:8000', credentials: creds});

var params = {
  AttributeDefinitions: [
    {
      AttributeName: 'Customer',
      AttributeType: 'S'
    },
    {
      AttributeName: 'Timestamp',
      AttributeType: 'S'
    }
  ],
  KeySchema: [
    {
      AttributeName: 'Customer',
      KeyType: 'HASH'
    },
    {
      AttributeName: 'Timestamp',
      KeyType: 'RANGE'
    }
  ],
  ProvisionedThroughput: {
    ReadCapacityUnits: 200,
    WriteCapacityUnits: 500
  },
  TableName: 'audit'
};

ddb.createTable(params, function(err, data) {
  if (err) {
    console.log("Error", err);
  } else {
    console.log("Success", data);
  }
});
