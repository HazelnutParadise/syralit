package syralit

import (
	"encoding/base64"
	"encoding/json"
	"strings"
)

// Table renders a static data table.
func Table(headers []string, rows [][]string) {
	current().add(&Node{Type: "table", Props: map[string]any{
		"headers": headers,
		"rows":    rows,
	}})
}

// Metric renders a big-number metric with optional delta indicator.
// Use Delta("1.2 °F") and DeltaColor("inverse") options.
func Metric(label, value string, opts ...Option) {
	o := applyOpts(opts)
	props := map[string]any{"label": label, "value": value}
	if o.delta != "" {
		props["delta"] = o.delta
	}
	if o.deltaColor != "" {
		props["delta_color"] = o.deltaColor
	}
	current().add(&Node{Type: "metric", Props: props})
}

// Code renders a preformatted code block. Use Language("go") for syntax hints.
func Code(code string, opts ...Option) {
	o := applyOpts(opts)
	props := map[string]any{"code": code}
	if o.language != "" {
		props["language"] = o.language
	}
	current().add(&Node{Type: "code", Props: props})
}

// Image renders an image. src can be a URL or data URI.
// Use Alt("desc"), Width(300), ImageCaption("caption") options.
func Image(src string, opts ...Option) {
	o := applyOpts(opts)
	props := map[string]any{"src": src}
	if o.alt != "" {
		props["alt"] = o.alt
	}
	if o.width > 0 {
		props["width"] = o.width
	}
	if o.caption != "" {
		props["caption"] = o.caption
	}
	current().add(&Node{Type: "image", Props: props})
}

// ImageFromBytes renders an image from raw bytes. The MIME type should be
// specified via MimeType("image/png") or similar; defaults to image/png.
func ImageFromBytes(data []byte, opts ...Option) {
	o := applyOpts(opts)
	mime := o.mime
	if mime == "" {
		mime = "image/png"
	}
	src := "data:" + mime + ";base64," + base64.StdEncoding.EncodeToString(data)
	Image(src, opts...)
}

// JSON renders a formatted JSON viewer for any serializable value.
func JSON(data any) {
	b, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		b = []byte(err.Error())
	}
	current().add(&Node{Type: "json", Props: map[string]any{"data": string(b)}})
}

// Progress renders a progress bar. value is 0.0 to 1.0.
func Progress(value float64) {
	if value < 0 {
		value = 0
	}
	if value > 1 {
		value = 1
	}
	current().add(&Node{Type: "progress", Props: map[string]any{"value": value}})
}

// Link renders a clickable hyperlink that opens in a new tab.
func Link(text, url string) {
	current().add(&Node{Type: "link", Props: map[string]any{"text": text, "url": url}})
}

// DownloadButton renders a button that downloads data as a file when clicked.
// Data is base64-encoded and sent to the client. Use MimeType("text/csv") to
// set the content type.
func DownloadButton(label string, data []byte, filename string, opts ...Option) {
	o := applyOpts(opts)
	mime := o.mime
	if mime == "" {
		mime = "application/octet-stream"
	}
	current().add(&Node{Type: "download_button", Props: map[string]any{
		"label":    label,
		"data":     base64.StdEncoding.EncodeToString(data),
		"filename": filename,
		"mime":     mime,
	}})
}

// DataFrame renders a sortable, interactive data table. Sorting is handled
// client-side. Rows can contain any printable values.
func DataFrame(headers []string, rows [][]any, opts ...Option) {
	o := applyOpts(opts)
	props := map[string]any{"headers": headers, "rows": rows}
	if o.height > 0 {
		props["height"] = o.height
	}
	current().add(&Node{Type: "dataframe", Props: props})
}

// HTML renders raw HTML content directly. Use with care — user-supplied
// content should be sanitized before passing to this function.
func HTML(html string) {
	current().add(&Node{Type: "html", Props: map[string]any{"html": html}})
}

// Dialog renders a modal dialog overlay. Content is always rendered in the
// tree (for consistent widget IDs) but only visible when open. Use
// ShowDialog/CloseDialog to toggle, or let the user click the backdrop/×.
//
// Key is required to identify the dialog for ShowDialog/CloseDialog.
func Dialog(title string, fn func(), opts ...Option) {
	rc := current()
	o := applyOpts(opts)
	key := o.key
	if key == "" {
		key = title
	}
	id := "dialog:" + key
	val, _ := rc.sess.widgetValue(id)
	open, _ := val.(bool)
	dialog := &Node{ID: id, Type: "dialog", Props: map[string]any{
		"title": title,
		"open":  open,
	}}
	rc.add(dialog)
	rc.stack = append(rc.stack, dialog)
	fn()
	rc.stack = rc.stack[:len(rc.stack)-1]
}

// ShowDialog opens a dialog by its key.
func ShowDialog(key string) {
	current().sess.setWidget("dialog:"+key, true)
}

// CloseDialog closes a dialog by its key.
func CloseDialog(key string) {
	current().sess.setWidget("dialog:"+key, false)
}

// ConfigLogo sets the sidebar logo image URL.
func ConfigLogo(src string) PageConfigOption {
	return func(c *pageConfig) { c.logo = src }
}

// LaTeX renders a mathematical formula using KaTeX (loaded from CDN on first
// use). The formula string should use LaTeX syntax without delimiters.
func LaTeX(formula string) {
	current().add(&Node{Type: "latex", Props: map[string]any{"formula": formula}})
}

// MapPoint represents a single marker on a Map widget.
type MapPoint struct {
	Lat  float64
	Lon  float64
	Text string // optional popup text
}

