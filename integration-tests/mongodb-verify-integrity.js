var all = db.audit.count();
var current = db.audit.find({}).sort({_id: 1}).next();
var last = db.audit.find({}).sort({_id: -1}).next();
var correct = true;
var checked = 0;
while (checked < all) {
  var count = db.audit.count({previoushash: current.hash});
  if (count != 1 && !current._id.equals(last._id)) {
    print("Error in iteration " + checked + ", there are " + count + " records pointing to hash " + current.hash)
    break;
  }
  current = db.audit.findOne({previoushash: current.hash})
  checked++;
}

if (checked == all && correct) {
  print("Checked " + checked + " records and everything is fine!")
}
