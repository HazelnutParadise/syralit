---
name: syralit-dev
description: Build interactive data apps, dashboards, and AI tool UIs in Go with the Syralit framework. Use when writing or editing Syralit apps — adding widgets, charts, layouts, forms, chat, state, caching, auth, or multi-page navigation with the `sy` (github.com/HazelnutParadise/syralit) package.
---

# Syralit App Development

You are an expert Syralit developer. Syralit is a Go-native framework for building interactive data apps — inspired by Streamlit, designed for Go.

## Core Concepts

### Rerun Model
Syralit re-executes the entire app function on every widget interaction. All UI is declared imperatively inside `sy.App(func(){ ... })`. There is no virtual DOM — the server sends a full UI tree to the browser via WebSocket on each rerun.

### Import Convention
```go
import sy "github.com/HazelnutParadise/syralit"
```

### Minimal App
```go
package main

import sy "github.com/HazelnutParadise/syralit"

func main() {
    sy.App(func() {
        sy.Title("My App")
        name := sy.TextInput("Name")
        if name != "" {
            sy.Success("Hello, " + name + "!")
        }
    })
}
```

### Multi-Page App
```go
func init() {
    sy.AddPage("Home", homePage, sy.PageIcon("🏠"), sy.PageOrder(1))
    sy.AddPage("Settings", settingsPage, sy.PageIcon("⚙️"), sy.PageOrder(2))
}

func main() { sy.App(nil) }
```

## API Reference

### Input Widgets (all return the current value)
| Function | Returns | Description |
|----------|---------|-------------|
| `Button(label, opts...)` | `bool` | True on click (one rerun only) |
| `TextInput(label, opts...)` | `string` | Single-line text |
| `PasswordInput(label, opts...)` | `string` | Masked text |
| `TextArea(label, opts...)` | `string` | Multi-line text |
| `NumberInput(label, opts...)` | `float64` | Number with min/max/step |
| `Slider(label, min, max, opts...)` | `float64` | Range slider |
| `RangeSlider(label, min, max, opts...)` | `(float64, float64)` | Two-handle (low, high) range |
| `DateSlider(label, minDate, maxDate, opts...)` | `string` | Slider over a date range → "YYYY-MM-DD" |
| `TimeSlider(label, minTime, maxTime, opts...)` | `string` | Slider over a time range → "HH:MM" |
| `SelectSlider(label, options, opts...)` | `string` | Discrete slider |
| `Checkbox(label, opts...)` | `bool` | Checkbox |
| `Toggle(label, opts...)` | `bool` | Toggle switch |
| `Radio(label, options, opts...)` | `string` | Radio group |
| `SelectBox(label, options, opts...)` | `string` | Dropdown (auto-searchable at 20+ items) |
| `MultiSelect(label, options, opts...)` | `[]string` | Multi-select |
| `DateInput(label, opts...)` | `string` | Date picker (YYYY-MM-DD) |
| `DateRangeInput(label, opts...)` | `(string, string)` | Start/end date pickers |
| `TimeInput(label, opts...)` | `string` | Time picker (HH:MM) |
| `ColorPicker(label, opts...)` | `string` | Color hex |
| `FileUploader(label, opts...)` | `*UploadedFile` | File upload (nil if empty) |
| `CameraInput(label, opts...)` | `string` | Webcam capture (base64 data URI) |
| `AudioInput(label, opts...)` | `string` | Microphone recording (base64) |
| `ChatInput(placeholder, opts...)` | `string` | Chat input box |
| `Feedback(opts...)` | `string` | Thumbs up/down; `sy.FeedbackStyle("stars"/"faces")` for other styles |
| `SegmentedControl(label, options, opts...)` | `string` | Segmented buttons (`SegmentedControlMulti` → `[]string`) |
| `Pills(label, options, opts...)` | `string` | Pill buttons (`PillsMulti` → `[]string`) |
| `Pagination(totalPages, opts...)` | `int` | Page selector (1-based) |
| `DownloadButton(label, data, filename, opts...)` | — | File download |
| `LinkButton(label, url, opts...)` | — | External link button |
| `PageLink(label, page, opts...)` | — | Internal page link |

