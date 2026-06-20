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
    var qs = location.search || "";
    ws = new WebSocket(proto + "//" + location.host + "/_syralit/ws" + qs);
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
        case "fragment_patch":
          patchFragment(msg.fragment_key, msg.nodes || []);
          if (msg.toasts) msg.toasts.forEach(handleToast);
          break;
        case "stream_append":
          appendStream(msg.id, msg.chunk);
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
    // The effective theme is the explicit data-theme if set, otherwise the OS
    // preference. Flipping from the effective theme makes the first click always
    // produce a visible change (a plain toggle of data-theme would no-op when the
    // attribute is unset but the OS is already in the target mode).
    function effectiveTheme() {
      var attr = html.getAttribute("data-theme");
      if (attr === "dark" || attr === "light") return attr;
      return window.matchMedia && window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
    }
    function icon() { return effectiveTheme() === "dark" ? "☀️" : "🌙"; }
    btn.textContent = icon();
    btn.onclick = function () {
      var next = effectiveTheme() === "dark" ? "light" : "dark";
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
      // Dual-input widgets carry both values under one id; collect the pair.
      if (inp.dataset.multi === "range") {
        var rs = inp.closest(".sy-range-slider").querySelectorAll("input[type=range]");
        var a = parseFloat(rs[0].value), b = parseFloat(rs[1].value);
        if (a > b) { var t = a; a = b; b = t; }
        changes.push({ widget_id: id, value: [a, b] });
        return;
      }
      if (inp.dataset.multi === "daterange") {
        var ds = inp.closest(".sy-date-range").querySelectorAll("input[type=date]");
        changes.push({ widget_id: id, value: [ds[0].value, ds[1].value] });
        return;
      }
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
      case "exception": return exceptionEl(node, p);
      // --- Input widgets ---
      case "text_input":    return textInput(node, p);
      case "checkbox":      return checkbox(node, p);
      case "select":        return selectBox(node, p);
      case "button":        return button(node, p);
      case "number_input":  return numberInput(node, p);
      case "slider":        return slider(node, p);
      case "range_slider":  return rangeSlider(node, p);
      case "textarea":      return textarea(node, p);
      case "radio":         return radio(node, p);
      case "multi_select":  return multiSelect(node, p);
      case "date_input":    return dateInput(node, p);
      case "date_range_input": return dateRangeInput(node, p);
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
      case "fragment":  return fragmentEl(node);
      case "form":      return formContainer(node);
      case "form_submit": return formSubmitBtn(node, p);
      case "divider":   return el("hr", "sy-divider");
      case "status_container": return statusContainer(node, p);
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
      case "data_editor": return dataEditorEl(node, p);
      case "dialog":     return dialogEl(node, p);
      case "html":       return htmlEl(node, p);
      case "component":  return componentEl(node, p);
      case "iframe":     return iframeEl(node, p);
      case "latex":      return latexEl(node, p);
      case "chat_message": return chatMessageEl(node, p);
      case "chat_input":   return chatInputEl(node, p);
      case "camera_input": return cameraInputEl(node, p);
      case "audio_input":  return audioInputEl(node, p);
      case "page_link":    return pageLinkEl(node, p);
      case "badge":        return badgeEl(node, p);
      case "pagination":   return paginationEl(node, p);
      case "feedback":     return feedbackEl(node, p);
      case "segmented_control": return segmentedControlEl(node, p);
      case "pills":        return pillsEl(node, p);
      case "write_stream": return writeStreamEl(node, p);
      case "map":          return mapEl(node, p);
      case "spinner":      return spinnerEl(node, p);
      case "popover":      return popoverEl(node, p);
      // --- Charts ---
      case "line_chart":  return lineChartEl(node, p);
      case "bar_chart":   return barChartEl(node, p);
      case "area_chart":    return areaChartEl(node, p);
      case "scatter_chart": return scatterChartEl(node, p);
      case "pie_chart":     return pieChartEl(node, p);
      case "histogram_chart": return histogramChartEl(node, p);
      case "doughnut_chart":  return doughnutChartEl(node, p);
      case "radar_chart":     return radarChartEl(node, p);
      case "graphviz_chart":  return graphvizChartEl(node, p);
      case "vega_lite_chart": return vegaLiteChartEl(node, p);
      case "plotly_chart":    return plotlyChartEl(node, p);
      case "pyplot_chart":    return pyplotChartEl(node, p);
      case "bokeh_chart":     return bokehChartEl(node, p);
      case "pydeck_chart":    return pydeckChartEl(node, p);
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

  function field(labelText, control, helpText, labelVis) {
    var wrap = el("div", "sy-field");
    if (labelText) {
      var lbl = el("label", "sy-label", labelText);
      if (labelVis === "hidden") lbl.style.visibility = "hidden";
      else if (labelVis === "collapsed") lbl.style.display = "none";
      wrap.appendChild(lbl);
    }
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
    input.type = p.input_type || "text";
    input.className = "sy-input";
    input.dataset.id = node.id;
    input.value = p.value || "";
    if (p.placeholder) input.placeholder = p.placeholder;
    if (p.disabled) input.disabled = true;
    if (p.max_chars) input.maxLength = p.max_chars;
    input.oninput = function () {
      if (!inForm(input)) send(node.id, input.value, false);
    };
    return field(p.label, input, p.help, p.label_visibility);
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
    var options = p.options || [];
    if (options.length <= 20) {
      var sel = document.createElement("select");
      sel.className = "sy-select";
      sel.dataset.id = node.id;
      if (p.disabled) sel.disabled = true;
      options.forEach(function (opt) {
        var o = document.createElement("option");
        o.value = opt;
        o.textContent = opt;
        if (opt === p.value) o.selected = true;
        sel.appendChild(o);
      });
      sel.onchange = function () {
        if (!inForm(sel)) send(node.id, sel.value, false);
      };
      return field(p.label, sel, p.help, p.label_visibility);
    }
    var wrap = el("div", "sy-searchable-select");
    wrap.dataset.id = node.id;
    var input = document.createElement("input");
    input.type = "text";
    input.className = "sy-input";
    input.value = p.value || "";
    input.placeholder = "Type to search...";
    if (p.disabled) input.disabled = true;
    var dropdown = el("div", "sy-searchable-dropdown");
    dropdown.style.display = "none";

    function showFiltered(query) {
      dropdown.innerHTML = "";
      var q = query.toLowerCase();
      var matches = options.filter(function (o) { return o.toLowerCase().indexOf(q) >= 0; });
      matches.slice(0, 50).forEach(function (opt) {
        var item = el("div", "sy-searchable-item", opt);
        item.onmousedown = function (e) {
          e.preventDefault();
          input.value = opt;
          dropdown.style.display = "none";
          if (!inForm(wrap)) send(node.id, opt, false);
        };
        dropdown.appendChild(item);
      });
      dropdown.style.display = matches.length ? "" : "none";
    }

    input.onfocus = function () { showFiltered(input.value); };
    input.oninput = function () { showFiltered(input.value); };
    input.onblur = function () { setTimeout(function () { dropdown.style.display = "none"; }, 150); };
    wrap.appendChild(input);
    wrap.appendChild(dropdown);
    return field(p.label, wrap, p.help, p.label_visibility);
  }

  // btnClass / btnLabel apply the shared button styling props (type, width,
  // icon) so Button, LinkButton and DownloadButton render consistently.
  function btnClass(base, p) {
    var cls = base;
    if (p.buttonType === "secondary") cls += " sy-button-outline";
    else if (p.buttonType === "tertiary") cls += " sy-button-tertiary";
    if (p.containerWidth) cls += " sy-button-block";
    return cls;
  }
  function btnLabel(p) { return p.icon ? (p.icon + " " + p.label) : p.label; }

  function button(node, p) {
    var b = el("button", btnClass("sy-button", p), btnLabel(p));
    b.dataset.id = node.id;
    if (p.disabled) b.disabled = true;
    b.onclick = function () { send(node.id, true, true); };
    return b;
  }

  function exceptionEl(node, p) {
    var box = el("div", "sy-exception");
    box.appendChild(el("div", "sy-exception-head", "⚠ Exception"));
    box.appendChild(el("pre", "sy-exception-body", p.text));
    return box;
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

  // rangeSlider renders two overlaid range inputs (low/high) sharing a track,
  // with a filled band between the handles. The handles are individually
  // draggable via pointer-events on the thumbs (see .sy-range-input CSS). It
  // sends [low, high] on change and is not form-batched.
  function rangeSlider(node, p) {
    var min = p.min != null ? p.min : 0;
    var max = p.max != null ? p.max : 100;
    var step = p.step || 1;

    var row = el("div", "sy-range-row");
    var slider = el("div", "sy-range-slider");
    var track = el("div", "sy-range-track");
    var fill = el("div", "sy-range-fill");
    track.appendChild(fill);

    function mkInput(cls, value) {
      var i = document.createElement("input");
      i.type = "range";
      i.className = "sy-range-input " + cls;
      i.min = min; i.max = max; i.step = step; i.value = value;
      if (p.disabled) i.disabled = true;
      return i;
    }
    var inLo = mkInput("sy-range-low", p.low);
    var inHi = mkInput("sy-range-high", p.high);
    inLo.dataset.id = node.id;
    inLo.dataset.multi = "range"; // tells submitForm to collect both handles

    var display = el("span", "sy-slider-value");
    function pct(v) { return max === min ? 0 : ((v - min) / (max - min)) * 100; }
    function ordered() {
      var a = parseFloat(inLo.value), b = parseFloat(inHi.value);
      return a <= b ? [a, b] : [b, a];
    }
    function paint() {
      var r = ordered();
      fill.style.left = pct(r[0]) + "%";
      fill.style.right = (100 - pct(r[1])) + "%";
      display.textContent = r[0] + " – " + r[1];
    }
    inLo.oninput = paint;
    inHi.oninput = paint;
    function commit() {
      if (inForm(slider)) return; // batched on form submit
      var r = ordered();
      send(node.id, [r[0], r[1]], false);
    }
    inLo.onchange = commit;
    inHi.onchange = commit;

    slider.appendChild(track);
    slider.appendChild(inLo);
    slider.appendChild(inHi);
    row.appendChild(slider);
    row.appendChild(display);
    paint();
    return field(p.label, row, p.help);
  }

  // dateRangeInput renders a start/end pair of native date pickers, sending
  // [start, end] on change. Not form-batched.
  function dateRangeInput(node, p) {
    var row = el("div", "sy-date-range");
    function mkDate(value) {
      var i = document.createElement("input");
      i.type = "date";
      i.className = "sy-input";
      i.value = value || "";
      if (p.min) i.min = p.min;
      if (p.max) i.max = p.max;
      if (p.disabled) i.disabled = true;
      return i;
    }
    var start = mkDate(p.start);
    var end = mkDate(p.end);
    start.dataset.id = node.id;
    start.dataset.multi = "daterange"; // tells submitForm to collect both dates
    function commit() {
      if (inForm(row)) return; // batched on form submit
      send(node.id, [start.value, end.value], false);
    }
    start.onchange = commit;
    end.onchange = commit;
    row.appendChild(start);
    row.appendChild(el("span", "sy-date-range-sep", "→"));
    row.appendChild(end);
    return field(p.label, row, p.help);
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
    if (p.min) input.min = p.min;
    if (p.max) input.max = p.max;
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
    if (p.gap) div.style.gap = p.gap + "px";
    if (p.vertical_alignment) {
      var va = p.vertical_alignment;
      div.style.alignItems = va === "center" ? "center" : va === "bottom" ? "flex-end" : "flex-start";
    }
    if (p.border) div.classList.add("sy-columns-bordered");
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

  function segmentedControlEl(node, p) {
    var wrap = el("div", "sy-segmented-control");
    (p.options || []).forEach(function (opt) {
      var btn = el("button", "sy-segmented-btn" + (opt === p.value ? " sy-segmented-active" : ""), opt);
      btn.disabled = p.disabled;
      btn.onclick = function () { send(node.id, opt, false); };
      wrap.appendChild(btn);
    });
    return field(p.label, wrap, p.help, p.label_visibility);
  }

  function pillsEl(node, p) {
    var wrap = el("div", "sy-pills");
    (p.options || []).forEach(function (opt) {
      var pill = el("button", "sy-pill" + (opt === p.value ? " sy-pill-active" : ""), opt);
      pill.disabled = p.disabled;
      pill.onclick = function () { send(node.id, opt, false); };
      wrap.appendChild(pill);
    });
    return field(p.label, wrap, p.help, p.label_visibility);
  }

  function feedbackEl(node, p) {
    var wrap = el("div", "sy-feedback");
    var current = p.value || "";
    var disabled = !!p.disabled;

    function makeBtn(type, emoji) {
      var btn = document.createElement("button");
      btn.className = "sy-feedback-btn" + (current === type ? " sy-feedback-active" : "");
      btn.textContent = emoji;
      btn.disabled = disabled;
      btn.onclick = function () {
        var next = current === type ? "" : type;
        send(node.id, next, false);
      };
      return btn;
    }

    wrap.appendChild(makeBtn("up", "👍"));
    wrap.appendChild(makeBtn("down", "👎"));
    return wrap;
  }

  function paginationEl(node, p) {
    var wrap = el("div", "sy-pagination");
    var total = p.total_pages || 1;
    var current = p.page || 1;

    var prev = el("button", "sy-pagination-btn", "←");
    prev.disabled = current <= 1 || p.disabled;
    prev.onclick = function () { send(node.id, current - 1, false); };

    var label = el("span", "sy-pagination-label", current + " / " + total);

    var next = el("button", "sy-pagination-btn", "→");
    next.disabled = current >= total || p.disabled;
    next.onclick = function () { send(node.id, current + 1, false); };

    wrap.appendChild(prev);
    wrap.appendChild(label);
    wrap.appendChild(next);
    return wrap;
  }

  function writeStreamEl(node, p) {
    var div = el("div", "sy-write-stream sy-markdown");
    div.setAttribute("data-stream-id", node.id);
    if (p.text) div.textContent = p.text;
    return div;
  }

  function appendStream(id, chunk) {
    var target = content.querySelector('[data-stream-id="' + id + '"]');
    if (!target) return;
    target.textContent += chunk;
  }

  function patchFragment(key, nodes) {
    var target = content.querySelector('[data-fragment-key="' + key + '"]');
    if (!target) return;
    target.replaceChildren();
    nodes.forEach(function (n) { target.appendChild(renderNode(n)); });
  }

  function fragmentEl(node) {
    var div = el("div", "sy-fragment");
    div.setAttribute("data-fragment-key", (node.props || {}).key || "");
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

  function statusContainer(node, p) {
    var wrap = el("div", "sy-status-container sy-status-container-" + (p.state || "running"));
    var header = el("div", "sy-status-container-header");
    var icon = el("span", "sy-status-container-icon");
    if (p.state === "complete") icon.textContent = "✓";
    else if (p.state === "error") icon.textContent = "✕";
    else icon.innerHTML = '<span class="sy-status-spinner"></span>';
    header.appendChild(icon);
    header.appendChild(el("span", "sy-status-container-label", p.label));
    wrap.appendChild(header);
    var body = el("div", "sy-status-container-body");
    childNodes(node).forEach(function (c) { body.appendChild(c); });
    wrap.appendChild(body);
    return wrap;
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
    var wrap = el("div", "sy-metric" + (p.border ? " sy-metric-bordered" : ""));
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
    a.className = btnClass("sy-button sy-link-btn", p);
    a.href = p.url;
    a.textContent = btnLabel(p);
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
    a.className = btnClass("sy-button sy-download-button", p);
    a.textContent = btnLabel(p);
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

  function dataEditorEl(node, p) {
    var headers = p.headers || [];
    var rows = (p.rows || []).map(function (r) { return (r || []).slice(); });
    var disabled = !!p.disabled;
    var colCfg = p.column_config || {};

    var wrap = el("div", "sy-dataframe-wrap sy-data-editor");
    if (p.height) wrap.style.maxHeight = p.height + "px";
    wrap.style.overflowY = "auto";

    var t = document.createElement("table");
    t.className = "sy-table sy-dataframe";
    var thead = document.createElement("thead");
    var tr = document.createElement("tr");
    headers.forEach(function (h) {
      var th = document.createElement("th");
      th.textContent = h;
      var cfg = colCfg[h] || {};
      if (cfg.width) th.style.width = cfg.width + "px";
      tr.appendChild(th);
    });
    thead.appendChild(tr);
    t.appendChild(thead);

    var tbody = document.createElement("tbody");
    rows.forEach(function (row, ri) {
      var tr2 = document.createElement("tr");
      (row || []).forEach(function (cell, ci) {
        var td = document.createElement("td");
        var colName = headers[ci] || "";
        var cfg = colCfg[colName] || {};
        var colType = cfg.type || "text";

        if (disabled) {
          if (colType === "checkbox") {
            td.textContent = cell ? "✓" : "✗";
          } else if (colType === "link") {
            var a = document.createElement("a");
            a.href = cell || "#";
            a.textContent = cell || "";
            a.target = "_blank";
            td.appendChild(a);
          } else if (colType === "image") {
            var img = document.createElement("img");
            img.src = cell || "";
            img.style.maxHeight = "40px";
            td.appendChild(img);
          } else if (colType === "progress") {
            var bar = document.createElement("div");
            bar.className = "sy-progress";
            var fill = document.createElement("div");
            fill.className = "sy-progress-fill";
            var pctMax = cfg.max || 100;
            var pctVal = Math.min(Math.max((parseFloat(cell) || 0) / pctMax, 0), 1) * 100;
            fill.style.width = pctVal + "%";
            bar.appendChild(fill);
            td.appendChild(bar);
          } else if (colType === "list") {
            td.textContent = Array.isArray(cell) ? cell.join(", ") : String(cell || "");
          } else {
            td.textContent = cell == null ? "" : String(cell);
          }
        } else if (colType === "checkbox") {
          var cb = document.createElement("input");
          cb.type = "checkbox";
          cb.checked = !!cell;
          cb.onchange = function () {
            rows[ri][ci] = cb.checked;
            send(node.id, rows, false);
          };
          td.appendChild(cb);
        } else if (colType === "select" && cfg.options) {
          var sel = document.createElement("select");
          sel.className = "sy-data-editor-input";
          cfg.options.forEach(function (opt) {
            var o = document.createElement("option");
            o.value = opt;
            o.textContent = opt;
            if (String(cell) === opt) o.selected = true;
            sel.appendChild(o);
          });
          sel.onchange = function () {
            rows[ri][ci] = sel.value;
            send(node.id, rows, false);
          };
          td.appendChild(sel);
        } else if (colType === "link") {
          var linkInp = document.createElement("input");
          linkInp.type = "url";
          linkInp.className = "sy-data-editor-input";
          linkInp.value = cell == null ? "" : String(cell);
          linkInp.onchange = function () { rows[ri][ci] = linkInp.value; send(node.id, rows, false); };
          td.appendChild(linkInp);
        } else if (colType === "date") {
          var dateInp = document.createElement("input");
          dateInp.type = "date";
          dateInp.className = "sy-data-editor-input";
          dateInp.value = cell == null ? "" : String(cell);
          dateInp.onchange = function () { rows[ri][ci] = dateInp.value; send(node.id, rows, false); };
          td.appendChild(dateInp);
        } else if (colType === "time") {
          var timeInp = document.createElement("input");
          timeInp.type = "time";
          timeInp.className = "sy-data-editor-input";
          timeInp.value = cell == null ? "" : String(cell);
          timeInp.onchange = function () { rows[ri][ci] = timeInp.value; send(node.id, rows, false); };
          td.appendChild(timeInp);
        } else if (colType === "datetime") {
          var dtInp = document.createElement("input");
          dtInp.type = "datetime-local";
          dtInp.className = "sy-data-editor-input";
          dtInp.value = cell == null ? "" : String(cell);
          dtInp.onchange = function () { rows[ri][ci] = dtInp.value; send(node.id, rows, false); };
          td.appendChild(dtInp);
        } else {
          var inp = document.createElement("input");
          inp.type = colType === "number" ? "number" : "text";
          inp.className = "sy-data-editor-input";
          inp.value = cell == null ? "" : String(cell);
          if (colType === "number") {
            if (cfg.min !== undefined) inp.min = cfg.min;
            if (cfg.max !== undefined) inp.max = cfg.max;
          }
          inp.onchange = function () {
            var v = inp.value;
            if (colType === "number") {
              rows[ri][ci] = v === "" ? 0 : parseFloat(v);
            } else {
              var num = parseFloat(v);
              rows[ri][ci] = (v !== "" && !isNaN(num) && String(num) === v) ? num : v;
            }
            send(node.id, rows, false);
          };
          td.appendChild(inp);
        }
        tr2.appendChild(td);
      });
      tbody.appendChild(tr2);
    });
    t.appendChild(tbody);
    wrap.appendChild(t);

    if (p.dynamic_rows && !disabled) {
      var toolbar = el("div", "sy-data-editor-toolbar");
      var addBtn = el("button", "sy-button", "+ Add row");
      addBtn.onclick = function () {
        var newRow = headers.map(function () { return ""; });
        rows.push(newRow);
        send(node.id, rows, false);
      };
      toolbar.appendChild(addBtn);
      wrap.appendChild(toolbar);

      var delHeader = document.createElement("th");
      delHeader.textContent = "";
      delHeader.style.width = "32px";
      tr.appendChild(delHeader);

      var trs = tbody.querySelectorAll("tr");
      for (var ri2 = 0; ri2 < trs.length; ri2++) {
        (function (idx) {
          var delTd = document.createElement("td");
          var delBtn = el("button", "sy-data-editor-del", "✕");
          delBtn.onclick = function () {
            rows.splice(idx, 1);
            send(node.id, rows, false);
          };
          delTd.appendChild(delBtn);
          trs[idx].appendChild(delTd);
        })(ri2);
      }
    }

    return wrap;
  }

  function dialogEl(node, p) {
    var outer = el("div", "sy-dialog-backdrop" + (p.open ? " sy-dialog-open" : ""));
    var dialog = el("div", "sy-dialog");
    if (p.width) dialog.style.width = p.width + "px";
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

  // --- Map (Leaflet) -----------------------------------------------------

  var leafletState = "idle";
  var leafletQueue = [];

  function loadLeaflet(cb) {
    if (window.L) { cb(); return; }
    leafletQueue.push(cb);
    if (leafletState !== "idle") return;
    leafletState = "loading";
    var css = document.createElement("link");
    css.rel = "stylesheet";
    css.href = "https://unpkg.com/leaflet@1.9.4/dist/leaflet.css";
    document.head.appendChild(css);
    var js = document.createElement("script");
    js.src = "https://unpkg.com/leaflet@1.9.4/dist/leaflet.js";
    js.onload = function () {
      leafletState = "ready";
      var q = leafletQueue; leafletQueue = [];
      q.forEach(function (f) { f(); });
    };
    document.head.appendChild(js);
  }

  function mapEl(node, p) {
    var wrap = el("div", "sy-map");
    var h = (p.height || 400);
    wrap.style.height = h + "px";
    loadLeaflet(function () {
      var pts = p.points || [];
      var center = pts.length > 0 ? [pts[0].lat, pts[0].lon] : [25.033, 121.565];
      var map = window.L.map(wrap).setView(center, 12);
      window.L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
        attribution: "&copy; OpenStreetMap"
      }).addTo(map);
      var bounds = [];
      pts.forEach(function (pt) {
        var m = window.L.marker([pt.lat, pt.lon]).addTo(map);
        if (pt.text) m.bindPopup(pt.text);
        bounds.push([pt.lat, pt.lon]);
      });
      if (bounds.length > 1) {
        map.fitBounds(bounds, { padding: [30, 30] });
      }
      setTimeout(function () { map.invalidateSize(); }, 100);
    });
    return wrap;
  }

  // --- Raw HTML ----------------------------------------------------------

  function htmlEl(node, p) {
    var div = el("div", "sy-html");
    div.innerHTML = p.html || "";
    return div;
  }

  // --- Custom Component / IFrame -----------------------------------------

  function componentEl(node, p) {
    var iframe = document.createElement("iframe");
    iframe.className = "sy-component";
    iframe.style.border = "none";
    iframe.style.width = (p.width || 100) + (p.width ? "px" : "%");
    iframe.style.height = (p.height || 300) + "px";
    iframe.srcdoc = p.html || "";
    iframe.sandbox = "allow-scripts allow-same-origin";

    window.addEventListener("message", function (ev) {
      if (ev.source === iframe.contentWindow && ev.data && ev.data.syralitValue !== undefined) {
        send(node.id, ev.data.syralitValue, false);
      }
    });
    return iframe;
  }

  function iframeEl(node, p) {
    var iframe = document.createElement("iframe");
    iframe.className = "sy-iframe";
    iframe.style.border = "none";
    iframe.style.width = (p.width || 100) + (p.width ? "px" : "%");
    iframe.style.height = (p.height || 400) + "px";
    iframe.src = p.url || "";
    return iframe;
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
    var r = document.documentElement;
    if (cfg.primary_color) r.style.setProperty("--sy-accent", cfg.primary_color);
    if (cfg.bg_color) r.style.setProperty("--sy-bg", cfg.bg_color);
    if (cfg.text_color) r.style.setProperty("--sy-fg", cfg.text_color);
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

  function cameraInputEl(node, p) {
    var wrap = el("div", "sy-camera-input");
    var video = document.createElement("video");
    video.className = "sy-camera-video";
    video.autoplay = true;
    video.playsInline = true;
    video.muted = true;
    var canvas = document.createElement("canvas");
    canvas.style.display = "none";
    var preview = document.createElement("img");
    preview.className = "sy-camera-preview";
    preview.style.display = "none";
    var btn = el("button", "sy-button", "📷 Take Photo");
    var retakeBtn = el("button", "sy-button sy-button-secondary", "Retake");
    retakeBtn.style.display = "none";
    var stream = null;

    function startCamera() {
      if (navigator.mediaDevices && navigator.mediaDevices.getUserMedia) {
        navigator.mediaDevices.getUserMedia({ video: { facingMode: "user" } }).then(function (s) {
          stream = s;
          video.srcObject = s;
          video.style.display = "";
          preview.style.display = "none";
          btn.style.display = "";
          retakeBtn.style.display = "none";
        }).catch(function () {
          wrap.appendChild(el("div", "sy-status-error", "Camera access denied"));
        });
      } else {
        wrap.appendChild(el("div", "sy-status-error", "Camera not supported"));
      }
    }

    btn.onclick = function () {
      canvas.width = video.videoWidth;
      canvas.height = video.videoHeight;
      canvas.getContext("2d").drawImage(video, 0, 0);
      var dataUrl = canvas.toDataURL("image/jpeg", 0.85);
      preview.src = dataUrl;
      preview.style.display = "";
      video.style.display = "none";
      btn.style.display = "none";
      retakeBtn.style.display = "";
      if (stream) stream.getTracks().forEach(function (t) { t.stop(); });
      send(node.id, dataUrl, false);
    };

    retakeBtn.onclick = function () { startCamera(); };

    wrap.appendChild(video);
    wrap.appendChild(preview);
    wrap.appendChild(canvas);
    var btns = el("div", "sy-camera-buttons");
    btns.appendChild(btn);
    btns.appendChild(retakeBtn);
    wrap.appendChild(btns);
    startCamera();
    return field(p.label, wrap, p.help, p.label_visibility);
  }

  function audioInputEl(node, p) {
    var wrap = el("div", "sy-audio-input");
    var btn = el("button", "sy-button", "🎙️ Record");
    var stopBtn = el("button", "sy-button sy-button-secondary", "⏹ Stop");
    stopBtn.style.display = "none";
    var audioPreview = document.createElement("audio");
    audioPreview.className = "sy-audio";
    audioPreview.controls = true;
    audioPreview.style.display = "none";
    var mediaRec = null;
    var chunks = [];

    btn.onclick = function () {
      if (navigator.mediaDevices && navigator.mediaDevices.getUserMedia) {
        navigator.mediaDevices.getUserMedia({ audio: true }).then(function (stream) {
          chunks = [];
          mediaRec = new MediaRecorder(stream);
          mediaRec.ondataavailable = function (e) { if (e.data.size > 0) chunks.push(e.data); };
          mediaRec.onstop = function () {
            stream.getTracks().forEach(function (t) { t.stop(); });
            var blob = new Blob(chunks, { type: "audio/webm" });
            var reader = new FileReader();
            reader.onloadend = function () {
              audioPreview.src = reader.result;
              audioPreview.style.display = "";
              send(node.id, reader.result, false);
            };
            reader.readAsDataURL(blob);
            btn.textContent = "🎙️ Re-record";
            btn.style.display = "";
            stopBtn.style.display = "none";
          };
          mediaRec.start();
          btn.style.display = "none";
          stopBtn.style.display = "";
        }).catch(function () {
          wrap.appendChild(el("div", "sy-status-error", "Microphone access denied"));
        });
      }
    };
    stopBtn.onclick = function () { if (mediaRec) mediaRec.stop(); };

    var btns = el("div", "sy-camera-buttons");
    btns.appendChild(btn);
    btns.appendChild(stopBtn);
    wrap.appendChild(btns);
    wrap.appendChild(audioPreview);
    return field(p.label, wrap, p.help, p.label_visibility);
  }

  function pageLinkEl(node, p) {
    var a = document.createElement("a");
    a.className = "sy-page-link";
    a.textContent = p.label || p.page || "";
    a.href = "#";
    if (p.disabled) {
      a.classList.add("sy-disabled");
      a.onclick = function (e) { e.preventDefault(); };
    } else {
      a.onclick = function (e) {
        e.preventDefault();
        if (p.page && p.page.match(/^https?:\/\//)) {
          window.open(p.page, "_blank");
        } else {
          var b = JSON.stringify({ type: "page_change", page: p.page });
          if (ws && ws.readyState === 1) ws.send(b);
        }
      };
    }
    return a;
  }

  function badgeEl(node, p) {
    var colorMap = {
      blue: "#1f77b4", green: "#2ca02c", red: "#d62728",
      orange: "#ff7f0e", gray: "#6b7280", violet: "#9467bd"
    };
    var c = p.color || "blue";
    var bg = colorMap[c] || c;
    var span = document.createElement("span");
    span.className = "sy-badge";
    span.textContent = p.text || "";
    span.style.background = bg;
    return span;
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

  // --- Charts (Chart.js) ------------------------------------------------

  var CHART_COLORS = ["#7c3aed", "#2563eb", "#16a34a", "#d97706", "#dc2626", "#0891b2", "#be185d", "#4f46e5"];
  var chartjsState = "idle";
  var chartjsQueue = [];

  function loadChartJS(cb) {
    if (chartjsState === "ready") { cb(); return; }
    chartjsQueue.push(cb);
    if (chartjsState === "loading") return;
    chartjsState = "loading";
    var s = document.createElement("script");
    s.src = "https://cdn.jsdelivr.net/npm/chart.js@4.4.7/dist/chart.umd.min.js";
    s.onload = function () {
      chartjsState = "ready";
      chartjsQueue.forEach(function (fn) { fn(); });
      chartjsQueue = [];
    };
    document.head.appendChild(s);
  }

  function makeChartJS(type, p, configFn) {
    var wrap = el("div", "sy-chart-wrap");
    var canvas = document.createElement("canvas");
    wrap.style.maxWidth = (p.width || 600) + "px";
    wrap.style.height = (p.height || 300) + "px";
    wrap.appendChild(canvas);
    loadChartJS(function () {
      var cfg = configFn();
      new Chart(canvas, cfg);
    });
    return wrap;
  }

  function seriesDatasets(p, type) {
    var series = p.series || {};
    var names = Object.keys(series);
    var maxLen = 0;
    names.forEach(function (n) { maxLen = Math.max(maxLen, series[n].length); });
    var labels = p.x_labels && p.x_labels.length > 0
      ? p.x_labels
      : Array.from({ length: maxLen }, function (_, i) { return String(i + 1); });
    var datasets = names.map(function (name, si) {
      var color = CHART_COLORS[si % CHART_COLORS.length];
      var ds = { label: name, data: series[name], borderColor: color, backgroundColor: color };
      if (type === "line") {
        ds.fill = false;
        ds.tension = 0.3;
        ds.pointRadius = 3;
      } else if (type === "area") {
        ds.fill = true;
        ds.backgroundColor = color + "26";
        ds.tension = 0.3;
        ds.pointRadius = 2;
      } else if (type === "bar") {
        ds.backgroundColor = color + "cc";
      }
      return ds;
    });
    return { labels: labels, datasets: datasets, title: p.title };
  }

  function chartOptions(title, type) {
    var opts = {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: "index", intersect: false },
      plugins: {
        legend: { display: true, position: "top" },
        tooltip: { enabled: true },
      },
      scales: type !== "pie" && type !== "doughnut" ? {
        x: { grid: { display: false } },
        y: { beginAtZero: type === "bar" }
      } : undefined,
    };
    if (title) {
      opts.plugins.title = { display: true, text: title, font: { size: 14 } };
    }
    return opts;
  }

  function lineChartEl(node, p) {
    return makeChartJS("line", p, function () {
      var d = seriesDatasets(p, "line");
      return { type: "line", data: { labels: d.labels, datasets: d.datasets }, options: chartOptions(d.title, "line") };
    });
  }

  function barChartEl(node, p) {
    return makeChartJS("bar", p, function () {
      var d = seriesDatasets(p, "bar");
      return { type: "bar", data: { labels: d.labels, datasets: d.datasets }, options: chartOptions(d.title, "bar") };
    });
  }

  function areaChartEl(node, p) {
    return makeChartJS("line", p, function () {
      var d = seriesDatasets(p, "area");
      return { type: "line", data: { labels: d.labels, datasets: d.datasets }, options: chartOptions(d.title, "area") };
    });
  }

  function pieChartEl(node, p) {
    var data = p.data || {};
    var names = Object.keys(data);
    return makeChartJS("pie", p, function () {
      return {
        type: "pie",
        data: {
          labels: names,
          datasets: [{
            data: names.map(function (n) { return data[n]; }),
            backgroundColor: names.map(function (_, i) { return CHART_COLORS[i % CHART_COLORS.length]; }),
          }],
        },
        options: chartOptions(p.title, "pie"),
      };
    });
  }

  function scatterChartEl(node, p) {
    var series = p.series || {};
    var names = Object.keys(series);
    return makeChartJS("scatter", p, function () {
      var datasets = names.map(function (name, si) {
        var pts = (series[name] || []).map(function (pt) { return { x: pt[0], y: pt[1] }; });
        return { label: name, data: pts, backgroundColor: CHART_COLORS[si % CHART_COLORS.length], pointRadius: 5 };
      });
      return { type: "scatter", data: { datasets: datasets }, options: chartOptions(p.title, "scatter") };
    });
  }

  function histogramChartEl(node, p) {
    var values = p.data || [];
    var bins = p.bins || 10;
    return makeChartJS("bar", p, function () {
      if (!values.length) return { type: "bar", data: { labels: [], datasets: [] }, options: {} };
      var mn = Math.min.apply(null, values);
      var mx = Math.max.apply(null, values);
      if (mn === mx) { mn -= 1; mx += 1; }
      var step = (mx - mn) / bins;
      var counts = new Array(bins).fill(0);
      var labels = [];
      for (var i = 0; i < bins; i++) {
        var lo = mn + i * step;
        var hi = lo + step;
        labels.push(lo.toFixed(1) + "–" + hi.toFixed(1));
      }
      values.forEach(function (v) {
        var idx = Math.min(Math.floor((v - mn) / step), bins - 1);
        counts[idx]++;
      });
      return {
        type: "bar",
        data: {
          labels: labels,
          datasets: [{ label: "Frequency", data: counts, backgroundColor: CHART_COLORS[0] + "cc" }],
        },
        options: chartOptions(p.title, "bar"),
      };
    });
  }

  function doughnutChartEl(node, p) {
    var data = p.data || {};
    var names = Object.keys(data);
    return makeChartJS("doughnut", p, function () {
      return {
        type: "doughnut",
        data: {
          labels: names,
          datasets: [{
            data: names.map(function (n) { return data[n]; }),
            backgroundColor: names.map(function (_, i) { return CHART_COLORS[i % CHART_COLORS.length]; }),
          }],
        },
        options: chartOptions(p.title, "doughnut"),
      };
    });
  }

  function radarChartEl(node, p) {
    var series = p.series || {};
    var names = Object.keys(series);
    var labels = p.labels || [];
    return makeChartJS("radar", p, function () {
      var datasets = names.map(function (name, si) {
        var color = CHART_COLORS[si % CHART_COLORS.length];
        return {
          label: name,
          data: series[name],
          borderColor: color,
          backgroundColor: color + "33",
          pointRadius: 3,
        };
      });
      return {
        type: "radar",
        data: { labels: labels, datasets: datasets },
        options: chartOptions(p.title, "radar"),
      };
    });
  }

  // --- Graphviz Chart (viz.js CDN) ----------------------------------------

  var vizState = "idle"; // idle | loading | ready
  var vizQueue = [];

  function loadViz(cb) {
    if (vizState === "ready") { cb(); return; }
    vizQueue.push(cb);
    if (vizState === "loading") return;
    vizState = "loading";
    var s = document.createElement("script");
    s.src = "https://cdn.jsdelivr.net/npm/@viz-js/viz@3.11.0/lib/viz-standalone.js";
    s.onload = function () {
      vizState = "ready";
      vizQueue.forEach(function (fn) { fn(); });
      vizQueue = [];
    };
    document.head.appendChild(s);
  }

  function graphvizChartEl(node, p) {
    var wrap = el("div", "sy-graphviz");
    if (p.height) wrap.style.height = p.height + "px";
    wrap.textContent = "Loading graph…";
    loadViz(function () {
      if (typeof Viz !== "undefined") {
        Viz.instance().then(function (viz) {
          var svgStr = viz.renderSVGElement(p.dot || "digraph {}");
          wrap.textContent = "";
          svgStr.style.maxWidth = "100%";
          wrap.appendChild(svgStr);
        }).catch(function (e) {
          wrap.textContent = "Graphviz error: " + e.message;
        });
      }
    });
    return wrap;
  }

  // --- Vega-Lite Chart (st.altair_chart equivalent) --------------------------

  var vegaState = "idle";
  var vegaQueue = [];

  function loadVegaLite(cb) {
    if (window.vegaEmbed) { cb(); return; }
    vegaQueue.push(cb);
    if (vegaState !== "idle") return;
    vegaState = "loading";
    var scripts = [
      "https://cdn.jsdelivr.net/npm/vega@5/build/vega.min.js",
      "https://cdn.jsdelivr.net/npm/vega-lite@5/build/vega-lite.min.js",
      "https://cdn.jsdelivr.net/npm/vega-embed@6/build/vega-embed.min.js"
    ];
    var idx = 0;
    function next() {
      if (idx >= scripts.length) {
        vegaState = "ready";
        var q = vegaQueue; vegaQueue = [];
        q.forEach(function (f) { f(); });
        return;
      }
      var s = document.createElement("script");
      s.src = scripts[idx++];
      s.onload = next;
      s.onerror = function () {
        vegaState = "idle";
        vegaQueue.forEach(function (f) { f(); });
        vegaQueue = [];
      };
      document.head.appendChild(s);
    }
    next();
  }

  function vegaLiteChartEl(node, p) {
    var wrap = el("div", "sy-vega-chart");
    if (p.height) wrap.style.height = p.height + "px";
    if (p.width) wrap.style.width = p.width + "px";
    wrap.textContent = "Loading Vega-Lite chart…";
    loadVegaLite(function () {
      if (window.vegaEmbed) {
        var spec = p.spec || {};
        if (!spec["$schema"]) {
          spec["$schema"] = "https://vega.github.io/schema/vega-lite/v5.json";
        }
        wrap.textContent = "";
        window.vegaEmbed(wrap, spec, { actions: false, renderer: "svg" })
          .catch(function (e) {
            wrap.textContent = "Vega-Lite error: " + e.message;
          });
      } else {
        wrap.textContent = "Failed to load Vega-Lite library.";
      }
    });
    return wrap;
  }

  // --- Plotly Chart (st.plotly_chart equivalent) -----------------------------

  var plotlyState = "idle";
  var plotlyQueue = [];

  function loadPlotly(cb) {
    if (window.Plotly) { cb(); return; }
    plotlyQueue.push(cb);
    if (plotlyState !== "idle") return;
    plotlyState = "loading";
    var s = document.createElement("script");
    s.src = "https://cdn.plot.ly/plotly-2.35.0.min.js";
    s.onload = function () {
      plotlyState = "ready";
      var q = plotlyQueue; plotlyQueue = [];
      q.forEach(function (f) { f(); });
    };
    s.onerror = function () {
      plotlyState = "idle";
      plotlyQueue.forEach(function (f) { f(); });
      plotlyQueue = [];
    };
    document.head.appendChild(s);
  }

  function plotlyChartEl(node, p) {
    var wrap = el("div", "sy-plotly-chart");
    if (p.height) wrap.style.height = p.height + "px";
    if (p.width) wrap.style.width = p.width + "px";
    wrap.textContent = "Loading Plotly chart…";
    loadPlotly(function () {
      if (window.Plotly) {
        var spec = p.spec || {};
        var data = spec.data || [];
        var layout = Object.assign({}, spec.layout || {});
        layout.autosize = true;
        if (!layout.margin) layout.margin = { l: 50, r: 30, t: 40, b: 40 };
        wrap.textContent = "";
        window.Plotly.newPlot(wrap, data, layout, {
          responsive: true,
          displayModeBar: true,
          displaylogo: false
        }).catch(function (e) {
          wrap.textContent = "Plotly error: " + (e.message || e);
        });
      } else {
        wrap.textContent = "Failed to load Plotly library.";
      }
    });
    return wrap;
  }

  // --- Pyplot Chart (st.pyplot equivalent) -----------------------------------

  function pyplotChartEl(node, p) {
    var wrap = el("div", "sy-pyplot-chart");
    if (p.height) wrap.style.height = p.height + "px";
    if (p.width) wrap.style.width = p.width + "px";
    var data = p.data || "";
    if (data.trimStart().startsWith("<svg")) {
      wrap.innerHTML = data;
      var svg = wrap.querySelector("svg");
      if (svg) {
        svg.style.maxWidth = "100%";
        svg.style.height = "auto";
      }
    } else {
      var img = document.createElement("img");
      img.className = "sy-image";
      if (data.startsWith("data:")) {
        img.src = data;
      } else {
        img.src = "data:image/png;base64," + data;
      }
      wrap.appendChild(img);
    }
    if (p.caption) wrap.appendChild(el("p", "sy-caption", p.caption));
    return wrap;
  }

  // --- Bokeh Chart (st.bokeh_chart equivalent) -------------------------------

  var bokehState = "idle";
  var bokehQueue = [];

  function loadBokeh(cb) {
    if (window.Bokeh) { cb(); return; }
    bokehQueue.push(cb);
    if (bokehState !== "idle") return;
    bokehState = "loading";
    var s = document.createElement("script");
    s.src = "https://cdn.bokeh.org/bokeh/release/bokeh-3.4.1.min.js";
    s.onload = function () {
      bokehState = "ready";
      var q = bokehQueue; bokehQueue = [];
      q.forEach(function (f) { f(); });
    };
    s.onerror = function () {
      bokehState = "idle";
      bokehQueue.forEach(function (f) { f(); });
      bokehQueue = [];
    };
    document.head.appendChild(s);
  }

  function bokehChartEl(node, p) {
    var wrap = el("div", "sy-bokeh-chart");
    if (p.height) wrap.style.height = p.height + "px";
    if (p.width) wrap.style.width = p.width + "px";
    wrap.textContent = "Loading Bokeh chart…";
    loadBokeh(function () {
      if (window.Bokeh) {
        wrap.textContent = "";
        try {
          window.Bokeh.embed.embed_item(p.spec || {}, wrap);
        } catch (e) {
          wrap.textContent = "Bokeh error: " + e.message;
        }
      } else {
        wrap.textContent = "Failed to load Bokeh library.";
      }
    });
    return wrap;
  }

  // --- PyDeck Chart (st.pydeck_chart equivalent, using deck.gl) --------------

  var deckState = "idle";
  var deckQueue = [];

  function loadDeckGL(cb) {
    if (window.deck) { cb(); return; }
    deckQueue.push(cb);
    if (deckState !== "idle") return;
    deckState = "loading";
    var scripts = [
      "https://unpkg.com/deck.gl@latest/dist.min.js",
      "https://api.mapbox.com/mapbox-gl-js/v3.1.2/mapbox-gl.js"
    ];
    var css = document.createElement("link");
    css.rel = "stylesheet";
    css.href = "https://api.mapbox.com/mapbox-gl-js/v3.1.2/mapbox-gl.css";
    document.head.appendChild(css);
    var idx = 0;
    function next() {
      if (idx >= scripts.length) {
        deckState = "ready";
        var q = deckQueue; deckQueue = [];
        q.forEach(function (f) { f(); });
        return;
      }
      var s = document.createElement("script");
      s.src = scripts[idx++];
      s.onload = next;
      s.onerror = function () {
        deckState = "idle";
        deckQueue.forEach(function (f) { f(); });
        deckQueue = [];
      };
      document.head.appendChild(s);
    }
    next();
  }

  function pydeckChartEl(node, p) {
    var wrap = el("div", "sy-pydeck-chart");
    var h = (p.height || 500);
    wrap.style.height = h + "px";
    if (p.width) wrap.style.width = p.width + "px";
    wrap.textContent = "Loading deck.gl…";

    loadDeckGL(function () {
      if (window.deck) {
        wrap.textContent = "";
        var spec = p.spec || {};
        var vs = spec.initialViewState || { latitude: 0, longitude: 0, zoom: 1 };

        try {
          new window.deck.DeckGL({
            container: wrap,
            initialViewState: vs,
            controller: true,
            layers: (spec.layers || []).map(function (l) {
              var typeName = l["@@type"] || "ScatterplotLayer";
              var LayerClass = window.deck[typeName];
              if (!LayerClass) return null;
              var layerProps = {};
              Object.keys(l).forEach(function (k) {
                if (k !== "@@type") layerProps[k] = l[k];
              });
              return new LayerClass(layerProps);
            }).filter(Boolean)
          });
        } catch (e) {
          wrap.textContent = "deck.gl error: " + e.message;
        }
      } else {
        wrap.textContent = "Failed to load deck.gl library.";
      }
    });
    return wrap;
  }

  connect();
})();
