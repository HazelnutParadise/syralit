# Changelog

All notable changes to Syralit are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/), and this project adheres to
[Semantic Versioning](https://semver.org/).

## [Unreleased]

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
- New example `examples/insyra-charts` — Sankey, gauge, funnel, pie and word
  cloud with an offline toggle.

### Fixed
- `Toggle` and `Checkbox` now honor `DefaultValue(true)` — previously they
  ignored the option and always started unchecked.

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