### Common Options
```go
sy.Key("unique_key")           // stable widget identity
sy.DefaultValue(val)           // initial value
sy.Placeholder("hint")        // placeholder text
sy.Help("tooltip text")       // help tooltip
sy.Disabled()                 // disable widget
sy.Min(0), sy.Max(100)        // numeric range
sy.Step(0.5)                  // numeric step
sy.Height(300)                // height in px
sy.Width(400)                 // width in px
sy.MaxChars(100)              // text length limit
sy.MaxSelections(3)           // multiselect limit
sy.LabelHidden()              // hide label
sy.LabelCollapsed()           // collapse label
sy.MimeType("text/csv")       // file type hint
sy.Language("go")             // code language
sy.Border()                   // container border
sy.Color("green")             // badge/element color
sy.DynamicRows()              // DataEditor add/delete rows
sy.Expanded()                 // expander starts open
sy.VerticalAlignment("center") // columns alignment
sy.Icon("🚀")                 // prefix a button label with an icon
sy.ButtonType("secondary")    // button style: primary/secondary/tertiary
sy.UseContainerWidth()        // button spans its container
sy.Border()                   // bordered Metric card (also container border)
sy.MinDate("2026-01-01")      // DateInput/DateRangeInput lower bound
sy.MaxDate("2026-12-31")      // DateInput/DateRangeInput upper bound
```

### Display
```go
sy.Title("text")
sy.Header("text")
sy.Subheader("text")
sy.Text("text")
sy.Textf("format %d", val)        // fmt.Sprintf style
sy.Markdown("**bold** _italic_")
sy.Caption("small text")
sy.Code(code, sy.Language("go"))   // syntax highlighted
sy.LaTeX(`E = mc^2`)              // KaTeX rendered
sy.JSON(anyValue)                  // interactive JSON tree
sy.HTML("<b>raw html</b>")
sy.Badge("label", sy.Color("green"))
sy.Image(src, opts...)
sy.ImageFromBytes(data, mime, opts...)
sy.Audio(src)
sy.Video(src)
sy.Link(text, url)
sy.Metric(label, value, sy.Delta("+5%"), sy.DeltaColor("normal"))
sy.Progress(0.75)                  // 0.0 to 1.0
sy.Spinner("Loading...")
sy.Write(args...)                  // auto-detect: string→markdown, error→error, else→JSON
sy.WriteStream(id, func(w func(string)) { w("token") })  // streaming text
sy.ArtifactCanvas(store, opts...)  // shared, animated agent-updatable canvas
sy.Component(html, opts...)        // custom HTML/JS in iframe
sy.IFrame(url, opts...)
```

### Data
```go
// Static table
sy.Table(headers []string, rows [][]string)

// Sortable data frame. Add sy.Selectable() to get row selection (returns
// selected indices into the original rows); sy.ColConfig renders cells by type.
sy.DataFrame(headers []string, rows [][]any, opts...)
selected := sy.DataFrame(headers, rows, sy.Selectable(), sy.Key("df")) // []int

// Editable data editor — returns current rows
edited := sy.DataEditor(headers, rows, opts...)

// Column configuration for DataEditor and DataFrame. Fields: Type, Options,
// Width, Min, Max, Step, Format ("$%.2f"/"%d%%"), Label, Help, Color.
// Types: text/number/checkbox/select/date/time/datetime/link/image/progress/
// list, plus display-only bar_chart / line_chart (cell value = []float64).
sy.ColConfig(map[string]sy.ColumnConfig{
    "Score":  {Type: "number", Min: 0, Max: 100, Format: "%.1f"},
    "Pass":   {Type: "checkbox"},
    "Grade":  {Type: "select", Options: []string{"A","B","C"}},
    "Due":    {Type: "date"},
    "Link":   {Type: "link"},
    "Trend":  {Type: "line_chart", Label: "30-day", Color: "#7c3aed"},
    "Avatar": {Type: "image"},
    "Done":   {Type: "progress", Max: 100},
})
```

