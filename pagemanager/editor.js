"use strict";

function toggleRemove(globalVariables) {
  const originalCursor = document.body.style.cursor;
  globalVariables.removeState = false;
  document.addEventListener("mousedown", function () {
    if (globalVariables.removeState) {
      globalVariables.removeState = false;
      document.body.style.cursor = originalCursor;
    }
  });
  return function () {
    globalVariables.removeState = true;
    document.body.style.cursor = "crosshair";
  };
}

function toolbarButtons(globalVariables) {
  return [
    {
      title: "Clear styles from selected text or text under caret",
      innerHTML: `Clear`,
      onclick: clearSelection,
    },
    {
      title: "apply header1 to selected text",
      // https://icons.getbootstrap.com/
      innerHTML:
        `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor"` +
        ` class="bi bi-type-h1" viewBox="0 0 16 16" aria-hidden="true">` +
        `<path d="M8.637 13V3.669H7.379V7.62H2.758V3.67H1.5V13h1.258V8.728h4.62V13h1.259zm5.329` +
        ` 0V3.669h-1.244L10.5 5.316v1.265l2.16-1.565h.062V13h1.244z"></path>` +
        `</svg><span class="pm-toolbar-button-label">header1</span>`,
      onclick: surroundSelection("h1"),
    },
    {
      title: "apply header2 to selected text",
      // https://icons.getbootstrap.com/
      innerHTML:
        `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor"` +
        ` class="bi bi-type-h2" viewBox="0 0 16 16" aria-hidden="true">` +
        `<path d="M7.638 13V3.669H6.38V7.62H1.759V3.67H.5V13h1.258V8.728h4.62V13h1.259zm3.022-6.733v-.048c0-.889.63-1.668` +
        ` 1.716-1.668.957 0 1.675.608 1.675 1.572 0 .855-.554 1.504-1.067 2.085l-3.513` +
        ` 3.999V13H15.5v-1.094h-4.245v-.075l2.481-2.844c.875-.998 1.586-1.784 1.586-2.953` +
        ` 0-1.463-1.155-2.556-2.919-2.556-1.941 0-2.966 1.326-2.966 2.74v.049h1.223z"></path>` +
        `</svg><span class="pm-toolbar-button-label">header2</span>`,
      onclick: surroundSelection("h2"),
    },
    {
      title: "bold selected text",
      // https://icons.getbootstrap.com/
      innerHTML:
        `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor"` +
        ` class="bi bi-type-bold" viewBox="0 0 16 16" aria-hidden="true">` +
        `<path d="M8.21 13c2.106 0 3.412-1.087 3.412-2.823 0-1.306-.984-2.283-2.324-2.386v-.055a2.176 2.176 0 0 0` +
        ` 1.852-2.14c0-1.51-1.162-2.46-3.014-2.46H3.843V13H8.21zM5.908 4.674h1.696c.963 0 1.517.451 1.517 1.244 0` +
        ` .834-.629 1.32-1.73 1.32H5.908V4.673zm0 6.788V8.598h1.73c1.217 0 1.88.492 1.88 1.415 0` +
        ` .943-.643 1.449-1.832 1.449H5.907z"></path>` +
        `</svg><span class="pm-toolbar-button-label">bold</span>`,
      onclick: surroundSelection("strong"),
    },
    {
      title: "italic selected text",
      // https://icons.getbootstrap.com/
      innerHTML:
        `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor"` +
        ` class="bi bi-type-italic" viewBox="0 0 16 16" aria-hidden="true">` +
        `<path d="M7.991 11.674L9.53 4.455c.123-.595.246-.71 1.347-.807l.11-.52H7.211l-.11.52c1.06.096 1.128.212` +
        ` 1.005.807L6.57 11.674c-.123.595-.246.71-1.346.806l-.11.52h3.774l.11-.52c-1.06-.095-1.129-.211-1.006-.806z"></path>` +
        `</svg><span class="pm-toolbar-button-label">italic</span>`,
      onclick: surroundSelection("em"),
    },
    {
      title: "underline selected text",
      // https://icons.getbootstrap.com/
      innerHTML:
        `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor"` +
        ` class="bi bi-type-underline" viewBox="0 0 16 16" aria-hidden="true">` +
        `<path d="M5.313 3.136h-1.23V9.54c0 2.105 1.47 3.623 3.917 3.623s3.917-1.518 3.917-3.623V3.136h-1.23v6.323c0` +
        ` 1.49-.978 2.57-2.687 2.57-1.709 0-2.687-1.08-2.687-2.57V3.136zM12.5 15h-9v-1h9v1z"></path>` +
        `</svg><span class="pm-toolbar-button-label">underline</span>`,
      onclick: surroundSelection("u"),
    },
    {
      title: "strikeout selected text",
      // https://icons.getbootstrap.com/
      innerHTML:
        `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor"` +
        ` class="bi bi-type-strikethrough" viewBox="0 0 16 16" aria-hidden="true">` +
        `<path d="M6.333 5.686c0 .31.083.581.27.814H5.166a2.776 2.776 0 0 1-.099-.76c0-1.627 1.436-2.768` +
        ` 3.48-2.768 1.969 0 3.39 1.175 3.445 2.85h-1.23c-.11-1.08-.964-1.743-2.25-1.743-1.23` +
        ` 0-2.18.602-2.18 1.607zm2.194 7.478c-2.153 0-3.589-1.107-3.705-2.81h1.23c.144 1.06 1.129 1.703` +
        ` 2.544 1.703 1.34 0 2.31-.705 2.31-1.675 0-.827-.547-1.374-1.914-1.675L8.046` +
        ` 8.5H1v-1h14v1h-3.504c.468.437.675.994.675 1.697 0 1.826-1.436 2.967-3.644 2.967z"></path>` +
        `</svg><span class="pm-toolbar-button-label">strikeout</span>`,
      onclick: surroundSelection("strike"),
    },
    {
      title: "Remove list under caret",
      innerHTML: `Un-list`,
      onclick: unlist,
    },
    {
      title: "bullet list selected text",
      // https://icons.getbootstrap.com/
      innerHTML:
        `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor"` +
        ` class="bi bi-list-ul" viewBox="0 0 16 16" aria-hidden="true">` +
        `<path fill-rule="evenodd" d="M5 11.5a.5.5 0 0 1 .5-.5h9a.5.5 0 0 1 0 1h-9a.5.5 0 0 1-.5-.5zm0-4a.5.5 0 0` +
        ` 1 .5-.5h9a.5.5 0 0 1 0 1h-9a.5.5 0 0 1-.5-.5zm0-4a.5.5 0 0 1 .5-.5h9a.5.5 0 0 1 0 1h-9a.5.5 0 0 1-.5-.5zm-3` +
        ` 1a1 1 0 1 0 0-2 1 1 0 0 0 0 2zm0 4a1 1 0 1 0 0-2 1 1 0 0 0 0 2zm0 4a1 1 0 1 0 0-2 1 1 0 0 0 0 2z"></path>` +
        `</svg><span class="pm-toolbar-button-label">bullet list</span>`,
      onclick: listifySelection("ul"),
    },
    {
      title: "number list selected text",
      // https://icons.getbootstrap.com/
      innerHTML:
        `<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor"` +
        ` class="bi bi-list-ol" viewBox="0 0 16 16" aria-hidden="true">` +
        `<path fill-rule="evenodd" d="M5 11.5a.5.5 0 0 1 .5-.5h9a.5.5 0 0 1 0 1h-9a.5.5 0 0 1-.5-.5zm0-4a.5.5 0 0` +
        ` 1 .5-.5h9a.5.5 0 0 1 0 1h-9a.5.5 0 0 1-.5-.5zm0-4a.5.5 0 0 1 .5-.5h9a.5.5 0 0 1 0 1h-9a.5.5 0 0 1-.5-.5z"></path>` +
        `<path d="M1.713 11.865v-.474H2c.217 0 .363-.137.363-.317` +
        ` 0-.185-.158-.31-.361-.31-.223 0-.367.152-.373.31h-.59c.016-.467.373-.787.986-.787.588-.002.954.291.957.703a.595.595` +
        ` 0 0 1-.492.594v.033a.615.615 0 0 1` +
        ` .569.631c.003.533-.502.8-1.051.8-.656 0-1-.37-1.008-.794h.582c.008.178.186.306.422.309.254 0` +
        ` .424-.145.422-.35-.002-.195-.155-.348-.414-.348h-.3zm-.004-4.699h-.604v-.035c0-.408.295-.844.958-.844.583 0` +
        ` .96.326.96.756 0 .389-.257.617-.476.848l-.537.572v.03h1.054V9H1.143v-.395l.957-.99c.138-.142.293-.304.293-.508` +
        ` 0-.18-.147-.32-.342-.32a.33.33 0 0 0-.342.338v.041zM2.564` +
        ` 5h-.635V2.924h-.031l-.598.42v-.567l.629-.443h.635V5z"></path>` +
        `</svg><span class="pm-toolbar-button-label">number list</span>`,
      onclick: listifySelection("ol"),
    },
    {
      title: "insert link under caret",
      // https://icones.netlify.app/collection/all
      innerHTML:
        `<svg xmlns="http://www.w3.org/2000/svg" width="1em" height="1em" fill="currentColor"` +
        ` xmlns:xlink="http://www.w3.org/1999/xlink" focusable="false" role="img" preserveAspectRatio="xMidYMid meet"` +
        ` class="iconify iconify--mdi" viewBox="0 0 24 24" aria-hidden="true">` +
        `<path d="M10.6 13.4a1 1 0 0 1-1.4 1.4a4.8 4.8 0 0 1 0-7l3.5-3.6a5.1 5.1 0 0 1 7.1 0a5.1 5.1 0 0 1 0` +
        ` 7.1l-1.5 1.5a6.4 6.4 0 0 0-.4-2.4l.5-.5a3.2 3.2 0 0 0 0-4.3a3.2 3.2 0 0 0-4.3 0l-3.5 3.6a2.9 2.9 0 0 0 0` +
        ` 4.2M23 18v2h-3v3h-2v-3h-3v-2h3v-3h2v3m-3.8-4.3a4.8 4.8 0 0 0-1.4-4.5a1 1 0 0 0-1.4 1.4a2.9 2.9 0 0 1 0` +
        ` 4.2l-3.5 3.6a3.2 3.2 0 0 1-4.3 0a3.2 3.2 0 0 1 0-4.3l.5-.4a7.3 7.3 0 0 1-.4-2.5l-1.5 1.5a5.1 5.1 0 0 0 0` +
        ` 7.1a5.1 5.1 0 0 0 7.1 0l1.8-1.8a6 6 0 0 1 3.1-4.3z"></path>` +
        `</svg><span class="pm-toolbar-button-label">insert link</span>`,
      onclick: undefined,
    },
    {
      title: "modify link under caret",
      // https://icones.netlify.app/collection/all
      innerHTML:
        `<svg xmlns="http://www.w3.org/2000/svg" width="1em" height="1em" fill="currentColor"` +
        ` xmlns:xlink="http://www.w3.org/1999/xlink" focusable="false" role="img" preserveAspectRatio="xMidYMid meet"` +
        ` class="iconify iconify--mdi" viewBox="0 0 24 24" aria-hidden="true">` +
        `<path d="M10.59 13.41c.41.39.41 1.03 0 1.42c-.39.39-1.03.39-1.42 0a5.003 5.003 0 0` +
        ` 1 0-7.07l3.54-3.54a5.003 5.003 0 0 1 7.07 0a5.003 5.003 0 0 1 0` +
        ` 7.07l-1.49 1.49c.01-.82-.12-1.64-.4-2.42l.47-.48a2.982 2.982 0 0 0 0-4.24a2.982 2.982 0 0` +
        ` 0-4.24 0l-3.53 3.53a2.982 2.982 0 0 0 0 4.24m2.82-4.24c.39-.39 1.03-.39 1.42 0a5.003 5.003 0 0 1 0` +
        ` 7.07l-3.54 3.54a5.003 5.003 0 0 1-7.07 0a5.003 5.003 0 0 1` +
        ` 0-7.07l1.49-1.49c-.01.82.12 1.64.4 2.43l-.47.47a2.982 2.982 0 0 0 0 4.24a2.982 2.982 0 0 0` +
        ` 4.24 0l3.53-3.53a2.982 2.982 0 0 0 0-4.24a.973.973 0 0 1 0-1.42z"></path>` +
        `</svg><span class="pm-toolbar-button-label">modify link</span>`,
      onclick: undefined,
    },
    {
      title: "save changes to page",
      innerHTML: `Save`,
      onclick: savedata2,
    },
    {
      title: "remove element on click",
      innerHTML: `Remove`,
      onclick: toggleRemove(globalVariables),
    },
  ];
}

