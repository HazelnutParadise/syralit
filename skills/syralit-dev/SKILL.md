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
| `MultiSelect(label, options, opts...)` | `[]string` | Multi-select; `sy.AcceptNewOptions()` allows typing new values |
| `DateInput(label, opts...)` | `string` | Date picker (YYYY-MM-DD) |
| `DatetimeInput(label, opts...)` | `string` | Date+time picker (YYYY-MM-DD HH:MM) |
| `DateRangeInput(label, opts...)` | `(string, string)` | Start/end date pickers |
| `TimeInput(label, opts...)` | `string` | Time picker (HH:MM) |
| `ColorPicker(label, opts...)` | `string` | Color hex |
| `FileUploader(label, opts...)` | `*UploadedFile` | File upload (nil if empty) |
| `FileUploaderMultiple(label, opts...)` | `[]*UploadedFile` | Multi-file upload (empty slice if none) |
| `CameraInput(label, opts...)` | `string` | Webcam capture (base64 data URI) |
| `AudioInput(label, opts...)` | `string` | Microphone recording (base64) |
| `ChatInput(placeholder, opts...)` | `string` | Chat input box |
| `Feedback(opts...)` | `string` | Thumbs up/down; `sy.FeedbackStyle("stars"/"faces")` for other styles |
| `SegmentedControl(label, options, opts...)` | `string` | Segmented buttons (`SegmentedControlMulti` → `[]string`) |
| `Pills(label, options, opts...)` | `string` | Pill buttons (`PillsMulti` → `[]string`) |
| `Pagination(totalPages, opts...)` | `int` | Page selector (1-based) |
| `MenuButton(label, options, opts...)` | `string` | Dropdown button; returns clicked option for one rerun |
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
sy.StartTime(2), sy.EndTime(5) // Audio/Video playback range (seconds)
sy.Subtitles("/subs.vtt")     // Video subtitle track (WebVTT)
sy.AcceptNewOptions()         // MultiSelect: allow typing new values
sy.Mono()                     // TextInput/TextArea in the code font (formulas, IDs)
sy.Formula()                  // formula-bar look: fx marker in the box, code surface (implies Mono)
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
sy.PDF(src, sy.Height(600))        // embedded PDF viewer (browser renderer)
```

### Data
```go
// Static table
sy.Table(headers []string, rows [][]string)

// Sortable data frame. Add sy.Selectable() to get row selection (returns
// selected indices into the original rows); sy.ColConfig renders cells by type.
sy.DataFrame(headers []string, rows [][]any, opts...)
selected := sy.DataFrame(headers, rows, sy.Selectable(), sy.Key("df")) // []int
// sy.SelectionMode("single-row") limits selection to one row;
// sy.ColumnOrder("B", "A") reorders/filters displayed columns.

// Editable data editor — returns current rows
edited := sy.DataEditor(headers, rows, opts...)

// Column configuration for DataEditor and DataFrame. Fields: Type, Options,
// Width, Min, Max, Step, Format ("$%.2f"/"%d%%"), Label, Help, Color.
// Types: text/number/checkbox/select/date/time/datetime/link/image/progress/
// list, json, plus display-only bar_chart / line_chart / area_chart
// (cell value = []float64).
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
// Line/Bar/Area/Scatter/Pie charts accept sy.Selectable() and then return a
// *sy.ChartSelection (nil until the user clicks a point):
if sel := sy.BarChart(data, sy.Selectable(), sy.Key("sales")); sel != nil {
    sy.Textf("%s at %s = %v", sel.Series, sel.X, sel.Value) // Series/Index/X/Value
}
// sy.RangeSelectable() on Line/Bar/Area also lets the user DRAG across the
// chart to select an x-range: sel.Range is true, Index..EndIndex / X..EndX
// span the interval. Point clicks still work.
if sel := sy.LineChart(data, sy.RangeSelectable(), sy.Key("ts")); sel != nil && sel.Range {
    sy.Textf("%s to %s", sel.X, sel.EndX)
}
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
sy.Space()                     // vertical whitespace (sy.Height(32) to size)
sy.Bottom(func() {             // pin content to the bottom of the viewport
    msg := sy.ChatInput("Message...")
    _ = msg
})
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
sy.Toast("Message", "success")  // level: "success"/"info"/"warning"/"error"; optional 3rd/4th args: icon, duration ("8s")
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
sy.SetQueryParam("page", "2")  // updates the browser URL (deep linking); "" removes
sy.ResetWidget("key")          // delete a widget's stored value (del st.session_state[key])
port := sy.GetOption("server.port") // read resolved config values

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