### Charts (all powered by Chart.js, interactive)
```go
sy.LineChart(map[string][]float64{"Series": {1,2,3}}, opts...)
sy.BarChart(data, opts...)
sy.AreaChart(data, opts...)
sy.PieChart(map[string]float64{"A": 30, "B": 70}, opts...)
sy.DoughnutChart(data, opts...)
sy.ScatterChart(map[string][][2]float64{"S": {{1,2},{3,4}}}, opts...)
sy.HistogramChart([]float64{...}, bins, opts...)
sy.RadarChart(labels, map[string][]float64{"S": {80,90,70}}, opts...)

// Chart options
sy.ChartTitle("Title")
sy.XLabels([]string{"Jan","Feb","Mar"})
sy.Height(400)
```

### External Charting Libraries (CDN-loaded, accept JSON specs)
```go
sy.VegaLiteChart(spec map[string]any, opts...)   // Altair/Vega-Lite
sy.PlotlyChart(spec map[string]any, opts...)     // Plotly.js
sy.PyplotChart(svgOrBase64 string, opts...)       // SVG/PNG image
sy.BokehChart(spec map[string]any, opts...)      // BokehJS
sy.PydeckChart(spec map[string]any, opts...)     // deck.gl 3D maps
sy.GraphvizChart(dot string, opts...)            // Graphviz DOT
sy.Map([]sy.MapPoint{{Lat: 25.03, Lon: 121.56, Text: "Taipei"}}, opts...)
```

### Layout
```go
// Equal columns
cols := sy.Columns(3)
cols[0](func() { sy.Text("Col 1") })
cols[1](func() { sy.Text("Col 2") })
cols[2](func() { sy.Text("Col 3") })

// Weighted columns
cols := sy.WeightedColumns(2, 1, 1)

// Tabs
tab := sy.Tabs([]string{"Tab1", "Tab2"})
tab("Tab1", func() { sy.Text("Content 1") })
tab("Tab2", func() { sy.Text("Content 2") })

// Sidebar
sy.Sidebar(func() { sy.Text("Sidebar content") })

// Expander
sy.Expander("Title", func() { sy.Text("Hidden content") })

// Container with optional border and height
sy.Container(func() { sy.Text("Boxed") }, sy.Border(), sy.Height(200))

// Popover
sy.Popover("Click me", func() { sy.Text("Floating content") })

// Empty placeholder
placeholder := sy.Empty()
// later: placeholder can be replaced

// Divider
sy.Divider()
```

### Forms (batch changes, no rerun until submit)
```go
sy.Form("form_key", func() {
    name := sy.TextInput("Name", sy.Key("f_name"))
    email := sy.TextInput("Email", sy.Key("f_email"))
    if sy.FormSubmitButton("Submit") {
        // process name, email
        sy.Toast("Submitted!", "success")
    }
}, sy.ClearOnSubmit()) // optional: reset inputs after submit
```

### Dialog (modal)
```go
sy.Dialog("Settings", func() {
    sy.TextInput("Name", sy.Key("d_name"))
    if sy.Button("Save", sy.Key("d_save")) {
        sy.CloseDialog("Settings")
    }
}, sy.Key("Settings"))

if sy.Button("Open Settings") {
    sy.ShowDialog("Settings")
}
```

### Fragment (partial rerun)
```go
sy.Fragment("counter", func() {
    count := sy.State("n", 0)
    if sy.Button("Add", sy.Key("frag_add")) {
        count.Set(count.Get() + 1)
    }
    sy.Textf("Count: %d", count.Get())
})

// Server-driven live refresh: re-run just this fragment every interval.
sy.Fragment("clock", func() {
    sy.Textf("%s", time.Now().Format("15:04:05"))
}, sy.RunEvery(time.Second))
```