function isContentEditable(node) {
  return !!node?.getAttribute && node.getAttribute("contenteditable") === "true";
}

function inContentEditable(range) {
  let [starttop, endtop] = [
    getContenteditableToplevelNode(range.startContainer),
    getContenteditableToplevelNode(range.endContainer),
  ];
  return starttop && endtop && starttop.parentNode === endtop.parentNode;
}

function removeEmptyTextnodes(nodes) {
  let result = [];
  for (const node of nodes) {
    if (node.nodeName === "#text" && node.textContent.trim() === "") {
      continue;
    }
    result.push(node);
  }
  return result;
}

const isEmptyElement = (function () {
  // https://developer.mozilla.org/en-US/docs/Glossary/Empty_element
  const tags = [];
  tags.push("AREA", "BASE", "BR", "COL", "EMBED", "HR", "IMG", "INPUT");
  tags.push("LINK", "META", "PARAM", "SOURCE", "TRACK", "WBR");
  const set = new Set(tags);
  return function (node) {
    return !!node?.nodeName && set.has(node.nodeName);
  };
})();

const isBlockElement = (function () {
  // https://developer.mozilla.org/en-US/docs/Web/HTML/Block-level_elements
  const tags = [];
  tags.push("ADDRESS", "ARTICLE", "ASIDE", "BLOCKQUOTE", "DETAILS", "DIALOG");
  tags.push("DD", "DIV", "DL", "DT", "FIELDSET", "FIGCAPTION", "FIGURE", "FOOTER");
  tags.push("FORM", "H1", "H2", "H3", "H4", "H5", "H6", "HEADER", "HGROUP", "HR");
  tags.push("LI", "MAIN", "NAV", "OL", "P", "PRE", "SECTION", "TABLE", "UL");
  const set = new Set(tags);
  return function (node) {
    return !!node?.nodeName && set.has(node.nodeName);
  };
})();

