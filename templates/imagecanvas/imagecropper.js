// import { MODES, STATES } from "./constants";
const STATES = {
  OFFLINE: 0,
  LOADING: 1,
  READY: 2,
};
const MODES = {
  SQUARE: "square",
  CIRCULAR: "circular",
};

// import { copyTo, hasValue } from "./utils/Object";
function values(obj) {
  return Object.keys(obj).reduce((acc, key) => {
    acc.push(obj[key]);
    return acc;
  }, []);
}
function copyTo(obj_to, obj_from) {
  Object.keys(obj_from || {}).forEach((key) => {
    obj_to[key] = obj_from[key];
  });
}
function hasValue(obj, val_to_find) {
  return !!(Array.isArray(obj) ? obj : values(obj).indexOf(val_to_find) !== -1);
}

// import { cell, isElement } from "./utils/Dom";
function cell(tag, class_name = false, attributes = {}, parent = null, is_svg = false) {
  //  Create element, use svg namespace if required
  const el = !is_svg ? document.createElement(tag) : document.createElementNS("http://www.w3.org/2000/svg", tag);
  //   Append to parent if passed
  if (parent) parent.appendChild(el);
  //   Set attributes
  if (class_name) {
    (Array.isArray(class_name) ? class_name : [class_name]).forEach((cname) => el.classList.add(cname));
  }
  Object.keys(attributes || {}).forEach((key) => el.setAttribute(key, attributes[key]));
  return el;
}
function isElement(obj) {
  return "HTMLElement" in window
    ? !!(obj && obj instanceof HTMLElement) // DOM, Level2
    : !!(obj && typeof obj === "object" && obj.nodeType === 1 && obj.nodeName); // Older browsers
}

// import { convertGlobalToLocal } from "./utils/Event"
function convertGlobalToLocal(evt, comparison) {
  const x = evt.clientX - comparison.left;
  const y = evt.clientY - comparison.top;
  return {
    //  Make sure X is always within the bounds of our dimensions
    x: x < 0 ? 0 : x > comparison.width ? comparison.width : x,
    //  Make sure Y is always within the bounds of our dimensions
    y: y < 0 ? 0 : y > comparison.height ? comparison.height : y,
  };
}

// import Content from "./components/Content";
class Content {
  constructor(scope) {
    this.$$view = cell("div", ["imgc-content"], {}, scope.$$parent);
    this.$$source = cell("img", null, {}, this.$$view);
    //  Load Image
    this.$$source.addEventListener("load", () => {
      this.$$source.dispatchEvent(new CustomEvent("source:fetched"));
    });
  }
  source(href) {
    this.$$source.src = href;
  }
}