notes := sy.NewArtifactStore("notes", notesSpec)
auth := sy.StaticAgentKey("local-agent", sy.Secrets("AGENT_KEY"))

// Default: one authenticated discovery/update API for selected stores.
sy.HandleArtifactAPI("/api/agent/artifacts", auth, board, notes)

sy.App(func() {
    sy.ArtifactCanvas(board, sy.Height(520))
    sy.ArtifactCanvas(notes, sy.Height(240))
})

// Unified full-replace update payload:
// {"artifact":"main","expected_revision":1,"spec":{"version":"v1","nodes":[{"id":"msg","component":"text","props":{"text":"Updated"}}]}}
```
Allowed artifact components: `text`, `markdown`, `metric`, `table`,
`dataframe`, `line_chart`, `bar_chart`, `pie_chart`, `image`, `progress`,
`container`, and — when the app imports `integrations/insyra/insyradsl` — the
optional `insyra` component for live Insyra DSL computation (see below). Data
binding uses JSON Pointer from `ArtifactSpec.Data`, e.g.
`Bind: map[string]string{"props.value": "/summary/revenue"}`. Every artifact
node needs a stable `ID` so the browser can animate enter/update/exit states.

Artifact API choices:

```go
// One route per store, useful for separate authenticators or permissions.
sy.HandleArtifactEndpoint("/api/private/report", board, auth)

// Mount on another http.Server, mux, or port.
apiHandler := sy.ArtifactAPIHandler(auth, board, notes)
singleHandler := sy.ArtifactHandler(board, auth)
```

The unified API uses one URL:

- `GET /api/agent/artifacts` discovers exposed stores, a `components` object
  (`builtin` + `custom`, plus `capabilities` — for `insyra` this includes the
  live safe DSL command catalog pulled from the linked Insyra version), and
  observed page
  placements.
- `GET /api/agent/artifacts?artifact=main` returns the current spec.
- `POST /api/agent/artifacts` accepts
  `{"artifact":"main","expected_revision":3,"spec":{...}}`.

Use the public Syralit App URL for these routes. Under `syralit dev`, `/api/`
is proxied to the hot-reload child; never expose or call the ephemeral child
port printed in internal diagnostics.

Successful updates return a monotonically increasing `revision`, `placements`,
and a `preview` with the page URL and a server-generated selector. A
browser-capable agent must wait until that element has both the returned
`data-artifact-revision` and `data-artifact-state="settled"` before taking and
returning a screenshot. Do not use the selector as the update target; updates
select a store by artifact ID.

`expected_revision` is required. A stale update receives `409 Conflict`; fetch
the current spec again, reconcile the intended change, and submit against the
new revision instead of blindly retrying. `data-artifact-readiness` is
`complete`, `partial`, or `timeout`; only `complete` proves all visual resources
loaded.

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
    sy.InitialSidebarState("collapsed"),  // sidebar starts hidden; floating button reopens
    sy.ConfigMenuItems(                   // top-right app menu (pass "" to omit an item)
        "https://example.com/help",       // "Get help" link
        "https://example.com/bugs",       // "Report a bug" link
        "**My App** v1.0",                // About dialog (markdown)
    ),
)

// Secrets (from syralit.toml [secrets] section)
apiKey := sy.Secrets("api_key")

// Read resolved config values ("title", "server.port", "theme.accent", ...)
port := sy.GetOption("server.port")
_ = port

// Embed the app in an existing Go HTTP server (instead of sy.App)
// mux.Handle("/dash/", http.StripPrefix("/dash", sy.Handler(sy.Config{}, myApp)))

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

// OIDC single sign-on (Google / Entra / Keycloak / Auth0 ...) — separate
// module so its dependency tree stays out of the core:
//   import syoidc "github.com/HazelnutParadise/syralit/integrations/oidc"
// handler, err := syoidc.Protect(sy.Handler(sy.Config{}, app), syoidc.Config{
//     Issuer: "https://accounts.google.com", ClientID: "...",
//     ClientSecret: "...", RedirectURL: "http://host/auth/callback",
//     CookieSecret: []byte("32+ random bytes"),
// })
// Visitors are redirected through the provider; sy.User() returns the verified
// claims (sub/email/name/picture). Sign-out link: /auth/logout.
// sy.SetUserResolver(fn) is the underlying core hook (request -> user).
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

### Desktop App (native window)
```go
// Ship the same app as a native desktop window (Wails v3) — separate module
// so the Wails dependency tree stays out of the core:
import sydesktop "github.com/HazelnutParadise/syralit/integrations/desktop"