// function that, when given a node, finds the top level node of the current contenteditable or undefined
function getContenteditableToplevelNode(node) {
  if (!node) {
    return undefined;
  }
  if (isContentEditable(node) || node.nodeName === "BODY" || node.nodeName === "HTML") {
    return undefined;
  }
  while (node.parentNode && node.parentNode.nodeName !== "BODY") {
    if (isContentEditable(node.parentNode)) {
      return node;
    }
    node = node.parentNode;
  }
  return undefined;
}

function getLiToplevelNode(node) {
  if (!node) {
    return undefined;
  }
  if (node.nodeName === "LI" || node.nodeName === "BODY" || node.nodeName === "HTML") {
    return undefined;
  }
  while (node.parentNode && node.parentNode.nodeName !== "BODY") {
    if (node.parentNode.nodeName === "LI") {
      return node;
    }
    node = node.parentNode;
  }
  return undefined;
}

// function that, when given a node, returns a DocumentFragment with all non-singleton tags removed.
function clearTags(node) {
  const fragment = document.createDocumentFragment();
  if (!node) {
    return fragment;
  }
  let stack = [];
  let texts = [];
  stack.unshift(node);
  while (stack.length !== 0) {
    let head = stack.shift();
    if (head.nodeName === "#text") {
      texts.push(head.textContent);
      continue;
    }
    if (isEmptyElement(head)) {
      if (texts.length > 0) {
        fragment.append(document.createTextNode(texts.join("")));
        texts = [];
      }
      fragment.append(head.cloneNode());
      continue;
    }
    stack.unshift(...head.childNodes);
  }
  if (texts.length > 0) {
    fragment.append(document.createTextNode(texts.join("")));
  }
  return fragment;
}

