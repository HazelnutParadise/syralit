package syralit

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
)

// Option configures a widget. See Key, Min, Max, Step, Placeholder, etc.
type Option func(*widgetOpts)

type widgetOpts struct {
	key         string
	min         *float64
	max         *float64
	step        *float64
	height      int
	width       int
	maxChars    int
	placeholder string
	disabled    bool
	delta       string
	deltaColor  string
	language    string
	alt         string
	caption     string
	mime        string
	helpText        string
	defaultVal      any
	border          bool
	maxSelections   int
	gap             int
	labelVisibility string // "visible" (default), "hidden", "collapsed"
}

func Key(k string) Option          { return func(o *widgetOpts) { o.key = k } }
func Min(v float64) Option         { return func(o *widgetOpts) { o.min = &v } }
func Max(v float64) Option         { return func(o *widgetOpts) { o.max = &v } }
func Step(v float64) Option        { return func(o *widgetOpts) { o.step = &v } }
func Height(v int) Option          { return func(o *widgetOpts) { o.height = v } }
func Width(v int) Option           { return func(o *widgetOpts) { o.width = v } }
func MaxChars(v int) Option        { return func(o *widgetOpts) { o.maxChars = v } }
func Placeholder(v string) Option  { return func(o *widgetOpts) { o.placeholder = v } }
func Disabled() Option             { return func(o *widgetOpts) { o.disabled = true } }
func Delta(v string) Option        { return func(o *widgetOpts) { o.delta = v } }
func DeltaColor(v string) Option   { return func(o *widgetOpts) { o.deltaColor = v } }
func Language(v string) Option     { return func(o *widgetOpts) { o.language = v } }
func Alt(v string) Option          { return func(o *widgetOpts) { o.alt = v } }
func ImageCaption(v string) Option { return func(o *widgetOpts) { o.caption = v } }
func MimeType(v string) Option     { return func(o *widgetOpts) { o.mime = v } }
func Help(v string) Option         { return func(o *widgetOpts) { o.helpText = v } }
func DefaultValue(v any) Option    { return func(o *widgetOpts) { o.defaultVal = v } }
func Border() Option               { return func(o *widgetOpts) { o.border = true } }
func MaxSelections(n int) Option   { return func(o *widgetOpts) { o.maxSelections = n } }
func Gap(px int) Option            { return func(o *widgetOpts) { o.gap = px } }
func LabelHidden() Option          { return func(o *widgetOpts) { o.labelVisibility = "hidden" } }
func LabelCollapsed() Option       { return func(o *widgetOpts) { o.labelVisibility = "collapsed" } }

func applyCommonProps(props map[string]any, o widgetOpts) {
	if o.disabled {
		props["disabled"] = true
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	if o.labelVisibility != "" {
		props["label_visibility"] = o.labelVisibility
	}
}

func applyOpts(opts []Option) widgetOpts {
	var o widgetOpts
	for _, fn := range opts {
		fn(&o)
	}
	return o
}

// --- Text (SPEC §6.2) ---

func Title(text string)     { current().add(textNode("title", text)) }
func Header(text string)    { current().add(textNode("header", text)) }
func Subheader(text string) { current().add(textNode("subheader", text)) }
func Text(text string)      { current().add(textNode("text", text)) }
func Caption(text string)   { current().add(textNode("caption", text)) }

func Markdown(text string) {
	var buf bytes.Buffer
	if err := goldmark.Convert([]byte(text), &buf); err != nil {
		current().add(textNode("markdown", text))
		return
	}
	current().add(&Node{Type: "markdown", Props: map[string]any{"html": buf.String()}})
}

func Textf(format string, a ...any) { Text(fmt.Sprintf(format, a...)) }

func textNode(typ, text string) *Node {
	return &Node{Type: typ, Props: map[string]any{"text": text}}
}

// --- Status (SPEC §6.3) ---

func Success(text string) { current().add(statusNode("success", text)) }
func Info(text string)    { current().add(statusNode("info", text)) }
func Warning(text string) { current().add(statusNode("warning", text)) }
func Error(text string)   { current().add(statusNode("error", text)) }

func statusNode(level, text string) *Node {
	return &Node{Type: "status", Props: map[string]any{"level": level, "text": text}}
}

// --- Inputs (SPEC §6.4) ---

func TextInput(label string, opts ...Option) string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("text_input", o.key)
	val, _ := rc.sess.widgetValue(id)
	s, _ := val.(string)
	props := map[string]any{"label": label, "value": s}
	if o.placeholder != "" {
		props["placeholder"] = o.placeholder
	}
	if o.maxChars > 0 {
		props["max_chars"] = o.maxChars
	}
	applyCommonProps(props, o)
	rc.add(&Node{ID: id, Type: "text_input", Props: props})
	return s
}