// import Handles from "./components/handles/index";
const HANDLES = Object.freeze([
  (pos, dim, opts) => {
    //  TOP LEFT
    const { x } = dim;
    HANDLES[7](pos, dim, opts);
    if (!opts.fixed_size) HANDLES[4](pos, dim, opts);
    else {
      if (dim.y + dim.x - x < 0) {
        dim.x = x - dim.y;
        dim.y = 0;
      } else {
        dim.y += dim.x - x;
      }
    }
  },
  (pos, dim, opts) => {
    //  TOP RIGHT
    const { x2 } = dim;
    HANDLES[5](pos, dim, opts);
    if (!opts.fixed_size) HANDLES[4](pos, dim, opts);
    else {
      if (dim.y - dim.x2 + x2 < 0) {
        dim.x2 = x2 + dim.y;
        dim.y = 0;
      } else {
        dim.y -= dim.x2 - x2;
      }
    }
  },
  (pos, dim, opts) => {
    //  BOTTOM RIGHT
    const { x2 } = dim;
    HANDLES[5](pos, dim, opts);
    if (!opts.fixed_size) HANDLES[6](pos, dim, opts);
    else {
      if (dim.y2 + dim.x2 - x2 > dim.h) {
        dim.x2 = x2 + (dim.h - dim.y2);
        dim.y2 = dim.h;
      } else {
        dim.y2 += dim.x2 - x2;
      }
    }
  },
  (pos, dim, opts) => {
    //  BOTTOM LEFT
    const { x } = dim;
    HANDLES[7](pos, dim, opts);
    if (!opts.fixed_size) HANDLES[6](pos, dim, opts);
    else {
      if (dim.y2 + (x - dim.x) > dim.h) {
        dim.x = x - (dim.h - dim.y2);
        dim.y2 = dim.h;
      } else {
        dim.y2 -= dim.x - x;
      }
    }
  },
  //  TOP
  (pos, dim, opts) => (dim.y = dim.y2 - pos.y < opts.min_crop_height ? dim.y2 - opts.min_crop_height : pos.y),
  //  RIGHT
  (pos, dim, opts) => (dim.x2 = pos.x - dim.x < opts.min_crop_width ? dim.x + opts.min_crop_width : pos.x),
  //  BOTTOM
  (pos, dim, opts) => (dim.y2 = pos.y - dim.y < opts.min_crop_height ? dim.y + opts.min_crop_height : pos.y),
  //  LEFT
  (pos, dim, opts) => (dim.x = dim.x2 - pos.x < opts.min_crop_width ? dim.x2 - opts.min_crop_width : pos.x),
]);
class Handle {
  constructor(parent, type, scope) {
    this.$$view = cell("span", ["imgc-handles-el", `imgc-handles-el-${~~(type / 4)}-${type % 4}`], {}, parent);
    //  Down handler
    function handleMouseDown(evt) {
      evt.stopPropagation();
      document.addEventListener("mouseup", handleMouseUp);
      document.addEventListener("mousemove", handleMouseMove);
    }
    //  Up handler
    function handleMouseUp(evt) {
      evt.stopPropagation();
      document.removeEventListener("mouseup", handleMouseUp);
      document.removeEventListener("mousemove", handleMouseMove);
    }
    //  Move handler
    function handleMouseMove(evt) {
      evt.stopPropagation();
      HANDLES[type](
        convertGlobalToLocal(evt, scope.$$parent.getBoundingClientRect()),
        scope.meta.dimensions,
        scope.options,
      );
      parent.dispatchEvent(new CustomEvent("source:dimensions"));
    }
    //  Bootstrap element
    this.$$view.addEventListener("mousedown", handleMouseDown);
  }
}
function move(pos, dim) {
  const w = ~~((dim.x2 - dim.x) * 0.5);
  const h = ~~((dim.y2 - dim.y) * 0.5);
  if (pos.x - w < 0) pos.x = w;
  if (pos.x + w > dim.w) pos.x = dim.w - w;
  if (pos.y - h < 0) pos.y = h;
  if (pos.y + h > dim.h) pos.y = dim.h - h;
  copyTo(dim, {
    x: pos.x - w,
    x2: pos.x + w,
    y: pos.y - h,
    y2: pos.y + h,
  });
}
class Handles {
  constructor(scope) {
    if (!hasValue(MODES, scope.options.mode)) throw new TypeError(`Mode ${scope.options.mode} doesnt exist`);
    this.$$view = cell("div", ["imgc-handles", `imgc-handles-${scope.options.mode}`], {}, scope.$$parent);
    for (let i = 0; i < (scope.options.fixed_size ? 4 : 8); i++) {
      new Handle(this.$$view, i, scope);
    }
    function onMouseDown(evt) {
      document.addEventListener("mousemove", documentMouseDown);
      document.addEventListener("mouseup", documentMouseUp);
      changeDimensions(evt);
    }
    function documentMouseDown(evt) {
      changeDimensions(evt);
    }
    function documentMouseUp() {
      document.removeEventListener("mouseup", documentMouseUp);
      document.removeEventListener("mousemove", documentMouseDown);
    }
    function changeDimensions(evt) {
      move(convertGlobalToLocal(evt, scope.$$parent.getBoundingClientRect()), scope.meta.dimensions);
      scope.$$parent.dispatchEvent(new CustomEvent("source:dimensions"));
    }
    this.$$view.addEventListener("mousedown", onMouseDown);
  }
  update({ x, x2, y, y2, w, h }) {
    copyTo(this.$$view.style, {
      top: `${y}px`,
      left: `${x}px`,
      right: `${~~(w - x2)}px`,
      bottom: `${~~(h - y2)}px`,
    });
  }
}

// import Overlay from "./components/Overlay";
class Overlay {
  constructor(scope) {
    this.$$view = cell("svg", ["imgc-overlay"], {}, scope.$$parent, true);
    this.$$path = cell("path", null, { "fill-rule": "evenodd" }, this.$$view, true);
  }
  update({ x, x2, y, y2, w, h }, { mode }) {
    const half_w = (x2 - x) * 0.5; //  Half width
    const half_h = (y2 - y) * 0.5; //  Half height
    const crop_w = x2 - x; //  Crop Width
    const crop_h = y2 - y; //  Crop Height
    this.$$path.setAttribute(
      "d",
      `M 0 0 v ${h} h ${w} v ${-h} H-0zM` +
        (mode === MODES.SQUARE
          ? `${x} ${y} h ${crop_w} v ${crop_h} h ${-crop_w} V ${-crop_h} z`
          : `${x + crop_w * 0.5} ${
              y + crop_h * 0.5
            } m ${-half_w},0 a ${half_w}, ${half_h} 0 1,0 ${crop_w},0 a ${half_w}, ${half_h} 0 1,0 ${-crop_w} ,0 z`),
    );
  }
}