function surroundSelection(tagType) {
  return function () {
    let sel = window.getSelection();
    if (!sel.anchorNode) {
      return; // nothing selected and caret not present in document
    }
    let range = sel.getRangeAt(0);
    if (range.collapsed) {
      return; // nothing selected
    }
    if (!inContentEditable(range)) {
      return; // not in contenteditable
    }
    let tag = document.createElement(tagType);
    tag.append(range.extractContents());
    range.insertNode(tag);
    // cleanup
    sel.removeAllRanges();
    range.collapse(true);
    sel.addRange(range);
  };
}

function listifySelection(tagType) {
  return function () {
    let sel = window.getSelection();
    if (!sel.anchorNode) {
      return; // nothing selected and caret not present in document
    }
    let range = sel.getRangeAt(0);
    if (range.collapsed) {
      return; // nothing selected
    }
    if (!inContentEditable(range)) {
      return; // not in contenteditable
    }
    const listNodes = [];
    const nodes = removeEmptyTextnodes(range.extractContents().childNodes);
    let fragment = null;
    {
      // *** isolate ***
      for (const node of nodes) {
        if (!isBlockElement(node) && node.nodeName !== "BR") {
          if (!fragment) {
            fragment = document.createDocumentFragment();
          }
          fragment.append(node);
          continue;
        }
        if (fragment) {
          listNodes.push(fragment);
          fragment = null;
        }
        if (node.nodeName !== "BR") {
          listNodes.push(node);
        }
      }
      if (fragment) {
        listNodes.push(fragment);
      }
      // *** isolate ***
    }
    const list = document.createElement(tagType);
    for (const node of listNodes) {
      const li = document.createElement("li");
      li.append(node);
      list.append(li);
    }
    range.insertNode(list);
    // cleanup
    sel.removeAllRanges();
    range.collapse(true);
    sel.addRange(range);
  };
}

