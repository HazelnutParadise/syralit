# Changelog

All notable changes to Syralit are documented here. The format is based on
[Keep a Changelog](https://keepachangelog.com/), and this project adheres to
[Semantic Versioning](https://semver.org/).

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
