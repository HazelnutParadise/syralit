// Syralit client runtime.
//
// Model: the server sends a full UI tree as JSON ({type:"ui_patch", nodes:[...]}).
// We render it into #syralit-app, reconciling by widget id so that the input the
// user is typing into keeps its focus and caret across reruns. Interactions send
// {type:"widget_change", widget_id, value, is_button} back over the WebSocket,
// which triggers a rerun on the server and another ui_patch in return.
//
// Multi-page: when the server includes "pages" and "active_page" in the
// ui_patch, we render a sidebar with page links. Each page has its own URL
// (/<slug>), so a page can be linked to, reloaded and reached with the back
// button. Clicking a link pushes that URL and sends {type:"page_change", page};
// the server reruns the target page. The transports carry only a query string,
// so the page in the address bar rides along as __sy_page on the socket URL —
// that is how a cold load of /reports renders reports instead of the first page.

(function () {
  var SY_BASE = window.__SY_BASE || "";
  // T resolves a built-in UI string against the [i18n] overrides.
  function T(key, fallback) {
    var m = window.__SY_I18N || {};
    return m[key] || fallback;
  }
  "use strict";

  var root = document.getElementById("syralit-app");
  var sidebar = document.getElementById("syralit-sidebar");
  var layoutRoot = document.getElementById("syralit-root");
  var ws;
  var lastPagesKey = "";
  var fragmentTimers = {}; // fragment key -> setInterval id (RunEvery)
  var usingSSE = false;    // true once we've fallen back to the SSE transport
  var wsEverOpened = false;
  var sessionId = "";      // SSE transport: correlates POSTs with the SSE stream

  // syAsset resolves a third-party library URL, honoring any override set via
  // sy.SetAssetURL (window.__SY_ASSETS) so libs can be self-hosted for offline.
  function syAsset(name, def) {
    return (window.__SY_ASSETS && window.__SY_ASSETS[name]) || def;
  }

  // Dispatch a server message, regardless of transport (WebSocket or SSE).
  function handleServerMsg(msg) {
    switch (msg.type) {
      case "ui_patch":
        hideOverlay();
        if (msg.pages || msg.sidebar) {
          renderSidebar(msg.pages || [], msg.active_page || "", msg.sidebar || []);
        }
        if (msg.page_config) applyPageConfig(msg.page_config);
        if (msg.set_query) applyQueryParams(msg.set_query);
        render(msg.nodes || []);
        if (msg.toasts) msg.toasts.forEach(handleToast);
        break;
      case "fragment_patch":
        patchFragment(msg.fragment_key, msg.nodes || []);
        if (msg.set_query) applyQueryParams(msg.set_query);
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
  }

  // --- Page URLs -------------------------------------------------------

  var knownPages = [];   // [{title, slug, icon}] from the last ui_patch

  // currentSlug reads the page slug out of the address bar: everything after
  // the mount prefix, with no slashes of its own. "" means the app root.
  function currentSlug() {
    var path = location.pathname;
    if (SY_BASE && path.indexOf(SY_BASE) === 0) path = path.slice(SY_BASE.length);
    return decodeURIComponent(path.replace(/^\/+|\/+$/g, ""));
  }

  function pageBySlug(slug) {
    for (var i = 0; i < knownPages.length; i++) {
      if (knownPages[i].slug === slug) return knownPages[i];
    }
    return null;
  }

  function pageURL(slug) {
    return SY_BASE + "/" + encodeURIComponent(slug) + (location.search || "");
  }

  // transportQuery appends the current page to the socket URL. The server strips
  // it before the app sees the query parameters.
  function transportQuery() {
    var qs = location.search || "";
    var slug = currentSlug();
    if (!slug) return qs;
    return qs + (qs ? "&" : "?") + "__sy_page=" + encodeURIComponent(slug);
  }

  // The back and forward buttons move between pages without a reload. An empty
  // slug is the app root, which is the first page in the sidebar.
  window.addEventListener("popstate", function () {
    var slug = currentSlug();
    var p = slug ? pageBySlug(slug) : knownPages[0];
    if (p) sendPageChange(p.title);
  });

  function connect() {
    var proto = location.protocol === "https:" ? "wss:" : "ws:";
    var qs = transportQuery();
    try {
      ws = new WebSocket(proto + "//" + location.host + SY_BASE + "/_syralit/ws" + qs);
    } catch (e) {
      startSSE();
      return;
    }
    ws.onopen = function () { wsEverOpened = true; };
    ws.onmessage = function (ev) { handleServerMsg(JSON.parse(ev.data)); };
    ws.onclose = function () {
      // Reconnect over WS if it ever worked; otherwise fall back to SSE (e.g. a
      // proxy that blocks WebSocket upgrades).
      if (wsEverOpened) setTimeout(connect, 1000);
      else startSSE();
    };
    ws.onerror = function () { /* onclose follows; handled there */ };
  }

  // SSE fallback: a plain-HTTP EventSource for downstream, POST for upstream.
  function startSSE() {
    if (usingSSE) return;
    usingSSE = true;
    ws = null;
    var es = new EventSource(SY_BASE + "/_syralit/sse" + transportQuery());
    es.addEventListener("session", function (e) { sessionId = e.data; });
    es.onmessage = function (e) { handleServerMsg(JSON.parse(e.data)); };
    // EventSource auto-reconnects; the server starts a fresh session on
    // reconnect, the same as a WebSocket reconnect.
  }

  // sendMsg routes a client frame to the active transport.
  function sendMsg(obj) {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(obj));
    } else if (usingSSE) {
      obj.session_id = sessionId;
      fetch(SY_BASE + "/_syralit/msg", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(obj),
        keepalive: true,
      });
    }
  }

  // --- Sidebar ---------------------------------------------------------

  function renderSidebar(pages, activePage, sidebarNodes) {
    layoutRoot.classList.add("has-sidebar");
    knownPages = pages;

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
          // A real href so the link can be copied, opened in a new tab or
          // middle-clicked; a plain click still navigates without a reload.
          a.href = p.slug ? pageURL(p.slug) : "#";
          if (p.icon) a.appendChild(el("span", "sy-sidebar-icon", p.icon));
          a.appendChild(document.createTextNode(p.title));
          if (p.title === activePage) a.classList.add("active");
          a.onclick = function (e) {
            if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return;
            e.preventDefault();
            if (p.slug) {
              try { history.pushState(null, "", pageURL(p.slug)); } catch (err) {}
            }
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
    btn.onclick = function () {
      if (layoutRoot.classList.contains("sidebar-collapsed")) {
        layoutRoot.classList.remove("sidebar-collapsed");
        return;
      }
      layoutRoot.classList.toggle("sidebar-open");
    };
    // Must live inside #syralit-root: the CSS that shows/hides it keys off
    // the root's has-sidebar / sidebar-collapsed classes with a descendant
    // selector. (position:fixed, so layout is unaffected.)
    layoutRoot.appendChild(btn);
  }

  function ensureBackdrop() {
    if (layoutRoot.querySelector(".sy-sidebar-backdrop")) return;
    var bd = document.createElement("div");
    bd.className = "sy-sidebar-backdrop";
    bd.onclick = function () { layoutRoot.classList.remove("sidebar-open"); };
    layoutRoot.appendChild(bd);
  }

  function sendPageChange(page) {
    sendMsg({ type: "page_change", page: page });
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
    sendMsg({
      type: "widget_change",
      widget_id: widgetID,
      value: value,
      is_button: !!isButton,
    });
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
      if (inp.dataset.multi === "dateslider") {
        var dayMs = parseInt(inp.value, 10) * 86400000;
        changes.push({ widget_id: id, value: new Date(dayMs).toISOString().slice(0, 10) });
        return;
      }
      if (inp.dataset.multi === "timeslider") {
        var tm = parseInt(inp.value, 10), th = Math.floor(tm / 60), tmm = tm % 60;
        changes.push({ widget_id: id, value: (th < 10 ? "0" : "") + th + ":" + (tmm < 10 ? "0" : "") + tmm });
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
    sendMsg({
      type: "form_submit",
      widget_id: submitId,
      changes: changes,
    });
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

    patchChildren(root, nodes.map(buildNode));

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

  // Replace parent's children with `next`, keeping any element that is already
  // a child of parent (a reused Embed) attached the whole time — detaching and
  // re-inserting it would reload every iframe the embed created. Elements not
  // in `next` are removed first, then each new element is inserted before
  // whatever currently sits at its index.
  function patchChildren(parent, next) {
    Array.prototype.slice.call(parent.childNodes).forEach(function (c) {
      if (next.indexOf(c) === -1) parent.removeChild(c);
    });
    next.forEach(function (n, i) {
      var cur = parent.childNodes[i];
      if (cur !== n) parent.insertBefore(n, cur || null);
    });
    pruneEmbeds();
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
      case "date_slider":   return dateSlider(node, p);
      case "time_slider":   return timeSlider(node, p);
      case "textarea":      return textarea(node, p);
      case "radio":         return radio(node, p);
      case "multi_select":  return multiSelect(node, p);
      case "date_input":    return dateInput(node, p);
      case "datetime_input": return datetimeInput(node, p);
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
      case "space":     return spaceEl(node, p);
      case "bottom":    return bottomEl(node);
      case "artifact_canvas": return artifactCanvasEl(node, p);
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
      case "embed":      return embedEl(node, p);
      case "component":  return componentEl(node, p);
      case "iframe":     return iframeEl(node, p);
      case "pdf":        return pdfEl(node, p);
      case "menu_button": return menuButtonEl(node, p);
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
    input.className = "sy-input" + (p.mono ? " sy-input-mono" : "");
    input.dataset.id = node.id;
    input.value = p.value || "";
    if (p.placeholder) input.placeholder = p.placeholder;
    if (p.disabled) input.disabled = true;
    if (p.max_chars) input.maxLength = p.max_chars;
    input.oninput = function () {
      if (!inForm(input)) send(node.id, input.value, false);
    };
    if (p.formula) {
      // Formula-bar look: fx marker inside the box, input borderless within.
      var wrap = el("div", "sy-formula-input");
      wrap.appendChild(el("span", "sy-formula-fx", "ƒx"));
      input.classList.add("sy-formula-field");
      wrap.appendChild(input);
      return field(p.label, wrap, p.help, p.label_visibility);
    }
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
    if (p.help) b.title = p.help;
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
    if (p.accept_new && !p.disabled && !atLimit) {
      var newInput = document.createElement("input");
      newInput.type = "text";
      newInput.className = "sy-input sy-multi-new";
      newInput.placeholder = T("add_new", "Add new…");
      newInput.onkeydown = function (ev) {
        if (ev.key !== "Enter") return;
        var v = newInput.value.trim();
        if (!v || vals.indexOf(v) >= 0) return;
        newInput.value = "";
        var next = vals.concat([v]);
        if (!inForm(wrap)) send(node.id, next, false);
      };
      wrap.appendChild(newInput);
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

  function datetimeInput(node, p) {
    var input = document.createElement("input");
    input.type = "datetime-local";
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
    summary.textContent = p.icon ? (p.icon + " " + p.label) : p.label;
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

  // toggleChoice computes the next value for a single/multi choice widget and
  // sends it: multi toggles membership in an array, single sends the option.
  function toggleChoice(node, p, opt) {
    if (p.multi) {
      var cur = p.value || [];
      var next = cur.indexOf(opt) >= 0 ? cur.filter(function (x) { return x !== opt; }) : cur.concat([opt]);
      send(node.id, next, false);
    } else {
      send(node.id, opt, false);
    }
  }
  function isChoiceActive(p, opt) {
    return p.multi ? (p.value || []).indexOf(opt) >= 0 : opt === p.value;
  }

  function segmentedControlEl(node, p) {
    var wrap = el("div", "sy-segmented-control");
    (p.options || []).forEach(function (opt) {
      var btn = el("button", "sy-segmented-btn" + (isChoiceActive(p, opt) ? " sy-segmented-active" : ""), opt);
      btn.disabled = p.disabled;
      btn.onclick = function () { toggleChoice(node, p, opt); };
      wrap.appendChild(btn);
    });
    return field(p.label, wrap, p.help, p.label_visibility);
  }

  function pillsEl(node, p) {
    var wrap = el("div", "sy-pills");
    (p.options || []).forEach(function (opt) {
      var pill = el("button", "sy-pill" + (isChoiceActive(p, opt) ? " sy-pill-active" : ""), opt);
      pill.disabled = p.disabled;
      pill.onclick = function () { toggleChoice(node, p, opt); };
      wrap.appendChild(pill);
    });
    return field(p.label, wrap, p.help, p.label_visibility);
  }

  function feedbackEl(node, p) {
    var wrap = el("div", "sy-feedback");
    var current = p.value || "";
    var disabled = !!p.disabled;
    var style = p.style || "thumbs";

    function makeBtn(type, emoji, active) {
      var btn = document.createElement("button");
      btn.className = "sy-feedback-btn" + (active ? " sy-feedback-active" : "");
      btn.textContent = emoji;
      btn.disabled = disabled;
      btn.onclick = function () {
        var next = current === type ? "" : type;
        send(node.id, next, false);
      };
      return btn;
    }

    if (style === "stars") {
      // ★ rating 1..5; clicking a star selects that rating (filled up to it).
      var cur = parseInt(current, 10) || 0;
      for (var s = 1; s <= 5; s++) {
        (function (rating) {
          var b = makeBtn(String(rating), rating <= cur ? "★" : "☆", false);
          b.classList.add("sy-feedback-star");
          if (rating <= cur) b.classList.add("sy-feedback-active");
          wrap.appendChild(b);
        })(s);
      }
    } else if (style === "faces") {
      var faces = ["😞", "🙁", "😐", "🙂", "😀"];
      for (var fi = 0; fi < faces.length; fi++) {
        (function (rating) {
          wrap.appendChild(makeBtn(String(rating), faces[rating - 1], current === String(rating)));
        })(fi + 1);
      }
    } else {
      wrap.appendChild(makeBtn("up", "👍", current === "up"));
      wrap.appendChild(makeBtn("down", "👎", current === "down"));
    }
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
    var target = root.querySelector('[data-stream-id="' + id + '"]');
    if (!target) return;
    target.textContent += chunk;
  }

  function patchFragment(key, nodes) {
    var target = root.querySelector('[data-fragment-key="' + key + '"]');
    if (!target) return;
    patchChildren(target, nodes.map(buildNode));
  }

  function fragmentEl(node) {
    var p = node.props || {};
    var key = p.key || "";
    var div = el("div", "sy-fragment");
    div.setAttribute("data-fragment-key", key);
    childNodes(node).forEach(function (c) { div.appendChild(c); });
    // RunEvery: poll the server to re-run just this fragment on an interval.
    // Reset any prior timer for this key (a full re-render recreates the div).
    if (fragmentTimers[key]) { clearInterval(fragmentTimers[key]); delete fragmentTimers[key]; }
    if (p.run_every > 0) {
      fragmentTimers[key] = setInterval(function () {
        sendMsg({ type: "fragment_rerun", fragment_key: key });
      }, p.run_every);
    }
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

  function spaceEl(node, p) {
    var div = el("div", "sy-space");
    div.style.height = (p.height || 16) + "px";
    if (p.width) { div.style.display = "inline-block"; div.style.width = p.width + "px"; }
    return div;
  }

  function bottomEl(node) {
    var bar = el("div", "sy-bottom");
    childNodes(node).forEach(function (c) { bar.appendChild(c); });
    // Reserve space in the main area so content never hides behind the bar.
    // Deferred: the node is appended to the DOM after this function returns.
    setTimeout(function () {
      var app = document.getElementById("syralit-app");
      if (app && bar.isConnected && bar.offsetHeight) {
        app.style.paddingBottom = (bar.offsetHeight + 24) + "px";
      }
    }, 50);
    return bar;
  }

  function artifactCanvasEl(node, p) {
    var id = node.id || ("artifact:" + (p.name || "default"));
    var wrap = root.querySelector('[data-artifact-id="' + cssEscape(id) + '"]');
    var isNewCanvas = !wrap;
    if (!wrap) {
      wrap = el("section", "sy-artifact-canvas");
      wrap.setAttribute("data-artifact-id", id);
      var body = el("div", "sy-artifact-grid");
      wrap.appendChild(body);
    }
    wrap.setAttribute("data-artifact-name", p.name || "");
    if (p.height) wrap.style.minHeight = p.height + "px";
    if (p.width) wrap.style.maxWidth = p.width + "px";

    var layout = p.layout || {};
    var grid = wrap.querySelector(".sy-artifact-grid");
    var cols = Math.max(1, Number(layout.columns || 1));
    grid.style.gridTemplateColumns = "repeat(" + cols + ", minmax(0, 1fr))";
    if (layout.gap) grid.style.gap = layout.gap + "px";
    else grid.style.gap = "";
    if (layout.padding) grid.style.padding = layout.padding + "px";
    else grid.style.padding = "";

    var revision = String(p.revision || 0);
    var revisionChanged = isNewCanvas || wrap.dataset.artifactRevision !== revision;
    if (revisionChanged) {
      if (typeof wrap.getAnimations === "function") {
        try {
          wrap.getAnimations({ subtree: true }).forEach(function (animation) {
            animation.cancel();
          });
        } catch (e) {}
      }
      wrap.dataset.artifactRevision = revision;
      wrap.dataset.artifactState = "transitioning";
      wrap.dataset.artifactReadiness = "pending";
      var token = String((Number(wrap.dataset.artifactTransitionToken || 0) + 1));
      wrap.dataset.artifactTransitionToken = token;
      var transition = reconcileArtifactChildren(grid, node.children || []);
      settleArtifactCanvas(wrap, p.name || "", id, revision, token, transition);
    }
    return wrap;
  }

  function reconcileArtifactChildren(grid, children) {
    var existing = {};
    var oldRects = {};
    var animations = [];
    var oldGridHeight = grid.getBoundingClientRect().height;
    Array.prototype.forEach.call(grid.children, function (el) {
      var id = el.getAttribute("data-artifact-node-id");
      if (id) {
        existing[id] = el;
        oldRects[id] = el.getBoundingClientRect();
      }
    });

    children.forEach(function (child) {
      var id = child.id || "";
      if (!id) return;
      var item = existing[id];
      if (!item) {
        item = el("div", "sy-artifact-item");
        item.setAttribute("data-artifact-node-id", id);
      }

      applyArtifactItemLayout(item, child.props || {});
      item.replaceChildren(buildNode(child));
      grid.appendChild(item);
      delete existing[id];
    });

    Object.keys(existing).forEach(function (id) {
      var item = existing[id];
      var rect = oldRects[id];
      var gridRect = grid.getBoundingClientRect();
      var ghost = item.cloneNode(true);
      copyArtifactCanvases(item, ghost);
      ghost.removeAttribute("data-artifact-node-id");
      ghost.classList.add("sy-artifact-ghost");
      ghost.style.left = (rect.left - gridRect.left) + "px";
      ghost.style.top = (rect.top - gridRect.top) + "px";
      ghost.style.width = rect.width + "px";
      ghost.style.height = rect.height + "px";
      item.remove();
      grid.appendChild(ghost);
      animations.push(runArtifactAnimation(ghost, [
        { opacity: 1, transform: "translateY(0) scale(1)" },
        { opacity: 0, transform: "translateY(-10px) scale(.975)" },
      ], 280, function () { ghost.remove(); }));
    });

    return nextArtifactFrame().then(function () {
      Array.prototype.forEach.call(grid.children, function (item) {
        var id = item.getAttribute("data-artifact-node-id");
        if (!id) return;
        var oldRect = oldRects[id];
        if (!oldRect) {
          animations.push(runArtifactAnimation(item, [
            { opacity: 0, transform: "translateY(14px) scale(.975)" },
            { opacity: 1, transform: "translateY(0) scale(1)" },
          ], 460));
          return;
        }
        var nextRect = item.getBoundingClientRect();
        var dx = oldRect.left - nextRect.left;
        var dy = oldRect.top - nextRect.top;
        var sx = nextRect.width ? oldRect.width / nextRect.width : 1;
        var sy = nextRect.height ? oldRect.height / nextRect.height : 1;
        animations.push(runArtifactAnimation(item, [
          {
            opacity: .72,
            transform: "translate(" + dx + "px," + dy + "px) scale(" + sx + "," + sy + ")",
            boxShadow: "0 0 0 0 color-mix(in srgb, var(--sy-accent) 28%, transparent)",
          },
          {
            opacity: 1,
            transform: "translate(0,0) scale(1,1)",
            boxShadow: "0 0 0 14px transparent",
          },
        ], 520));
      });

      var nextGridHeight = grid.getBoundingClientRect().height;
      if (Math.abs(oldGridHeight - nextGridHeight) > 1) {
        animations.push(runArtifactAnimation(grid, [
          { minHeight: oldGridHeight + "px" },
          { minHeight: nextGridHeight + "px" },
        ], 460));
      }
      return Promise.all(animations);
    });
  }

  function copyArtifactCanvases(source, clone) {
    var sourceCanvases = source.querySelectorAll("canvas");
    var cloneCanvases = clone.querySelectorAll("canvas");
    Array.prototype.forEach.call(sourceCanvases, function (canvas, i) {
      var target = cloneCanvases[i];
      if (!target) return;
      target.width = canvas.width;
      target.height = canvas.height;
      try { target.getContext("2d").drawImage(canvas, 0, 0); } catch (e) {}
    });
  }

  function runArtifactAnimation(target, frames, duration, done) {
    if (!target || typeof target.animate !== "function" ||
        window.matchMedia("(prefers-reduced-motion: reduce)").matches) {
      if (done) done();
      return Promise.resolve();
    }
    var animation = target.animate(frames, {
      duration: duration,
      easing: "cubic-bezier(.22,1,.36,1)",
      fill: "both",
    });
    return animation.finished.catch(function () {}).then(function () {
      animation.cancel();
      if (done) done();
    });
  }

  function nextArtifactFrame() {
    return new Promise(function (resolve) {
      requestAnimationFrame(function () { requestAnimationFrame(resolve); });
    });
  }

  function settleArtifactCanvas(wrap, name, canvasID, revision, token, transition) {
    var readiness = Promise.resolve(transition)
      .then(function () { return waitForArtifactResources(wrap); })
      .then(function (result) {
        return nextArtifactFrame().then(function () { return result; });
      });
    Promise.race([
      readiness,
      new Promise(function (resolve) {
        setTimeout(function () { resolve("timeout"); }, 6000);
      }),
    ]).then(function (result) {
      if (wrap.dataset.artifactTransitionToken !== token) return;
      wrap.dataset.artifactReadiness = result;
      wrap.dataset.artifactState = "settled";
      wrap.dispatchEvent(new CustomEvent("syralit:artifact-settled", {
        bubbles: true,
        detail: {
          artifact: name,
          canvas_id: canvasID,
          revision: Number(revision),
          readiness: result,
        },
      }));
    });
  }

  function waitForArtifactResources(wrap) {
    var pending = [];
    Array.prototype.forEach.call(wrap.querySelectorAll("img"), function (img) {
      if (img.complete) {
        if (!img.naturalWidth) {
          pending.push(Promise.resolve(false));
        } else if (typeof img.decode === "function") {
          pending.push(img.decode().then(function () { return true; }).catch(function () { return false; }));
        }
        return;
      }
      pending.push(new Promise(function (resolve) {
        img.addEventListener("load", function () { resolve(true); }, { once: true });
        img.addEventListener("error", function () { resolve(false); }, { once: true });
      }));
    });
    Array.prototype.forEach.call(wrap.querySelectorAll("[data-chart-state]"), function (chart) {
      if (chart.dataset.chartState === "settled") {
        pending.push(Promise.resolve(true));
        return;
      }
      if (chart.dataset.chartState === "error") {
        pending.push(Promise.resolve(false));
        return;
      }
      pending.push(new Promise(function (resolve) {
        chart.addEventListener("syralit:chart-settled", function () {
          resolve(chart.dataset.chartState === "settled");
        }, { once: true });
      }));
    });
    if (document.fonts && document.fonts.ready) {
      pending.push(document.fonts.ready.then(function () { return true; }).catch(function () { return false; }));
    }
    return Promise.all(pending).then(function (results) {
      return results.every(function (ok) { return ok; }) ? "complete" : "partial";
    });
  }

  function applyArtifactItemLayout(item, props) {
    var layout = props.artifact_layout || {};
    if (layout.column_span) item.style.gridColumn = "span " + Math.max(1, Number(layout.column_span));
    else item.style.gridColumn = "";
    if (layout.row_span) item.style.gridRow = "span " + Math.max(1, Number(layout.row_span));
    else item.style.gridRow = "";
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
    if (p.help) wrap.title = p.help;
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
    link.href = syAsset("highlight_css", "https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@11.11.1/build/styles/github.min.css");
    var linkDark = document.createElement("link");
    linkDark.rel = "stylesheet";
    linkDark.media = "(prefers-color-scheme: dark)";
    linkDark.href = syAsset("highlight_css_dark", "https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@11.11.1/build/styles/github-dark.min.css");
    document.head.appendChild(link);
    document.head.appendChild(linkDark);
    var script = document.createElement("script");
    script.src = syAsset("highlight_js", "https://cdn.jsdelivr.net/gh/highlightjs/cdn-release@11.11.1/build/highlight.min.js");
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
    if (p.wrap) pre.classList.add("sy-code-wrapped");
    var code = document.createElement("code");
    if (p.language) code.className = "language-" + p.language;
    code.textContent = p.code;
    pre.appendChild(code);

    if (p.line_numbers) {
      var row = el("div", "sy-code-row");
      var gutter = el("div", "sy-code-gutter");
      var lines = String(p.code).replace(/\n$/, "").split("\n");
      lines.forEach(function (_, i) { gutter.appendChild(el("span", null, String(i + 1))); });
      row.appendChild(gutter);
      row.appendChild(pre);
      wrap.appendChild(row);
    } else {
      wrap.appendChild(pre);
    }

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
    if (p.containerWidth) img.style.width = "100%";
    else if (p.width) img.style.maxWidth = p.width + "px";
    wrap.appendChild(img);
    if (p.caption) wrap.appendChild(el("p", "sy-caption", p.caption));
    return wrap;
  }

  function jsonView(node, p) {
    var wrap = el("div", "sy-json");
    var data;
    try { data = JSON.parse(p.data); }
    catch (e) { wrap.textContent = p.data; return wrap; }
    wrap.appendChild(buildJsonNode(data, null, p.expanded !== false));
    return wrap;
  }

  function jsonPrimitive(v) {
    var span = document.createElement("span");
    if (v === null) { span.className = "sy-json-null"; span.textContent = "null"; }
    else if (typeof v === "string") { span.className = "sy-json-str"; span.textContent = JSON.stringify(v); }
    else if (typeof v === "number") { span.className = "sy-json-num"; span.textContent = String(v); }
    else if (typeof v === "boolean") { span.className = "sy-json-bool"; span.textContent = String(v); }
    else { span.textContent = String(v); }
    return span;
  }

  function jsonKey(k) {
    var s = el("span", "sy-json-key", (typeof k === "number" ? k : JSON.stringify(k)) + ": ");
    return s;
  }

  // buildJsonNode renders a value as a collapsible tree row. Objects/arrays get
  // a disclosure toggle; expand cascades to descendants.
  function buildJsonNode(value, keyLabel, expand) {
    var isArr = Array.isArray(value);
    var isObj = value !== null && typeof value === "object";
    if (!isObj) {
      var leaf = el("div", "sy-json-row");
      if (keyLabel !== null) leaf.appendChild(jsonKey(keyLabel));
      leaf.appendChild(jsonPrimitive(value));
      return leaf;
    }
    var keys = isArr ? value.map(function (_, i) { return i; }) : Object.keys(value);
    var open = isArr ? "[" : "{", close = isArr ? "]" : "}";

    var box = el("div", "sy-json-node");
    var head = el("div", "sy-json-head");
    var toggle = el("span", "sy-json-toggle", expand ? "▾" : "▸");
    head.appendChild(toggle);
    if (keyLabel !== null) head.appendChild(jsonKey(keyLabel));
    head.appendChild(el("span", "sy-json-punct", open));
    var summary = el("span", "sy-json-count", keys.length + (isArr ? " items" : " keys"));
    head.appendChild(summary);

    var children = el("div", "sy-json-children");
    if (!expand) children.style.display = "none";
    keys.forEach(function (k) { children.appendChild(buildJsonNode(value[k], k, expand)); });
    var tail = el("div", "sy-json-punct sy-json-tail", close);
    if (!expand) tail.style.display = "none";

    head.onclick = function () {
      var shown = children.style.display !== "none";
      children.style.display = shown ? "none" : "";
      tail.style.display = shown ? "none" : "";
      toggle.textContent = shown ? "▸" : "▾";
      summary.style.display = shown ? "" : "none";
    };
    if (expand) summary.style.display = "none";

    box.appendChild(head);
    box.appendChild(children);
    box.appendChild(tail);
    return box;
  }

  function progressBar(node, p) {
    var wrap = el("div", "sy-progress-wrap");
    if (p.text) wrap.appendChild(el("div", "sy-progress-text", p.text));
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

  // applyNumFormat renders a number through a printf-style format string
  // (a useful subset: %d, %f, %.Nf, %e, %g, and literal %%), e.g. "$%.2f".
  function applyNumFormat(fmt, value) {
    if (!fmt) return String(value);
    return fmt.replace(/%(\.\d+)?([dfeg%])/g, function (m, prec, verb) {
      if (verb === "%") return "%";
      var n = parseFloat(value);
      if (isNaN(n)) return String(value);
      var p = prec ? parseInt(prec.slice(1), 10) : (verb === "d" ? 0 : 6);
      if (verb === "d") return String(Math.round(n));
      if (verb === "f") return n.toFixed(p);
      if (verb === "e") return n.toExponential(prec ? p : 6);
      return String(n); // g
    });
  }

  // sparkline draws a tiny inline bar/line chart from an array of numbers, for
  // the "bar_chart" / "line_chart" column types (st.column_config equivalents).
  function sparkline(values, type, color) {
    var w = 84, h = 24, pad = 2, ns = "http://www.w3.org/2000/svg";
    var svg = document.createElementNS(ns, "svg");
    svg.setAttribute("width", w);
    svg.setAttribute("height", h);
    svg.setAttribute("class", "sy-spark");
    var nums = (Array.isArray(values) ? values : []).map(function (v) { return parseFloat(v) || 0; });
    if (!nums.length) return svg;
    var min = Math.min.apply(null, nums), max = Math.max.apply(null, nums), range = (max - min) || 1;
    var c = (color || "var(--sy-accent)");
    function x(i) { return pad + (nums.length === 1 ? (w - 2 * pad) / 2 : i * (w - 2 * pad) / (nums.length - 1)); }
    function y(v) { return h - pad - (v - min) / range * (h - 2 * pad); }
    if (type === "line_chart" || type === "area_chart") {
      var pts = nums.map(function (v, i) { return x(i) + "," + y(v); }).join(" ");
      if (type === "area_chart") {
        var pg = document.createElementNS(ns, "polygon");
        pg.setAttribute("points", x(0) + "," + (h - pad) + " " + pts + " " + x(nums.length - 1) + "," + (h - pad));
        pg.setAttribute("fill", c);
        pg.setAttribute("opacity", "0.25");
        svg.appendChild(pg);
      }
      var pl = document.createElementNS(ns, "polyline");
      pl.setAttribute("points", pts);
      pl.setAttribute("fill", "none");
      pl.setAttribute("stroke", c);
      pl.setAttribute("stroke-width", "1.5");
      svg.appendChild(pl);
    } else {
      var bw = Math.max(1, (w - 2 * pad) / nums.length * 0.7);
      var base = h - pad;
      nums.forEach(function (v, i) {
        var r = document.createElementNS(ns, "rect");
        var yy = y(v);
        r.setAttribute("x", x(i) - bw / 2);
        r.setAttribute("y", yy);
        r.setAttribute("width", bw);
        r.setAttribute("height", Math.max(0, base - yy));
        r.setAttribute("fill", c);
        svg.appendChild(r);
      });
    }
    return svg;
  }

  // renderReadonlyCell formats a non-editable cell by its column type. Shared by
  // DataFrame (column_config display) and DataEditor's disabled mode.
  function renderReadonlyCell(td, cell, colType, cfg) {
    cfg = cfg || {};
    if (colType === "checkbox") {
      td.textContent = cell ? "✓" : "✗";
    } else if (colType === "link") {
      var a = document.createElement("a");
      a.href = cell || "#";
      a.textContent = cfg.format ? applyNumFormat(cfg.format, cell) : (cell || "");
      a.target = "_blank";
      a.rel = "noopener noreferrer";
      td.appendChild(a);
    } else if (colType === "image") {
      var img = document.createElement("img");
      img.src = cell || "";
      img.style.maxHeight = "40px";
      td.appendChild(img);
    } else if (colType === "progress") {
      var bar = el("div", "sy-progress-bar sy-progress-cell");
      var fill = el("div", "sy-progress-fill");
      var pctMax = cfg.max || 100;
      fill.style.width = Math.min(Math.max((parseFloat(cell) || 0) / pctMax, 0), 1) * 100 + "%";
      bar.appendChild(fill);
      td.appendChild(bar);
      if (cfg.format) { var lbl = el("span", "sy-progress-label", applyNumFormat(cfg.format, cell)); td.appendChild(lbl); }
    } else if (colType === "list") {
      td.textContent = Array.isArray(cell) ? cell.join(", ") : String(cell == null ? "" : cell);
    } else if (colType === "bar_chart" || colType === "line_chart" || colType === "area_chart") {
      td.appendChild(sparkline(cell, colType, cfg.color));
    } else if (colType === "json") {
      var codeEl = document.createElement("code");
      codeEl.className = "sy-json-cell";
      try { codeEl.textContent = typeof cell === "string" ? cell : JSON.stringify(cell); }
      catch (e) { codeEl.textContent = String(cell); }
      td.appendChild(codeEl);
    } else if (colType === "number" && cfg.format) {
      td.textContent = applyNumFormat(cfg.format, cell);
    } else {
      td.textContent = cell == null ? "" : String(cell);
    }
  }

  function dataframeEl(node, p) {
    var headers = p.headers || [];
    var colCfg = p.column_config || {};
    // column_order reorders and filters columns; colIdx maps display position
    // back to the original column index.
    var colIdx = headers.map(function (_, i) { return i; });
    if (p.column_order && p.column_order.length) {
      colIdx = p.column_order.map(function (h) { return headers.indexOf(h); })
        .filter(function (i) { return i >= 0; });
    }
    var singleRow = p.selection_mode === "single-row";
    var colMode = p.selection_mode === "single-column" || p.selection_mode === "multi-column";
    var singleCol = p.selection_mode === "single-column";
    var selectable = !!p.selectable;
    var selected = {};
    (p.selected || []).forEach(function (i) { selected[i] = true; });
    // Pair each row with its original index so selection is stable across sorts.
    var indexed = (p.rows || []).map(function (r, i) { return { r: r, i: i }; });
    var sortCol = -1, sortAsc = true;

    var wrap = el("div", "sy-dataframe-wrap");
    if (p.height) wrap.style.maxHeight = p.height + "px";
    wrap.style.overflowY = "auto";

    function sendSelection() {
      var ids = Object.keys(selected).filter(function (k) { return selected[k]; })
        .map(Number).sort(function (a, b) { return a - b; });
      send(node.id, ids, false);
    }

    function rebuild() {
      var t = document.createElement("table");
      t.className = "sy-table sy-dataframe";
      var thead = document.createElement("thead");
      var tr = document.createElement("tr");
      if (selectable && !colMode) tr.appendChild(el("th", "sy-df-header sy-df-select"));
      colIdx.forEach(function (ci) {
        var h = headers[ci];
        var hcfg = colCfg[h] || {};
        var label = hcfg.label || h;
        var th = el("th", "sy-df-header", label + (!colMode && ci === sortCol ? (sortAsc ? " ▲" : " ▼") : ""));
        if (hcfg.help) th.title = hcfg.help;
        if (hcfg.width) th.style.width = hcfg.width + "px";
        if (selectable && colMode) {
          if (selected[ci]) th.classList.add("sy-df-col-selected");
          th.onclick = function () {
            if (singleCol) {
              var was = !!selected[ci];
              selected = {};
              if (!was) selected[ci] = true;
            } else {
              selected[ci] = !selected[ci];
            }
            sendSelection();
            wrap.replaceChildren();
            rebuild();
          };
          tr.appendChild(th);
          return;
        }
        th.onclick = function () {
          if (sortCol === ci) { sortAsc = !sortAsc; } else { sortCol = ci; sortAsc = true; }
          indexed.sort(function (a, b) {
            var va = a.r[ci], vb = b.r[ci];
            var na = parseFloat(va), nb = parseFloat(vb);
            if (!isNaN(na) && !isNaN(nb)) return sortAsc ? na - nb : nb - na;
            return sortAsc ? String(va).localeCompare(String(vb)) : String(vb).localeCompare(String(va));
          });
          wrap.replaceChildren();
          rebuild();
        };
        tr.appendChild(th);
      });
      thead.appendChild(tr);
      t.appendChild(thead);

      var tbody = document.createElement("tbody");
      indexed.forEach(function (item) {
        var row = item.r, oi = item.i;
        var tr2 = document.createElement("tr");
        if (selectable && !colMode) {
          if (selected[oi]) tr2.className = "sy-df-row-selected";
          var selTd = el("td", "sy-df-select");
          var cb = document.createElement("input");
          cb.type = "checkbox";
          cb.checked = !!selected[oi];
          function toggle(checked) {
            if (singleRow && checked) {
              selected = {};
              // Clear every other row's visual state on next rebuild; do it
              // live for the current table too.
              [].forEach.call(tbody.querySelectorAll("tr"), function (row) {
                row.classList.remove("sy-df-row-selected");
                var box = row.querySelector(".sy-df-select input");
                if (box) box.checked = false;
              });
            }
            selected[oi] = checked;
            cb.checked = checked;
            tr2.classList.toggle("sy-df-row-selected", checked);
            sendSelection();
          }
          cb.onclick = function (e) { e.stopPropagation(); };
          cb.onchange = function () { toggle(cb.checked); };
          selTd.appendChild(cb);
          tr2.appendChild(selTd);
          tr2.onclick = function () { toggle(!cb.checked); };
        }
        colIdx.forEach(function (ci) {
          var cell = (row || [])[ci];
          var td = document.createElement("td");
          if (colMode && selected[ci]) td.classList.add("sy-df-col-selected");
          var cfg = colCfg[headers[ci]] || {};
          renderReadonlyCell(td, cell, cfg.type || "text", cfg);
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

  function dateSlider(node, p) {
    function toDays(s) { return Math.round(new Date(s + "T00:00:00Z").getTime() / 86400000); }
    function fromDays(n) { return new Date(n * 86400000).toISOString().slice(0, 10); }
    var minN = toDays(p.min), maxN = toDays(p.max);
    var curN = toDays(p.value || p.min);

    var wrap = el("div", "sy-slider-wrap");
    var input = document.createElement("input");
    input.type = "range";
    input.className = "sy-slider";
    input.dataset.id = node.id;
    input.dataset.multi = "dateslider"; // submitForm converts day-offset -> date
    input.min = minN; input.max = maxN; input.step = 1; input.value = curN;
    if (p.disabled) input.disabled = true;
    var display = el("span", "sy-slider-value", fromDays(curN));
    input.oninput = function () { display.textContent = fromDays(parseInt(input.value, 10)); };
    input.onchange = function () {
      if (!inForm(input)) send(node.id, fromDays(parseInt(input.value, 10)), false);
    };
    wrap.appendChild(input);
    wrap.appendChild(display);
    return field(p.label, wrap, p.help);
  }

  function timeSlider(node, p) {
    function toMin(s) { var a = (s || "00:00").split(":"); return (parseInt(a[0], 10) || 0) * 60 + (parseInt(a[1], 10) || 0); }
    function fromMin(m) { var h = Math.floor(m / 60), mm = m % 60; return (h < 10 ? "0" : "") + h + ":" + (mm < 10 ? "0" : "") + mm; }
    var minN = toMin(p.min), maxN = toMin(p.max), curN = toMin(p.value || p.min);
    var step = p.step || 15;

    var wrap = el("div", "sy-slider-wrap");
    var input = document.createElement("input");
    input.type = "range";
    input.className = "sy-slider";
    input.dataset.id = node.id;
    input.dataset.multi = "timeslider"; // submitForm converts minute-offset -> HH:MM
    input.min = minN; input.max = maxN; input.step = step; input.value = curN;
    if (p.disabled) input.disabled = true;
    var display = el("span", "sy-slider-value", fromMin(curN));
    input.oninput = function () { display.textContent = fromMin(parseInt(input.value, 10)); };
    input.onchange = function () {
      if (!inForm(input)) send(node.id, fromMin(parseInt(input.value, 10)), false);
    };
    wrap.appendChild(input);
    wrap.appendChild(display);
    return field(p.label, wrap, p.help);
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
      var cfg = colCfg[h] || {};
      th.textContent = cfg.label || h;
      if (cfg.help) th.title = cfg.help;
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
          renderReadonlyCell(td, cell, colType, cfg);
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
            if (cfg.step !== undefined) inp.step = cfg.step;
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
    link.href = syAsset("katex_css", "https://cdn.jsdelivr.net/npm/katex@0.16.18/dist/katex.min.css");
    document.head.appendChild(link);
    var script = document.createElement("script");
    script.src = syAsset("katex_js", "https://cdn.jsdelivr.net/npm/katex@0.16.18/dist/katex.min.js");
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
    css.href = syAsset("leaflet_css", "https://unpkg.com/leaflet@1.9.4/dist/leaflet.css");
    document.head.appendChild(css);
    var js = document.createElement("script");
    js.src = syAsset("leaflet_js", "https://unpkg.com/leaflet@1.9.4/dist/leaflet.js");
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
      var map = window.L.map(wrap).setView(center, p.zoom || 12);
      window.L.tileLayer("https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png", {
        attribution: "&copy; OpenStreetMap"
      }).addTo(map);
      var bounds = [];
      pts.forEach(function (pt) {
        var m;
        if (pt.size || pt.color) {
          m = window.L.circleMarker([pt.lat, pt.lon], {
            radius: pt.size || 6,
            color: pt.color || "#7c3aed",
            fillColor: pt.color || "#7c3aed",
            fillOpacity: 0.6,
            weight: 1,
          }).addTo(map);
        } else {
          m = window.L.marker([pt.lat, pt.lon]).addTo(map);
        }
        if (pt.text) m.bindPopup(pt.text);
        bounds.push([pt.lat, pt.lon]);
      });
      // An explicit zoom takes precedence over auto-fitting the bounds.
      if (bounds.length > 1 && !p.zoom) {
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

  // --- Embed (third-party markup + scripts) -----------------------------

  // id -> {html, el}. An embed is built once per id; later reruns with the
  // same html hand back the same element so its scripts run only once and
  // whatever the widget mounted survives. pruneEmbeds drops entries whose
  // element left the document.
  var embeds = {};

  function embedEl(node, p) {
    var html = p.html || "";
    var hit = embeds[node.id];
    if (hit && hit.html === html) return hit.el;
    var div = el("div", "sy-embed");
    div.innerHTML = html;
    // <script> inserted via innerHTML never executes; recreate each one in
    // place so the browser runs it once the element joins the document.
    // async=false keeps external scripts in markup order (config, then loader).
    Array.prototype.slice.call(div.querySelectorAll("script")).forEach(function (old) {
      var s = document.createElement("script");
      Array.prototype.slice.call(old.attributes).forEach(function (a) { s.setAttribute(a.name, a.value); });
      if (old.src) s.async = false;
      s.textContent = old.textContent;
      old.parentNode.replaceChild(s, old);
    });
    embeds[node.id] = { html: html, el: div };
    return div;
  }

  function pruneEmbeds() {
    Object.keys(embeds).forEach(function (id) {
      if (!embeds[id].el.isConnected) delete embeds[id];
    });
  }

  // --- Custom Component / IFrame -----------------------------------------

  // Embedded iframes are separate documents, so the page's themed scrollbars
  // (runtime.css) don't reach them — a scrolling Component/echarts chart would
  // show the default OS scrollbar. Inject a matching themed scrollbar using the
  // parent's current theme colors. Same-origin (srcdoc) only; cross-origin
  // access throws and is ignored.
  function injectScrollbarTheme(iframe) {
    iframe.addEventListener("load", function () {
      try {
        var doc = iframe.contentDocument;
        if (!doc || !doc.head) return;
        var cs = getComputedStyle(document.documentElement);
        var border = (cs.getPropertyValue("--sy-border") || "#e5e7eb").trim();
        var muted = (cs.getPropertyValue("--sy-muted") || "#6b7280").trim();
        var st = doc.createElement("style");
        st.setAttribute("data-sy-scrollbar", "");
        st.textContent =
          "*{scrollbar-width:thin;scrollbar-color:" + border + " transparent}" +
          "::-webkit-scrollbar{width:12px;height:12px}" +
          "::-webkit-scrollbar-track{background:transparent}" +
          "::-webkit-scrollbar-thumb{background:" + border +
          ";border-radius:8px;border:3px solid transparent;background-clip:padding-box}" +
          "::-webkit-scrollbar-thumb:hover{background:" + muted + ";background-clip:padding-box}" +
          "::-webkit-scrollbar-corner{background:transparent}";
        doc.head.appendChild(st);
      } catch (e) {}
    });
  }

  function componentEl(node, p) {
    var iframe = document.createElement("iframe");
    iframe.className = "sy-component";
    iframe.style.border = "none";
    iframe.style.width = (p.width || 100) + (p.width ? "px" : "%");
    iframe.style.height = (p.height || 300) + "px";
    iframe.srcdoc = p.html || "";
    iframe.sandbox = "allow-scripts allow-same-origin";
    injectScrollbarTheme(iframe);

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
    injectScrollbarTheme(iframe); // same-origin only; cross-origin is ignored
    return iframe;
  }

  function pdfEl(node, p) {
    var iframe = document.createElement("iframe");
    iframe.className = "sy-pdf";
    iframe.style.border = "1px solid var(--sy-border)";
    iframe.style.borderRadius = "var(--sy-radius)";
    iframe.style.width = (p.width ? p.width + "px" : "100%");
    iframe.style.height = (p.height || 600) + "px";
    iframe.src = p.src || "";
    return iframe;
  }

  function menuButtonEl(node, p) {
    var wrap = el("div", "sy-menu-button-wrap");
    var btn = document.createElement("button");
    btn.className = "sy-button sy-menu-button";
    btn.textContent = (p.label || "") + " ▾";
    if (p.disabled) btn.disabled = true;
    var dd = el("div", "sy-menu-button-dropdown");
    (p.options || []).forEach(function (opt) {
      var item = document.createElement("button");
      item.className = "sy-app-menu-item";
      item.textContent = opt;
      item.onclick = function (ev) {
        ev.stopPropagation();
        dd.classList.remove("open");
        send(node.id, opt, false);
      };
      dd.appendChild(item);
    });
    btn.onclick = function (ev) { ev.stopPropagation(); dd.classList.toggle("open"); };
    document.addEventListener("click", function () { dd.classList.remove("open"); });
    wrap.appendChild(btn);
    wrap.appendChild(dd);
    return wrap;
  }

  // --- Audio / Video -----------------------------------------------------

  function mediaSrc(p) {
    // Clip playback with a media fragment: #t=start[,end]
    var src = p.src || "";
    if (p.start_time || p.end_time) {
      src += "#t=" + (p.start_time || 0) + (p.end_time ? "," + p.end_time : "");
    }
    return src;
  }

  function audioEl(node, p) {
    var audio = document.createElement("audio");
    audio.className = "sy-audio";
    audio.src = mediaSrc(p);
    audio.controls = true;
    if (p.autoplay) audio.autoplay = true;
    if (p.loop) audio.loop = true;
    if (p.muted) audio.muted = true;
    return audio;
  }

  function videoEl(node, p) {
    var wrap = el("div", "sy-video-wrap");
    var video = document.createElement("video");
    video.className = "sy-video";
    video.src = mediaSrc(p);
    if (p.subtitles) {
      var track = document.createElement("track");
      track.kind = "subtitles";
      track.src = p.subtitles;
      track.default = true;
      video.appendChild(track);
      video.crossOrigin = "anonymous";
    }
    video.controls = true;
    if (p.width) video.style.maxWidth = p.width + "px";
    if (p.autoplay) video.autoplay = true;
    if (p.loop) video.loop = true;
    if (p.muted) video.muted = true;
    wrap.appendChild(video);
    return wrap;
  }

  // --- Toast / Celebrations -----------------------------------------------

  function handleToast(t) {
    if (t.type === "balloons") { showBalloons(); return; }
    if (t.type === "snow")     { showSnow(); return; }
    var toast = el("div", "sy-toast sy-toast-" + (t.level || "info"));
    if (t.icon) toast.appendChild(el("span", "sy-toast-icon", t.icon));
    toast.appendChild(el("span", "sy-toast-text", t.text));
    var container = document.getElementById("syralit-toasts");
    if (!container) {
      container = document.createElement("div");
      container.id = "syralit-toasts";
      container.className = "sy-toast-container";
      document.body.appendChild(container);
    }
    container.appendChild(toast);
    // setTimeout, not requestAnimationFrame: rAF doesn't fire in hidden or
    // backgrounded tabs, which would leave the toast permanently invisible
    // while its removal timer still runs.
    setTimeout(function () { toast.classList.add("sy-toast-show"); }, 20);
    setTimeout(function () {
      toast.classList.remove("sy-toast-show");
      toast.classList.add("sy-toast-hide");
      setTimeout(function () { toast.remove(); }, 300);
    }, t.duration || 3000);
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

  // --- App menu (SetPageConfig ConfigMenuItems) ---------------------------

  function ensureAppMenu(cfg) {
    var has = cfg.menu_help_url || cfg.menu_bug_url || cfg.menu_about;
    var existing = document.getElementById("syralit-app-menu");
    if (!has) { if (existing) existing.remove(); return; }
    var sig = JSON.stringify([cfg.menu_help_url || "", cfg.menu_bug_url || "", cfg.menu_about || ""]);
    if (existing) {
      if (existing.dataset.sig === sig) return; // unchanged — keep open state
      existing.remove();
    }

    var wrap = document.createElement("div");
    wrap.id = "syralit-app-menu";
    wrap.className = "sy-app-menu";
    var btn = document.createElement("button");
    btn.className = "sy-app-menu-btn";
    btn.textContent = "⋮";
    btn.title = T("menu", "Menu");
    var dd = document.createElement("div");
    dd.className = "sy-app-menu-dropdown";

    function addLink(label, url) {
      var a = document.createElement("a");
      a.className = "sy-app-menu-item";
      a.textContent = label;
      a.href = url;
      a.target = "_blank";
      a.rel = "noopener";
      dd.appendChild(a);
    }
    if (cfg.menu_help_url) addLink(T("menu_get_help", "Get help"), cfg.menu_help_url);
    if (cfg.menu_bug_url) addLink(T("menu_report_bug", "Report a bug"), cfg.menu_bug_url);
    if (cfg.menu_about) {
      var about = document.createElement("button");
      about.className = "sy-app-menu-item";
      about.textContent = T("menu_about", "About");
      about.onclick = function () {
        dd.classList.remove("open");
        var overlay = document.createElement("div");
        overlay.className = "sy-app-about-overlay";
        var box = document.createElement("div");
        box.className = "sy-app-about-box sy-markdown";
        box.innerHTML = cfg.menu_about; // server-rendered markdown HTML
        overlay.onclick = function (ev) { if (ev.target === overlay) overlay.remove(); };
        overlay.appendChild(box);
        document.body.appendChild(overlay);
      };
      dd.appendChild(about);
    }

    btn.onclick = function (ev) { ev.stopPropagation(); dd.classList.toggle("open"); };
    document.addEventListener("click", function () { dd.classList.remove("open"); });
    wrap.dataset.sig = sig;
    wrap.appendChild(btn);
    wrap.appendChild(dd);
    document.body.appendChild(wrap);
  }

  // --- Page Config -------------------------------------------------------

  function applyQueryParams(params) {
    var qs = Object.keys(params).map(function (k) {
      return encodeURIComponent(k) + "=" + encodeURIComponent(params[k]);
    }).join("&");
    var url = location.pathname + (qs ? "?" + qs : "") + location.hash;
    try { history.replaceState(null, "", url); } catch (e) {}
  }

  var sidebarStateApplied = false;

  function applyPageConfig(cfg) {
    if (cfg.title) document.title = cfg.title;
    if (cfg.sidebar_state === "collapsed" && !sidebarStateApplied) {
      sidebarStateApplied = true;
      layoutRoot.classList.add("sidebar-collapsed");
      ensureMobileToggle();
    } else if (cfg.sidebar_state) {
      sidebarStateApplied = true;
    }
    ensureAppMenu(cfg);
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
    if (p.avatar && /^(https?:|data:|\/)/.test(p.avatar)) {
      var aimg = document.createElement("img");
      aimg.src = p.avatar;
      aimg.className = "sy-chat-avatar-img";
      avatar.appendChild(aimg);
    } else {
      avatar.textContent = p.avatar || (p.role === "assistant" ? "🤖" : "👤");
    }
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
      blue: "var(--sy-color-blue)", green: "var(--sy-color-green)",
      red: "var(--sy-color-red)", orange: "var(--sy-color-orange)",
      yellow: "var(--sy-color-yellow)", gray: "var(--sy-color-gray)",
      violet: "var(--sy-color-violet)"
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
    var label = p.text || T("loading", "Loading…");
    var txt = el("span", "sy-spinner-text", label);
    wrap.appendChild(txt);
    if (p.show_time) {
      var start = Date.now();
      var timer = setInterval(function () {
        if (!txt.isConnected) { clearInterval(timer); return; }
        txt.textContent = label + " (" + ((Date.now() - start) / 1000).toFixed(1) + "s)";
      }, 100);
    }
    return wrap;
  }

  // --- Popover -----------------------------------------------------------

  function popoverEl(node, p) {
    var wrap = el("div", "sy-popover-wrap");
    var btn = el("button", btnClass("sy-button sy-popover-trigger", p), btnLabel(p));
    btn.dataset.id = node.id;
    if (p.disabled) btn.disabled = true;
    if (p.help) btn.title = p.help;
    btn.onclick = function () { if (!p.disabled) send(node.id, !p.open, false); };
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
    var maxSize = p.max_size || 10 * 1024 * 1024;
    if (p.file_names && p.file_names.length) {
      var infoM = el("div", "sy-file-info");
      infoM.textContent = p.file_names.join(", ") + " (" + formatSize(p.file_size || 0) + ")";
      wrap.appendChild(infoM);
    }
    if (p.file_name) {
      var info = el("div", "sy-file-info");
      info.textContent = "📄 " + p.file_name + " (" + formatSize(p.file_size || 0) + ")";
      wrap.appendChild(info);
    }
    var input = document.createElement("input");
    input.type = "file";
    if (p.multiple) input.multiple = true;
    input.className = "sy-file-input";
    input.dataset.id = node.id;

    function readOne(file) {
      return new Promise(function (resolve) {
        var reader = new FileReader();
        reader.onload = function () {
          var b64 = reader.result.split(",")[1] || "";
          resolve({ name: file.name, size: file.size, type: file.type, data: b64 });
        };
        reader.readAsDataURL(file);
      });
    }

    input.onchange = function () {
      if (!input.files || !input.files.length) return;
      var files = Array.prototype.slice.call(input.files);
      var total = files.reduce(function (sum, f) { return sum + f.size; }, 0);
      if (total > maxSize) {
        alert(T("file_too_large", "File too large") + " (max " + formatSize(maxSize) + ")");
        return;
      }
      if (p.multiple) {
        Promise.all(files.map(readOne)).then(function (payloads) {
          send(node.id, payloads, false);
        });
      } else {
        readOne(files[0]).then(function (payload) {
          send(node.id, payload, false);
        });
      }
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

  var DEFAULT_CHART_COLORS = ["#7c3aed", "#2563eb", "#16a34a", "#d97706", "#dc2626", "#0891b2", "#be185d", "#4f46e5"];
  // Theme override (theme.chart_categorical_colors) injected via window.__SY_THEME.
  function CHART_PALETTE() {
    var t = (window.__SY_THEME || {}).chart_categorical_colors;
    return (t && t.length) ? t : DEFAULT_CHART_COLORS;
  }
  var chartjsState = "idle";
  var chartjsQueue = [];

  function loadChartJS(cb) {
    if (chartjsState === "ready") { cb(); return; }
    if (chartjsState === "error") { cb(new Error("Chart.js failed to load")); return; }
    chartjsQueue.push(cb);
    if (chartjsState === "loading") return;
    chartjsState = "loading";
    var s = document.createElement("script");
    s.src = syAsset("chartjs", "https://cdn.jsdelivr.net/npm/chart.js@4.4.7/dist/chart.umd.min.js");
    s.onload = function () {
      chartjsState = "ready";
      chartjsQueue.forEach(function (fn) { fn(); });
      chartjsQueue = [];
    };
    s.onerror = function () {
      chartjsState = "error";
      var err = new Error("Chart.js failed to load");
      chartjsQueue.forEach(function (fn) { fn(err); });
      chartjsQueue = [];
    };
    document.head.appendChild(s);
  }

  function makeChartJS(type, p, configFn) {
    var wrap = el("div", "sy-chart-wrap");
    wrap.dataset.chartState = "loading";
    var canvas = document.createElement("canvas");
    wrap.style.maxWidth = (p.width || 600) + "px";
    wrap.style.height = (p.height || 300) + "px";
    wrap.appendChild(canvas);
    loadChartJS(function (loadErr) {
      if (loadErr) {
        wrap.dataset.chartState = "error";
        wrap.dispatchEvent(new CustomEvent("syralit:chart-settled"));
        return;
      }
      var cfg = configFn();
      cfg.options = cfg.options || {};
      var animation = cfg.options.animation;
      if (animation === false) {
        wrap.dataset.chartState = "settled";
      } else {
        if (!animation || typeof animation !== "object") animation = {};
        var originalComplete = animation.onComplete;
        animation.onComplete = function (ctx) {
          wrap.dataset.chartState = "settled";
          wrap.dispatchEvent(new CustomEvent("syralit:chart-settled"));
          if (typeof originalComplete === "function") originalComplete(ctx);
        };
        cfg.options.animation = animation;
      }
      try {
        new Chart(canvas, cfg);
        if (cfg.options.animation === false) {
          wrap.dispatchEvent(new CustomEvent("syralit:chart-settled"));
        }
      } catch (e) {
        wrap.dataset.chartState = "error";
        wrap.dispatchEvent(new CustomEvent("syralit:chart-settled"));
      }
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
    var palette = (p.colors && p.colors.length) ? p.colors : CHART_PALETTE();
    var datasets = names.map(function (name, si) {
      var color = palette[si % palette.length];
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

  function chartOptions(title, type, p) {
    p = p || {};
    var stacked = !!p.stacked;
    var scales;
    if (type !== "pie" && type !== "doughnut") {
      scales = {
        x: { grid: { display: false }, stacked: stacked },
        y: { beginAtZero: type === "bar", stacked: stacked },
      };
    }
    var opts = {
      responsive: true,
      maintainAspectRatio: false,
      interaction: { mode: "index", intersect: false },
      plugins: {
        legend: { display: true, position: "top" },
        tooltip: { enabled: true },
      },
      scales: scales,
    };
    if (type === "bar" && p.horizontal) opts.indexAxis = "y";
    if (title) {
      opts.plugins.title = { display: true, text: title, font: { size: 14 } };
    }
    return opts;
  }

  // Drag-to-select on index-based charts (line/bar/area): dragging
  // horizontally selects an x-axis range and sends
  // {range:true, index, x, end_index, end_x}. One document-level handler pair
  // serves every chart — per-render listeners would pile up stale closures
  // because reruns rebuild the chart elements.
  var rangeDrag = null; // {wrap, canvas, node, startPx, overlay}
  var rangeDocHandlersInstalled = false;
  // Set when a drag-range gesture completes: the mouseup also produces a
  // Chart.js click on the same canvas, which would immediately overwrite the
  // just-sent range selection with a point selection.
  var suppressNextChartClick = false;

  function attachRangeSelect(wrap, canvas, node) {
    canvas.addEventListener("mousedown", function (ev) {
      var overlay = el("div", "sy-chart-range-overlay");
      wrap.appendChild(overlay);
      rangeDrag = { wrap: wrap, canvas: canvas, node: node, startPx: ev.clientX, overlay: overlay };
      updateRangeOverlay(ev.clientX);
      ev.preventDefault();
    });
    installRangeDocHandlers();
  }

  function updateRangeOverlay(curX) {
    if (!rangeDrag) return;
    var rect = rangeDrag.wrap.getBoundingClientRect();
    var a = Math.min(rangeDrag.startPx, curX) - rect.left;
    var b = Math.max(rangeDrag.startPx, curX) - rect.left;
    rangeDrag.overlay.style.left = a + "px";
    rangeDrag.overlay.style.width = (b - a) + "px";
  }

  function installRangeDocHandlers() {
    if (rangeDocHandlersInstalled) return;
    rangeDocHandlersInstalled = true;

    document.addEventListener("mousemove", function (ev) {
      if (rangeDrag) updateRangeOverlay(ev.clientX);
    });

    document.addEventListener("mouseup", function (ev) {
      if (!rangeDrag) return;
      var d = rangeDrag;
      rangeDrag = null;
      d.overlay.remove();
      var chart = window.Chart && window.Chart.getChart(d.canvas);
      if (!chart) return;
      // Small drags fall through to Chart.js's own click handling.
      if (Math.abs(ev.clientX - d.startPx) < 8) return;
      var rect = d.canvas.getBoundingClientRect();
      function idxAt(clientX) {
        var v = chart.scales.x.getValueForPixel(clientX - rect.left);
        var n = chart.data.labels.length;
        return Math.max(0, Math.min(n - 1, Math.round(v)));
      }
      var i0 = idxAt(Math.min(d.startPx, ev.clientX));
      var i1 = idxAt(Math.max(d.startPx, ev.clientX));
      var labels = chart.data.labels || [];
      suppressNextChartClick = true;
      setTimeout(function () { suppressNextChartClick = false; }, 100);
      send(d.node.id, {
        range: true,
        index: i0, x: String(labels[i0]),
        end_index: i1, end_x: String(labels[i1]),
        series: "", value: 0,
      }, false);
    });
  }

  // applyChartSelect wires a click handler onto a selectable chart config: the
  // nearest element's series/index/label/value is sent as the widget value.
  function applyChartSelect(cfg, node, p, kind) {
    if (!p.selectable || !node.id) return cfg;
    cfg.options = cfg.options || {};
    cfg.options.onClick = function (evt, elements, chart) {
      if (suppressNextChartClick) { suppressNextChartClick = false; return; }
      var els = elements && elements.length ? elements
        : chart.getElementsAtEventForMode(evt.native || evt, "nearest", { intersect: true }, true);
      if (!els || !els.length) return;
      var dsi = els[0].datasetIndex, idx = els[0].index;
      var ds = chart.data.datasets[dsi] || {};
      var sel;
      if (kind === "pie") {
        sel = { series: String(chart.data.labels[idx]), index: idx,
                x: String(chart.data.labels[idx]), value: Number(ds.data[idx]) || 0 };
      } else if (kind === "scatter") {
        var pt = ds.data[idx] || {};
        sel = { series: ds.label || "", index: idx, x: String(pt.x), value: Number(pt.y) || 0 };
      } else {
        sel = { series: ds.label || "", index: idx,
                x: String((chart.data.labels || [])[idx]), value: Number(ds.data[idx]) || 0 };
      }
      send(node.id, sel, false);
    };
    return cfg;
  }

  function lineChartEl(node, p) {
    var wrap = makeChartJS("line", p, function () {
      var d = seriesDatasets(p, "line");
      return applyChartSelect({ type: "line", data: { labels: d.labels, datasets: d.datasets }, options: chartOptions(d.title, "line", p) }, node, p, "line");
    });
    if (p.range_selectable && node.id) attachRangeSelect(wrap, wrap.querySelector("canvas"), node);
    return wrap;
  }

  function barChartEl(node, p) {
    var wrap = makeChartJS("bar", p, function () {
      var d = seriesDatasets(p, "bar");
      return applyChartSelect({ type: "bar", data: { labels: d.labels, datasets: d.datasets }, options: chartOptions(d.title, "bar", p) }, node, p, "bar");
    });
    if (p.range_selectable && node.id) attachRangeSelect(wrap, wrap.querySelector("canvas"), node);
    return wrap;
  }

  function areaChartEl(node, p) {
    var wrap = makeChartJS("line", p, function () {
      var d = seriesDatasets(p, "area");
      return applyChartSelect({ type: "line", data: { labels: d.labels, datasets: d.datasets }, options: chartOptions(d.title, "area", p) }, node, p, "area");
    });
    if (p.range_selectable && node.id) attachRangeSelect(wrap, wrap.querySelector("canvas"), node);
    return wrap;
  }

  function pieChartEl(node, p) {
    var data = p.data || {};
    var names = Object.keys(data);
    return makeChartJS("pie", p, function () {
      return applyChartSelect({
        type: "pie",
        data: {
          labels: names,
          datasets: [{
            data: names.map(function (n) { return data[n]; }),
            backgroundColor: names.map(function (_, i) { var cp = CHART_PALETTE(); return cp[i % cp.length]; }),
          }],
        },
        options: chartOptions(p.title, "pie"),
      }, node, p, "pie");
    });
  }

  function scatterChartEl(node, p) {
    var series = p.series || {};
    var names = Object.keys(series);
    return makeChartJS("scatter", p, function () {
      var datasets = names.map(function (name, si) {
        var pts = (series[name] || []).map(function (pt) { return { x: pt[0], y: pt[1] }; });
        return { label: name, data: pts, backgroundColor: CHART_PALETTE()[si % CHART_PALETTE().length], pointRadius: 5 };
      });
      return applyChartSelect({ type: "scatter", data: { datasets: datasets }, options: chartOptions(p.title, "scatter") }, node, p, "scatter");
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
          datasets: [{ label: "Frequency", data: counts, backgroundColor: CHART_PALETTE()[0] + "cc" }],
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
            backgroundColor: names.map(function (_, i) { var cp = CHART_PALETTE(); return cp[i % cp.length]; }),
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
        var color = CHART_PALETTE()[si % CHART_PALETTE().length];
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
    s.src = syAsset("viz", "https://cdn.jsdelivr.net/npm/@viz-js/viz@3.11.0/lib/viz-standalone.js");
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
      syAsset("vega", "https://cdn.jsdelivr.net/npm/vega@5/build/vega.min.js"),
      syAsset("vega_lite", "https://cdn.jsdelivr.net/npm/vega-lite@5/build/vega-lite.min.js"),
      syAsset("vega_embed", "https://cdn.jsdelivr.net/npm/vega-embed@6/build/vega-embed.min.js")
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
    s.src = syAsset("plotly", "https://cdn.plot.ly/plotly-2.35.0.min.js");
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
    s.src = syAsset("bokeh", "https://cdn.bokeh.org/bokeh/release/bokeh-3.4.1.min.js");
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
      syAsset("deckgl", "https://unpkg.com/deck.gl@latest/dist.min.js"),
      syAsset("mapbox_js", "https://api.mapbox.com/mapbox-gl-js/v3.1.2/mapbox-gl.js")
    ];
    var css = document.createElement("link");
    css.rel = "stylesheet";
    css.href = syAsset("mapbox_css", "https://api.mapbox.com/mapbox-gl-js/v3.1.2/mapbox-gl.css");
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
