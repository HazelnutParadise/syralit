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
          if (msg.page_config) applyPageConfig(msg.page_config);
          render(msg.nodes || []);
          if (msg.toasts) msg.toasts.forEach(handleToast);
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

    ensureThemeToggle();
    ensureMobileToggle();
    ensureBackdrop();
  }

  function ensureThemeToggle() {
    if (sidebar.querySelector(".sy-theme-toggle")) return;
    var footer = el("div", "sy-sidebar-footer");
    var btn = document.createElement("button");
    btn.className = "sy-theme-toggle sy-btn";
    var html = document.documentElement;
    function icon() { return html.getAttribute("data-theme") === "dark" ? "☀️" : "🌙"; }
    btn.textContent = icon();
    btn.onclick = function () {
      var cur = html.getAttribute("data-theme");
      var next = cur === "dark" ? "light" : "dark";
      html.setAttribute("data-theme", next);
      btn.textContent = icon();
      try { localStorage.setItem("sy-theme", next); } catch (e) {}
    };
    footer.appendChild(btn);
    sidebar.appendChild(footer);
    try {
      var saved = localStorage.getItem("sy-theme");
      if (saved && !html.getAttribute("data-theme")) {
        html.setAttribute("data-theme", saved);
        btn.textContent = icon();
      }
    } catch (e) {}
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
      case "markdown": {
        var md = el("div", "sy-markdown");
        if (p.html) { md.innerHTML = p.html; } else { md.textContent = p.text || ""; }
        return md;
      }
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
      case "toggle":          return toggle(node, p);
      case "select_slider":   return selectSlider(node, p);
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
      case "link":        return linkEl(node, p);
      case "link_button": return linkBtnEl(node, p);
      case "download_button": return downloadBtn(node, p);
      case "file_uploader":   return fileUploader(node, p);
      case "audio":      return audioEl(node, p);
      case "video":      return videoEl(node, p);
      case "dataframe":  return dataframeEl(node, p);
      case "dialog":     return dialogEl(node, p);
      case "html":       return htmlEl(node, p);
      case "latex":      return latexEl(node, p);
      case "chat_message": return chatMessageEl(node, p);
      case "chat_input":   return chatInputEl(node, p);
      case "spinner":      return spinnerEl(node, p);
      case "popover":      return popoverEl(node, p);
      // --- Charts ---
      case "line_chart":  return lineChartEl(node, p);
      case "bar_chart":   return barChartEl(node, p);
      case "area_chart":    return areaChartEl(node, p);
      case "scatter_chart": return scatterChartEl(node, p);
      case "pie_chart":     return pieChartEl(node, p);
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

    var maxSel = p.max_selections || 0;
    var atLimit = maxSel > 0 && vals.length >= maxSel;
    var available = options.filter(function (o) { return vals.indexOf(o) < 0; });
    if (available.length > 0 && !p.disabled && !atLimit) {
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
    if (p.template) {
      div.style.gridTemplateColumns = p.template;
    } else {
      var n = p.count || 2;
      div.style.gridTemplateColumns = "repeat(" + n + ", 1fr)";
    }
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
    var p = node.props || {};
    var div = el("div", "sy-container");
    if (p.border) div.classList.add("sy-container-bordered");
    if (p.height) { div.style.maxHeight = p.height + "px"; div.style.overflowY = "auto"; }
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

  var hljsLoaded = false;
  var hljsQueue = [];

  function loadHighlightJS(cb) {
    if (hljsLoaded) { cb(); return; }
    hljsQueue.push(cb);
    if (hljsQueue.length > 1) return;
    var link = document.createElement("link");
    link.rel = "stylesheet";
    link.href = "https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@11.11.1/build/styles/github.min.css";
    var linkDark = document.createElement("link");
    linkDark.rel = "stylesheet";
    linkDark.media = "(prefers-color-scheme: dark)";
    linkDark.href = "https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@11.11.1/build/styles/github-dark.min.css";
    document.head.appendChild(link);
    document.head.appendChild(linkDark);
    var script = document.createElement("script");
    script.src = "https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@11.11.1/build/highlight.min.js";
    script.onload = function () {
      hljsLoaded = true;
      hljsQueue.forEach(function (fn) { fn(); });
      hljsQueue = [];
    };
    document.head.appendChild(script);
  }

  function codeBlock(node, p) {
    var wrap = el("div", "sy-code-wrap");
    var pre = document.createElement("pre");
    pre.className = "sy-code";
    var code = document.createElement("code");
    if (p.language) code.className = "language-" + p.language;
    code.textContent = p.code;
    pre.appendChild(code);
    wrap.appendChild(pre);

    var copyBtn = el("button", "sy-code-copy", "Copy");
    copyBtn.onclick = function () {
      navigator.clipboard.writeText(p.code).then(function () {
        copyBtn.textContent = "Copied!";
        setTimeout(function () { copyBtn.textContent = "Copy"; }, 2000);
      });
    };
    wrap.appendChild(copyBtn);

    if (p.language) {
      loadHighlightJS(function () {
        try { window.hljs.highlightElement(code); } catch (e) {}
      });
    }

    return wrap;
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

  function linkBtnEl(node, p) {
    var a = document.createElement("a");
    a.className = "sy-btn sy-link-btn";
    a.href = p.url;
    a.textContent = p.label;
    a.target = "_blank";
    a.rel = "noopener noreferrer";
    if (p.disabled) { a.classList.add("sy-disabled"); a.removeAttribute("href"); }
    return a;
  }

  function selectSlider(node, p) {
    var wrap = el("div", "sy-input-group");
    wrap.appendChild(el("label", "sy-label", p.label));
    var opts = p.options || [];
    var idx = opts.indexOf(p.value);
    if (idx < 0) idx = 0;
    var track = el("div", "sy-select-slider");
    var input = document.createElement("input");
    input.type = "range";
    input.min = "0";
    input.max = String(opts.length - 1);
    input.value = String(idx);
    input.className = "sy-slider-input";
    if (p.disabled) input.disabled = true;
    var display = el("span", "sy-select-slider-value", opts[idx] || "");
    input.oninput = function () {
      display.textContent = opts[+input.value] || "";
    };
    input.onchange = function () {
      send(node.id, opts[+input.value] || "", false);
    };
    track.appendChild(input);
    track.appendChild(display);
    wrap.appendChild(track);
    if (p.help) wrap.appendChild(el("div", "sy-help", p.help));
    return wrap;
  }

  function downloadBtn(node, p) {
    var a = document.createElement("a");
    a.className = "sy-button sy-download-button";
    a.textContent = p.label;
    a.href = "data:" + (p.mime || "application/octet-stream") + ";base64," + p.data;
    a.download = p.filename || "download";
    return a;
  }

  // --- DataFrame (sortable table) ----------------------------------------

  function dataframeEl(node, p) {
    var headers = p.headers || [];
    var rows = (p.rows || []).slice();
    var sortCol = -1, sortAsc = true;

    var wrap = el("div", "sy-dataframe-wrap");
    if (p.height) wrap.style.maxHeight = p.height + "px";
    wrap.style.overflowY = "auto";

    function rebuild() {
      var t = document.createElement("table");
      t.className = "sy-table sy-dataframe";
      var thead = document.createElement("thead");
      var tr = document.createElement("tr");
      headers.forEach(function (h, ci) {
        var th = document.createElement("th");
        th.className = "sy-df-header";
        th.textContent = h;
        if (ci === sortCol) th.textContent += sortAsc ? " ▲" : " ▼";
        th.onclick = function () {
          if (sortCol === ci) { sortAsc = !sortAsc; }
          else { sortCol = ci; sortAsc = true; }
          rows.sort(function (a, b) {
            var va = a[ci], vb = b[ci];
            var na = parseFloat(va), nb = parseFloat(vb);
            if (!isNaN(na) && !isNaN(nb)) return sortAsc ? na - nb : nb - na;
            va = String(va); vb = String(vb);
            return sortAsc ? va.localeCompare(vb) : vb.localeCompare(va);
          });
          wrap.replaceChildren();
          rebuild();
        };
        tr.appendChild(th);
      });
      thead.appendChild(tr);
      t.appendChild(thead);
      var tbody = document.createElement("tbody");
      rows.forEach(function (row) {
        var tr2 = document.createElement("tr");
        (row || []).forEach(function (cell) {
          var td = document.createElement("td");
          td.textContent = cell == null ? "" : String(cell);
          tr2.appendChild(td);
        });
        tbody.appendChild(tr2);
      });
      t.appendChild(tbody);
      wrap.appendChild(t);
    }
    rebuild();
    return wrap;
  }

  // --- Dialog (modal) ---------------------------------------------------

  function dialogEl(node, p) {
    var outer = el("div", "sy-dialog-backdrop" + (p.open ? " sy-dialog-open" : ""));
    var dialog = el("div", "sy-dialog");
    var header = el("div", "sy-dialog-header");
    header.appendChild(el("h3", "sy-dialog-title", p.title));
    var closeBtn = el("button", "sy-dialog-close", "×");
    closeBtn.onclick = function () { send(node.id, false, false); };
    header.appendChild(closeBtn);
    dialog.appendChild(header);
    var body = el("div", "sy-dialog-body");
    childNodes(node).forEach(function (c) { body.appendChild(c); });
    dialog.appendChild(body);
    outer.appendChild(dialog);
    outer.onclick = function (e) {
      if (e.target === outer) send(node.id, false, false);
    };
    return outer;
  }

  // --- LaTeX (KaTeX) ------------------------------------------------------

  var kaTexLoaded = false;
  var kaTexQueue = [];

  function loadKaTeX(cb) {
    if (kaTexLoaded) { cb(); return; }
    kaTexQueue.push(cb);
    if (kaTexQueue.length > 1) return;
    var link = document.createElement("link");
    link.rel = "stylesheet";
    link.href = "https://cdn.jsdelivr.net/npm/katex@0.16.18/dist/katex.min.css";
    document.head.appendChild(link);
    var script = document.createElement("script");
    script.src = "https://cdn.jsdelivr.net/npm/katex@0.16.18/dist/katex.min.js";
    script.onload = function () {
      kaTexLoaded = true;
      kaTexQueue.forEach(function (fn) { fn(); });
      kaTexQueue = [];
    };
    document.head.appendChild(script);
  }

  function latexEl(node, p) {
    var div = el("div", "sy-latex");
    div.textContent = p.formula || "";
    loadKaTeX(function () {
      try { window.katex.render(p.formula || "", div, { throwOnError: false, displayMode: true }); }
      catch (e) { div.textContent = p.formula || ""; }
    });
    return div;
  }

  // --- Raw HTML ----------------------------------------------------------

  function htmlEl(node, p) {
    var div = el("div", "sy-html");
    div.innerHTML = p.html || "";
    return div;
  }

  // --- Audio / Video -----------------------------------------------------

  function audioEl(node, p) {
    var audio = document.createElement("audio");
    audio.className = "sy-audio";
    audio.src = p.src;
    audio.controls = true;
    return audio;
  }

  function videoEl(node, p) {
    var wrap = el("div", "sy-video-wrap");
    var video = document.createElement("video");
    video.className = "sy-video";
    video.src = p.src;
    video.controls = true;
    if (p.width) video.style.maxWidth = p.width + "px";
    wrap.appendChild(video);
    return wrap;
  }

  // --- Toast / Celebrations -----------------------------------------------

  function handleToast(t) {
    if (t.type === "balloons") { showBalloons(); return; }
    if (t.type === "snow")     { showSnow(); return; }
    var toast = el("div", "sy-toast sy-toast-" + (t.level || "info"), t.text);
    var container = document.getElementById("syralit-toasts");
    if (!container) {
      container = document.createElement("div");
      container.id = "syralit-toasts";
      container.className = "sy-toast-container";
      document.body.appendChild(container);
    }
    container.appendChild(toast);
    requestAnimationFrame(function () { toast.classList.add("sy-toast-show"); });
    setTimeout(function () {
      toast.classList.remove("sy-toast-show");
      toast.classList.add("sy-toast-hide");
      setTimeout(function () { toast.remove(); }, 300);
    }, 3000);
  }

  function showBalloons() {
    var count = 30;
    var colors = ["#7c3aed", "#2563eb", "#16a34a", "#d97706", "#dc2626", "#f472b6", "#facc15"];
    for (var i = 0; i < count; i++) {
      (function (idx) {
        setTimeout(function () {
          var b = document.createElement("div");
          b.className = "sy-balloon";
          b.style.left = (Math.random() * 100) + "vw";
          b.style.background = colors[idx % colors.length];
          b.style.animationDuration = (2 + Math.random() * 2) + "s";
          b.style.width = b.style.height = (20 + Math.random() * 20) + "px";
          document.body.appendChild(b);
          setTimeout(function () { b.remove(); }, 4500);
        }, idx * 80);
      })(i);
    }
  }

  function showSnow() {
    var count = 50;
    for (var i = 0; i < count; i++) {
      (function (idx) {
        setTimeout(function () {
          var s = document.createElement("div");
          s.className = "sy-snowflake";
          s.textContent = "❄";
          s.style.left = (Math.random() * 100) + "vw";
          s.style.fontSize = (10 + Math.random() * 14) + "px";
          s.style.animationDuration = (3 + Math.random() * 3) + "s";
          s.style.opacity = String(0.4 + Math.random() * 0.6);
          document.body.appendChild(s);
          setTimeout(function () { s.remove(); }, 6500);
        }, idx * 60);
      })(i);
    }
  }

  // --- Page Config -------------------------------------------------------

  function applyPageConfig(cfg) {
    if (cfg.title) document.title = cfg.title;
    if (cfg.icon) setFavicon(cfg.icon);
    if (cfg.layout === "wide") {
      root.style.maxWidth = "100%";
    } else if (cfg.layout === "centered") {
      root.style.maxWidth = "";
    }
    if (cfg.logo) {
      var header = sidebar ? sidebar.querySelector(".sy-sidebar-header") : null;
      if (header) {
        var existing = header.querySelector(".sy-sidebar-logo");
        if (!existing) {
          var img = document.createElement("img");
          img.className = "sy-sidebar-logo";
          img.src = cfg.logo;
          img.alt = "";
          header.insertBefore(img, header.firstChild);
        } else {
          existing.src = cfg.logo;
        }
      }
    }
  }

  function setFavicon(icon) {
    var link = document.querySelector("link[rel='icon']");
    if (!link) {
      link = document.createElement("link");
      link.rel = "icon";
      document.head.appendChild(link);
    }
    if (icon.startsWith("http") || icon.startsWith("/") || icon.startsWith("data:")) {
      link.href = icon;
    } else {
      var svg = '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><text y="80" font-size="80">' + icon + '</text></svg>';
      link.href = "data:image/svg+xml," + encodeURIComponent(svg);
    }
  }

  // --- Chat Interface ----------------------------------------------------

  function chatMessageEl(node, p) {
    var wrap = el("div", "sy-chat-message sy-chat-" + (p.role || "user"));
    var avatar = el("div", "sy-chat-avatar");
    avatar.textContent = p.role === "assistant" ? "🤖" : "👤";
    var content = el("div", "sy-chat-content");
    childNodes(node).forEach(function (c) { content.appendChild(c); });
    wrap.appendChild(avatar);
    wrap.appendChild(content);
    return wrap;
  }

  function chatInputEl(node, p) {
    var wrap = el("div", "sy-chat-input-wrap");
    var input = document.createElement("input");
    input.type = "text";
    input.className = "sy-chat-input";
    input.dataset.id = node.id;
    input.placeholder = p.placeholder || "Type a message…";
    if (p.disabled) input.disabled = true;
    var btn = el("button", "sy-chat-send", "➤");
    function doSubmit() {
      var val = input.value.trim();
      if (!val) return;
      send(node.id, val, false);
      input.value = "";
    }
    input.addEventListener("keydown", function (e) {
      if (e.key === "Enter") { e.preventDefault(); doSubmit(); }
    });
    btn.onclick = doSubmit;
    wrap.appendChild(input);
    wrap.appendChild(btn);
    return wrap;
  }

  // --- Spinner -----------------------------------------------------------

  function spinnerEl(node, p) {
    var wrap = el("div", "sy-spinner-wrap");
    var dot = el("div", "sy-spinner");
    wrap.appendChild(dot);
    wrap.appendChild(el("span", "sy-spinner-text", p.text || "Loading…"));
    return wrap;
  }

  // --- Popover -----------------------------------------------------------

  function popoverEl(node, p) {
    var wrap = el("div", "sy-popover-wrap");
    var btn = el("button", "sy-button sy-popover-trigger", p.label);
    btn.dataset.id = node.id;
    btn.onclick = function () { send(node.id, !p.open, false); };
    wrap.appendChild(btn);
    if (p.open) {
      var panel = el("div", "sy-popover-panel");
      childNodes(node).forEach(function (c) { panel.appendChild(c); });
      wrap.appendChild(panel);
      setTimeout(function () {
        document.addEventListener("click", function close(e) {
          if (!wrap.contains(e.target)) {
            send(node.id, false, false);
            document.removeEventListener("click", close);
          }
        });
      }, 0);
    }
    return wrap;
  }

  // --- File Uploader ----------------------------------------------------

  function fileUploader(node, p) {
    var wrap = el("div", "sy-file-uploader-wrap");
    if (p.file_name) {
      var info = el("div", "sy-file-info");
      info.textContent = "📄 " + p.file_name + " (" + formatSize(p.file_size || 0) + ")";
      wrap.appendChild(info);
    }
    var input = document.createElement("input");
    input.type = "file";
    input.className = "sy-file-input";
    input.dataset.id = node.id;
    input.onchange = function () {
      if (!input.files || !input.files.length) return;
      var file = input.files[0];
      if (file.size > 10 * 1024 * 1024) {
        alert("File too large (max 10 MB)");
        return;
      }
      var reader = new FileReader();
      reader.onload = function () {
        var b64 = reader.result.split(",")[1] || "";
        send(node.id, { name: file.name, size: file.size, type: file.type, data: b64 }, false);
      };
      reader.readAsDataURL(file);
    };
    wrap.appendChild(input);
    return field(p.label, wrap, p.help);
  }

  function formatSize(bytes) {
    if (bytes < 1024) return bytes + " B";
    if (bytes < 1048576) return (bytes / 1024).toFixed(1) + " KB";
    return (bytes / 1048576).toFixed(1) + " MB";
  }

  // --- Charts (SVG) -----------------------------------------------------

  var CHART_COLORS = ["#7c3aed", "#2563eb", "#16a34a", "#d97706", "#dc2626", "#0891b2", "#be185d", "#4f46e5"];

  function svgNS(tag, attrs) {
    var e = document.createElementNS("http://www.w3.org/2000/svg", tag);
    if (attrs) { for (var k in attrs) { e.setAttribute(k, attrs[k]); } }
    return e;
  }

  function formatNum(n) {
    if (Math.abs(n) >= 1e6) return (n / 1e6).toFixed(1) + "M";
    if (Math.abs(n) >= 1e3) return (n / 1e3).toFixed(1) + "K";
    if (n === Math.floor(n)) return String(n);
    return n.toFixed(1);
  }

  function chartSetup(p) {
    var series = p.series || {};
    var names = Object.keys(series);
    var w = p.width || 600, h = p.height || 300;
    var pad = { top: 20, right: 20, bottom: 30, left: 55 };
    var plotW = w - pad.left - pad.right;
    var plotH = h - pad.top - pad.bottom;
    var allVals = [], maxLen = 0;
    names.forEach(function (name) {
      allVals = allVals.concat(series[name]);
      maxLen = Math.max(maxLen, series[name].length);
    });
    var minV = allVals.length ? Math.min.apply(null, allVals) : 0;
    var maxV = allVals.length ? Math.max.apply(null, allVals) : 1;
    if (minV === maxV) { minV -= 1; maxV += 1; }
    return { series: series, names: names, w: w, h: h, pad: pad, plotW: plotW, plotH: plotH, minV: minV, maxV: maxV, range: maxV - minV, maxLen: maxLen };
  }

  function chartAxes(svg, c) {
    var cs = getComputedStyle(document.documentElement);
    var borderC = cs.getPropertyValue("--sy-border").trim() || "#e5e7eb";
    var mutedC = cs.getPropertyValue("--sy-muted").trim() || "#6b7280";
    var ticks = 5;
    for (var i = 0; i <= ticks; i++) {
      var y = c.pad.top + c.plotH - (i / ticks) * c.plotH;
      svg.appendChild(svgNS("line", { x1: c.pad.left, y1: y, x2: c.w - c.pad.right, y2: y, stroke: borderC, "stroke-dasharray": "3,3", "stroke-width": "1" }));
      var val = c.minV + (i / ticks) * c.range;
      var txt = svgNS("text", { x: c.pad.left - 8, y: y + 4, fill: mutedC, "font-size": "11", "text-anchor": "end", "font-family": "inherit" });
      txt.textContent = formatNum(val);
      svg.appendChild(txt);
    }
    svg.appendChild(svgNS("line", { x1: c.pad.left, y1: c.h - c.pad.bottom, x2: c.w - c.pad.right, y2: c.h - c.pad.bottom, stroke: borderC, "stroke-width": "1" }));
  }

  function chartLegend(names) {
    if (names.length < 2) return null;
    var legend = el("div", "sy-chart-legend");
    names.forEach(function (name, i) {
      var item = el("span", "sy-legend-item");
      var swatch = el("span", "sy-legend-swatch");
      swatch.style.background = CHART_COLORS[i % CHART_COLORS.length];
      item.appendChild(swatch);
      item.appendChild(document.createTextNode(name));
      legend.appendChild(item);
    });
    return legend;
  }

  function wrapChart(svg, c) {
    var wrap = el("div", "sy-chart-wrap");
    wrap.appendChild(svg);
    var legend = chartLegend(c.names);
    if (legend) wrap.appendChild(legend);
    return wrap;
  }

  function lineChartEl(node, p) {
    var c = chartSetup(p);
    if (!c.names.length) return el("div", "sy-chart-empty", "No data");
    var svg = svgNS("svg", { viewBox: "0 0 " + c.w + " " + c.h, class: "sy-chart" });
    svg.style.width = "100%";
    svg.style.maxWidth = c.w + "px";
    chartAxes(svg, c);
    c.names.forEach(function (name, si) {
      var vals = c.series[name];
      var n = vals.length;
      if (n === 0) return;
      var pts = [];
      for (var j = 0; j < n; j++) {
        var x = c.pad.left + (j / (n - 1 || 1)) * c.plotW;
        var y = c.pad.top + c.plotH - ((vals[j] - c.minV) / c.range) * c.plotH;
        pts.push(x + "," + y);
      }
      svg.appendChild(svgNS("polyline", {
        points: pts.join(" "), fill: "none",
        stroke: CHART_COLORS[si % CHART_COLORS.length],
        "stroke-width": "2", "stroke-linejoin": "round", "stroke-linecap": "round",
      }));
    });
    return wrapChart(svg, c);
  }

  function barChartEl(node, p) {
    var c = chartSetup(p);
    if (!c.names.length) return el("div", "sy-chart-empty", "No data");
    c.minV = Math.min(0, c.minV);
    c.range = c.maxV - c.minV;
    var svg = svgNS("svg", { viewBox: "0 0 " + c.w + " " + c.h, class: "sy-chart" });
    svg.style.width = "100%";
    svg.style.maxWidth = c.w + "px";
    chartAxes(svg, c);
    var groupW = c.plotW / c.maxLen;
    var barW = groupW / (c.names.length + 1);
    var baseY = c.pad.top + c.plotH - ((-c.minV) / c.range) * c.plotH;
    c.names.forEach(function (name, si) {
      var vals = c.series[name];
      for (var j = 0; j < vals.length; j++) {
        var x = c.pad.left + j * groupW + (si + 0.5) * barW;
        var barH = (vals[j] / c.range) * c.plotH;
        var y = vals[j] >= 0 ? baseY - barH : baseY;
        svg.appendChild(svgNS("rect", {
          x: x, y: y, width: barW * 0.8, height: Math.abs(barH),
          fill: CHART_COLORS[si % CHART_COLORS.length], rx: "2",
        }));
      }
    });
    return wrapChart(svg, c);
  }

  function areaChartEl(node, p) {
    var c = chartSetup(p);
    if (!c.names.length) return el("div", "sy-chart-empty", "No data");
    var svg = svgNS("svg", { viewBox: "0 0 " + c.w + " " + c.h, class: "sy-chart" });
    svg.style.width = "100%";
    svg.style.maxWidth = c.w + "px";
    chartAxes(svg, c);
    var baselineY = c.pad.top + c.plotH;
    c.names.forEach(function (name, si) {
      var vals = c.series[name];
      var n = vals.length;
      if (n === 0) return;
      var pts = [];
      var linePoints = [];
      for (var j = 0; j < n; j++) {
        var x = c.pad.left + (j / (n - 1 || 1)) * c.plotW;
        var y = c.pad.top + c.plotH - ((vals[j] - c.minV) / c.range) * c.plotH;
        pts.push(x + "," + y);
        linePoints.push(x + "," + y);
      }
      var firstX = c.pad.left;
      var lastX = c.pad.left + c.plotW;
      var areaPath = "M" + firstX + "," + baselineY + " L" + pts.join(" L") + " L" + lastX + "," + baselineY + " Z";
      var color = CHART_COLORS[si % CHART_COLORS.length];
      svg.appendChild(svgNS("path", {
        d: areaPath, fill: color, opacity: "0.15",
      }));
      svg.appendChild(svgNS("polyline", {
        points: linePoints.join(" "), fill: "none",
        stroke: color, "stroke-width": "2", "stroke-linejoin": "round",
      }));
    });
    return wrapChart(svg, c);
  }

  function pieChartEl(node, p) {
    var data = p.data || {};
    var names = Object.keys(data);
    if (!names.length) return el("div", "sy-chart-empty", "No data");
    var total = 0;
    names.forEach(function (n) { total += data[n]; });
    if (total === 0) return el("div", "sy-chart-empty", "No data");

    var w = p.width || 400, h = p.height || 300;
    var cx = w / 2, cy = h / 2, r = Math.min(cx, cy) - 20;
    var svg = svgNS("svg", { viewBox: "0 0 " + w + " " + h, class: "sy-chart" });
    svg.style.width = "100%";
    svg.style.maxWidth = w + "px";

    var angle = -Math.PI / 2;
    names.forEach(function (name, si) {
      var frac = data[name] / total;
      var endAngle = angle + frac * 2 * Math.PI;
      var large = frac > 0.5 ? 1 : 0;
      var x1 = cx + r * Math.cos(angle), y1 = cy + r * Math.sin(angle);
      var x2 = cx + r * Math.cos(endAngle), y2 = cy + r * Math.sin(endAngle);
      var d = "M " + cx + " " + cy +
              " L " + x1 + " " + y1 +
              " A " + r + " " + r + " 0 " + large + " 1 " + x2 + " " + y2 + " Z";
      var path = svgNS("path", { d: d, fill: CHART_COLORS[si % CHART_COLORS.length] });
      svg.appendChild(path);

      var midAngle = angle + frac * Math.PI;
      var labelR = r * 0.65;
      var lx = cx + labelR * Math.cos(midAngle);
      var ly = cy + labelR * Math.sin(midAngle);
      if (frac >= 0.05) {
        var t = svgNS("text", { x: lx, y: ly, fill: "#fff", "font-size": "12", "text-anchor": "middle", "dominant-baseline": "central" });
        t.textContent = Math.round(frac * 100) + "%";
        svg.appendChild(t);
      }
      angle = endAngle;
    });

    return wrapChart(svg, { names: names });
  }

  function scatterChartEl(node, p) {
    var series = p.series || {};
    var names = Object.keys(series);
    if (!names.length) return el("div", "sy-chart-empty", "No data");

    var w = p.width || 600, h = p.height || 300;
    var pad = { top: 20, right: 20, bottom: 30, left: 55 };
    var plotW = w - pad.left - pad.right;
    var plotH = h - pad.top - pad.bottom;
    var allX = [], allY = [];
    names.forEach(function (name) {
      (series[name] || []).forEach(function (pt) { allX.push(pt[0]); allY.push(pt[1]); });
    });
    if (!allX.length) return el("div", "sy-chart-empty", "No data");
    var minX = Math.min.apply(null, allX), maxX = Math.max.apply(null, allX);
    var minY = Math.min.apply(null, allY), maxY = Math.max.apply(null, allY);
    if (minX === maxX) { minX -= 1; maxX += 1; }
    if (minY === maxY) { minY -= 1; maxY += 1; }
    var rangeX = maxX - minX, rangeY = maxY - minY;

    var svg = svgNS("svg", { viewBox: "0 0 " + w + " " + h, class: "sy-chart" });
    svg.style.width = "100%";
    svg.style.maxWidth = w + "px";
    var cs = getComputedStyle(document.documentElement);
    var borderC = cs.getPropertyValue("--sy-border").trim() || "#e5e7eb";
    var mutedC = cs.getPropertyValue("--sy-muted").trim() || "#6b7280";

    var ticks = 5;
    for (var i = 0; i <= ticks; i++) {
      var yy = pad.top + plotH - (i / ticks) * plotH;
      svg.appendChild(svgNS("line", { x1: pad.left, y1: yy, x2: w - pad.right, y2: yy, stroke: borderC, "stroke-dasharray": "3,3" }));
      var valY = minY + (i / ticks) * rangeY;
      var tY = svgNS("text", { x: pad.left - 8, y: yy + 4, fill: mutedC, "font-size": "11", "text-anchor": "end" });
      tY.textContent = formatNum(valY);
      svg.appendChild(tY);
    }
    svg.appendChild(svgNS("line", { x1: pad.left, y1: h - pad.bottom, x2: w - pad.right, y2: h - pad.bottom, stroke: borderC }));
    svg.appendChild(svgNS("line", { x1: pad.left, y1: pad.top, x2: pad.left, y2: h - pad.bottom, stroke: borderC }));

    names.forEach(function (name, si) {
      var color = CHART_COLORS[si % CHART_COLORS.length];
      (series[name] || []).forEach(function (pt) {
        var cx = pad.left + ((pt[0] - minX) / rangeX) * plotW;
        var cy = pad.top + plotH - ((pt[1] - minY) / rangeY) * plotH;
        svg.appendChild(svgNS("circle", { cx: cx, cy: cy, r: "4", fill: color, opacity: "0.7" }));
      });
    });

    return wrapChart(svg, { names: names });
  }

  connect();
})();


