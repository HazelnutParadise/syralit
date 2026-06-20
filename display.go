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