func main() {
    sydesktop.App(func() {          // desktop counterpart of sy.App (fatal on error)
        sy.Title("My tool")
        // ... any Syralit app; nil + sy.AddPage for multi-page
    },
        sydesktop.WindowSize(1200, 800),   // initial size (default 1024×768)
        sydesktop.MinSize(640, 480),       // minimum size
        sydesktop.WindowTitle("My tool"),  // default: resolved app title
        sydesktop.Config(sy.Config{}),     // theme etc.; Host ignored (always loopback),
                                           //   explicit Port pins the port for /api/ agents
        sydesktop.Frameless(),             // remove the native frame
        sydesktop.Icon(pngBytes),          // app icon (PNG bytes)
        sydesktop.AllowBrowser(),          // opt out of browser lockdown (see below)
    )
}
// sydesktop.Run(fn, opts...) error  — same, error returned instead of fatal.
```
The app serves on a loopback-only random port and the window points at it;
closing the window shuts the server down. **Browser lockdown (default)**: the
server only answers its own window (per-launch token; other local browsers get
403) — pass sydesktop.AllowBrowser() to open it up. `/api/` endpoints are
exempt so agent artifact endpoints stay reachable with their own bearer auth;
SYRALIT_URL is set in the app's environment so spawned agent subprocesses can
find them, and Config(sy.Config{Port: N}) pins the port for external agents. Call from the main goroutine (macOS
needs the event loop on the main thread). The Go code runs on the user's
machine, so local paths (os.ReadFile etc.) work directly — no FileUploader
round-trip. Build needs: nothing extra on Windows (WebView2 is preinstalled on
10/11), Xcode CLT on macOS, webkit2gtk on Linux. Example: examples/desktop-demo.

`syralit dev` hot reload works for desktop apps: the window connects to the
supervisor and survives rebuilds exactly like a browser tab (state preserved,
build-error overlay). One window opens per dev session; if the user closes it,
the session keeps running and stays reachable in a browser at the printed URL.
The window auto-quits a few seconds after the supervisor stops.

### File Config (syralit.toml)
```toml
title = "My App"
host = "0.0.0.0"
port = 8600

