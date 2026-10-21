(function () {
  document.addEventListener("click", function (e) {
    var sw = e.target.closest(".switch");
    if (sw) {
      sw.classList.toggle("on");
      var item = sw.closest(".perm-item");
      if (item) item.classList.toggle("on", sw.classList.contains("on"));
    }
    var t = e.target.closest("[data-toast]");
    if (t) {
      e.preventDefault();
      var msg = t.getAttribute("data-toast");
      var bar = document.getElementById("toast");
      if (!bar) {
        bar = document.createElement("div");
        bar.id = "toast";
        bar.style.cssText = "position:fixed;top:72px;right:24px;background:#13283d;color:#fff;padding:10px 14px;border-radius:10px;font-size:.82rem;z-index:50;box-shadow:0 8px 24px rgba(0,0,0,.2)";
        document.body.appendChild(bar);
      }
      bar.textContent = msg;
      bar.style.display = "block";
      clearTimeout(bar._t);
      bar._t = setTimeout(function () { bar.style.display = "none"; }, 2200);
    }
  });
})();
