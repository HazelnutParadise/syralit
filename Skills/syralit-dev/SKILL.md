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
| `SelectSlider(label, options, opts...)` | `string` | Discrete slider |
| `Checkbox(label, opts...)` | `bool` | Checkbox |
| `Toggle(label, opts...)` | `bool` | Toggle switch |
| `Radio(label, options, opts...)` | `string` | Radio group |
| `SelectBox(label, options, opts...)` | `string` | Dropdown (auto-searchable at 20+ items) |
| `MultiSelect(label, options, opts...)` | `[]string` | Multi-select |
| `DateInput(label, opts...)` | `string` | Date picker (YYYY-MM-DD) |
| `TimeInput(label, opts...)` | `string` | Time picker (HH:MM) |
| `ColorPicker(label, opts...)` | `string` | Color hex |
| `FileUploader(label, opts...)` | `*UploadedFile` | File upload (nil if empty) |
| `CameraInput(label, opts...)` | `string` | Webcam capture (base64 data URI) |
| `AudioInput(label, opts...)` | `string` | Microphone recording (base64) |
| `ChatInput(placeholder, opts...)` | `string` | Chat input box |
| `Feedback(opts...)` | `string` | Thumbs up/down ("up"/"down"/"") |
| `SegmentedControl(label, options, opts...)` | `string` | Segmented buttons |
| `Pills(label, options, opts...)` | `string` | Pill buttons |
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
sy.Component(html, opts...)        // custom HTML/JS in iframe
sy.IFrame(url, opts...)
```

### Data
```go
// Static table
sy.Table(headers []string, rows [][]string)

// Sortable data frame
sy.DataFrame(headers []string, rows [][]any, opts...)

// Editable data editor — returns current rows
edited := sy.DataEditor(headers, rows, opts...)

// Column configuration for DataEditor
sy.ColConfig(map[string]sy.ColumnConfig{
    "Score":  {Type: "number", Min: 0, Max: 100},
    "Pass":   {Type: "checkbox"},
    "Grade":  {Type: "select", Options: []string{"A","B","C"}},
    "Due":    {Type: "date"},
    "Link":   {Type: "link"},
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
})
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
```

### Status & Feedback
```go
sy.Success("Done!")
sy.Info("Note: ...")
sy.Warning("Watch out")
sy.Error("Failed")
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

syi.Table(dt)                           // render DataTable
syi.Preview(dt, 5)                      // first N rows
syi.EditableTable(dt, sy.Key("edit"))   // editable
syi.Metrics(dt, "column")              // statistics
syi.BarChart(dt, "x_col", "y_col")     // chart from columns
col := syi.ColumnSelect("Pick", dt)    // column picker
```

## Common Mistakes
1. **Missing Key in loops/conditionals**: Widgets in `if` blocks or loops need explicit `sy.Key()` for stable identity.
2. **Button returns true only once**: `sy.Button()` is true for exactly one rerun, then resets. Store the result in state if needed.
3. **Form widgets need Key**: Every widget inside `sy.Form()` must have a unique `sy.Key()`.
4. **State default type matters**: `sy.State("x", 0)` creates `int`, `sy.State("x", 0.0)` creates `float64`.
5. **Don't use goroutines for UI**: All widget calls must happen on the main rerun goroutine. Use `CacheData` for async work.