// Map renders an interactive map with markers using Leaflet.js (CDN).
// Points are displayed as markers with optional popup text.
//
//	sy.Map([]sy.MapPoint{
//	    {Lat: 25.033, Lon: 121.565, Text: "Taipei 101"},
//	    {Lat: 25.047, Lon: 121.517, Text: "Taipei Main Station"},
//	}, sy.Height(400))
func Map(points []MapPoint, opts ...Option) {
	o := applyOpts(opts)
	ps := make([]map[string]any, len(points))
	for i, p := range points {
		m := map[string]any{"lat": p.Lat, "lon": p.Lon}
		if p.Text != "" {
			m["text"] = p.Text
		}
		ps[i] = m
	}
	props := map[string]any{"points": ps}
	if o.height > 0 {
		props["height"] = o.height
	}
	current().add(&Node{Type: "map", Props: props})
}

// WriteStream renders text that streams in token by token. The function receives
// a yield callback; call it with each text chunk. Useful for displaying LLM
// responses as they arrive.
//
//	sy.WriteStream(func(yield func(string)) {
//	    for token := range llm.Stream(prompt) {
//	        yield(token)
//	    }
//	})
func WriteStream(fn func(yield func(string)), opts ...Option) {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("write_stream", o.key)

	rc.add(&Node{ID: id, Type: "write_stream", Props: map[string]any{"text": ""}})

	var buf strings.Builder
	fn(func(chunk string) {
		buf.WriteString(chunk)
		if rc.streamer != nil {
			rc.streamer(id, chunk)
		}
	})
	rc.sess.setWidget(id, buf.String())
}

// Audio renders an HTML audio player. src can be a URL or data URI.
func Audio(src string, opts ...Option) {
	current().add(&Node{Type: "audio", Props: map[string]any{"src": src}})
}

// Video renders an HTML video player. src can be a URL or data URI.
// Use Width(640) to constrain the player width.
func Video(src string, opts ...Option) {
	o := applyOpts(opts)
	props := map[string]any{"src": src}
	if o.width > 0 {
		props["width"] = o.width
	}
	current().add(&Node{Type: "video", Props: props})
}

// Echo displays source code alongside its output. Pass the code text and a
// function that produces the output widgets.
func Echo(code string, fn func()) {
	Code(code, Language("go"))
	fn()
}

// DataEditor renders an editable data table. Returns the current rows,
// reflecting any edits made by the user. Each cell edit triggers a rerun.
func DataEditor(headers []string, rows [][]any, opts ...Option) [][]any {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("data_editor", o.key)
	val, hasVal := rc.sess.widgetValue(id)
	if hasVal {
		if edited, ok := val.([]any); ok {
			result := make([][]any, len(edited))
			for i, row := range edited {
				if r, ok := row.([]any); ok {
					result[i] = r
				}
			}
			rows = result
		}
	}
	props := map[string]any{"headers": headers, "rows": rows}
	if o.height > 0 {
		props["height"] = o.height
	}
	if o.disabled {
		props["disabled"] = true
	}
	rc.add(&Node{ID: id, Type: "data_editor", Props: props})
	return rows
}

// Toast shows a brief notification that auto-dismisses. Level is one of
// "info", "success", "warning", "error".
func Toast(text string, level ...string) {
	rc := current()
	lvl := "info"
	if len(level) > 0 && level[0] != "" {
		lvl = level[0]
	}
	rc.sess.mu.Lock()
	rc.sess.pendingToasts = append(rc.sess.pendingToasts, map[string]any{
		"text":  text,
		"level": lvl,
	})
	rc.sess.mu.Unlock()
}

// PageConfigOption configures the page via SetPageConfig.
type PageConfigOption func(*pageConfig)

// PageTitle sets the browser tab title.
func PageTitle(t string) PageConfigOption { return func(c *pageConfig) { c.title = t } }

// ConfigIcon sets the favicon (emoji or URL).
func ConfigIcon(i string) PageConfigOption { return func(c *pageConfig) { c.icon = i } }

// PageLayout sets the content layout. "centered" (default) or "wide".
func PageLayout(l string) PageConfigOption { return func(c *pageConfig) { c.layout = l } }

// PrimaryColor sets the accent/primary color for the theme.
func PrimaryColor(color string) PageConfigOption {
	return func(c *pageConfig) { c.primaryColor = color }
}

// BackgroundColor sets the main background color.
func BackgroundColor(color string) PageConfigOption {
	return func(c *pageConfig) { c.bgColor = color }
}

// TextColor sets the primary text color.
func TextColor(color string) PageConfigOption {
	return func(c *pageConfig) { c.textColor = color }
}

// SetPageConfig configures page-level settings such as the browser tab title
// and content layout. Call this at the top of your page function.
func SetPageConfig(opts ...PageConfigOption) {
	rc := current()
	rc.sess.mu.Lock()
	if rc.sess.pageConfig == nil {
		rc.sess.pageConfig = &pageConfig{}
	}
	for _, opt := range opts {
		opt(rc.sess.pageConfig)
	}
	rc.sess.mu.Unlock()
}

// Balloons renders a celebratory balloon animation.
func Balloons() {
	rc := current()
	rc.sess.mu.Lock()
	rc.sess.pendingToasts = append(rc.sess.pendingToasts, map[string]any{
		"type": "balloons",
	})
	rc.sess.mu.Unlock()
}

// Snow renders a falling snow animation.
func Snow() {
	rc := current()
	rc.sess.mu.Lock()
	rc.sess.pendingToasts = append(rc.sess.pendingToasts, map[string]any{
		"type": "snow",
	})
	rc.sess.mu.Unlock()
}