[theme]
mode = "system"        # "light" | "dark" | "system"
accent = "#7C3AED"     # primary/accent color
radius = "12px"        # base corner radius
button_radius = "999px"             # button corners; defaults to radius
background_color = "#ffffff"        # app background
secondary_background_color = "#f8f9fb"  # widget/code/sidebar surface
text_color = "#1f2329"
link_color = "#2563eb"              # defaults to accent
link_underline = true               # true: always, false: never, unset: on hover
code_text_color = "#0f766e"
code_background_color = "#f1f5f9"
border_color = "#e5e7eb"
dataframe_border_color = "#e5e7eb"      # defaults to border_color
dataframe_header_background_color = "#f3f4f6"
show_widget_border = true           # false hides input borders
show_sidebar_border = true          # false removes the sidebar divider
# Basic palette (used by badges, alerts, status colors). Each color also has
# <name>_background_color (alert surface tint) and <name>_text_color:
red_color = "#dc2626"               # also orange/yellow/blue/green/violet/gray
blue_background_color = "#eff6ff"
green_text_color = "#166534"
# Chart palettes: categorical drives built-in chart series colors;
# sequential/diverging are published to window.__SY_THEME for custom components.
chart_categorical_colors = ["#7c3aed", "#2563eb", "#16a34a"]
chart_sequential_colors = ["#f0fdfa", "#0f766e"]
chart_diverging_colors = ["#dc2626", "#f8fafc", "#2563eb"]
# Fonts: "sans-serif" | "serif" | "monospace" pick the built-in Source Sans 3 /
# Source Serif 4 / Source Code Pro (embedded, no CDN); any other value is used
# as a CSS font-family list.
font = "sans-serif"
heading_font = "serif"              # defaults to font
code_font = "monospace"
base_font_size = 16                 # root font size in px
base_font_weight = 400
heading_font_sizes = ["2rem", "1.5rem", "1.15rem"]  # h1..h6 (also Title/Header/Subheader)
heading_font_weights = [700, 650, 600]
code_font_size = "0.875rem"
code_font_weight = 400

[[theme.font_faces]]   # load custom fonts (otf/ttf/woff/woff2) — repeatable
family = "Inter"
url = "/fonts/inter.woff2"          # public/ path or absolute URL
weight = "100 900"                  # optional
style = "normal"                    # optional: normal | italic | oblique
unicode_range = "U+0-10FFFF"        # optional

[theme.sidebar]        # sidebar-only overrides — supports every color/font/radius
font = "sans-serif"    # key above (inherits the main theme when unset)
accent = "#f59e0b"

[secrets]
api_key = "sk-..."
db_dsn = "postgres://..."

[server]
max_upload_size_mb = 50   # FileUploader/CameraInput cap (default 10 MB)
ssl_cert_file = "cert.pem"  # serve HTTPS when both are set
ssl_key_file = "key.pem"

[i18n]                    # localize built-in UI text (all keys optional)
connecting = "連線中…"     # keys: connecting, loading, add_new, file_too_large,
loading = "載入中…"        #       menu, menu_get_help, menu_report_bug, menu_about
```

### Headless App Testing (sy.AppTest)
```go
at := sy.NewAppTest(func() {
    name := sy.TextInput("Name", sy.Key("name"))
    if sy.Button("Greet", sy.Key("greet")) {
        sy.Success("Hello, " + name)
    }
})
at.Run()
at.SetValue("name", "Ada")   // set widget value by sy.Key
at.Click("greet")            // or at.ClickLabel("Greet")
at.Run()
at.Texts("status")           // -> ["Hello, Ada"]
at.FindAll("title")          // nodes by type; at.FindByLabel(type, label)
at.SwitchToPage("Settings")  // multi-page apps
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
syi.BarChart(dt, "x_col", "y_col")     // chart from columns (x_col = axis labels);
                                        // also LineChart / AreaChart / ScatterChart / PieChart.
                                        // All accept sy.Option and return *sy.ChartSelection
                                        // when sy.Selectable() is set.
syi.MultiLineChart(dt, "x", nil)       // all numeric columns as series (st.line_chart(df));
                                        // also MultiBarChart / MultiAreaChart; pass []string to pick columns
col := syi.ColumnSelect("Pick", dt)    // column picker

// Click-to-filter dashboards: GroupBy chart + selection + filter
sel := syi.GroupedBarChart(dt, "region", "revenue", insyra.OpSum,
    sy.Selectable(), sy.Key("by_region"))     // one bar per group (also GroupedPieChart)