func Checkbox(label string, opts ...Option) bool {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("checkbox", o.key)
	val, _ := rc.sess.widgetValue(id)
	b, _ := val.(bool)
	props := map[string]any{"label": label, "value": b}
	if o.disabled {
		props["disabled"] = true
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	rc.add(&Node{ID: id, Type: "checkbox", Props: props})
	return b
}

func SelectBox(label string, options []string, opts ...Option) string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("select", o.key)
	val, _ := rc.sess.widgetValue(id)
	sel, _ := val.(string)
	if sel == "" && len(options) > 0 {
		sel = options[0]
	}
	props := map[string]any{"label": label, "options": options, "value": sel}
	if o.disabled {
		props["disabled"] = true
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	rc.add(&Node{ID: id, Type: "select", Props: props})
	return sel
}

func Button(label string, opts ...Option) bool {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("button", o.key)
	pressed := rc.sess.buttonPressed(id)
	props := map[string]any{"label": label}
	if o.disabled {
		props["disabled"] = true
	}
	rc.add(&Node{ID: id, Type: "button", Props: props})
	return pressed
}

func NumberInput(label string, opts ...Option) float64 {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("number_input", o.key)
	val, exists := rc.sess.widgetValue(id)
	var f float64
	if exists {
		f = toFloat64(val)
	} else if o.defaultVal != nil {
		f = toFloat64(o.defaultVal)
	}
	props := map[string]any{"label": label, "value": f}
	if o.min != nil {
		props["min"] = *o.min
	}
	if o.max != nil {
		props["max"] = *o.max
	}
	if o.step != nil {
		props["step"] = *o.step
	}
	if o.placeholder != "" {
		props["placeholder"] = o.placeholder
	}
	if o.disabled {
		props["disabled"] = true
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	rc.add(&Node{ID: id, Type: "number_input", Props: props})
	return f
}

func Slider(label string, min, max float64, opts ...Option) float64 {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("slider", o.key)
	val, exists := rc.sess.widgetValue(id)
	var f float64
	if exists {
		f = toFloat64(val)
	} else if o.defaultVal != nil {
		f = toFloat64(o.defaultVal)
	} else {
		f = min
	}
	step := 1.0
	if o.step != nil {
		step = *o.step
	}
	props := map[string]any{"label": label, "value": f, "min": min, "max": max, "step": step}
	if o.disabled {
		props["disabled"] = true
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	rc.add(&Node{ID: id, Type: "slider", Props: props})
	return f
}

func TextArea(label string, opts ...Option) string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("textarea", o.key)
	val, _ := rc.sess.widgetValue(id)
	s, _ := val.(string)
	props := map[string]any{"label": label, "value": s}
	if o.height > 0 {
		props["height"] = o.height
	}
	if o.maxChars > 0 {
		props["max_chars"] = o.maxChars
	}
	if o.placeholder != "" {
		props["placeholder"] = o.placeholder
	}
	if o.disabled {
		props["disabled"] = true
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	rc.add(&Node{ID: id, Type: "textarea", Props: props})
	return s
}

func Radio(label string, options []string, opts ...Option) string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("radio", o.key)
	val, _ := rc.sess.widgetValue(id)
	sel, _ := val.(string)
	if sel == "" && len(options) > 0 {
		sel = options[0]
	}
	props := map[string]any{"label": label, "options": options, "value": sel}
	if o.disabled {
		props["disabled"] = true
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	rc.add(&Node{ID: id, Type: "radio", Props: props})
	return sel
}

func MultiSelect(label string, options []string, opts ...Option) []string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("multi_select", o.key)
	val, _ := rc.sess.widgetValue(id)
	var selected []string
	switch v := val.(type) {
	case []string:
		selected = v
	case []any:
		for _, item := range v {
			if s, ok := item.(string); ok {
				selected = append(selected, s)
			}
		}
	}
	if selected == nil {
		selected = []string{}
	}
	props := map[string]any{"label": label, "options": options, "value": selected}
	if o.disabled {
		props["disabled"] = true
	}
	if o.maxSelections > 0 {
		props["max_selections"] = o.maxSelections
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	rc.add(&Node{ID: id, Type: "multi_select", Props: props})
	return selected
}

func DateInput(label string, opts ...Option) string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("date_input", o.key)
	val, _ := rc.sess.widgetValue(id)
	s, _ := val.(string)
	props := map[string]any{"label": label, "value": s}
	if o.disabled {
		props["disabled"] = true
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	rc.add(&Node{ID: id, Type: "date_input", Props: props})
	return s
}

func TimeInput(label string, opts ...Option) string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("time_input", o.key)
	val, _ := rc.sess.widgetValue(id)
	s, _ := val.(string)
	props := map[string]any{"label": label, "value": s}
	if o.disabled {
		props["disabled"] = true
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	rc.add(&Node{ID: id, Type: "time_input", Props: props})
	return s
}

func ColorPicker(label string, opts ...Option) string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("color_picker", o.key)
	val, _ := rc.sess.widgetValue(id)
	s, _ := val.(string)
	if s == "" {
		s = "#000000"
	}
	props := map[string]any{"label": label, "value": s}
	if o.disabled {
		props["disabled"] = true
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	rc.add(&Node{ID: id, Type: "color_picker", Props: props})
	return s
}

func Toggle(label string, opts ...Option) bool {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("toggle", o.key)
	val, _ := rc.sess.widgetValue(id)
	b, _ := val.(bool)
	props := map[string]any{"label": label, "value": b}
	if o.disabled {
		props["disabled"] = true
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	rc.add(&Node{ID: id, Type: "toggle", Props: props})
	return b
}

// UploadedFile holds the data from a FileUploader widget.
type UploadedFile struct {
	Name string
	Size int64
	Type string
	Data []byte
}

// FileUploader renders a file upload widget and returns the uploaded file (or
// nil if nothing has been uploaded). The file data is sent as base64 over the
// WebSocket, so this is suited for files up to a few MB.
func FileUploader(label string, opts ...Option) *UploadedFile {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("file_uploader", o.key)
	val, _ := rc.sess.widgetValue(id)

	var file *UploadedFile
	if m, ok := val.(map[string]any); ok {
		name, _ := m["name"].(string)
		size := toFloat64(m["size"])
		typ, _ := m["type"].(string)
		dataStr, _ := m["data"].(string)
		data, err := base64.StdEncoding.DecodeString(dataStr)
		if err == nil && name != "" {
			file = &UploadedFile{Name: name, Size: int64(size), Type: typ, Data: data}
		}
	}

	props := map[string]any{"label": label}
	if file != nil {
		props["file_name"] = file.Name
		props["file_size"] = file.Size
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	rc.add(&Node{ID: id, Type: "file_uploader", Props: props})
	return file
}

// LinkButton renders a button-styled hyperlink that opens in a new tab.
func LinkButton(label, url string, opts ...Option) {
	o := applyOpts(opts)
	props := map[string]any{"label": label, "url": url}
	if o.disabled {
		props["disabled"] = true
	}
	current().add(&Node{Type: "link_button", Props: props})
}

// SelectSlider renders a slider that snaps to labelled options, returning
// the selected label string. options must have at least 2 elements.
func SelectSlider(label string, options []string, opts ...Option) string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("select_slider", o.key)
	val, _ := rc.sess.widgetValue(id)
	s, _ := val.(string)
	if s == "" {
		if dv, ok := o.defaultVal.(string); ok {
			s = dv
		} else if len(options) > 0 {
			s = options[0]
		}
	}
	props := map[string]any{"label": label, "options": options, "value": s}
	if o.disabled {
		props["disabled"] = true
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	rc.add(&Node{ID: id, Type: "select_slider", Props: props})
	return s
}

// Write renders any value: strings are treated as Markdown, errors as Error
// blocks, and everything else is formatted as JSON.
func Write(args ...any) {
	for _, arg := range args {
		switch v := arg.(type) {
		case string:
			Markdown(v)
		case error:
			Error(v.Error())
		case nil:
			Text("None")
		case fmt.Stringer:
			Text(v.String())
		default:
			JSON(v)
		}
	}
}

// ChatMessage renders a chat bubble with avatar and role styling.
// Use inside a loop over your message history:
//
//	sy.ChatMessage("user", func() { sy.Text(msg.Content) })
//	sy.ChatMessage("assistant", func() { sy.Markdown(msg.Content) })
func ChatMessage(role string, fn func()) {
	rc := current()
	msg := &Node{Type: "chat_message", Props: map[string]any{"role": role}}
	rc.add(msg)
	rc.stack = append(rc.stack, msg)
	fn()
	rc.stack = rc.stack[:len(rc.stack)-1]
}

// ChatInput renders a chat text input pinned to the bottom of the content area.
// Returns the submitted text (non-empty only on the rerun triggered by Enter).
func ChatInput(placeholder string, opts ...Option) string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("chat_input", o.key)
	val, _ := rc.sess.widgetValue(id)
	s, _ := val.(string)
	props := map[string]any{"placeholder": placeholder}
	if o.disabled {
		props["disabled"] = true
	}
	rc.add(&Node{ID: id, Type: "chat_input", Props: props})
	return strings.TrimSpace(s)
}

// Spinner renders a loading indicator with optional text.
func Spinner(text ...string) {
	label := "Loading..."
	if len(text) > 0 && text[0] != "" {
		label = text[0]
	}
	current().add(&Node{Type: "spinner", Props: map[string]any{"text": label}})
}

// Popover renders a button that shows a floating panel with the content
// produced by fn.
func Popover(label string, fn func(), opts ...Option) {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("popover", o.key)
	val, _ := rc.sess.widgetValue(id)
	open, _ := val.(bool)
	pop := &Node{ID: id, Type: "popover", Props: map[string]any{
		"label": label,
		"open":  open,
	}}
	rc.add(pop)
	rc.stack = append(rc.stack, pop)
	fn()
	rc.stack = rc.stack[:len(rc.stack)-1]
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case float64:
		return n
	case float32:
		return float64(n)
	case int:
		return float64(n)
	case int64:
		return float64(n)
	case int32:
		return float64(n)
	case int16:
		return float64(n)
	case int8:
		return float64(n)
	case uint:
		return float64(n)
	case uint64:
		return float64(n)
	case uint32:
		return float64(n)
	}
	return 0
}
