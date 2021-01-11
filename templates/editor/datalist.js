const linkdiv = document.querySelector("#link");
document.querySelector("#browser").addEventListener("input", function (event) {
  if (event.target.value === "") {
    linkdiv.innerHTML = "";
    return;
  }
  const a = document.createElement("a");
  let href = event.target.value;
  if (!href.startsWith("/") && !href.startsWith("http")) {
    href = "https://" + href;
  }
  a.setAttribute("href", href);
  a.innerText = "preview";
  linkdiv.innerHTML = "";
  linkdiv.append(a);
});
