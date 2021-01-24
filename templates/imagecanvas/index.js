"use strict";

// const img = document.querySelector("[data-img-fallback]");
// if (img) {
//   const fallbackLink = img.getAttribute("data-img-fallback");
//   img.setAttribute("src", fallbackLink);
//   const canvas = document.createElement("canvas");
//   canvas.setAttribute("height", "400");
//   canvas.setAttribute("width", "400");
//   img.insertAdjacentHTML("afterend", canvas.outerHTML);
//   const ctx = canvas.getContext("2d");
//   img.addEventListener("load", function () {
//     ctx.drawImage(img, 0, 0);
//   });
// }

const img = document.createElement("img");
img.setAttribute("src", `${Env("StaticPrefix")}/templates/imagecanvas/face.jpg`);
const canvas = document.createElement("canvas");
canvas.setAttribute("height", "400");
canvas.setAttribute("width", "400");
const ctx = canvas.getContext("2d");
document.querySelector("body")?.prepend(canvas);
img.addEventListener("load", function() {
  ctx.drawImage(img, 0, 0, 200, 200);
});
