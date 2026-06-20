# Syralit

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

## Features

### Input Widgets

TextInput, PasswordInput, NumberInput, Slider, SelectSlider, TextArea, Checkbox, Toggle, Radio, SelectBox (auto-searchable for 20+ options), MultiSelect, DateInput, TimeInput, ColorPicker, FileUploader, CameraInput, AudioInput, Button, ChatInput, SegmentedControl, Pills, Pagination, Feedback

### Display

Title, Header, Subheader, Text, Markdown, Caption, Badge, Code (with syntax highlighting), LaTeX (KaTeX), JSON, HTML, Image, ImageFromBytes, Audio, Video, Table, DataFrame, DataEditor (editable tables with 11 column types: text, number, checkbox, select, date, time, datetime, link, image, progress, list), Link, LinkButton, PageLink, Metric (with delta), Progress, Spinner, Map (Leaflet.js), WriteStream (token-by-token streaming for LLM output), Component (custom HTML/JS widgets), IFrame

### Layout

Columns, WeightedColumns, Tabs, Sidebar, Expander, Container, Form, Status, Popover, Empty, Fragment, Divider

### Charts

Built-in charts powered by **Chart.js** with interactive tooltips, hover highlights, and responsive sizing:

LineChart, BarChart, AreaChart, ScatterChart, PieChart, DoughnutChart, HistogramChart, RadarChart, GraphvizChart (Graphviz DOT via viz.js)

Charting library integrations (CDN-loaded, accepting JSON specs):

- **VegaLiteChart** — Vega-Lite specs (Go equivalent of `st.altair_chart`)
- **PlotlyChart** — Plotly figure specs via Plotly.js (Go equivalent of `st.plotly_chart`)
- **PyplotChart** — SVG or base64 PNG chart images (Go equivalent of `st.pyplot`)
- **BokehChart** — Bokeh JSON docs via BokehJS (Go equivalent of `st.bokeh_chart`)

### State & Navigation

- Session state: `sy.State("key", defaultVal)` with `.Get()` / `.Set()`
- Multi-page apps: `sy.AddPage(title, fn, PageIcon(), PageOrder())`
- Declarative navigation: `sy.Navigation([]sy.Page{...})`
- Page switching: `sy.SwitchPage(title)`
- Query parameters: `sy.QueryParam("key")`, `sy.QueryParams()`
- Flow control: `sy.Stop()`, `sy.Rerun()`
- Database: `sy.Connection("name")` + `sy.SQLQuery(db, query)`
- Auth: `sy.LoginGate(checkFn)`, `sy.User()`, `sy.Login()`, `sy.Logout()`

### Feedback

Toast notifications, Balloons, Snow, Dialog (modal), Error/Warning/Info/Success status blocks

### Performance

- Fragment: wrap a function with `sy.Fragment(key, fn)` for partial reruns — only the fragment re-renders on widget change, not the entire app
- Caching: `sy.CacheData(key, fn)` and `sy.CacheResource(key, fn)` with optional `sy.TTL()`

### Configuration

- `syralit.toml` for file-based config (title, host, port, theme)
- Runtime: `sy.SetPageConfig(PageTitle(), PageLayout(), ConfigIcon(), ConfigLogo())`
- Theme: built-in light/dark toggle, custom colors via `PrimaryColor()`, `BackgroundColor()`, `TextColor()`
- Secrets management: `sy.Secrets("key")` reads from `[secrets]` in config

### Developer Experience

- `syralit dev` — hot reload with state preservation across rebuilds
- `syralit new <name>` — scaffold a new project
- `syralit run` — build and run in production mode
- Responsive mobile layout, loading states, error recovery with stack traces

## Installation

```bash
go install github.com/HazelnutParadise/syralit/cmd/syralit@latest
```

## Quick Start

```bash
syralit new myapp
cd myapp
syralit dev
```

## Insyra Integration

Syralit has first-class support for [Insyra](https://github.com/HazelnutParadise/insyra) DataTables via the `syinsyra` adapter package. The core framework never imports Insyra — the integration is cleanly separated.

```go
import syi "github.com/HazelnutParadise/syralit/integrations/insyra"

syi.Table(dt)                           // render a DataTable
syi.Preview(dt, 5)                      // first 5 rows
syi.EditableTable(dt)                   // editable DataTable
col := syi.ColumnSelect("Column", dt)   // column picker dropdown
syi.Metrics(dt, col)                    // count, mean, min, max
syi.BarChart(dt, "Category", "Value")   // chart from DataTable columns
```

## Example

Multi-page app with sidebar, charts, and state:

```go
package main

import sy "github.com/HazelnutParadise/syralit"

func init() {
    sy.AddPage("Dashboard", dashboard, sy.PageIcon("📊"), sy.PageOrder(1))
    sy.AddPage("Settings", settings, sy.PageIcon("⚙️"), sy.PageOrder(2))
}

func main() { sy.App(nil) }

func dashboard() {
    sy.Title("Dashboard")
    
    cols := sy.Columns(3)
    cols[0](func() { sy.Metric("Users", "1,234", sy.Delta("+12%")) })
    cols[1](func() { sy.Metric("Revenue", "$5.6K", sy.Delta("+8%")) })
    cols[2](func() { sy.Metric("Uptime", "99.9%") })
    
    sy.LineChart([]float64{10, 25, 18, 42, 35, 60}, sy.ChartTitle("Growth"))
}

func settings() {
    sy.Title("Settings")
    theme := sy.SelectBox("Theme", []string{"Light", "Dark", "Auto"})
    sy.Info("Selected: " + theme)
}
```

## Requirements

- Go 1.25+

## License

MIT