### Status & Feedback
```go
sy.Success("Done!")
sy.Info("Note: ...")
sy.Warning("Watch out")
sy.Error("Failed")
sy.Exception(err)               // styled monospace error box; nil renders nothing
sy.Toast("Message", "success")  // "success", "info", "warning", "error"
sy.Balloons()
sy.Snow()
sy.Status("Loading data", "running", func() {
    sy.Text("Processing...")
    sy.Progress(0.5)
})
// Status levels: "running", "complete", "error"
```

### Chat UI
```go
msgs := sy.State("msgs", []map[string]string{})
for _, m := range msgs.Get() {
    sy.ChatMessage(m["role"], func() {
        sy.Markdown(m["content"])
    })
}
if input := sy.ChatInput("Ask something..."); input != "" {
    msgs.Set(append(msgs.Get(), map[string]string{"role": "user", "content": input}))
}
```

### State Management
```go
// Typed state (persists across reruns within a session)
count := sy.State("count", 0)       // *StateVar[int]
count.Get()                          // read
count.Set(42)                        // write (triggers rerun)

// Raw session store
sy.Session().Set("key", value)
sy.Session().Get("key")

// Query parameters
val := sy.QueryParam("page")
all := sy.QueryParams()

// Request context (headers, cookies, host, IP, locale) — st.context
ctx := sy.Context()
lang := ctx.Locale
```

### Caching
```go
// Cache expensive computation
data := sy.CacheData("key", func() []Row {
    return fetchFromDB()
}, sy.TTL(5 * time.Minute))

// Cache resources (DB connections, ML models)
db := sy.CacheResource("db", func() *sql.DB {
    return openDB()
})

sy.ClearCache()  // invalidate all
```

### Background tasks (beyond Streamlit)
```go
// Runs fn once in a goroutine, keyed per session. The page stays responsive;
// the server pushes a rerun when the job finishes. Returns a typed handle.
job := sy.Task("report", func() Report { return buildReport() })
if job.Running() {
    sy.Spinner("Working…")
} else if job.Err() != nil {
    sy.Exception(job.Err())
} else {
    render(job.Result()) // Result() is typed (Report)
}
```

### Offline / self-hosted libraries (beyond Streamlit)
```go
// Repoint a CDN lib to a self-hosted copy (served from public/), so syralit
// build produces a fully offline / air-gapped / CSP-safe single binary.
sy.SetAssetURL("chartjs", "/chart.umd.min.js")
// names: chartjs, leaflet_js/css, katex_js/css, highlight_js/css/css_dark,
// viz, vega/vega_lite/vega_embed, plotly, bokeh, deckgl, mapbox_js/css
```

### Shared state (beyond Streamlit — real-time, cross-session)
```go
// App-wide value shared by ALL sessions; Set/Update live-pushes to everyone.
online := sy.Shared("online", 0)
online.Update(func(v int) int { return v + 1 }) // atomic read-modify-write
sy.Metric("Online now", fmt.Sprint(online.Get()))
```

