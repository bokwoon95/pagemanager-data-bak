"use strict";

// entrypoint function
document.addEventListener("DOMContentLoaded", function main() {
  const img = document.querySelector("img[data-img-fallback]");
  if (!img) {
    return;
  }
  const canvas = newCanvas(img);
  document.querySelector("#resize")?.addEventListener("input", function (event) {
    const input = event.currentTarget;
    const magnitude = Math.abs(input.value);
    if (input.value === 0) {
      canvas.scaleX = 1;
      canvas.scaleY = 1;
    } else if (input.value < 0) {
      canvas.scaleX = 1 / magnitude;
      canvas.scaleY = 1 / magnitude;
    } else if (input.value > 0) {
      canvas.scaleX = magnitude;
      canvas.scaleY = magnitude;
    }
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
  img.addEventListener("load", function () {
    render(canvas);
  });
  canvas.classList.add("ba", "b--dark-red");
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
    imgWidth: parseInt(imgstyle.width, 10),
    imgHeight: parseInt(imgstyle.height, 10),
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
  return canvas;
}

function render(canvas) {
  const ctx = canvas.getContext("2d");
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  ctx.drawImage(
    canvas.img,
    0,
    0,
    canvas.imgWidth,
    canvas.imgHeight,
    canvas.dx,
    canvas.dy,
    canvas.dWidth * canvas.scaleX,
    canvas.dHeight * canvas.scaleY,
  );
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
