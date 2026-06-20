// Syralit client runtime.
//
// Model: the server sends a full UI tree as JSON ({type:"ui_patch", nodes:[...]}).
// We render it into #syralit-app, reconciling by widget id so that the input the
// user is typing into keeps its focus and caret across reruns. Interactions send
// {type:"widget_change", widget_id, value, is_button} back over the WebSocket,
// which triggers a rerun on the server and another ui_patch in return.
//
// Multi-page: when the server includes "pages" and "active_page" in the
// ui_patch, we render a sidebar with page links. Clicking a link sends
// {type:"page_change", page} and the server reruns the target page.

(function () {
  "use strict";

  var root = document.getElementById("syralit-app");
  var sidebar = document.getElementById("syralit-sidebar");
  var layoutRoot = document.getElementById("syralit-root");
  var ws;
  var lastPagesKey = "";

  function connect() {
    var proto = location.protocol === "https:" ? "wss:" : "ws:";
    ws = new WebSocket(proto + "//" + location.host + "/_syralit/ws");
    ws.onmessage = function (ev) {
      var msg = JSON.parse(ev.data);
      switch (msg.type) {
        case "ui_patch":
          hideOverlay();
          if (msg.pages || msg.sidebar) {
            renderSidebar(msg.pages || [], msg.active_page || "", msg.sidebar || []);
          }
          render(msg.nodes || []);
          break;
        case "__dev_status":
          setBadge(msg.state === "building" ? "Reloading…" : "");
          break;
        case "__dev_build_error":
          showOverlay(msg.error || "Build failed");
          break;
        case "__dev_asset_reload":
          if (msg.asset === "css") reloadCSS();
          else location.reload();
          break;
      }
    };
    ws.onclose = function () { setTimeout(connect, 1000); };
  }

  // --- Sidebar ---------------------------------------------------------

  function renderSidebar(pages, activePage, sidebarNodes) {
    layoutRoot.classList.add("has-sidebar");

    var header = sidebar.querySelector(".sy-sidebar-header");
    if (!header) {
      header = el("div", "sy-sidebar-header", document.title);
      sidebar.appendChild(header);
    }

    // Page links
    if (pages.length > 0) {
      var list = sidebar.querySelector(".sy-sidebar-pages");
      if (!list) {
        list = document.createElement("ul");
        list.className = "sy-sidebar-pages";
        sidebar.appendChild(list);
      }
      var key = JSON.stringify(pages) + "|" + activePage;
      if (lastPagesKey !== key) {
        lastPagesKey = key;
        list.replaceChildren();
        pages.forEach(function (p) {
          var li = document.createElement("li");
          var a = document.createElement("a");
          a.href = "#";
          if (p.icon) a.appendChild(el("span", "sy-sidebar-icon", p.icon));
          a.appendChild(document.createTextNode(p.title));
          if (p.title === activePage) a.classList.add("active");
          a.onclick = function (e) {
            e.preventDefault();
            sendPageChange(p.title);
            layoutRoot.classList.remove("sidebar-open");
          };
          li.appendChild(a);
          list.appendChild(li);
        });
      }
    }

    // User sidebar content
    var section = sidebar.querySelector(".sy-sidebar-content");
    if (sidebarNodes.length > 0) {
      if (!section) {
        section = el("div", "sy-sidebar-content");
        sidebar.appendChild(section);
      }
      section.replaceChildren.apply(section, sidebarNodes.map(buildNode));
    } else if (section) {
      section.remove();
    }

    ensureMobileToggle();
    ensureBackdrop();
  }

  function ensureMobileToggle() {
    if (document.getElementById("syralit-sidebar-toggle")) return;
    var btn = document.createElement("button");
    btn.id = "syralit-sidebar-toggle";
    btn.className = "sy-sidebar-toggle";
    btn.textContent = "☰";
    btn.onclick = function () { layoutRoot.classList.toggle("sidebar-open"); };
    document.body.appendChild(btn);
  }

  function ensureBackdrop() {
    if (layoutRoot.querySelector(".sy-sidebar-backdrop")) return;
    var bd = document.createElement("div");
    bd.className = "sy-sidebar-backdrop";
    bd.onclick = function () { layoutRoot.classList.remove("sidebar-open"); };
    layoutRoot.appendChild(bd);
  }

  function sendPageChange(page) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    ws.send(JSON.stringify({ type: "page_change", page: page }));
  }

  // --- Dev overlay / badge ---------------------------------------------

  function setBadge(text) {
    var b = document.getElementById("syralit-badge");
    if (!text) { if (b) b.remove(); return; }
    if (!b) {
      b = document.createElement("div");
      b.id = "syralit-badge";
      b.className = "sy-dev-badge";
      document.body.appendChild(b);
    }
    b.textContent = text;
  }

  function showOverlay(errText) {
    setBadge("");
    var o = document.getElementById("syralit-overlay");
    if (!o) {
      o = document.createElement("div");
      o.id = "syralit-overlay";
      o.className = "sy-dev-overlay";
      document.body.appendChild(o);
    }
    o.replaceChildren();
    var h = document.createElement("div");
    h.className = "sy-dev-overlay-title";
    h.textContent = "Build failed";
    var pre = document.createElement("pre");
    pre.className = "sy-dev-overlay-body";
    pre.textContent = errText;
    o.appendChild(h);
    o.appendChild(pre);
  }

  function hideOverlay() {
    var o = document.getElementById("syralit-overlay");
    if (o) o.remove();
  }

  function reloadCSS() {
    document.querySelectorAll('link[rel="stylesheet"]').forEach(function (link) {
      var url = new URL(link.href, location.href);
      url.searchParams.set("_sy", Date.now().toString());
      link.href = url.toString();
    });
  }

  // --- Widget communication --------------------------------------------

  function send(widgetID, value, isButton) {
    if (!ws || ws.readyState !== WebSocket.OPEN) return;
    ws.send(JSON.stringify({
      type: "widget_change",
      widget_id: widgetID,
      value: value,
      is_button: !!isButton,
    }));
  }

  function submitForm(formEl, submitId) {
    var changes = [];
    formEl.querySelectorAll("[data-id]").forEach(function (inp) {
      var id = inp.dataset.id;
      if (id === submitId) return;
      if (inp.type === "checkbox") {
        changes.push({ widget_id: id, value: inp.checked });
      } else if (inp.tagName === "SELECT") {
        if (inp.multiple) {
          var vals = [];
          for (var i = 0; i < inp.options.length; i++) {
            if (inp.options[i].selected) vals.push(inp.options[i].value);
          }
          changes.push({ widget_id: id, value: vals });
        } else {
          changes.push({ widget_id: id, value: inp.value });
        }
      } else if (inp.type === "number" || inp.type === "range") {
        changes.push({ widget_id: id, value: parseFloat(inp.value) || 0 });
      } else {
        changes.push({ widget_id: id, value: inp.value });
      }
    });
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify({
        type: "form_submit",
        widget_id: submitId,
        changes: changes,
      }));
    }
  }

  function inForm(el) {
    return !!el.closest(".sy-form");
  }

  // --- Render ----------------------------------------------------------

  function render(nodes) {
    var active = document.activeElement;
    var activeID = active && active.dataset ? active.dataset.id : null;
    var selStart = active && "selectionStart" in active ? active.selectionStart : null;
    var selEnd = active && "selectionEnd" in active ? active.selectionEnd : null;

    root.replaceChildren.apply(root, nodes.map(buildNode));

    if (activeID) {
      var next = root.querySelector('[data-id="' + cssEscape(activeID) + '"]');
      if (next && typeof next.focus === "function") {
        next.focus();
        if (selStart != null && "setSelectionRange" in next) {
          try { next.setSelectionRange(selStart, selEnd); } catch (e) {}
        }
      }
    }
  }

  function buildNode(node) {
    var p = node.props || {};
    var result;
    switch (node.type) {
      // --- Text ---
      case "title":     return el("h1", "sy-title", p.text);
      case "header":    return el("h2", "sy-header", p.text);
      case "subheader": return el("h3", "sy-subheader", p.text);
      case "text":      return el("p", "sy-text", p.text);
      case "markdown":  return el("div", "sy-markdown", p.text);
      case "caption":   return el("p", "sy-caption", p.text);
      // --- Status ---
      case "status":    return el("div", "sy-status sy-status-" + p.level, p.text);
      case "error":     return el("div", "sy-status sy-status-error", p.text);
      // --- Input widgets ---
      case "text_input":    return textInput(node, p);
      case "checkbox":      return checkbox(node, p);
      case "select":        return selectBox(node, p);
      case "button":        return button(node, p);
      case "number_input":  return numberInput(node, p);
      case "slider":        return slider(node, p);
      case "textarea":      return textarea(node, p);
      case "radio":         return radio(node, p);
      case "multi_select":  return multiSelect(node, p);
      case "date_input":    return dateInput(node, p);
      case "time_input":    return timeInput(node, p);
      case "color_picker":  return colorPicker(node, p);
      case "toggle":        return toggle(node, p);
      // --- Layout ---
      case "columns":   return columns(node, p);
      case "column":    return column(node);
      case "expander":  return expander(node, p);
      case "tabs":      return tabs(node, p);
      case "tab_panel": return tabPanel(node, p);
      case "container": return container(node);
      case "form":      return formContainer(node);
      case "form_submit": return formSubmitBtn(node, p);
      case "divider":   return el("hr", "sy-divider");
      // --- Display ---
      case "table":     return tableEl(node, p);
      case "metric":    return metric(node, p);
      case "code":      return codeBlock(node, p);
      case "image":     return imageEl(node, p);
      case "json":      return jsonView(node, p);
      case "progress":  return progressBar(node, p);
      case "link":      return linkEl(node, p);
      case "download_button": return downloadBtn(node, p);
      default:          return el("div", "sy-unknown", "[unknown: " + node.type + "]");
    }
  }

  // --- Helpers ----------------------------------------------------------

  function el(tag, cls, text) {
    var e = document.createElement(tag);
    if (cls) e.className = cls;
    if (text != null) e.textContent = text;
    return e;
  }

  function field(labelText, control, helpText) {
    var wrap = el("div", "sy-field");
    if (labelText) wrap.appendChild(el("label", "sy-label", labelText));
    wrap.appendChild(control);
    if (helpText) wrap.appendChild(el("div", "sy-help", helpText));
    return wrap;
  }

  function cssEscape(s) {
    return (window.CSS && CSS.escape) ? CSS.escape(s) : s.replace(/"/g, '\\"');
  }

  function childNodes(node) {
    return (node.children || []).map(buildNode);
  }

  // --- Input widgets ----------------------------------------------------

  function textInput(node, p) {
    var input = document.createElement("input");
    input.type = "text";
    input.className = "sy-input";
    input.dataset.id = node.id;
    input.value = p.value || "";
    if (p.placeholder) input.placeholder = p.placeholder;
    if (p.disabled) input.disabled = true;
    if (p.max_chars) input.maxLength = p.max_chars;
    input.oninput = function () {
      if (!inForm(input)) send(node.id, input.value, false);
    };
    return field(p.label, input, p.help);
  }

  function checkbox(node, p) {
    var input = document.createElement("input");
    input.type = "checkbox";
    input.className = "sy-checkbox";
    input.dataset.id = node.id;
    input.checked = !!p.value;
    if (p.disabled) input.disabled = true;
    input.onchange = function () {
      if (!inForm(input)) send(node.id, input.checked, false);
    };
    var wrap = el("label", "sy-field sy-field-inline");
    wrap.appendChild(input);
    wrap.appendChild(el("span", "sy-label", p.label));
    if (p.help) wrap.appendChild(el("div", "sy-help", p.help));
    return wrap;
  }

  function selectBox(node, p) {
    var sel = document.createElement("select");
    sel.className = "sy-select";
    sel.dataset.id = node.id;
    if (p.disabled) sel.disabled = true;
    (p.options || []).forEach(function (opt) {
      var o = document.createElement("option");
      o.value = opt;
      o.textContent = opt;
      if (opt === p.value) o.selected = true;
      sel.appendChild(o);
    });
    sel.onchange = function () {
      if (!inForm(sel)) send(node.id, sel.value, false);
    };
    return field(p.label, sel, p.help);
  }

  function button(node, p) {
    var b = el("button", "sy-button", p.label);
    b.dataset.id = node.id;
    if (p.disabled) b.disabled = true;
    b.onclick = function () { send(node.id, true, true); };
    return b;
  }

  function numberInput(node, p) {
    var input = document.createElement("input");
    input.type = "number";
    input.className = "sy-input";
    input.dataset.id = node.id;
    input.value = p.value;
    if (p.min != null) input.min = p.min;
    if (p.max != null) input.max = p.max;
    if (p.step) input.step = p.step;
    if (p.placeholder) input.placeholder = p.placeholder;
    if (p.disabled) input.disabled = true;
    input.onchange = function () {
      if (!inForm(input)) send(node.id, parseFloat(input.value) || 0, false);
    };
    return field(p.label, input, p.help);
  }

  function slider(node, p) {
    var wrap = el("div", "sy-slider-wrap");
    var input = document.createElement("input");
    input.type = "range";
    input.className = "sy-slider";
    input.dataset.id = node.id;
    input.min = p.min != null ? p.min : 0;
    input.max = p.max != null ? p.max : 100;
    input.step = p.step || 1;
    input.value = p.value;
    if (p.disabled) input.disabled = true;
    var display = el("span", "sy-slider-value", String(p.value));
    input.oninput = function () {
      display.textContent = input.value;
    };
    input.onchange = function () {
      if (!inForm(input)) send(node.id, parseFloat(input.value), false);
    };
    wrap.appendChild(input);
    wrap.appendChild(display);
    return field(p.label, wrap, p.help);
  }

  function textarea(node, p) {
    var ta = document.createElement("textarea");
    ta.className = "sy-input sy-textarea";
    ta.dataset.id = node.id;
    ta.value = p.value || "";
    if (p.height) ta.style.height = p.height + "px";
    if (p.max_chars) ta.maxLength = p.max_chars;
    if (p.placeholder) ta.placeholder = p.placeholder;
    if (p.disabled) ta.disabled = true;
    ta.oninput = function () {
      if (!inForm(ta)) send(node.id, ta.value, false);
    };
    return field(p.label, ta, p.help);
  }

  function radio(node, p) {
    var group = el("div", "sy-radio-group");
    (p.options || []).forEach(function (opt) {
      var label = el("label", "sy-radio-label");
      var input = document.createElement("input");
      input.type = "radio";
      input.name = node.id;
      input.className = "sy-radio";
      input.value = opt;
      if (opt === p.value) input.checked = true;
      if (p.disabled) input.disabled = true;
      input.onchange = function () {
        if (!inForm(input)) send(node.id, opt, false);
      };
      label.appendChild(input);
      label.appendChild(el("span", "", opt));
      group.appendChild(label);
    });
    return field(p.label, group, p.help);
  }

  function multiSelect(node, p) {
    var vals = p.value || [];
    var options = p.options || [];
    var wrap = el("div", "sy-multi-select-wrap");
    wrap.dataset.id = node.id;

    var chips = el("div", "sy-chips");
    vals.forEach(function (v) {
      var chip = el("span", "sy-chip", v);
      if (!p.disabled) {
        var x = el("span", "sy-chip-x", "×");
        x.onclick = function () {
          var next = vals.filter(function (vv) { return vv !== v; });
          if (!inForm(wrap)) send(node.id, next, false);
        };
        chip.appendChild(x);
      }
      chips.appendChild(chip);
    });
    wrap.appendChild(chips);

    var available = options.filter(function (o) { return vals.indexOf(o) < 0; });
    if (available.length > 0 && !p.disabled) {
      var sel = document.createElement("select");
      sel.className = "sy-select sy-multi-add";
      var ph = document.createElement("option");
      ph.value = "";
      ph.textContent = "Add…";
      sel.appendChild(ph);
      available.forEach(function (opt) {
        var o = document.createElement("option");
        o.value = opt;
        o.textContent = opt;
        sel.appendChild(o);
      });
      sel.onchange = function () {
        if (sel.value) {
          var next = vals.concat([sel.value]);
          if (!inForm(wrap)) send(node.id, next, false);
        }
      };
      wrap.appendChild(sel);
    }

    return field(p.label, wrap, p.help);
  }

  function dateInput(node, p) {
    var input = document.createElement("input");
    input.type = "date";
    input.className = "sy-input";
    input.dataset.id = node.id;
    input.value = p.value || "";
    if (p.disabled) input.disabled = true;
    input.onchange = function () {
      if (!inForm(input)) send(node.id, input.value, false);
    };
    return field(p.label, input, p.help);
  }

  function timeInput(node, p) {
    var input = document.createElement("input");
    input.type = "time";
    input.className = "sy-input";
    input.dataset.id = node.id;
    input.value = p.value || "";
    if (p.disabled) input.disabled = true;
    input.onchange = function () {
      if (!inForm(input)) send(node.id, input.value, false);
    };
    return field(p.label, input, p.help);
  }

  function colorPicker(node, p) {
    var wrap = el("div", "sy-color-wrap");
    var input = document.createElement("input");
    input.type = "color";
    input.className = "sy-color-picker";
    input.dataset.id = node.id;
    input.value = p.value || "#000000";
    if (p.disabled) input.disabled = true;
    var hex = el("span", "sy-color-hex", p.value || "#000000");
    input.oninput = function () {
      hex.textContent = input.value;
    };
    input.onchange = function () {
      if (!inForm(input)) send(node.id, input.value, false);
    };
    wrap.appendChild(input);
    wrap.appendChild(hex);
    return field(p.label, wrap, p.help);
  }

  function toggle(node, p) {
    var wrap = el("label", "sy-field sy-field-inline sy-toggle-wrap");
    var input = document.createElement("input");
    input.type = "checkbox";
    input.className = "sy-toggle-input";
    input.dataset.id = node.id;
    input.checked = !!p.value;
    if (p.disabled) input.disabled = true;
    input.onchange = function () {
      if (!inForm(input)) send(node.id, input.checked, false);
    };
    var track = el("span", "sy-toggle");
    var knob = el("span", "sy-toggle-knob");
    track.appendChild(input);
    track.appendChild(knob);
    wrap.appendChild(track);
    wrap.appendChild(el("span", "sy-label", p.label));
    if (p.help) wrap.appendChild(el("div", "sy-help", p.help));
    return wrap;
  }

  // --- Layout -----------------------------------------------------------

  function columns(node, p) {
    var div = el("div", "sy-columns");
    var n = p.count || 2;
    div.style.gridTemplateColumns = "repeat(" + n + ", 1fr)";
    childNodes(node).forEach(function (c) { div.appendChild(c); });
    return div;
  }

  function column(node) {
    var div = el("div", "sy-column");
    childNodes(node).forEach(function (c) { div.appendChild(c); });
    return div;
  }

  function expander(node, p) {
    var details = document.createElement("details");
    details.className = "sy-expander";
    if (p.expanded) details.open = true;
    var summary = document.createElement("summary");
    summary.className = "sy-expander-summary";
    summary.textContent = p.label;
    details.appendChild(summary);
    var content = el("div", "sy-expander-content");
    childNodes(node).forEach(function (c) { content.appendChild(c); });
    details.appendChild(content);
    details.addEventListener("toggle", function () {
      send(node.id, details.open, false);
    });
    return details;
  }

  function tabs(node, p) {
    var wrap = el("div", "sy-tabs");
    var bar = el("div", "sy-tabs-bar");
    (p.labels || []).forEach(function (label) {
      var btn = el("button", "sy-tab-button", label);
      if (label === p.active) btn.classList.add("active");
      btn.onclick = function () { send(node.id, label, false); };
      bar.appendChild(btn);
    });
    wrap.appendChild(bar);
    var panels = el("div", "sy-tabs-panels");
    (node.children || []).forEach(function (c) {
      var panel = el("div", "sy-tab-panel");
      var cp = c.props || {};
      if (cp.label !== p.active) panel.style.display = "none";
      (c.children || []).forEach(function (gc) { panel.appendChild(buildNode(gc)); });
      panels.appendChild(panel);
    });
    wrap.appendChild(panels);
    return wrap;
  }

  function tabPanel(node, p) {
    var div = el("div", "sy-tab-panel");
    childNodes(node).forEach(function (c) { div.appendChild(c); });
    return div;
  }

  function container(node) {
    var div = el("div", "sy-container");
    childNodes(node).forEach(function (c) { div.appendChild(c); });
    return div;
  }

  function formContainer(node) {
    var div = el("div", "sy-form");
    div.dataset.formId = node.id;
    childNodes(node).forEach(function (c) { div.appendChild(c); });
    return div;
  }

  function formSubmitBtn(node, p) {
    var btn = el("button", "sy-button sy-form-submit", p.label);
    btn.dataset.id = node.id;
    btn.onclick = function () {
      var formEl = btn.closest(".sy-form");
      if (formEl) {
        submitForm(formEl, node.id);
      } else {
        send(node.id, true, true);
      }
    };
    return btn;
  }

  // --- Display ----------------------------------------------------------

  function tableEl(node, p) {
    var t = document.createElement("table");
    t.className = "sy-table";
    if (p.headers && p.headers.length) {
      var thead = document.createElement("thead");
      var tr = document.createElement("tr");
      p.headers.forEach(function (h) {
        var th = document.createElement("th");
        th.textContent = h;
        tr.appendChild(th);
      });
      thead.appendChild(tr);
      t.appendChild(thead);
    }
    var tbody = document.createElement("tbody");
    (p.rows || []).forEach(function (row) {
      var tr = document.createElement("tr");
      (row || []).forEach(function (cell) {
        var td = document.createElement("td");
        td.textContent = String(cell);
        tr.appendChild(td);
      });
      tbody.appendChild(tr);
    });
    t.appendChild(tbody);
    return t;
  }

  function metric(node, p) {
    var wrap = el("div", "sy-metric");
    wrap.appendChild(el("div", "sy-metric-label", p.label));
    wrap.appendChild(el("div", "sy-metric-value", p.value));
    if (p.delta) {
      var deltaNum = parseFloat(p.delta);
      var cls = "sy-metric-delta";
      var color = p.delta_color || "normal";
      if (!isNaN(deltaNum) && deltaNum !== 0) {
        var positive = deltaNum > 0;
        if (color === "inverse") positive = !positive;
        cls += positive ? " sy-delta-positive" : " sy-delta-negative";
      }
      var arrow = "";
      if (!isNaN(deltaNum) && deltaNum !== 0) {
        arrow = deltaNum > 0 ? "↑ " : "↓ ";
      }
      wrap.appendChild(el("div", cls, arrow + p.delta));
    }
    return wrap;
  }

  function codeBlock(node, p) {
    var pre = document.createElement("pre");
    pre.className = "sy-code";
    var code = document.createElement("code");
    if (p.language) code.className = "language-" + p.language;
    code.textContent = p.code;
    pre.appendChild(code);
    return pre;
  }

  function imageEl(node, p) {
    var wrap = el("div", "sy-image-wrap");
    var img = document.createElement("img");
    img.className = "sy-image";
    img.src = p.src;
    if (p.alt) img.alt = p.alt;
    if (p.width) img.style.maxWidth = p.width + "px";
    wrap.appendChild(img);
    if (p.caption) wrap.appendChild(el("p", "sy-caption", p.caption));
    return wrap;
  }

  function jsonView(node, p) {
    var pre = document.createElement("pre");
    pre.className = "sy-code sy-json";
    var code = document.createElement("code");
    code.textContent = p.data;
    pre.appendChild(code);
    return pre;
  }

  function progressBar(node, p) {
    var wrap = el("div", "sy-progress-wrap");
    var bar = el("div", "sy-progress-bar");
    var fill = el("div", "sy-progress-fill");
    var pct = Math.max(0, Math.min(1, p.value || 0)) * 100;
    fill.style.width = pct + "%";
    bar.appendChild(fill);
    wrap.appendChild(bar);
    return wrap;
  }

  function linkEl(node, p) {
    var a = document.createElement("a");
    a.className = "sy-link";
    a.href = p.url;
    a.textContent = p.text;
    a.target = "_blank";
    a.rel = "noopener noreferrer";
    return a;
  }

  function downloadBtn(node, p) {
    var a = document.createElement("a");
    a.className = "sy-button sy-download-button";
    a.textContent = p.label;
    a.href = "data:" + (p.mime || "application/octet-stream") + ";base64," + p.data;
    a.download = p.filename || "download";
    return a;
  }

  connect();
})();
