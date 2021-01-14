"use strict";

var data = {};
var prefix = "templates/editor/";
var templates = {
  "editor.html": ["editor.css", "editor.js", "datalist.js"],
};
Object.keys(templates).forEach(function (name) {
  var include = [];
  templates[name].forEach(function (dependency) {
    include.push(prefix + dependency);
  });
  data[prefix + name] = { include: include };
});

return data;
// return {
//   "templates/editor/editor.html": {
//     include: ["templates/editor/editor.css", "templates/editor/editor.js", "templates/editor/datalist.js"],
//   },
// };
