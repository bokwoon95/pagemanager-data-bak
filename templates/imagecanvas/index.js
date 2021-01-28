"use strict";

// entrypoint: start reading here
document.addEventListener("DOMContentLoaded", function main() {
  const img = document.querySelector("img[data-img-fallback]");
  if (!img) {
    return;
  }
  const canvas = newCanvas(img);
  document.querySelector("#resize")?.addEventListener("input", function (event) {
    const prevScaleX = canvas.scaleX;
    const prevScaleY = canvas.scaleY;
    const input = event.currentTarget;
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
      canvas.scaleX = 1;
      canvas.scaleY = 1;
    } else {
      canvas.scaleX = 1 + Math.abs(value * unit);
      canvas.scaleY = 1 + Math.abs(value * unit);
    }
    const deltaX = (canvas.scaleX - prevScaleX) * canvas.dWidth;
    const deltaY = (canvas.scaleY - prevScaleY) * canvas.dHeight;
    canvas.dx -= deltaX / 2;
    canvas.dy -= deltaY / 2;
    render(canvas);
  });
  document.querySelector("#imgpicker")?.addEventListener("input", async function (event) {
    const input = event.currentTarget;
    const file = input.files[0];
    if (file === null || file === undefined) {
      return;
    }
    canvas.img = await createImageBitmap(file);
    render(canvas);
  });
  img.replaceWith(canvas);
});

function newCanvas(img) {
  const imgstyle = window.getComputedStyle(img);
  const canvas = document.createElement("canvas");
  canvas.classList.add("db", "ba", "b--dark-red");
  canvas.setAttribute("height", imgstyle.height);
  canvas.setAttribute("width", imgstyle.width);
  canvas.addEventListener("mousedown", mousedown(canvas));
  canvas.addEventListener("mousemove", mousemove(canvas));
  canvas.addEventListener("mouseup", mouseup(canvas));
  canvas.addEventListener("mouseout", mouseout(canvas));
  canvas.addEventListener("mouseenter", mouseenter(canvas));
  document.addEventListener("mouseup", function (event) {
    if (event.target === canvas) {
      return;
    }
    canvas.dragging = false;
    canvas.outOfBoundsDragging = false;
  });
  Object.assign(canvas, {
    dragging: false,
    outOfBoundsDragging: false,
    img: img,
    imgWidth: img.naturalWidth,
    imgHeight: img.naturalHeight,
    dx: 0,
    dy: 0,
    prevX: 0,
    prevY: 0,
    scaleX: 1,
    scaleY: 1,
    dWidth: parseInt(imgstyle.width, 10),
    dHeight: parseInt(imgstyle.height, 10),
  });
  Object.seal(canvas);
  const img2 = document.createElement("img");
  img2.addEventListener("load", function () {
    canvas.img = img2;
    render(canvas);
  });
  img2.src = img.src;
  window.img = canvas.img;
  return canvas;
}

function render(canvas) {
  if (canvas.dx > 0) {
    canvas.dx = 0;
  }
  if (canvas.dx + canvas.width * canvas.scaleX < canvas.width) {
    canvas.dx = canvas.width - canvas.width * canvas.scaleX;
  }
  if (canvas.dy > 0) {
    canvas.dy = 0;
  }
  if (canvas.dy + canvas.height * canvas.scaleY < canvas.height) {
    canvas.dy = canvas.height - canvas.height * canvas.scaleY;
  }
  const ctx = canvas.getContext("2d");
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  ctx.drawImage(
    canvas.img, // img
    0, // sx
    0, // sy
    canvas.imgWidth, // sWidth
    canvas.imgHeight, // sHeight
    canvas.dx, // dx
    canvas.dy, // dy
    canvas.dWidth * canvas.scaleX, // dWidth
    // 533 * canvas.scaleX, // dWidth
    canvas.dHeight * canvas.scaleY, // dHeight
  );
  window.canvas = canvas;
}

function mousedown(canvas) {
  return function (event) {
    const rect = event.currentTarget.getBoundingClientRect();
    const X = event.clientX - rect.left;
    const Y = event.clientY - rect.top;
    canvas.prevX = X;
    canvas.prevY = Y;
    canvas.dragging = true;
    render(canvas);
  };
}

function mousemove(canvas) {
  return function (event) {
    if (!canvas.dragging) {
      return;
    }
    const rect = event.currentTarget.getBoundingClientRect();
    const X = event.clientX - rect.left;
    const Y = event.clientY - rect.top;
    const dx = X - canvas.prevX;
    const dy = Y - canvas.prevY;
    {
      // NOTE: don't really understand the shenanigans here, arrived at it through trial-and-error
      const withinTopBorder = canvas.dy + dy < 0;
      const withinBottomBorder = canvas.dHeight + Math.abs(canvas.dy + dy) < canvas.dHeight * canvas.scaleY;
      const withinRightBorder = canvas.dWidth + Math.abs(canvas.dx + dx) < canvas.dWidth * canvas.scaleX;
      const withinLeftBorder = canvas.dx + dx < 0;
      if (withinLeftBorder && withinRightBorder) {
        canvas.prevX = X;
        canvas.dx = canvas.dx + dx;
      }
      if (withinTopBorder && withinBottomBorder) {
        canvas.prevY = Y;
        canvas.dy = canvas.dy + dy;
      }
    }
    render(canvas);
  };
}

function mouseup(canvas) {
  return function () {
    canvas.dragging = false;
    render(canvas);
  };
}

function mouseout(canvas) {
  return function () {
    if (canvas.dragging) {
      canvas.dragging = false;
      canvas.outOfBoundsDragging = true;
    }
  };
}

function mouseenter(canvas) {
  return function () {
    if (canvas.outOfBoundsDragging) {
      canvas.outOfBoundsDragging = false;
      canvas.dragging = true;
      render(canvas);
    }
  };
}