function clearSelection() {
  const sel = window.getSelection();
  if (!sel.anchorNode) {
    return; // nothing selected and caret not present in document
  }
  const range = sel.getRangeAt(0);
  if (!inContentEditable(range)) {
    return; // not in contenteditable
  }

  // case 1: single caret (nothing selected)
  if (range.collapsed) {
    // keep going up until the first contenteditable element or <li> element
    const contenteditableToplevel = getContenteditableToplevelNode(range.startContainer);
    const liToplevel = getLiToplevelNode(range.startContainer);
    if (liToplevel) {
      liToplevel.replaceWith(clearTags(liToplevel));
    } else if (contenteditableToplevel) {
      contenteditableToplevel.replaceWith(clearTags(contenteditableToplevel));
    }
    return;
  }

  // case 2: selection within a single (non-contenteditable) node
  const parent = isContentEditable(range.startContainer.parentNode) ? undefined : range.startContainer.parentNode;
  if (parent) {
    const isTextNodeSelection = range.startContainer === range.endContainer;
    const isNodeSelection = parent.firstChild === range.startContainer && parent.lastChild === range.endContainer;
    if (isTextNodeSelection || isNodeSelection) {
      parent.replaceWith(clearTags(parent));
      return;
    }
  }

  let toplevels = range.commonAncestorContainer.childNodes;
  const toplevelStart = getContenteditableToplevelNode(range.startContainer);
  const toplevelEnd = getContenteditableToplevelNode(range.endContainer);
  // const fragment = document.createDocumentFragment();
  let [startIndex, endIndex] = [undefined, undefined];
  let liCount = 0;
  for (const [i, toplevel] of toplevels.entries()) {
    // if toplevel nodes
    if (toplevel == toplevelStart) {
      startIndex = i;
    } else if (toplevel == toplevelEnd) {
      endIndex = i;
    }
    // if list
    if (toplevel.nodeName === "LI") {
      liCount++;
      // const contents = toplevel.childNodes[0].cloneNode();
      // if (contents.nodeName === "#text" && i > 0) {
      //   fragment.append(document.createElement("br"));
      // }
      // fragment.append(contents);
    }
  }

  // case 3: selection spanning multiple toplevel nodes
  if (startIndex !== undefined && endIndex !== undefined) {
    for (let i = startIndex; i <= endIndex; i++) {
      toplevels[i].replaceWith(clearTags(toplevels[i]));
    }
    return;
  }

  // case 4: selection within a list
  if (liCount === toplevels.length) {
    // TODO: this should clear the styles across multiple <li> elements without purging the <li> tags themselves.
    // constrain our actions to the parent <li> element
    const liToplevel = getLiToplevelNode(range.startContainer);
    liToplevel?.replaceWith(clearTags(liToplevel));
    // range.commonAncestorContainer.replaceWith(fragment);
    return;
  }
}