// imagecrop.js
const scopes = {};
function __scope(id, opts) {
  let _state = STATES.OFFLINE;
  const scope = Object.seal(
    Object.defineProperties(
      {
        $$parent: null,
        el_content: null,
        el_handles: null,
        el_overlay: null,
        meta: {
          dimensions: {
            x: 0,
            x2: 0,
            y: 0,
            y2: 0,
            w: 0,
            h: 0,
          },
          ratio: {
            w: 1,
            h: 1,
          },
        },
        options: {
          update_cb: () => {},
          create_cb: () => {},
          destroy_cb: () => {},
          min_crop_width: 100,
          min_crop_height: 100,
          max_width: 500,
          max_height: 500,
          fixed_size: false,
          mode: MODES.SQUARE,
        },
      },
      {
        state: {
          get: () => _state,
          set: (state) => {
            _state = state;
            if (scope.$$parent) scope.$$parent.setAttribute("data-imgc-state", state);
          },
        },
      },
    ),
  );
  //  Configure scope
  copyTo(scope.options, opts);
  scopes[id] = Object.seal(scope);
  return scope;
}
function __render() {
  const scope = scopes[this.$$id];
  if (scope.state !== STATES.LOADING) return;
  const img = scope.el_content.$$source;
  //  Calculate width and height based on max-width and max-height
  let { naturalWidth: w, naturalHeight: h } = img;
  const { max_width: max_w, max_height: max_h } = scope.options;
  if (w > max_w) {
    h = ~~((max_w * h) / w);
    w = max_w;
  }
  if (h > max_h) {
    w = ~~((max_h * w) / h);
    h = max_h;
  }
  //  Set ratio to use in processing afterwards ( this is based on original image size )
  scope.meta.ratio = {
    w: Math.round((img.naturalWidth / w) * 100) / 100,
    h: Math.round((img.naturalHeight / h) * 100) / 100,
  };
  //  Set width/height
  scope.meta.dimensions.w = img.width = w;
  scope.meta.dimensions.h = img.height = h;
  scope.state = STATES.READY;
  //  Initialize dimensions
  if (scope.options.fixed_size) {
    const { min_crop_width: mcw, min_crop_height: mch } = scope.options;
    const rad = (mcw > mch ? mcw : mch) * 0.5;
    copyTo(scope.meta.dimensions, {
      x: w * 0.5 - rad,
      x2: w * 0.5 + rad,
      y: h * 0.5 - rad,
      y2: h * 0.5 + rad,
    });
  } else {
    copyTo(scope.meta.dimensions, {
      x2: w,
      y2: h,
    });
  }
  __update.call(this);
  scope.options.create_cb({ w, h });
}
function __update(evt) {
  const scope = scopes[this.$$id];
  if (scope.state !== STATES.READY) return;
  if (evt) evt.stopPropagation();
  const { dimensions: dim } = scope.meta;
  //  boundary collision checks
  if (dim.x < 0) dim.x = 0;
  if (dim.y < 0) dim.y = 0;
  if (dim.x2 > dim.w) dim.x2 = dim.w;
  if (dim.y2 > dim.h) dim.y2 = dim.h;
  //  Patch updates
  scope.el_overlay.update(dim, scope.options);
  scope.el_handles.update(dim, scope.options);
  scope.options.update_cb(dim);
}
class ImageCropper {
  constructor(selector, href, opts = {}) {
    if (!href || !selector) return;
    this.$$id = Math.random().toString(36).substring(2);
    const scope = __scope(this.$$id, opts);
    //  Set parent
    const el = selector instanceof HTMLElement ? selector : document.querySelector(selector);
    if (!isElement(el)) throw new TypeError("Does the parent exist?");
    //  Setup parent
    scope.$$parent = el;
    scope.$$parent.classList.add("imgc");
    scope.$$parent.addEventListener("DOMNodeRemovedFromDocument", this.destroy.bind(this));
    scope.$$parent.addEventListener("source:fetched", __render.bind(this), true);
    scope.$$parent.addEventListener("source:dimensions", __update.bind(this), true);
    //  Create Wrapper elements
    scope.el_content = new Content(scope);
    scope.el_overlay = new Overlay(scope);
    scope.el_handles = new Handles(scope);
    this.setImage(href);
  }
  setImage(href) {
    const scope = scopes[this.$$id];
    scope.state = STATES.LOADING;
    scope.el_content.source(href);
  }
  destroy() {
    const scope = scopes[this.$$id];
    scope.state = STATES.OFFLINE;
    if (isElement(scope.$$parent)) {
      while (scope.$$parent.firstChild) {
        scope.$$parent.removeChild(scope.$$parent.firstChild);
      }
      //  Clean parent
      scope.$$parent.classList.remove("imgc");
    }
    scope.options.destroy_cb();
    delete scopes[this.$$id];
  }
  crop(mime_type = "image/jpeg", quality = 1) {
    const scope = scopes[this.$$id];
    mime_type = hasValue(["image/jpeg", "image/png"], mime_type) ? "image/jpeg" : mime_type;
    quality = quality < 0 || quality > 1 ? 1 : quality;
    const { x, y, x2, y2 } = scope.meta.dimensions;
    const { w: rw, h: rh } = scope.meta.ratio;
    const w = x2 - x; //  width
    const h = y2 - y; //  height
    const canvas = cell("canvas", null, {
      width: w,
      height: h,
    });
    canvas.getContext("2d").drawImage(scope.el_content.$$source, rw * x, rh * y, rw * w, rh * h, 0, 0, w, h);
    return canvas.toDataURL(mime_type, quality);
  }
}
window.ImageCropper = ImageCropper;
