"use strict";

document.addEventListener("DOMContentLoaded", function main() {
  for (const img of document.querySelectorAll("img[data-pm\\.img\\.upload]")) {
    if (!img) {
      return;
    }
    initImagePicker(img);
  }
});

function initImagePicker(img) {
  const canvas = createElement("canvas", { width: img.width, height: img.height });
  const keepAspectRatio = createElement("input", {
    id: Math.random().toString(36).substring(2),
    type: "checkbox",
    style: { "margin-right": "0.5rem" },
    checked: true,
  });
  const scaleMax = 2;
  const sliderMax = 100;
  const sliderMin = 0;
  const sliderStep = (scaleMax - 1) / (sliderMax - sliderMin);
  const widthSlider = createElement("input", { type: "range", min: sliderMin, max: sliderMax, value: 0 });
  const heightSlider = createElement("input", { type: "range", min: sliderMin, max: sliderMax, value: 0 });
  const imgUpload = createElement("input", {
    type: "file",
    accept: "image/png, image/jpeg",
    style: {
      position: "absolute",
      top: "10px",
      left: "10px",
      color: "white",
      "font-family": "sans-serif",
      "text-shadow": "-1px 0 black, 0 1px black, 1px 0 black, 0 -1px black",
    },
  });
  const overlay = createElement(
    "div",
    {
      style: {
        position: "absolute",
        bottom: "10px",
        left: "10px",
        color: "white",
        "font-family": "sans-serif",
        "text-shadow": "-1px 0 black, 0 1px black, 1px 0 black, 0 -1px black",
      },
    },
    createElement("div", {}, keepAspectRatio, createElement("label", { for: keepAspectRatio.id }, "Lock aspect ratio")),
    createElement(
      "div",
      { style: { display: "flex", "align-items": "center", "justify-content": "space-between" } },
      createElement("span", { style: { "margin-right": "0.5rem" } }, "Width"),
      widthSlider,
    ),
    createElement(
      "div",
      { style: { display: "flex", "align-items": "center", "justify-content": "space-between" } },
      createElement("span", { style: { "margin-right": "0.5rem" } }, "Height"),
      heightSlider,
    ),
  );
  // const { display: imgDisplay, position: imgPosition } = window.getComputedStyle(img);
  const imgpicker = createElement(
    "div",
    {
      class: img.classList,
      style: {
        display: "inline-block",
        position: "relative",
        width: `${canvas.width}px`,
        height: `${canvas.height}px`,
      },
    },
    canvas,
    imgUpload,
    overlay,
  );
  imgpicker.classList.add("imgpicker");
  let sourceImage = document.createElement("img");
  let imageWidth = img.naturalWidth;
  let imageHeight = img.naturalHeight;
  let dragging = false;
  let outOfBoundsDragging = false;
  let destX = 0;
  let destY = 0;
  let scaleX = 1;
  let scaleY = 1;
  let lastWidthSliderValue = 0;
  let lastHeightSliderValue = 0;
  let lastMouseX, lastMouseY;
  sourceImage.src = img.src;
  sourceImage.setAttribute("data-pm.img.upload", img.getAttribute("data-pm.img.upload"));
  sourceImage.setAttribute("data-pm.img.fallback", img.getAttribute("data-pm.img.fallback"));
  sourceImage.addEventListener("load", function () {
    render();
    img.replaceWith(imgpicker);
  });
  sourceImage.addEventListener("error", function () {
    const fallbackSrc = sourceImage.getAttribute("data-pm.img.fallback");
    const fallbackImage = document.createElement("img");
    fallbackImage.src = fallbackSrc;
    fallbackImage.addEventListener("load", function () {
      sourceImage = fallbackImage;
      imageWidth = fallbackImage.naturalWidth;
      imageHeight = fallbackImage.naturalHeight;
      render();
    });
    img.replaceWith(imgpicker);
  });
  imgUpload.addEventListener("input", uploadimage);
  canvas.addEventListener("mousedown", mousedown);
  canvas.addEventListener("mousemove", mousemove);
  canvas.addEventListener("mouseup", mouseup);
  canvas.addEventListener("mouseout", mouseout);
  canvas.addEventListener("mouseenter", mouseenter);
  widthSlider.addEventListener("input", resizewidth);
  heightSlider.addEventListener("input", resizeheight);
  document.addEventListener("mouseup", function (event) {
    if (imgpicker.contains(event.target)) {
      return;
    }
    dragging = false;
    outOfBoundsDragging = false;
  });

  function createElement(tag, attributes, ...children) {
    const element = document.createElement(tag);
    for (const [attribute, value] of Object.entries(attributes)) {
      if (attribute === "style") {
        for (const [k, v] of Object.entries(value)) {
          element.style[k] = v;
        }
        continue;
      }
      element.setAttribute(attribute, value);
    }
    element.append(...children);
    return element;
  }

  function render() {
    const scaledWidth = canvas.width * scaleX;
    const scaledHeight = canvas.height * scaleY;
    const minDestX = canvas.width - scaledWidth;
    const maxDestX = 0;
    const minDestY = canvas.height - scaledHeight;
    const maxDestY = 0;
    const ctx = canvas.getContext("2d");
    if (destX < minDestX) {
      destX = minDestX;
    }
    if (destX > maxDestX) {
      destX = maxDestX;
    }
    if (destY < minDestY) {
      destY = minDestY;
    }
    if (destY > maxDestY) {
      destY = maxDestY;
    }
    ctx.clearRect(0, 0, canvas.width, canvas.height);
    ctx.drawImage(
      sourceImage, // image
      0, // sx
      0, // sy
      imageWidth, // sWidth
      imageHeight, // sHeight
      destX, // dx
      destY, // dy
      scaledWidth, // dWidth
      scaledHeight, // dHeight
    );
  }

  async function uploadimage() {
    const file = imgUpload.files[0];
    if (file === null || file === undefined) {
      return;
    }
    sourceImage = await createImageBitmap(file);
    imageWidth = sourceImage.width;
    imageHeight = sourceImage.height;
    destX = 0;
    destY = 0;
    scaleX = 1;
    scaleY = 1;
    widthSlider.value = 0;
    heightSlider.value = 0;
    render();
  }

  function resizewidth() {
    const widthSliderValue = parseInt(widthSlider.value, 10);
    const sliderDelta = widthSliderValue - lastWidthSliderValue;
    const widthDelta = canvas.width * sliderStep * sliderDelta;
    scaleX = 1 + widthSliderValue * sliderStep;
    destX -= widthDelta / 2;
    lastWidthSliderValue = widthSliderValue;
    if (keepAspectRatio.checked) {
      let heightSliderValue = parseInt(heightSlider.value, 10) + sliderDelta;
      if (heightSliderValue < sliderMin) {
        heightSliderValue = sliderMin;
      }
      if (heightSliderValue > sliderMax) {
        heightSliderValue = sliderMax;
      }
      heightSlider.value = `${heightSliderValue}`;
      const heightDelta = canvas.width * sliderStep * (heightSliderValue - lastHeightSliderValue);
      scaleY = 1 + heightSliderValue * sliderStep;
      destY -= heightDelta / 2;
      lastHeightSliderValue = heightSliderValue;
    }
    render();
  }

  function resizeheight() {
    const heightSliderValue = parseInt(heightSlider.value, 10);
    const sliderDelta = heightSliderValue - lastHeightSliderValue;
    const heightDelta = canvas.width * sliderStep * sliderDelta;
    scaleY = 1 + heightSliderValue * sliderStep;
    destY -= heightDelta / 2;
    lastHeightSliderValue = heightSliderValue;
    if (keepAspectRatio.checked) {
      let widthSliderValue = parseInt(widthSlider.value, 10) + sliderDelta;
      if (widthSliderValue < sliderMin) {
        widthSliderValue = sliderMin;
      }
      if (widthSliderValue > sliderMax) {
        widthSliderValue = sliderMax;
      }
      widthSlider.value = `${widthSliderValue}`;
      const widthDelta = canvas.width * sliderStep * (widthSliderValue - lastWidthSliderValue);
      scaleX = 1 + widthSliderValue * sliderStep;
      destX -= widthDelta / 2;
      lastWidthSliderValue = widthSliderValue;
    }
    render();
  }

  function mousedown(event) {
    const rect = canvas.getBoundingClientRect();
    lastMouseX = event.clientX - rect.left;
    lastMouseY = event.clientY - rect.top;
    dragging = true;
  }

  function mousemove(event) {
    if (!dragging) {
      return;
    }
    const rect = canvas.getBoundingClientRect();
    const mouseX = event.clientX - rect.left;
    const mouseY = event.clientY - rect.top;
    const deltaX = mouseX - lastMouseX;
    const deltaY = mouseY - lastMouseY;
    destX += deltaX;
    destY += deltaY;
    lastMouseX = mouseX;
    lastMouseY = mouseY;
    render();
  }

  function mouseup() {
    dragging = false;
  }

  function mouseout() {
    if (dragging) {
      dragging = false;
      outOfBoundsDragging = true;
    }
  }

  function mouseenter() {
    if (outOfBoundsDragging) {
      outOfBoundsDragging = false;
      dragging = true;
    }
  }
}
