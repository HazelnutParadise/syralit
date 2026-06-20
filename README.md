# Syralit

**Interactive data apps in Go.**

Syralit is a Go-native framework for building interactive data apps, dashboards, and AI tool interfaces — inspired by Streamlit, designed for Go.

Write Go functions, get a live web app. No JavaScript, no HTML templates, no frontend build step.

```go
package main

import sy "github.com/HazelnutParadise/syralit"

func main() {
    sy.App(func() {
        sy.Title("Hello Syralit")
        name := sy.TextInput("Your name")
        if name != "" {
            sy.Success("Hello, " + name + "!")
        }
    })
}
```

## Installation

```bash
go install github.com/HazelnutParadise/syralit/cmd/syralit@latest
```

## Quick Start

```bash
syralit new myapp    # scaffold a new project
cd myapp
syralit dev          # hot reload with state preservation
```

Or manually:

```go
package main

import sy "github.com/HazelnutParadise/syralit"

func main() {
    sy.App(func() {
        sy.Title("My App")
        if sy.Button("Click me") {
            sy.Balloons()
        }
    })
}
```

```bash
go run .
# Open http://localhost:8600
```

## Features

### Input Widgets

| Widget | Returns | Description |
|--------|---------|-------------|
| `Button` | `bool` | Clickable button (true for one rerun) |
| `TextInput` | `string` | Single-line text |
| `PasswordInput` | `string` | Masked text input |
| `TextArea` | `string` | Multi-line text |
| `NumberInput` | `float64` | Number with min/max/step |
| `Slider` | `float64` | Range slider |
| `SelectSlider` | `string` | Discrete slider with labels |
| `Checkbox` | `bool` | Checkbox |
| `Toggle` | `bool` | Toggle switch |
| `Radio` | `string` | Radio button group |
| `SelectBox` | `string` | Dropdown (auto-searchable at 20+ items) |
| `MultiSelect` | `[]string` | Multi-select dropdown |
| `DateInput` | `string` | Date picker (YYYY-MM-DD) |
| `TimeInput` | `string` | Time picker (HH:MM) |
| `ColorPicker` | `string` | Color hex picker |
| `FileUploader` | `*UploadedFile` | File upload |
| `CameraInput` | `string` | Webcam capture |
| `AudioInput` | `string` | Microphone recording |
| `ChatInput` | `string` | Chat message input |
| `Feedback` | `string` | Thumbs up/down |
| `SegmentedControl` | `string` | Segmented buttons |
| `Pills` | `string` | Pill-style buttons |
| `Pagination` | `int` | Page selector |

Plus: `DownloadButton`, `LinkButton`, `PageLink`, `Badge`.

### Display

Title, Header, Subheader, Text, Textf, Markdown, Caption, Code (syntax highlighting via highlight.js), LaTeX (KaTeX), JSON (interactive tree), HTML, Image, ImageFromBytes, Audio, Video, Link, Metric (with delta indicators), Progress, Spinner, WriteStream (token-by-token streaming), Component (custom HTML/JS), IFrame.

### Data

| Widget | Description |
|--------|-------------|
| `Table` | Static string table |
| `DataFrame` | Sortable data table with mixed types |
| `DataEditor` | Editable table with 11 column types |

DataEditor supports column configuration with types: `text`, `number`, `checkbox`, `select`, `date`, `time`, `datetime`, `link`, `image`, `progress`, `list`. Dynamic row add/delete with `sy.DynamicRows()`.

```go
sy.DataEditor(headers, rows,
    sy.ColConfig(map[string]sy.ColumnConfig{
        "Score": {Type: "number", Min: 0, Max: 100},
        "Pass":  {Type: "checkbox"},
        "Grade": {Type: "select", Options: []string{"A", "B", "C"}},
    }),
    sy.DynamicRows(),
)
```

### Charts

Built-in interactive charts powered by **Chart.js**:

| Chart | Input | Description |
|-------|-------|-------------|
| `LineChart` | `map[string][]float64` | Line chart with multiple series |
| `BarChart` | `map[string][]float64` | Bar chart |
| `AreaChart` | `map[string][]float64` | Filled line chart |
| `ScatterChart` | `map[string][][2]float64` | Scatter plot with xy pairs |
| `PieChart` | `map[string]float64` | Pie chart |
| `DoughnutChart` | `map[string]float64` | Doughnut chart |
| `HistogramChart` | `[]float64, bins` | Histogram from raw data |
| `RadarChart` | `labels, map[string][]float64` | Radar/spider chart |
| `GraphvizChart` | `dot string` | Graphviz DOT via viz.js |

External charting library integrations (CDN-loaded, accepting JSON specs):

| Chart | Library | Streamlit Equivalent |
|-------|---------|---------------------|
| `VegaLiteChart` | Vega-Lite / vega-embed | `st.altair_chart` |
| `PlotlyChart` | Plotly.js | `st.plotly_chart` |
| `PyplotChart` | SVG/PNG images | `st.pyplot` |
| `BokehChart` | BokehJS | `st.bokeh_chart` |
| `PydeckChart` | deck.gl | `st.pydeck_chart` |

### Layout

```go
// Columns (equal or weighted)
cols := sy.Columns(3)
cols[0](func() { sy.Text("Col 1") })

cols := sy.WeightedColumns(2, 1, 1)

// Tabs
tab := sy.Tabs([]string{"Tab1", "Tab2"})
tab("Tab1", func() { sy.Text("Content 1") })

// Other containers
sy.Sidebar(func() { ... })
sy.Expander("Title", func() { ... })
sy.Container(func() { ... }, sy.Border())
sy.Form("key", func() { ... })
sy.Status("Loading", "running", func() { ... })
sy.Fragment("key", func() { ... })  // partial rerun
```

