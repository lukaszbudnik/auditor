var AWS = require('aws-sdk');
AWS.config.update({region: 'us-west-2'});

var creds = new AWS.Credentials('abc', 'def')
var ddb = new AWS.DynamoDB({apiVersion: '2012-08-10', endpoint: 'http://dynamodb:8000', credentials: creds});

var scanParams = {
  TableName: 'audit',
  Select: 'COUNT',
 };

var firstParams = {
 ExpressionAttributeValues: {
   ':c': {S: 'abc'},
 },
 KeyConditionExpression: 'Customer = :c',
 ExpressionAttributeNames: {
   "#h": "Hash",
   "#t": "Timestamp"
 },
 ConsistentRead: true,
 ScanIndexForward: true,
 ProjectionExpression: '#h, #t, PreviousHash',
 Limit: 1,
 TableName: 'audit',
};

var lastParams = {
 ExpressionAttributeValues: {
   ':c': {S: 'abc'},
 },
 KeyConditionExpression: 'Customer = :c',
 ExpressionAttributeNames: {
   "#h": "Hash",
   "#t": "Timestamp"
 },
 ConsistentRead: true,
 ScanIndexForward: false,
 ProjectionExpression: '#h, #t, PreviousHash',
 Limit: 1,
 TableName: 'audit',
};

var queryParams = {
 ExpressionAttributeValues: {
   ':c': {S: 'abc'},
   ':p': {S: ''},
 },
 KeyConditionExpression: 'Customer = :c',
 ExpressionAttributeNames: {
   '#h': 'Hash',
   '#t': 'Timestamp'
 },
 FilterExpression: 'PreviousHash = :p',
 ConsistentRead: true,
 ProjectionExpression: '#h, #t, PreviousHash',
 Limit: 1000,
 TableName: 'audit',
};

var all = ddb.scan(scanParams).promise();
var first = ddb.query(firstParams).promise();
var last = ddb.query(lastParams).promise();

Promise.all([all, first, last]).then(function([all, first, last]) {
  verifyIntegrity(all.Count, first.Items[0], last.Items[0])
}).catch(function(err) {
  console.log("Got error: " + err)
});

function verifyIntegrity(all, first, last) {
  var previoushash = first.Hash.S;
  var checked = 1;

  queryParams.ExpressionAttributeValues[':p'].S = previoushash;

  ddb.query(queryParams, function loop(err, data) {
    if (err) {
      console.log("Error query DynamoDB: " + err);
    } else {
      var count = data.Items.length;
      if (data.Items.length != 1) {
        console.log("Error in iteration " + checked + ", there are " + count + " records pointing to hash " + previoushash)
        return
      }

      previoushash = data.Items[0].Hash.S
      checked++;

      if (previoushash != last.Hash.S) {
        queryParams.ExpressionAttributeValues[':p'].S = previoushash;
        ddb.query(queryParams, loop);
      } else {
        console.log("Checked " + checked + " records and everything is fine!")
      }
    }
  });

}