function unlist() {
  const sel = window.getSelection();
  if (!sel.anchorNode) {
    return; // nothing selected and caret not present in document
  }
  const range = sel.getRangeAt(0);
  if (!inContentEditable(range)) {
    return; // not in contenteditable
  }
  // case 1: single caret (nothing selected)
  if (range.collapsed) {
    // get the parent <ul>/<ol> element, if any
    const list = getLiToplevelNode(range.startContainer)?.parentNode?.parentNode;
    let prevItemIsBlockElement = true;
    if (list) {
      const fragment = document.createDocumentFragment();
      const lis = removeEmptyTextnodes(list.childNodes);
      for (const li of lis) {
        if (li.nodeName !== "LI") {
          continue;
        }
        // if the previous item is not a block element, we insert a manual
        // linebreak <br> so that the current element starts on a new line
        if (!prevItemIsBlockElement) {
          fragment.append(document.createElement("br"));
        }
        prevItemIsBlockElement = isBlockElement(li.childNodes[li.childNodes.length - 1]);
        fragment.append(...li.childNodes);
      }
      list.replaceWith(fragment);
    }
    return;
  }

  // case 2: selection spans multiple <li>s
  // TODO: merge each selected <li> into the previous <li>
  // if no previous <li>, just dump the contents outside the <ul> (right before it)
  // handle the shitty <br> edge cases again
}

function pathToKeys(path) {
  let keys = path
    .replace(/\[|\]\[|\]/g, ".") // replace array brackets with dot
    .replace(/\.{2,}/g, ".") // replace multiple dots with one dot
    .replace(/^\.+|\.+$|\s*/g, "") // don't want leading dots, trailing dots or whitespaces
    .split(".");
  // convert number strings into numbers
  for (const [index, value] of keys.entries()) {
    const number = parseInt(value, 10);
    if (!isNaN(number)) {
      keys[index] = number;
    }
  }
  return keys;
}

function set(obj, path, value) {
  const keys = typeof path === "string" ? pathToKeys(path) : [...path];
  let key = keys.shift();
  while (keys.length > 0) {
    let nextkey = keys.shift();
    if (Number.isInteger(nextkey) && !Array.isArray(obj[key])) {
      obj[key] = [];
    } else if (typeof nextkey === "string" && typeof obj[key] !== "object" && obj[key] !== null) {
      obj[key] = {};
    }
    obj = obj[key]; // point the obj reference at the next element
    key = nextkey;
  }
  obj[key] = value;
}

function get(obj, path) {
  const keys = typeof path === "string" ? pathToKeys(path) : [...path];
  while (keys.length > 0) {
    const key = keys.shift();
    obj = obj[key];
    if (obj === null || obj === undefined) {
      return obj;
    }
  }
  return obj;
}

async function savedata() {
  const data = {};
  const arrayify = (function () {
    const arraytracker = {};
    return function (keys, key) {
      const path = keys.concat(key);
      const count = get(arraytracker, path) || 0;
      key = `[${count}]` + key.slice(2, key.length);
      set(arraytracker, path, count + 1);
      return key;
    };
  })();
  const pageid = window.Env("PageID"); //  || window.location.pathname;
  for (const node of document.querySelectorAll("[data-key]")) {
    const id = node.dataset.id || pageid;
    if (data[id] === null || data[id] === undefined) {
      data[id] = {};
    }
    set(data[id], node.dataset.key, node.innerHTML);
  }
  console.log(data);
}

async function savedata2() {
  const data = {};
  const pageid = window.Env("PageID");
  if (!pageid) {
    throw new Error("PageID not provided for this page");
  }
  const indextracker = {};
  for (const node of document.querySelectorAll("[data-row]")) {
    const rowname = node.getAttribute("data-row");
    const index = indextracker[rowname] || 0;
    node.setAttribute("data-row-index", index);
    indextracker[rowname] = index + 1;
  }
  for (const node of document.querySelectorAll("[data-value],[data-row-value],[data-row-href]")) {
    const id = node.getAttribute("data-id") || pageid;
    if (data[id] === null || data[id] === undefined) {
      data[id] = {};
    }
    const value = node.getAttribute("data-value");
    const rowvalue = node.getAttribute("data-row-value");
    const rowhref = node.getAttribute("data-row-href");
    if (value !== null) {
      set(data[id], value, node.innerHTML);
      continue;
    }
    if (rowvalue !== null || rowhref !== null) {
      let rowindex;
      let rowname;
      let currentNode = node;
      // Find parent row
      while (currentNode.parentNode && currentNode.parentNode.nodeName !== "BODY") {
        const index = currentNode.getAttribute("data-row-index");
        if (index !== null) {
          rowname = currentNode.getAttribute("data-row");
          if (rowname === null || rowname === "") {
            throw new Error(`data-row=${rowname} is not a valid row name`);
          }
          rowindex = parseInt(index, 10);
          if (isNaN(index)) {
            throw new Error(`data-row-index=${index} is not a number`);
          }
          break;
        }
        currentNode = currentNode.parentNode;
      }
      if (rowname === undefined || rowindex === undefined) {
        throw new Error(`data-row-value=${rowvalue} found without parent data-row`);
      }
      if (rowvalue !== null) {
        set(data, [rowname, rowindex, rowvalue], node.innerHTML);
      }
      if (rowhref !== null) {
        set(data, [rowname, rowindex, rowhref], node.getAttribute("href"));
      }
    }
  }
  const formdata = new FormData();
  formdata.append("username", "Groucho");
  formdata.append("accountnum", 123456); // number 123456 is immediately converted to a string "123456"
  formdata.append("data", JSON.stringify(data));
  const _ = await fetch("/upload", {
    method: "POST",
    body: formdata,
  });
}

