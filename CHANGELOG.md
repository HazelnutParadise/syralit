# Changelog

All notable changes to Syralit are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/), and this project adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added
- **`sy.Embed(html, opts...)`** (#5) — inserts third-party markup into the
  main document and executes its `<script>`s, for "mount point plus loader"
  integrations (ad slots, comment threads, social embeds, chat widgets) that
  `sy.HTML` (never runs scripts) and `sy.Component` (sandboxed iframe) cannot
  host. The node is built once per key; reruns with unchanged html reuse the
  element and do not re-run its scripts, and at the top level or inside a
  `Fragment` the element stays attached to the DOM so iframes the widget
  created don't reload. Changing the html rebuilds and re-runs. Inside layout
  containers the container is rebuilt, so the embed is re-attached there.
- **`sy.ResolveConfig(cfg)`** (#4) — returns `cfg` with unset fields filled
  from `syralit.toml` and then the built-in defaults, the same resolution
  `sy.App` and `sy.Handler` perform internally. An app that mounts `sy.Handler`
  in its own server can now read the configured `Host`/`Port` to listen on.

### Fixed
- `syralit.toml` now also accepts `host`/`port` under `[server]`; previously
  only the top-level keys were read and `[server] port` was silently ignored,
  so apps using it bound the default 8600. Top-level keys win when both are
  present. The `embed-scroll`, `insyra-charts` and `insyra-demo` examples used
  the ignored form and now declare `port` at the top level.
- Documented that `sy.GetOption` only reflects the serving app's config inside
  the page function. Since 0.10.0 (#2) it has no process-wide fallback, so
  outside a rerun it returns the zero `Config`'s values (`""`/`0` for
  `server.host`/`server.port`, not the defaults); embedded apps that used it
  to pick a listen address silently bound `:0`. Use `sy.ResolveConfig` there.

## [0.10.1] - 2026-08-23

### Fixed
- `syralit dev <dir>` and `syralit run <dir>` now run the app with the project
  directory as its working directory. Before, the child inherited the
  directory `syralit` was launched from, so a project's `syralit.toml` went
  unread and relative paths the app opened resolved against the wrong place
  unless the command was run from inside the project.

## [0.10.0] - 2026-08-22

One handler, many documents: `Config.DocumentFunc` lets the title and `<head>`
depend on the request, and `syralit dev` now serves the document from the
child so the hook works there too. Also finishes #2 — the upload cap and
`sy.GetOption` are per handler.

### Added
- **Per-request document** (#3) — `Config.DocumentFunc func(*http.Request)
  Document` is called once per document request, before the shell is rendered
  and before any session exists. Non-empty fields of the returned `Document`
  (`Title`, `Lang`, `Dir`, `HeadHTML`) override the `Config` values for that
  response only, so one handler can send a different `<title>` and `<head>`
  per query parameter or path — the case the static shell settings from #1
  could not cover. `Title` also wins over the page-URL title. The returned
  `HeadHTML` is verbatim: escape anything taken from the request. Under
  `syralit dev` document requests are now forwarded to the child process
  while it is up (only it can run `DocumentFunc`, and only it knows which page
  paths exist, so dev gets real 404s too); the supervisor's own shell is the
  fallback while the child is rebuilding or failed to compile.

### Fixed
- **Upload cap and `sy.GetOption` are per handler** — the last two
  process-wide values from #2. `MaxUploadSizeMB` (the cap an uploader widget
  advertises and the socket enforces) and the config behind `sy.GetOption`
  were package-level, so with two `sy.Handler` values the later one's settings
  applied to both. A session now carries the config of the server that
  created it, and both read from there. `sy.GetOption` outside a page function
  now returns the zero config's values rather than whatever server started
  last.

## [0.9.1] - 2026-08-22

### Fixed
- **Shell settings are per handler** (#2) — `Config.Lang`, `Config.Dir`,
  `Config.HeadHTML` and `Config.UIStrings` were held in package-level
  variables, so building a second `sy.Handler` silently rewrote the shell of
  every handler built before it, and constructing one while another was
  serving was a data race. They are now resolved once onto the handler (and
  onto the dev supervisor) and only read after that, so a `Config` fully
  determines the document of the handler it is passed to. `UIStrings` had the
  same flaw before v0.9.0; it is fixed by the same change.

## [0.9.0] - 2026-08-22

The document release: an app can finally control the HTML around it. The shell's
`lang`, writing direction and `<head>` are configurable, and every page in a
multi-page app gets its own URL — so pages can be linked to, reloaded and
reached with the back button, and the first response carries the right title.
Also fixes `[i18n]` being dropped under `syralit dev`, and bumps Insyra to
v0.3.1.

### Added
- **Page URLs** — every page registered with `sy.AddPage` now has its own URL,
  derived from the title (`"Data Explorer"` → `/data_explorer`) or set with the
  new `sy.PageSlug` option. Pages can be linked to, bookmarked and reloaded, the
  back and forward buttons move between them, and the first HTML response for a
  page path carries that page's `<title>` — so a link preview no longer shows
  the app-wide name for every page. Static files in `public/` still win over a
  page of the same name, and an unknown path is a 404. Under `syralit dev` the
  supervisor serves the shell for any plain path, since the page registry lives
  in the child process; paths that look like a filename remain 404.
- **Configurable app shell** (#1) — `Config.Lang` sets the `lang` attribute on
  `<html>` (default `"en"`), `Config.Dir` sets `dir` (`"ltr"` / `"rtl"` /
  `"auto"`; omitted when empty), and `Config.HeadHTML` is inserted verbatim at
  the end of `<head>` for description and Open Graph tags, a favicon link or
  resource hints. All three land in the first HTML response, so crawlers and
  link previews see them; all three are also settable from `syralit.toml`
  (`lang`, `dir`, `head_html`). `HeadHTML` is neither escaped nor validated —
  same trust level as `sy.HTML()` — while an invalid `Lang` or `Dir` is logged
  and ignored. `DevOptions` gains `Lang` / `TextDir` / `HeadHTML` / `UIStrings`
  so `syralit dev` renders the same shell as `syralit run`.

### Fixed
- `syralit dev` now applies the `[i18n]` table to its own shell. The supervisor
  reads `syralit.toml` separately from `Config`, and `applyToDev` never copied
  the UI string overrides, so localized built-in text vanished in dev.

### Changed
- Insyra bumped to v0.3.1.

## [0.8.0] - 2026-07-18

Desktop release: ship the same Syralit app as a native desktop window via the
new `integrations/desktop` module (Wails v3) — with `syralit dev` hot reload
attached to the window, browser lockdown by default, an agent artifact channel
that works under lockdown, and a real-window e2e suite in CI. Also bumps
Insyra to v0.3.0 and makes `DefaultValue` work on text inputs.

### Added
- **Desktop apps** — new standalone module `integrations/desktop`
  (import alias `sydesktop`) ships any Syralit app as a native desktop window
  via Wails v3: `sydesktop.App(fn, opts...)` / `sydesktop.Run` start the app on
  a loopback-only random port and open a native webview pointing at it; closing
  the window shuts the server down. Options: `WindowTitle`, `WindowSize`,
  `MinSize`, `Frameless`, `Icon`, `Config`. Multi-page apps (`sy.AddPage`) work
  unchanged. New example `examples/desktop-demo` shows direct local-file access
  (the Go process runs on the user's machine — no upload round-trip).
  **Hot reload included**: under `syralit dev` the native window attaches to
  the supervisor and survives rebuilds like a browser tab (state preserved,
  build-error overlay); it auto-quits when the supervisor stops. The
  supervisor now exports `SYRALIT_URL` and `SYRALIT_DEV_SESSION` to its child
  to support this. **Browser lockdown by default**: the loopback server only
  answers its own window (per-launch token; opt out with
  `sydesktop.AllowBrowser()`); `/api/` endpoints stay open for agents with
  their own bearer auth, `SYRALIT_URL` is set for agent subprocesses, and an
  explicit `Config` Port pins the listen port. A Windows e2e suite
  (`SYRALIT_DESKTOP_E2E=1`, run in CI) opens real windows to cover production
  launch, lockdown, artifact auth, graceful close, and dev hot reload.

### Fixed
- `sy.TextInput`, `sy.TextArea` and `sy.PasswordInput` now honor
  `sy.DefaultValue("...")` as the initial value, as the docs already promised.
  The default applies only until the user first edits the field, so clearing it
  sticks.

### Changed
- **Insyra bumped to v0.3.0** (from v0.2.19). Two upstream behaviour changes
  surface through the integration: CSV/JSON files loaded via
  `syi.UploadTable`/`syi.ParseTable` now use column-level type inference, so
  integer columns arrive as `int64` instead of `float64` (charts, metrics and
  the DSL already coerce both); and `syi.ListDescribe` quartiles now follow
  R type-7, matching pandas' default `describe()`.

## [0.7.0] - 2026-07-05

Interactive-data release: click-to-filter and drag-range chart selections
wired end-to-end into Insyra (GroupBy charts, filters, time-series helpers,
CSV export, guarded CCL formula columns), OIDC single sign-on as an opt-in
module, dataframe column selection, UI-string localization, and a fuller
scaffold.

### Added
- **Interactive Insyra integration**: DataTable chart helpers now pass options
  through, use the x column as axis labels, and return `*sy.ChartSelection`
  with `sy.Selectable()`. New `MultiLineChart`/`MultiBarChart`/`MultiAreaChart`
  (nil column list = every numeric column, like `st.line_chart(df)`),
  `GroupedBarChart`/`GroupedPieChart` (Insyra GroupBy + Aggregate per group),
  `FilterEquals`/`FilterBySelection` (pure data helpers for click-to-filter
  dashboards), and `EditableDataTable` (DataEditor edits back as a new
  `*insyra.DataTable`). New examples: `examples/insyra-interactive` and
  `examples/data-studio` (the full upload → group → click-to-drill-down
  workflow, with README screenshots).
- **`sy.ResetWidget(key)`**: delete a widget's stored value (the counterpart
  of Streamlit's `del st.session_state[key]`), e.g. to clear a chart selection.
- **Chart range selection**: `sy.RangeSelectable()` on Line/Bar/Area charts —
  drag across the chart to select an x-axis interval; `ChartSelection` gains
  `Range`/`EndIndex`/`EndX`. The completing drag suppresses the Chart.js click
  that would otherwise overwrite the range with a point selection.
- **DataFrame column selection**: `sy.SelectionMode("single-column" /
  "multi-column")` — header clicks select columns and return their indices.
- **`[i18n]` config**: localize built-in UI strings (connecting/loading/menu
  labels/upload errors), published to the front end as `window.__SY_I18N`.
- **Insyra time-series & export helpers**: `syi.DownloadCSV`,
  `syi.RollingMeanChart`, `syi.CumSumChart`, `syi.PctChangeChart`, and
  `syi.AddFormulaColumn` (interactive CCL formula column — monospace `ƒx`
  editor, columns addressable as `A`/`B` or `["Name"]`, timeout-guarded
  evaluation since CCL can hang on malformed input). The editor shows a live
  `A = Name` column legend (CCL names are case-sensitive), and the guarded
  evaluator is exported as `syi.ComputeColumn` for custom formula UIs. New
  `sy.Mono()` option renders TextInput/TextArea in the code font, and
  `sy.Formula()` gives an input the formula-bar look (fx marker inside the
  box, accent edge, code surface) — used by the CCL editor by default.
- Table/dataframe headers no longer render uppercase: the true column casing
  must be visible because data tooling (CCL formulas) matches names
  case-sensitively.
- `syralit new` scaffold: generated syralit.toml now documents the full
  theme/server/secrets option surface.
- UI test suite: added chart range-drag and Insyra click-to-filter coverage
  (real mouse-drag CDP events).
- **OIDC single sign-on** (`integrations/oidc`, separate module): wrap the app
  handler with `syoidc.Protect` and visitors authenticate through any OIDC
  provider (auth-code flow with state/nonce, HMAC-signed identity cookie);
  `sy.User()` returns the verified claims. Built on the new core hook
  `sy.SetUserResolver` (request context -> user, dependency-free). New
  example: `examples/oidc-login` (standalone module).

## [0.6.0] - 2026-07-05

Streamlit parity release: full theming (fonts, color palette, chart colors,
per-sidebar overrides), the remaining widget/display gaps (PDF viewer,
multi-file upload, selectable charts, menu button, datetime input, bottom
container), deployment features (sub-path mounting via `sy.Handler`, HTTPS,
upload limits), a public headless testing API (`sy.AppTest`), and a
browser-level UI test suite.

### Added
- **Font theming** (Streamlit parity): `Theme` gains `Font`, `HeadingFont`,
  `CodeFont` (the keywords `"sans-serif"` / `"serif"` / `"monospace"` select
  the embedded Source Sans 3 / Source Serif 4 / Source Code Pro — served
  locally, no CDN; any other value is a CSS font-family list),
  `BaseFontSize`, `BaseFontWeight`, `HeadingFontSizes`, `HeadingFontWeights`,
  `CodeFontSize`, `CodeFontWeight`, custom `FontFaces` (`@font-face` from
  OTF/TTF/WOFF/WOFF2 files or URLs) and sidebar-scoped overrides via
  `Theme.Sidebar`. All configurable from `syralit.toml` (`[theme]`,
  `[[theme.font_faces]]`, `[theme.sidebar]`). New example:
  `examples/theme-fonts`.
- **Full theme parity with Streamlit**: `Theme` (via the embedded `ThemeStyle`)
  now also supports `BackgroundColor`, `SecondaryBackgroundColor`, `TextColor`,
  `LinkColor`, `LinkUnderline`, `CodeTextColor`, `CodeBackgroundColor`,
  `BorderColor`, `DataframeBorderColor`, `DataframeHeaderBackgroundColor`,
  `ButtonRadius`, `ShowWidgetBorder`, `ShowSidebarBorder`, the basic color
  palette (`red/orange/yellow/blue/green/violet/gray` base + background + text
  variants, wired to badges, alerts and status colors via new `--sy-color-*`
  CSS variables) and chart palettes (`ChartCategoricalColors` drives the
  built-in Chart.js series colors; sequential/diverging are published on
  `window.__SY_THEME`). Every color/font/radius option can be overridden for
  the sidebar only via `Theme.Sidebar` / `[theme.sidebar]`.
- **`sy.PDF(src)`**: embedded PDF viewer (browser renderer); `sy.Height`/`sy.Width`.
- **`sy.FileUploaderMultiple`**: multi-file upload returning `[]*UploadedFile`.
- **`[server] max_upload_size_mb`** (and `Config.MaxUploadSizeMB`): configurable
  upload cap (default 10 MB), enforced in the browser and in the socket read limit.
- **`sy.InitialSidebarState("collapsed")`**: sidebar starts hidden; the floating
  toggle reopens it. Also fixes the mobile sidebar toggle, which was appended
  outside `#syralit-root` and never matched its show/hide CSS.
- **`sy.ConfigMenuItems(helpURL, bugURL, aboutMarkdown)`**: top-right app menu
  with "Get help" / "Report a bug" links and an About dialog.
- **`sy.AcceptNewOptions()`**: MultiSelect free entry (type new values + Enter).
- **`sy.StartTime` / `sy.EndTime` / `sy.Subtitles`**: Audio/Video playback
  clipping (media fragments) and WebVTT subtitle tracks.
- **Toast duration**: optional 4th argument, e.g. `sy.Toast(msg, "success", "⏱", "8s")`.
- New example: `examples/streamlit-parity`.
- **`sy.Space()`**: vertical/horizontal whitespace element.
- **`sy.Bottom(fn)`**: viewport-bottom pinned container (chat-input layouts);
  the main area gains matching padding automatically.
- **`sy.DatetimeInput`**: combined date+time picker returning "YYYY-MM-DD HH:MM".
- **`sy.MenuButton(label, options)`**: dropdown button returning the clicked
  option for exactly one rerun.
- **`sy.Handler(cfg, fn)`**: mount a Syralit app as an `http.Handler` inside an
  existing Go server.
- **`sy.GetOption(key)`**: read resolved config values at runtime.
- **`sy.AppTest`**: headless app-testing harness (`NewAppTest`, `Run`,
  `SetValue`, `Click`/`ClickLabel`, `FindAll`/`FindByLabel`/`Texts`,
  `SwitchToPage`) — the Go counterpart of Streamlit's `st.testing.v1.AppTest`.
- **Selectable charts**: `LineChart`/`BarChart`/`AreaChart`/`ScatterChart`/
  `PieChart` accept `sy.Selectable()` and return a `*ChartSelection`
  (Series/Index/X/Value) when the user clicks a data point.
- **`sy.SetQueryParam`**: writable URL query parameters — the browser address
  bar updates via history.replaceState, making app state shareable.
- **Sub-path mounting fixed**: assets, WebSocket, SSE and message endpoints now
  resolve against the mount prefix, so `sy.Handler` works behind
  `http.StripPrefix` (e.g. `/dashboard/`).
- **DataFrame**: `sy.ColumnOrder(...)` reorders/filters columns;
  `sy.SelectionMode("single-row")` limits row selection.
- **Column config**: new `area_chart` and `json` cell types.
- **HTTPS**: `[server] ssl_cert_file` / `ssl_key_file` (and
  `Config.SSLCertFile/SSLKeyFile`) serve the app over TLS.
- **`sy.ShowTime()`** on `SpinnerWith`: shows elapsed time next to the spinner.
- **Browser UI test suite** (`uitest/`, separate Go module): headless-Chrome
  tests covering widget round-trips, sidebar collapse, toast visibility and
  duration, canvas chart click selection, multi-file upload, and sub-path
  mounting.

### Fixed
- Toasts never became visible in hidden/backgrounded tabs (the show class was
  added via requestAnimationFrame, which does not fire there).

### Fixed
- README and skill docs showed nonexistent `[theme]` keys (`primary_color`,
  `background_color`, `text_color`); corrected to the real `mode` / `accent` /
  `radius` keys.

## [0.5.0] - 2026-07-03

Insyra DSL brings dynamic, safe, server-side computation to Syralit widgets and
agent Artifacts. Agents can embed a `.isr` script in an artifact and have it
computed and rendered live, and discovery advertises the live command catalog.

### Added
- **Insyra DSL computation** (`integrations/insyra/insyradsl`, a new opt-in
  subpackage): `RunDSL` executes an Insyra CLI DSL (`.isr`) script in an
  isolated, ephemeral environment and returns the produced variables plus
  textual output. It runs in **safe mode** by default — a default-deny allowlist
  of pure, in-memory compute commands; `load`/`save`/`db`/`fetch`/`run`/`env`/
  `plot` are rejected (`Unrestricted()` lifts this for trusted scripts).
  `syidsl.DSL(script, opts...)` is a Go widget that runs a script and
  auto-renders the result (cached by script hash). Kept in a separate package
  because the Insyra DSL engine pulls in the full Insyra CLI dependency tree.
- **`insyra` Artifact component**: importing `insyradsl` registers an `insyra`
  artifact component so agents can embed a DSL script directly in an
  `ArtifactSpec` for live server-side computation (group-by, filter, stats, CCL)
  rendered as a table, chart, metric, or text node — the artifact path is always
  safe mode.
- **`RegisterArtifactComponent`**: a core extension point that lets integrations
  add custom artifact components without the core package importing them,
  preserving the core/Insyra boundary.
- **Artifact discovery reports components**: `GET` on the unified artifact API
  now returns a `components` object (`builtin` + `custom`) so an agent can detect
  whether an opt-in component such as `insyra` is enabled before using it.
- **Live component capabilities in discovery**: `RegisterArtifactComponentInfo`
  lets a component publish discovery metadata under `components.capabilities`.
  The `insyra` component reports the safe DSL command catalog
  (`syidsl.SafeCommandCatalog`) pulled live from Insyra's registry, so an agent
  learns the exact vocabulary for the app's Insyra version and version bumps need
  no doc edits.
- Example `examples/insyra-artifact` demonstrating both the `syidsl.DSL` widget
  and the `insyra` artifact component.

## [0.4.0] - 2026-07-02

Agent Artifacts turn Syralit apps into safe, live canvases that AI agents can
discover, update, and visually verify.

### Added
- **Agent Artifact Canvas**: `ArtifactSpec`, `ArtifactNode`,
  `NewArtifactStore`, `ArtifactCanvas`, and `HandleArtifactEndpoint` let apps
  expose an opt-in bearer-token endpoint that replaces a shared, animated
  canvas from a controlled DSL. The DSL maps only to safe Syralit components
  (`text`, `markdown`, `metric`, `table`, `dataframe`, line/bar/pie charts,
  `image`, `progress`, `container`) and supports JSON Pointer data binding.
- **Unified artifact discovery API**: `HandleArtifactAPI` exposes explicitly
  selected stores through one authenticated GET/POST route, while
  `ArtifactAPIHandler` and `ArtifactHandler` allow the same APIs to run on a
  separate mux or port. Discovery reports observed pages and stable selectors.
  The hot-reload supervisor proxies `/api/` to its child, so the public dev App
  URL works without exposing or documenting the internal child port.
- **Deterministic artifact previews**: every successful update returns a
  revision and preview metadata. Artifact canvases now expose
  `transitioning`/`settled` DOM states after keyed layout transitions, charts,
  images, and fonts finish so agents can capture the final rendered result.
  Full-replace writes require `expected_revision` and return `409 Conflict`
  instead of losing concurrent updates; failed visual resources produce
  `partial` readiness rather than a false `complete`.
- **Artifact DSL documentation and skill**: added `docs/artifact-dsl.md` as the
  formal schema reference and a dedicated `skills/syralit-artifact-dsl`
  generator skill for agents that need to output valid DSL payloads.
- **Agent key hooks**: `AgentAuthenticator`, `AgentKeyStore`,
  `StaticAgentKey`, and `AgentKeyManager` provide both hardcoded/secrets-backed
  auth and app-owned key-management UI without making Syralit persist keys.
- New example: `examples/agent-artifact` demonstrates the artifact canvas,
  POST updates, static auth, and a user-provided in-memory key store.

## [0.3.0] - 2026-06-22

Deeper Insyra integration — turning the adapter from "render Insyra data" into
"do data science in Go, in the browser". All additions stay inside
`integrations/insyra/`; the core framework still never imports Insyra.

### Added
- **Statistical analysis** (`integrations/insyra`, over `insyra/stats`):
  `Describe` (full per-column summary table), `Correlation` (r + p),
  `CorrelationMatrix`, `LinearRegression` (R²/adj-R²/p + coefficient table, plus
  a scatter for a single predictor), `TTest` (two-sample). New
  `renderTableWithRowNames` renders an Insyra DataTable with row labels.
- **File upload → DataTable**: `UploadTable` (FileUploader → `*DataTable`) and
  `ParseTable(name, bytes)` for CSV / Excel (`.xlsx`/`.xls`, via in-memory
  excelize) / JSON. Parquet is intentionally excluded to avoid forcing Apache
  Arrow on adapter users.
- **Interactive transforms**: `FilterBuilder` (column/operator/value row filter)
  and `CCLBuilder` (add a computed column with Insyra's CCL). Both non-
  destructive (operate on a `Clone`). CCL is applied only on an explicit Apply
  and every evaluation is time-bounded in a goroutine, so a malformed formula
  that runs away in the CCL engine can never hang or crash the app.
- **Native go-echarts charts** in a new opt-in subpackage
  `integrations/insyra/eplot` (`syiplot`): `EChart(plot.Renderable)` bridges any
  `insyra/plot` chart into a sandboxed Component iframe, and `WordCloud`. Unlocks
  Sankey, word cloud, K-line, gauge, funnel, theme-river, box plot, radar — chart
  types the built-in Chart.js layer doesn't have. Kept separate because
  `insyra/plot` drags in go-echarts and (transitively) chromedp; apps using only
  the core adapter stay free of those deps.
- **Offline echarts** — `syiplot.SetOffline(true)` inlines the echarts
  JavaScript (core, v4, word-cloud extension; bundled via `go:embed`) directly
  into each chart's iframe, so native charts render with no CDN — air-gapped,
  strict-CSP, or as a `syralit build` single binary. Default stays CDN (lighter
  HTML).
- **Themed scrollbars in embedded iframes** — `Component` and same-origin
  `IFrame` content now gets a scrollbar matching the page's light/dark theme
  (thin, rounded, transparent track) instead of the default OS one, injected
  from the parent's live theme colors.
- New examples: `examples/insyra-charts` (Sankey, gauge, funnel, pie and word
  cloud with an offline toggle) and `examples/embed-scroll` (themed embedded
  scrollbars).

### Fixed
- `Toggle` and `Checkbox` now honor `DefaultValue(true)` — previously they
  ignored the option and always started unchecked.
- Native go-echarts charts now fit their frame and follow the Syralit theme:
  the rendered chart is made responsive (no fixed-px overflow / horizontal
  scrollbar), its background is transparent, text recolors to the page's
  light/dark theme, and Sankey node labels are shown.

### Changed
- Bumped Insyra to **v0.2.19**.

## [0.2.0] - 2026-06-21

Beyond Streamlit — capabilities that lean on Go and have no Streamlit
equivalent. See `docs/STREAMLIT_PARITY.md`.

### Added
- **`Task[T]`** — background jobs in a goroutine; the page stays responsive and
  the server pushes the result when ready (a Streamlit rerun blocks). Typed
  handle: `Running`/`Done`/`Result`/`Err`.
- **`Shared[T]`** — app-wide state across all sessions; `Set`/`Update` live-push
  to every connected client (real-time collaboration).
- **Offline / air-gapped builds** — `SetAssetURL(name, url)` repoints any
  third-party library to a self-hosted copy; with `syralit build` the app runs
  as one binary with no CDN/internet (strict-CSP friendly).
- **Automatic SSE transport fallback** — when WebSocket can't connect, the
  client transparently switches to plain HTTP (Server-Sent Events downstream +
  POST upstream), no app changes.

### Internal
- Server event loop can push without a client message (session `wake` channel);
  render output is abstracted behind a `uiSink` (WebSocket or SSE).

## [0.1.1] - 2026-06-21

A broad Streamlit-parity pass. See `docs/STREAMLIT_PARITY.md` for the full map.

### Added
- Inputs: `RangeSlider`, `DateSlider`, `TimeSlider`, `DateRangeInput`;
  `PillsMulti` / `SegmentedControlMulti` (multi-select); `Feedback` styles
  (`FeedbackStyle` thumbs/stars/faces); date bounds (`MinDate`/`MaxDate`).
- Buttons: `Icon`, `ButtonType`, `UseContainerWidth`, `Help` on Button,
  LinkButton, DownloadButton, Popover.
- Data: `DataFrame` row selection (`Selectable`) and typed display via
  `ColConfig` (now shared with DataEditor); richer `ColumnConfig` (`Format`,
  `Label`, `Help`, `Step`, `Color`) plus `bar_chart`/`line_chart` sparkline
  columns. Interactive collapsible `JSON` tree.
- Charts: `Stacked`, `Horizontal`, `Colors`. `Map` `Zoom` + per-point
  Size/Color.
- Media: `Image` `UseContainerWidth`; `Audio`/`Video` `Autoplay`/`Loop`/`Muted`.
- Display: `Exception`, `Metric` `Border`/`Help`, `Code` `LineNumbers`/`Wrap`,
  `Progress` text, `ChatMessage` `Avatar`, `Toast` icon, `Expander` `Icon`.
- Platform: `Context` (request headers/cookies/host/IP/locale); `Fragment`
  `RunEvery` auto-refresh; `Form` `ClearOnSubmit`.
- Testing: `RenderOnce` + `Node.Find`; unit tests for the Insyra adapter.
- Insyra: `ListDescribe`; the full symmetric DataList helper set.

### Fixed
- WriteStream token streaming and all fragment partial updates were broken
  (`appendStream`/`patchFragment` referenced an undefined `content`/`renderNode`);
  now wired to the real container/builder.
- `--sy-primary` was referenced but never defined, breaking SegmentedControl/
  Pills active states and several accents.
- Themed scrollbars; stray vertical scrollbar on the tabs bar.
- Stats render `—` instead of `NaN` for non-numeric columns.

## [0.1.0] - 2026-06-21

First tagged release. Syralit is installable with
`go install github.com/HazelnutParadise/syralit/cmd/syralit@v0.1.0`.

### Core
- Rerun model: write a Go function, get a live web app over WebSocket — no
  JavaScript, HTML templates, or frontend build step.
- Session state (`State`, `Session`), caching (`CacheData`, `CacheResource`),
  query params, `Stop`/`Rerun`, `Fragment` partial reruns.
- Multi-page apps (`AddPage`, `Navigation`, `PageLink`, `SwitchPage`), auth
  (`LoginGate`, `User`, `Login`, `Logout`), config & secrets (`syralit.toml`,
  `SetPageConfig`, `Secrets`), DB (`Connection`, `SQLQuery`).
- `RenderOnce(appFn) *Node` plus `Node.Find` for unit-testing widgets and
  integrations without a server.

### Widgets
- Inputs: Button (with `Icon`/`ButtonType`/`UseContainerWidth`), TextInput,
  PasswordInput, TextArea, NumberInput, Slider, RangeSlider, DateSlider,
  SelectSlider, Checkbox, Toggle, Radio, SelectBox, MultiSelect, DateInput,
  DateRangeInput (with `MinDate`/`MaxDate`), TimeInput, ColorPicker,
  FileUploader, CameraInput, AudioInput, ChatInput, Feedback, SegmentedControl,
  Pills, Pagination, DownloadButton, LinkButton, PageLink, Badge.
- Display: Title/Header/Subheader/Text/Markdown/Caption, Code, LaTeX, JSON,
  HTML, Image, Audio, Video, Metric (with delta and `Border`), Progress,
  Spinner, WriteStream (token streaming), Exception, Component, IFrame.
- Data: Table; DataFrame (sortable, optional row selection via `Selectable`,
  typed display via `ColConfig`); DataEditor (11 column types, dynamic rows).
- Charts: Chart.js line/bar/area/scatter/pie/doughnut/histogram/radar/graphviz,
  plus CDN integrations VegaLite, Plotly, Pyplot, Bokeh, PyDeck.
- Layout: Columns/WeightedColumns, Tabs, Sidebar, Expander, Container, Form
  (batched inputs incl. RangeSlider/DateRangeInput/DateSlider), Status, Popover,
  Dialog, Empty. Maps via Leaflet.

### CLI
- `syralit new <name>` / `syralit new .` (scaffold, in-place mode).
- `syralit dev` (hot reload with state preservation), `syralit run`.
- `syralit build` — compile to a single self-contained executable, folding the
  framework front-end plus the project's `public/` and `assets/` into the binary
  via generated `//go:embed`.

### Insyra integration (`integrations/insyra`)
- DataTable helpers: Table, Preview, EditableTable, ColumnSelect, Metrics, and
  charts (Bar/Line/Area/Scatter/Pie).
- DataList helpers: List, ListPreview, EditableList, ListMetrics, ListDescribe,
  ListBarChart/ListLineChart/ListAreaChart, Histogram.
- The core framework never imports Insyra; the adapter is fully isolated.

### Theming
- Light/dark with system preference and a theme toggle; CSS-variable theming
  (`PrimaryColor`, accent, radius); themed scrollbars.
