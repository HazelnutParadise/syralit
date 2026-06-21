# Streamlit parity

How Syralit's API maps to Streamlit's, and the handful of intentional gaps.
Syralit aims for Streamlit-level coverage of the commonly-used surface, in
idiomatic Go (multiple return values instead of Python's dynamic returns).

## Covered

### Text & display
`st.write`→`Write`, `st.write_stream`→`WriteStream`, `st.markdown`→`Markdown`,
`st.title/header/subheader/caption/text`→same, `st.code`→`Code` (+ `LineNumbers`,
`Wrap`), `st.latex`→`LaTeX`, `st.divider`→`Divider`, `st.json`→`JSON`
(interactive collapsible tree), `st.metric`→`Metric` (+ `Delta`, `Border`,
`Help`), `st.echo`→`Echo`, `st.exception`→`Exception`, `st.html`→`HTML`,
`st.badge`→`Badge`.

### Input widgets
`button`(+`Icon`/`ButtonType`/`UseContainerWidth`/`Help`), `text_input`,
`text_area`, `password`, `number_input`, `slider`, **range slider**→`RangeSlider`,
**date slider**→`DateSlider`, **time slider**→`TimeSlider`, `select_slider`,
`checkbox`, `toggle`, `radio`, `selectbox`, `multiselect`, `date_input`
(+`MinDate`/`MaxDate`), **date range**→`DateRangeInput`, `time_input`,
`color_picker`, `file_uploader`, `camera_input`, `audio_input`, `chat_input`,
`feedback`(thumbs/stars/faces), `pills`(+`PillsMulti`),
`segmented_control`(+`SegmentedControlMulti`), `download_button`, `link_button`,
`page_link`. Pagination helper too.

### Data
`st.table`→`Table`, `st.dataframe`→`DataFrame` (sortable, row selection via
`Selectable`, typed display via `ColConfig`), `st.data_editor`→`DataEditor`
(11 column types + dynamic rows). `st.column_config`→`ColConfig` with Type,
Options, Width, Min/Max/Step, Format, Label, Help, Color, and the display-only
`bar_chart`/`line_chart` sparkline columns.

### Charts
`line/bar/area/scatter`→built-in Chart.js (+ `Stacked`, `Horizontal`, `Colors`,
`XLabels`), plus `pie/doughnut/histogram/radar/graphviz`. `st.map`→`Map`
(+ `Zoom`, per-point Size/Color). External: `altair/vega_lite`→`VegaLiteChart`,
`plotly`→`PlotlyChart`, `pyplot`→`PyplotChart`, `bokeh`→`BokehChart`,
`pydeck`→`PydeckChart`.

### Media
`image`(+`UseContainerWidth`), `audio`/`video`(+`Autoplay`/`Loop`/`Muted`),
`logo`→`ConfigLogo`.

### Layout & containers
`columns`(+weighted), `tabs`, `expander`(+`Icon`), `container`(+`Border`/`Height`
scroll), `sidebar`, `popover`(+icon/width/disabled/help), `empty`, `form`(+`ClearOnSubmit`) +
`form_submit_button` (batched inputs), `status`, `dialog`.

### Flow, state, platform
`session_state`→`State`/`Session`, `cache_data`/`cache_resource`,
`rerun`/`stop`, `fragment`→`Fragment` (+ **`RunEvery`** auto-refresh),
`query_params`→`QueryParam(s)`, `secrets`→`Secrets`, `connection`→`Connection`,
`context`→`Context` (headers/cookies/host/IP/locale), `navigation`/`Page`→
`AddPage`/`Navigation`, `switch_page`→`SwitchPage`, `set_page_config`→
`SetPageConfig`, auth (`LoginGate`/`User`/`Login`/`Logout`).

### Status & chrome
`success`/`info`/`warning`/`error`, `toast`(+icon), `balloons`/`snow`,
`progress`(+text), `spinner`, themed light/dark with toggle.

### Insyra integration (`integrations/insyra`)
DataTable + DataList helpers (`Table`, `List`, `ListMetrics`, `ListDescribe`,
charts, `Histogram`, …) — no equivalent in Streamlit; Syralit-specific.

## Beyond Streamlit
Capabilities with no Streamlit equivalent:
- **`Task[T]`** — background jobs in a goroutine; the server pushes the result
  when done, so the UI never blocks (a Streamlit rerun blocks the session).
- **`Shared[T]`** — app-wide state across sessions; `Set`/`Update` live-pushes to
  every client (real-time collaboration; Streamlit state is per-session only).
- **`Fragment` + `RunEvery`** — server-driven live refresh of one fragment.
- **`syralit build`** — single self-contained binary (front-end + backend +
  `public/`), no runtime/interpreter or dependencies.
- **Typed** state and task results via Go generics.

## Intentional / known gaps
- **`st.form(clear_on_submit=True)`** — supported via `ClearOnSubmit`. Caveat:
  inputs reset to type defaults (text→"", number→0, select→first option), not to
  a widget's custom `DefaultValue`.
- **`st.help`** — relies on Python runtime docstrings; no Go equivalent.
- **`st.number_input(format=...)`** display formatting — native numeric input;
  use a `Text` line or DataFrame `ColConfig` Format for formatted display.
- **pandas `Styler` / `st.dataframe` cell & column selection** — only row
  selection is supported.
- Deprecated `st.experimental_*` APIs — not ported.
