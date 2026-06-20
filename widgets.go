package syralit

import "fmt"

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
	helpText    string
	defaultVal  any
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

func Markdown(text string) { current().add(textNode("markdown", text)) }

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
	if o.disabled {
		props["disabled"] = true
	}
	if o.maxChars > 0 {
		props["max_chars"] = o.maxChars
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
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
