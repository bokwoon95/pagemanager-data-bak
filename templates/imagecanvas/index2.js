"use strict";

document.addEventListener("DOMContentLoaded", function main() {
  const img = document.querySelector("img[data-img-fallback]");
  if (!img) {
    return;
  }
  initImagePicker(img);

  function initImagePicker(img, opts) {
    if (opts === null || opts === undefined) {
      opts = {};
    }
    const imgstyle = window.getComputedStyle(img);
    const canvas = createElement("canvas", { class: "ba b--dark-red", width: imgstyle.width, height: imgstyle.height });
    const overlay = newOverlay();
    const imgpicker = createElement(
      "div",
      {
        class: "imgpicker",
        style: {
          display: "inline-block",
          position: "relative",
          width: `${imgstyle.width}px`,
          height: `${imgstyle.height}px`,
        },
      },
      canvas,
      overlay,
    );
    Object.assign(imgpicker, {
      canvas: canvas,
      overlay: overlay,
      dragging: false, // track dragging inside of imgpicker
      outOfBoundsDragging: false, // track dragging outside of imgpicker
      img: img, // source img
      imgWidth: null, // source img's original width (will be set later)
      imgHeight: null, // source img's original height (will be set later)
      destX: 0, // destination x-coordinate on canvas
      destY: 0, // destination y-coordinate on canvas
      widthSliderValue: 0,
      heightSliderValue: 0,
      scaleX: 1, // horizontal scaling factor
      scaleY: 1, // vertical scaling factor
      lastMouseX: 0, // last x-coordinate of mouse in imgpicker
      lastMouseY: 0, // last y-coordinate of mouse in imgpicker
    });
    Object.seal(imgpicker);
    canvas.addEventListener("mousedown", mousedown(imgpicker));
    canvas.addEventListener("mousemove", mousemove(imgpicker));
    canvas.addEventListener("mouseup", mouseup(imgpicker));
    canvas.addEventListener("mouseout", mouseout(imgpicker));
    canvas.addEventListener("mouseenter", mouseenter(imgpicker));
    imgpicker.overlay.widthSlider.addEventListener("input", resizewidth(imgpicker));
    imgpicker.overlay.heightSlider.addEventListener("input", resizeheight(imgpicker));
    document.addEventListener("mouseup", function (event) {
      if (imgpicker.contains(event.target)) {
        return;
      }
      imgpicker.dragging = false;
      imgpicker.outOfBoundsDragging = false;
    });
    {
      // initial render shenanigans
      const img2 = document.createElement("img");
      img2.addEventListener("load", function () {
        imgpicker.img = img2;
        imgpicker.imgWidth = img2.naturalWidth;
        imgpicker.imgHeight = img2.naturalHeight;
        render(imgpicker);
        const replaceimg = opts.replaceimg !== undefined ? opts.replaceimg : true;
        if (replaceimg) {
          img.replaceWith(imgpicker);
        }
      });
      img2.src = img.src;
    }
    return imgpicker;
  }

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

  function newOverlay() {
    const checkboxID = Math.random().toString(36).substring(2);
    const checkbox = createElement("input", {
      id: checkboxID,
      type: "checkbox",
      style: { "margin-right": "0.5rem" },
      checked: true,
    });
    const widthSlider = createElement("input", { type: "range", min: 0, max: 100 });
    const heightSlider = createElement("input", { type: "range", min: 0, max: 100 });
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
      createElement("div", {}, checkbox, createElement("label", { for: checkboxID }, "Lock aspect ratio")),
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
    overlay.checkbox = checkbox;
    overlay.widthSlider = widthSlider;
    overlay.heightSlider = heightSlider;
    return overlay;
  }

  function render(imgpicker) {
    window.imgpicker = imgpicker;
    if (imgpicker.destX > 0) {
      imgpicker.destX = 0;
    }
    if (imgpicker.destX + imgpicker.canvas.width * imgpicker.scaleX < imgpicker.canvas.width) {
      imgpicker.destX = imgpicker.canvas.width - imgpicker.canvas.width * imgpicker.scaleX;
    }
    if (imgpicker.destY > 0) {
      imgpicker.destY = 0;
    }
    if (imgpicker.destY + imgpicker.canvas.height * imgpicker.scaleY < imgpicker.canvas.height) {
      imgpicker.destY = imgpicker.canvas.height - imgpicker.canvas.height * imgpicker.scaleY;
    }
    const ctx = imgpicker.canvas.getContext("2d");
    if (imgpicker.overlay.widthSlider.value !== `${imgpicker.widthSliderValue}`) {
      imgpicker.overlay.widthSlider.value = `${imgpicker.widthSliderValue}`;
    }
    if (imgpicker.overlay.heightSlider.value !== `${imgpicker.heightSliderValue}`) {
      imgpicker.overlay.heightSlider.value = `${imgpicker.heightSliderValue}`;
    }
    ctx.clearRect(0, 0, imgpicker.canvas.width, imgpicker.canvas.height);
    ctx.drawImage(
      imgpicker.img, // img
      0, // sx
      0, // sy
      imgpicker.imgWidth, // sWidth
      imgpicker.imgHeight, // sHeight
      imgpicker.destX, // dx
      imgpicker.destY, // dy
      imgpicker.canvas.width * imgpicker.scaleX, // dWidth
      imgpicker.canvas.height * imgpicker.scaleY, // dHeight
    );
  }

  function resizewidth(imgpicker) {
    return function () {
      const prevScaleX = imgpicker.scaleX;
      const prevScaleY = imgpicker.scaleY;
      const input = imgpicker.overlay.widthSlider;
      const value = parseInt(input.value, 10);
      if (isNaN(value)) {
        throw new Error(`value (${input.value}) is not a number`);
      }
      const min = parseInt(input.min, 10);
      const max = parseInt(input.max, 10);
      const range = max - min;
      if (isNaN(range)) {
        throw new Error(`max (${input.max}) or min (${input.min}) is not a number`);
      }
      const scaleMax = 2;
      const unit = (scaleMax - 1) / range;
      if (value <= min) {
        imgpicker.scaleX = 1;
        if (imgpicker.overlay.checkbox.checked) {
          imgpicker.scaleY = 1;
        }
      } else {
        imgpicker.scaleX = 1 + Math.abs(value * unit);
        if (imgpicker.overlay.checkbox.checked) {
          imgpicker.scaleY = 1 + Math.abs(value * unit);
        }
      }
      const deltaX = (imgpicker.scaleX - prevScaleX) * imgpicker.canvas.width;
      const deltaY = (imgpicker.scaleY - prevScaleY) * imgpicker.canvas.height;
      imgpicker.destX -= deltaX / 2;
      if (imgpicker.overlay.checkbox.checked) {
        imgpicker.destY -= deltaY / 2;
      }
      const prevWidthSliderValue = imgpicker.widthSliderValue;
      imgpicker.widthSliderValue = value;
      if (imgpicker.overlay.checkbox.checked) {
        const delta = imgpicker.widthSliderValue - prevWidthSliderValue;
        imgpicker.heightSliderValue += delta;
      }
      render(imgpicker);
    };
  }

  function resizeheight(imgpicker) {
    return function () {
      const prevScaleX = imgpicker.scaleX;
      const prevScaleY = imgpicker.scaleY;
      const input = imgpicker.overlay.heightSlider;
      imgpicker.widthSliderValue = input.value;
      imgpicker.heightSliderValue = input.value;
      const value = parseInt(input.value, 10);
      if (isNaN(value)) {
        throw new Error(`value (${input.value}) is not a number`);
      }
      const min = parseInt(input.min, 10);
      const max = parseInt(input.max, 10);
      const range = max - min;
      if (isNaN(range)) {
        throw new Error(`max (${input.max}) or min (${input.min}) is not a number`);
      }
      const scaleMax = 2;
      const unit = (scaleMax - 1) / range;
      if (value <= min) {
        imgpicker.scaleX = 1;
        imgpicker.scaleY = 1;
      } else {
        imgpicker.scaleX = 1 + Math.abs(value * unit);
        imgpicker.scaleY = 1 + Math.abs(value * unit);
      }
      const deltaX = (imgpicker.scaleX - prevScaleX) * imgpicker.canvas.width;
      const deltaY = (imgpicker.scaleY - prevScaleY) * imgpicker.canvas.height;
      imgpicker.destX -= deltaX / 2;
      imgpicker.destY -= deltaY / 2;
      render(imgpicker);
    };
  }

  function mousedown(imgpicker) {
    return function (event) {
      const rect = event.currentTarget.getBoundingClientRect();
      const mouseX = event.clientX - rect.left;
      const mouseY = event.clientY - rect.top;
      imgpicker.lastMouseX = mouseX;
      imgpicker.lastMouseY = mouseY;
      imgpicker.dragging = true;
      render(imgpicker);
    };
  }

  function mousemove(imgpicker) {
    return function (event) {
      if (!imgpicker.dragging) {
        return;
      }
      const rect = event.currentTarget.getBoundingClientRect();
      const mouseX = event.clientX - rect.left;
      const mouseY = event.clientY - rect.top;
      const deltaX = mouseX - imgpicker.lastMouseX;
      const deltaY = mouseY - imgpicker.lastMouseY;
      {
        // NOTE: don't really understand the shenanigans here, arrived at it through trial-and-error
        const withinTopBorder = imgpicker.destY + deltaY < 0;
        const withinBottomBorder =
          imgpicker.canvas.height + Math.abs(imgpicker.destY + deltaY) < imgpicker.canvas.height * imgpicker.scaleY;
        const withinRightBorder =
          imgpicker.canvas.width + Math.abs(imgpicker.destX + deltaX) < imgpicker.canvas.width * imgpicker.scaleX;
        const withinLeftBorder = imgpicker.destX + deltaX < 0;
        if (withinLeftBorder && withinRightBorder) {
          imgpicker.lastMouseX = mouseX;
          imgpicker.destX += deltaX;
        }
        if (withinTopBorder && withinBottomBorder) {
          imgpicker.lastMouseY = mouseY;
          imgpicker.destY += deltaY;
        }
      }
      render(imgpicker);
    };
  }

  function mouseup(imgpicker) {
    return function () {
      imgpicker.dragging = false;
      render(imgpicker);
    };
  }

  function mouseout(imgpicker) {
    return function () {
      if (imgpicker.dragging) {
        imgpicker.dragging = false;
        imgpicker.outOfBoundsDragging = true;
      }
    };
  }

  function mouseenter(imgpicker) {
    return function () {
      if (imgpicker.outOfBoundsDragging) {
        imgpicker.outOfBoundsDragging = false;
        imgpicker.dragging = true;
        render(imgpicker);
      }
    };
  }
});
