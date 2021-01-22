// console.log("post-index.js");
const imgc = new ImageCropper("#imagecropper", "/static/templates/plainsimple/face.jpg");
document.querySelector("#crop")?.addEventListener("click", async function () {
  const data = await imgc.cropv2();
  console.log(data);
});
