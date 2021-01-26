var canvas = document.getElementById("canvas");
var ctx = canvas.getContext("2d");
ctx.strokeStyle = "red";
ctx.lineWidth = 5;
var rect = canvas.getBoundingClientRect();
var offsetX = rect.left + document.body.scrollLeft;
var offsetY = rect.top + document.body.scrollTop;
var lastX = 0;
var lastY = 0;
var panX = 0;
var panY = 0;
var dragging = [];
var isDown = false;
// create green & pink "images" (we just use rects instead of images)
var images = [];
images.push({ x: 200, y: 150, width: 25, height: 25, color: "green" });
images.push({ x: 80, y: 235, width: 25, height: 25, color: "pink" });
// load the tiger image
var tiger = new Image();
tiger.onload = function () {
  draw();
};
tiger.src = "/static/templates/imagecanvas/face.jpg";
function draw() {
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  // draw tiger
  ctx.globalAlpha = 0.25;
  ctx.drawImage(tiger, panX, panY, tiger.width, tiger.height);
  // draw color images
  ctx.globalAlpha = 1.0;
  for (var i = 0; i < images.length; i++) {
    var img = images[i];
    ctx.beginPath();
    ctx.rect(img.x + panX, img.y + panY, img.width, img.height);
    ctx.fillStyle = img.color;
    ctx.fill();
    ctx.stroke();
  }
}
// create an array of any "hit" colored-images
function imagesHitTests(x, y) {
  // adjust for panning
  x -= panX;
  y -= panY;
  // create var to hold any hits
  var hits = [];
  // hit-test each image
  // add hits to hits[]
  for (var i = 0; i < images.length; i++) {
    var img = images[i];
    if (x > img.x && x < img.x + img.width && y > img.y && y < img.y + img.height) {
      hits.push(i);
    }
  }
  return hits;
}

canvas.addEventListener("mousedown", function (e) {
  // get mouse coordinates
  var mouseX = parseInt(e.clientX - offsetX);
  var mouseY = parseInt(e.clientY - offsetY);
  // set the starting drag position
  lastX = mouseX;
  lastY = mouseY;
  // test if we're over any of the images
  dragging = imagesHitTests(mouseX, mouseY);
  // set the dragging flag
  isDown = true;
});
canvas.addEventListener("mousemove", function (e) {
  // if we're not dragging, exit
  if (!isDown) {
    return;
  }
  //get mouse coordinates
  var mouseX = parseInt(e.clientX - offsetX);
  var mouseY = parseInt(e.clientY - offsetY);
  // calc how much the mouse has moved since we were last here
  var dx = mouseX - lastX;
  var dy = mouseY - lastY;
  // set the lastXY for next time we're here
  lastX = mouseX;
  lastY = mouseY;
  // handle drags/pans
  if (dragging.length > 0) {
    // we're dragging images
    // move all affected images by how much the mouse has moved
    for (var i = 0; i < dragging.length; i++) {
      var img = images[dragging[i]];
      img.x += dx;
      img.y += dy;
    }
  } else {
    // we're panning the tiger
    // set the panXY by how much the mouse has moved
    panX += dx;
    panY += dy;
  }
  draw();
});
canvas.addEventListener("mouseup", function () {
  isDown = false;
});
canvas.addEventListener("mouseout", function () {
  isDown = false;
});