### Agent Artifact Canvas
```go
// Safe DSL for an agent-updatable canvas. The DSL maps to a curated subset of
// Syralit components, not raw HTML/JS or the internal Node protocol.
board := sy.NewArtifactStore("main", sy.ArtifactSpec{
    Version: "v1",
    Layout:  sy.ArtifactLayout{Columns: 2, Gap: 14, Padding: 16},
    Data: map[string]any{
        "summary": map[string]any{"revenue": "$42k"},
    },
    Nodes: []sy.ArtifactNode{{
        ID:        "revenue",
        Component: "metric",
        Props:     map[string]any{"label": "Revenue"},
        Bind:      map[string]string{"props.value": "/summary/revenue"},
    }},
})

// Opt-in POST endpoint. Requires Authorization: Bearer <token>.
sy.HandleArtifactEndpoint(
    "/api/agent/artifacts/main",
    board,
    sy.StaticAgentKey("local-agent", sy.Secrets("AGENT_KEY")),
)

sy.App(func() {
    sy.ArtifactCanvas(board, sy.Height(520))
})

// Full-replace update payload:
// {"spec":{"version":"v1","nodes":[{"id":"msg","component":"text","props":{"text":"Updated"}}]}}
```
Allowed artifact components: `text`, `markdown`, `metric`, `table`,
`dataframe`, `line_chart`, `bar_chart`, `pie_chart`, `image`, `progress`,
`container`. Data binding uses JSON Pointer from `ArtifactSpec.Data`, e.g.
`Bind: map[string]string{"props.value": "/summary/revenue"}`. Every artifact
node needs a stable `ID` so the browser can animate enter/update/exit states.

For app-owned key storage, implement:
```go
type AgentAuthenticator interface {
    AuthenticateAgent(ctx context.Context, token string) (sy.AgentPrincipal, bool, error)
}

type AgentKeyStore interface {
    AgentAuthenticator
    ListAgentKeys(ctx context.Context) ([]sy.AgentKeyInfo, error)
    CreateAgentKey(ctx context.Context, name string) (plainToken string, info sy.AgentKeyInfo, err error)
    RevokeAgentKey(ctx context.Context, id string) error
}

sy.AgentKeyManager(store, sy.Key("agent-keys"))
```
Syralit supplies the UI/callback contract, but does not persist keys for the
app.

For agents that need to generate Artifact DSL JSON directly, use the dedicated
`skills/syralit-artifact-dsl/SKILL.md` skill. Keep `syralit-dev` for framework
API usage and app wiring, and use the artifact skill when the task is "return a
valid payload" rather than "build a whole app".

### Configuration
```go
sy.SetPageConfig(
    sy.PageTitle("My App"),
    sy.PageLayout("wide"),      // "centered" (default) or "wide"
    sy.ConfigIcon("🚀"),
    sy.ConfigLogo("/logo.png"),
    sy.PrimaryColor("#ff4b4b"),
    sy.BackgroundColor("#0e1117"),
    sy.TextColor("#fafafa"),
)

// Secrets (from syralit.toml [secrets] section)
apiKey := sy.Secrets("api_key")

// Database connection
db := sy.Connection("mydb")  // DSN from Secrets
rows := sy.SQLQuery(db, "SELECT * FROM users")
```

### Auth
```go
// Simple login gate
username := sy.LoginGate(func(user, pass string) bool {
    return user == "admin" && pass == "secret"
})
sy.Textf("Welcome, %s", username)

// Manual auth
sy.Login(map[string]string{"name": "admin", "role": "admin"})
user := sy.User()  // map[string]string or nil
sy.Logout()
```

### Navigation
```go
// Declarative (newer API)
active := sy.Navigation([]sy.Page{
    {Title: "Home", Fn: homePage, Icon: "🏠"},
    {Title: "About", Fn: aboutPage, Icon: "ℹ️"},
})

// Programmatic switching
sy.SwitchPage("About")
```

### Developer Tools
```bash
syralit new myapp    # scaffold project
syralit dev          # hot reload with state preservation
syralit run          # production mode
```

### File Config (syralit.toml)
```toml
title = "My App"
host = "0.0.0.0"
port = 8600

[theme]
primary_color = "#ff4b4b"

[secrets]
api_key = "sk-..."
db_dsn = "postgres://..."
```

## Patterns

### Conditional UI
```go
mode := sy.Radio("Mode", []string{"Simple", "Advanced"})
if mode == "Advanced" {
    sy.NumberInput("Threshold", sy.Key("threshold"))
}
```

### Guard Pattern
```go
if sy.User() == nil {
    sy.LoginGate(checkCredentials)
    sy.Stop()  // halt rendering here
}
// authenticated content below
```

