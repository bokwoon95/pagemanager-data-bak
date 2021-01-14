const ENV = (function () {
  const error = function(name, message) {
    const err = new Error(message);
    if (name != "") {
      err.name = name;
    }
    return err;
  }
  const obj = {};
  obj["name1"] = "55";
  obj["apple"] = "jingle bells";
  obj["pear"] = "{jingle bells";
  obj["ohno"] = error("", "parse error at ...");
  obj["ohnov2"] = error("", "parse error at ...");
  return function (key) {
    const value = obj[key];
    try {
      return JSON.parse(value);
    } catch (e) {
      return value || {};
    }
  };
})();
