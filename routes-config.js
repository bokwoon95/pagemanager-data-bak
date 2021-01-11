"use strict";

route("/hellojs", function (req, res) {
  var age = req.query.age;
  res.send("Hello World! Your age is " + age);
});