### Streaming LLM Output
```go
sy.WriteStream("response", func(w func(string)) {
    for _, word := range strings.Fields(response) {
        w(word + " ")
        time.Sleep(30 * time.Millisecond)
    }
})
```

## Insyra Integration
```go
import syi "github.com/HazelnutParadise/syralit/integrations/insyra"

// DataTable (multi-column)
syi.Table(dt)                           // render DataTable
syi.Preview(dt, 5)                      // first N rows
syi.EditableTable(dt, sy.Key("edit"))   // editable
syi.Metrics(dt, "column")              // statistics for one column
syi.BarChart(dt, "x_col", "y_col")     // chart from columns
col := syi.ColumnSelect("Pick", dt)    // column picker

// DataList (single series) — symmetric helpers
syi.List(dl)                            // single-column table
syi.ListPreview(dl, 5)                  // first N values
syi.EditableList(dl, sy.Key("edl"))     // editable → []any
syi.ListMetrics(dl)                     // count, mean, min, max
syi.ListDescribe(dl)                    // count/mean/std/min/25%/50%/75%/max
syi.ListBarChart(dl)                    // also ListLineChart / ListAreaChart
syi.Histogram(dl, 20)                   // distribution (list-only)

// Statistical analysis (insyra/stats) — rendered in the UI
syi.Describe(dt)                            // per-column summary table
syi.Correlation(dt, "X", "Y", "pearson")   // r + p metrics (pearson/spearman/kendall)
syi.CorrelationMatrix(dt, "pearson")       // correlation matrix
syi.LinearRegression(dt, "Y", "X1", "X2")  // R²/coeff table (+ scatter if 1 predictor)
syi.TTest(dt, "A", "B", false)             // two-sample t-test

// File upload → DataTable (CSV / Excel / JSON; NOT parquet)
dt := syi.UploadTable("Upload data")       // uploader → *DataTable (nil until uploaded)
dt, err := syi.ParseTable(name, bytes)     // parse bytes from any source

// Interactive transforms (operate on a Clone — source untouched)
out := syi.FilterBuilder(dt)               // column/operator/value row filter
out := syi.CCLBuilder(dt)                  // add computed column via CCL; applied on "Apply"
```
Note: name a column/list with `insyra.NewDataList(vals...).SetName("X")` so
`GetColByName` and table/chart headers work.

### Native go-echarts charts (opt-in subpackage)
Interactive chart types Chart.js lacks (Sankey, word cloud, K-line, gauge,
funnel, theme-river, box plot, radar). Separate package because it pulls in
go-echarts + chromedp:
```go
import syiplot "github.com/HazelnutParadise/syralit/integrations/insyra/eplot"
import "github.com/HazelnutParadise/insyra/plot"

syiplot.WordCloud(dl, "Tags")
syiplot.EChart(plot.CreateSankeyChart(cfg, links...), sy.Height(560))
syiplot.SetOffline(true) // inline echarts JS — no CDN (air-gapped / syralit build)
```
`EChart` renders any `insyra/plot` chart via a sandboxed iframe. By default
go-echarts loads its JS from a CDN; call `syiplot.SetOffline(true)` to embed the
echarts JavaScript in each chart so it works with no internet (heavier HTML).

## Common Mistakes
1. **Missing Key in loops/conditionals**: Widgets in `if` blocks or loops need explicit `sy.Key()` for stable identity.
2. **Button returns true only once**: `sy.Button()` is true for exactly one rerun, then resets. Store the result in state if needed.
3. **Form widgets need Key**: Every widget inside `sy.Form()` must have a unique `sy.Key()`.
4. **State default type matters**: `sy.State("x", 0)` creates `int`, `sy.State("x", 0.0)` creates `float64`.
5. **Don't use goroutines for UI**: All widget calls must happen on the main rerun goroutine. Use `CacheData` for async work.
