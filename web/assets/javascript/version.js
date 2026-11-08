/*
 * Shows which version of speedtest-go the server runs next to the source code link.
 * The footer has an empty, hidden element whose data-version-url is the backend's
 * version address. The element stays hidden when the backend doesn't answer.
 */
(function () {
  function show() {
    var el = document.querySelector("[data-version-url]");
    if (!el || !window.fetch) return;
    fetch(el.getAttribute("data-version-url"), { cache: "no-store" })
      .then(function (r) {
        return r.ok ? r.json() : null;
      })
      .then(function (info) {
        if (!info || !info.label) return;
        var v = document.createElement(info.url ? "a" : "span");
        v.textContent = info.label;
        if (info.url) v.href = info.url;
        el.textContent = " · ";
        el.appendChild(v);
        el.hidden = false;
      })
      .catch(function () {});
  }

  if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", show);
  else show();
})();
