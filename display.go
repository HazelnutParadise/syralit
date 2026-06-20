package syralit

import (
	"encoding/base64"
	"encoding/json"
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
