/* 原型交互辅助 */
(function () {
  function showToast(msg, ms) {
    var el = document.querySelector(".toast");
    if (!el) {
      el = document.createElement("div");
      el.className = "toast";
      var screen = document.querySelector(".phone-screen");
      if (screen) screen.appendChild(el);
    }
    el.textContent = msg;
    el.classList.add("show");
    clearTimeout(el._t);
    el._t = setTimeout(function () {
      el.classList.remove("show");
    }, ms || 2200);
  }

  document.addEventListener("click", function (e) {
    var t = e.target.closest("[data-toast]");
    if (t) {
      e.preventDefault();
      showToast(t.getAttribute("data-toast"));
    }
    var chip = e.target.closest(".chip");
    if (chip && chip.parentElement && chip.parentElement.classList.contains("chips")) {
      var multi = chip.parentElement.dataset.multi === "1";
      if (!multi) {
        chip.parentElement.querySelectorAll(".chip").forEach(function (c) {
          c.classList.remove("on");
        });
        chip.classList.add("on");
      } else {
        chip.classList.toggle("on");
      }
    }
    var sw = e.target.closest(".switch");
    if (sw) sw.classList.toggle("on");
  });

  window.HWProto = { showToast: showToast };
})();