function setup() {
  const globalVariables = {};
  for (const node of document.querySelectorAll("[data-pm\\.key],[data-pm\\.row\\.key],[data-pm\\.row\\.href]")) {
    node.setAttribute("contenteditable", true);
    node.classList.add("contenteditable");
    node.classList.remove("contenteditable-selected");
    if (node.dataset.keyHref && !node.dataset.key) {
      node.classList.add("contenteditable-readonly");
      node.classList.remove("contenteditable");
      node.addEventListener("keydown", function (event) {
        event.preventDefault();
      });
    }
    node.addEventListener("mouseover", function () {
      if (!globalVariables.removeState) {
        return;
      }
      node.classList.add("contenteditable-selected");
      node.classList.remove("contenteditable");
    });
    node.addEventListener("mouseout", function () {
      if (!globalVariables.removeState) {
        return;
      }
      node.classList.add("contenteditable-selected");
      node.classList.remove("contenteditable");
    });
  }
  for (const node of document.querySelectorAll("[data-template]")) {
    // node.
  }
  // for (const node of document.querySelectorAll("[data-endkey]")) {
  //   node.setAttribute("contenteditable", true);
  //   node.classList.add("contenteditable");
  //   node.classList.remove("contenteditable-selected");
  // }
  const toolbar = document.createElement("div");
  toolbar.classList.add("pm-toolbar");
  const buttonsData = toolbarButtons(globalVariables);
  for (const data of buttonsData) {
    const btn = document.createElement("button");
    btn.classList.add("pm-toolbar-button");
    btn.setAttribute("title", data.title);
    btn.innerHTML = data.innerHTML;
    btn.addEventListener("mousedown", function (event) {
      event.preventDefault();
    });
    btn.addEventListener("click", data.onclick);
    toolbar.append(btn);
  }
  const toolbarBacking = document.createElement("div");
  toolbarBacking.id = "pm-toolbar-backing";
  toolbarBacking.classList.add("pm-toolbar-backing");
  toolbarBacking.append(toolbar);
  document.querySelector("body")?.append(toolbarBacking);
  const resizeToolbarBacking = function () {
    let height = toolbar.offsetHeight;
    height += parseInt(window.getComputedStyle(toolbar).getPropertyValue("margin-top"));
    height += parseInt(window.getComputedStyle(toolbar).getPropertyValue("margin-bottom"));
    toolbarBacking.style.marginBottom = `${height}px`;
  };
  resizeToolbarBacking();
  window.addEventListener("resize", resizeToolbarBacking);
}
setup();

// for debugging
window.addEventListener("keypress", function (event) {
  // boilerplate
  let sel = window.getSelection();
  if (!sel.anchorNode) {
    return;
  }
  let range = sel.getRangeAt(0);
  if (event.ctrlKey && event.key === "0") {
    window.sel = sel;
    window.range = range;
    console.log("sel", sel);
    console.log("range", range);
    console.log("startContainer", range.startContainer);
    console.log("endContainer", range.endContainer);
    console.log("commonAncestorContainer", range.commonAncestorContainer.childNodes);
  } else if (event.ctrlKey && event.key === "9") {
    window.docfrag = range.cloneContents();
    console.log("cloneContents", range.cloneContents());
  }
});