### State & Session

```go
// Typed state (persists across reruns)
count := sy.State("count", 0)
count.Get()
count.Set(42)

// Query parameters
val := sy.QueryParam("page")

// Flow control
sy.Stop()   // halt rendering
sy.Rerun()  // force rerun
```

### Multi-Page Apps

```go
func init() {
    sy.AddPage("Home", homePage, sy.PageIcon("🏠"), sy.PageOrder(1))
    sy.AddPage("About", aboutPage, sy.PageIcon("ℹ️"), sy.PageOrder(2))
}

func main() { sy.App(nil) }
```

### Auth

```go
// Login gate blocks rendering until authenticated
username := sy.LoginGate(func(user, pass string) bool {
    return user == "admin" && pass == "secret"
})

// Role-based access
user := sy.User()  // map[string]string or nil
sy.Login(map[string]string{"name": "admin", "role": "admin"})
sy.Logout()
```

### Caching

```go
data := sy.CacheData("key", func() []Row {
    return fetchFromDB()
}, sy.TTL(5 * time.Minute))

db := sy.CacheResource("db", func() *sql.DB {
    return openDB()
})
```

### Feedback & Notifications

```go
sy.Success("Done!")
sy.Error("Failed!")
sy.Warning("Watch out")
sy.Info("Note")
sy.Toast("Message", "success")
sy.Balloons()
sy.Snow()
sy.Dialog("Settings", func() { ... })
```

### Streaming (LLM Output)

```go
sy.WriteStream(func(yield func(string)) {
    for _, word := range words {
        yield(word + " ")
        time.Sleep(30 * time.Millisecond)
    }
})
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
    msgs.Set(append(msgs.Get(), map[string]string{
        "role": "user", "content": input,
    }))
}
```

### Maps

```go
sy.Map([]sy.MapPoint{
    {Lat: 25.0330, Lon: 121.5654, Text: "Taipei 101"},
}, sy.Height(450))
```

### Database

```go
db := sy.Connection("mydb")
rows := sy.SQLQuery(db, "SELECT * FROM users")
```

## Configuration

### syralit.toml

```toml
title = "My App"
host = "0.0.0.0"
port = 8600

[theme]
primary_color = "#ff4b4b"
background_color = "#0e1117"
text_color = "#fafafa"

[secrets]
api_key = "sk-..."
db_dsn = "postgres://..."
```

### Runtime Configuration

```go
sy.SetPageConfig(
    sy.PageTitle("My App"),
    sy.PageLayout("wide"),
    sy.ConfigIcon("🚀"),
    sy.PrimaryColor("#ff4b4b"),
)

apiKey := sy.Secrets("api_key")
```

## Common Options

```go
sy.Key("unique_key")        // stable widget identity
sy.DefaultValue(val)         // initial value
sy.Placeholder("hint")      // placeholder text
sy.Help("tooltip")          // help tooltip
sy.Disabled()               // disable widget
sy.Min(0), sy.Max(100)      // numeric range
sy.Step(0.5)                // numeric step
sy.Height(300), sy.Width(400)
sy.ChartTitle("Title")      // chart title
sy.Border()                 // container border
sy.Color("green")           // element color
sy.Language("go")           // code language
```

## Insyra Integration

Syralit has first-class support for [Insyra](https://github.com/HazelnutParadise/insyra) DataTables via a cleanly separated adapter package. The core framework never imports Insyra.

```go
import syi "github.com/HazelnutParadise/syralit/integrations/insyra"

syi.Table(dt)                           // render DataTable
syi.Preview(dt, 5)                      // first N rows
syi.EditableTable(dt, sy.Key("edit"))   // editable DataTable
col := syi.ColumnSelect("Column", dt)   // column picker
syi.Metrics(dt, col)                    // count, mean, min, max
syi.BarChart(dt, "Category", "Value")   // chart from columns
syi.LineChart(dt, "Month", "Revenue")
syi.ScatterChart(dt, "X", "Y")
```

## Examples

The [`examples/`](examples/) directory contains runnable demo apps:

| Example | Description |
|---------|-------------|
| [`hello`](examples/hello/) | Minimal single-page app with basic widgets |
| [`showcase`](examples/showcase/) | Comprehensive 6-page demo of all features |
| [`chatbot`](examples/chatbot/) | Chat UI with simulated streaming AI responses |
| [`form-app`](examples/form-app/) | Conference registration form with validation |
| [`data-explorer`](examples/data-explorer/) | 3-page sales dashboard with charts, filters, and data editing |
| [`auth-demo`](examples/auth-demo/) | Authentication with LoginGate and role-based access control |
| [`mega-demo`](examples/mega-demo/) | 10-page app showcasing every feature: all widgets, charts, layout, forms, data tables, chat, maps, state |
| [`insyra-demo`](examples/insyra-demo/) | Insyra DataTable integration demo |

Run any example:

```bash
cd examples/chatbot
go run .
```

## Agent Skills

The [`Skills/`](Skills/) directory contains [Agent Skills](https://docs.claude.com/en/docs/claude-code/skills) for building Syralit apps with AI coding assistants. The [`syralit-dev`](Skills/syralit-dev/SKILL.md) skill provides a complete API reference — copy the `Skills/syralit-dev/` folder into your project's skills directory (e.g. `.claude/skills/`).

## CLI Commands

| Command | Description |
|---------|-------------|
| `syralit new <name>` | Scaffold a new project |
| `syralit dev` | Hot reload with state preservation |
| `syralit run` | Build and run in production mode |

## Requirements

- Go 1.25+

## License

MIT