syi.Table(syi.FilterBySelection(dt, "region", sel)) // nil sel = unfiltered
sub := syi.FilterEquals(dt, "region", "north")      // pure data helper
edited := syi.EditableDataTable(dt, sy.Key("e"))    // DataEditor → new *insyra.DataTable
syi.DownloadCSV("Export", dt, "data.csv")           // download button for a DataTable
syi.RollingMeanChart(dt, "Month", "Revenue", 7)     // raw + rolling mean overlay
syi.CumSumChart(dt, "Month", "Revenue")             // cumulative sum line
syi.PctChangeChart(dt, "Month", "Revenue", 1)       // percent change bars
out := syi.AddFormulaColumn(dt, "ccl")              // interactive CCL formula column;
                                                    // "ccl" = widget-key prefix (inputs stored as
                                                    // ccl_formula / ccl_name — unique per page).
                                                    // Columns as A/B or ["Name"] (case-sensitive);
                                                    // shows a live letter=name legend; timeout-guarded.
out, err := syi.ComputeColumn(dt, "Profit", `+'`'+`["Revenue"] - ["Cost"]`+'`'+`)
                                                    // the guarded primitive for building a custom
                                                    // formula UI (own labels/layout/i18n)
// sy.ResetWidget("by_region") clears a stored selection (del st.session_state equivalent)

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

### Insyra DSL — dynamic computation (opt-in subpackage)
Run the Insyra CLI DSL (`.isr`) from Go and render the result. Separate package
because it pulls in the full Insyra CLI dependency tree (cobra, DB drivers,
parquet/arrow, readline):
```go
import syidsl "github.com/HazelnutParadise/syralit/integrations/insyra/insyradsl"

// Go widget: run a script (safe mode) and auto-render. Cached by script hash.
syidsl.DSL(`
newdl Q1 Q2 Q3 Q4 as quarter
newdl 42 55 61 78 as revenue
newdt quarter revenue as t
setcolnames t quarter revenue
`, syidsl.Render("bar_chart"), syidsl.Output("t"), syidsl.X("quarter"), syidsl.Y("revenue"))

syidsl.DSL("newdl 42 55 61 78 as revenue\nsummary revenue") // no opts → prints transcript

// Low-level: run and inspect the produced variables yourself.
res := syidsl.RunDSL(script, syidsl.WithVars(map[string]any{"t": dt}))
dt2 := res.Vars["report"].(*insyra.DataTable)  // res.Err, res.Output also available
```
Widget options: `Render`, `Output`, `X`, `Y`, `Label`, `Value`, `MetricLabel`,
`Title`, `Height`, `Input`. `RunDSL` options: `WithVars`, `DSLTimeout`,
`MaxLines`, `Unrestricted`, `EnvRoot`. Each run uses a throwaway temp
environment by default; `EnvRoot(path)` runs in a persistent
`path/envs/default/` that survives across calls (variables restored from
`state.json`) and is not deleted.

`RunDSL` runs in **safe mode** by default: only pure, in-memory compute commands
are allowed; `load`/`save`/`db`/`fetch`/`run`/`env`/`plot` are rejected. Pass
`syidsl.Unrestricted()` only for trusted, app-authored scripts. Each run uses an
isolated, ephemeral environment (no shared `~/.insyra` state).

Importing this package also registers the **`insyra` Artifact component**, so
agents can embed a DSL script in an `ArtifactSpec` for live computation:
```go
{ID: "chart", Component: "insyra", Props: map[string]any{
    "script": "newdl North South West as region\nnewdl 12 18 9 as deals\n" +
        "newdt region deals as t\nsetcolnames t region deals",
    "render": "bar_chart", "output": "t", "x": "region", "y": "deals"}}
```
The artifact path is always safe mode. Columns from `newdt` are positional —
reference by Excel letter (`A`, `B`) or name them with `setcolnames`. See
`skills/syralit-artifact-dsl/SKILL.md` for the full `insyra` component reference.

The discovery endpoint advertises the live safe DSL vocabulary under
`components.capabilities.insyra.commands` (pulled from Insyra's registry via
`syidsl.SafeCommandCatalog()`), so bumping the Insyra version refreshes it
without editing docs.

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
