package syralit

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

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
	title           string
	xLabels         []string
	colConfig       map[string]ColumnConfig
	color           string
	dynamicRows     bool
	verticalAlign   string
	icon              string
	buttonType        string // "primary" (default), "secondary", "tertiary"
	useContainerWidth bool
	minDate           string
	maxDate           string
	selectable        bool
	feedbackStyle     string
	horizontal        bool
	stacked           bool
	colors            []string
	autoplay          bool
	loop              bool
	muted             bool
	lineNumbers       bool
	wrap              bool
	avatar            string
	zoom              int
	runEvery          int // fragment auto-refresh interval in ms
	clearOnSubmit     bool
	startTime         float64
	endTime           float64
	subtitles         string
	acceptNew         bool
	multipleFiles     bool
	columnOrder       []string
	selectionMode     string
	showTime          bool
	rangeSelectable   bool
	mono              bool
	formula           bool
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
func ChartTitle(t string) Option   { return func(o *widgetOpts) { o.title = t } }
func Color(c string) Option        { return func(o *widgetOpts) { o.color = c } }
func XLabels(l []string) Option    { return func(o *widgetOpts) { o.xLabels = l } }
func Expanded() Option             { return DefaultValue(true) }
func DynamicRows() Option          { return func(o *widgetOpts) { o.dynamicRows = true } }
func VerticalAlignment(align string) Option { return func(o *widgetOpts) { o.verticalAlign = align } }

// Icon prefixes a button's label with an icon (emoji or short string).
func Icon(v string) Option { return func(o *widgetOpts) { o.icon = v } }

// ButtonType selects a button's visual style: "primary" (default, accent
// fill), "secondary" (outlined), or "tertiary" (text only).
func ButtonType(v string) Option { return func(o *widgetOpts) { o.buttonType = v } }

// UseContainerWidth makes a button span the full width of its container.
func UseContainerWidth() Option { return func(o *widgetOpts) { o.useContainerWidth = true } }

// MinDate / MaxDate bound a DateInput or DateRangeInput to a "YYYY-MM-DD" range.
func MinDate(d string) Option { return func(o *widgetOpts) { o.minDate = d } }
func MaxDate(d string) Option { return func(o *widgetOpts) { o.maxDate = d } }

// Selectable enables row selection on a DataFrame; the call then returns the
// selected row indices.
func Selectable() Option { return func(o *widgetOpts) { o.selectable = true } }

// RangeSelectable lets the user drag across a Line/Bar/Area chart to select an
// x-axis range; the chart then returns a *ChartSelection with Range true and
// Index..EndIndex spanning the dragged interval. Point clicks still work.
func RangeSelectable() Option {
	return func(o *widgetOpts) { o.selectable = true; o.rangeSelectable = true }
}

// FeedbackStyle selects the Feedback widget's rating style: "thumbs" (default,
// 👍/👎 → "up"/"down"), "stars" (★ → "1".."5"), or "faces" (→ "1".."5").
func FeedbackStyle(s string) Option { return func(o *widgetOpts) { o.feedbackStyle = s } }

// Horizontal lays out a BarChart with horizontal bars (st.bar_chart horizontal).
func Horizontal() Option { return func(o *widgetOpts) { o.horizontal = true } }

// Stacked stacks the series of a BarChart or AreaChart (st.bar_chart stack).
func Stacked() Option { return func(o *widgetOpts) { o.stacked = true } }

// Colors overrides the chart's series color palette with the given CSS colors.
func Colors(c []string) Option { return func(o *widgetOpts) { o.colors = c } }

// Autoplay / Loop / Muted control Audio and Video playback.
func Autoplay() Option { return func(o *widgetOpts) { o.autoplay = true } }
func Loop() Option     { return func(o *widgetOpts) { o.loop = true } }
func Muted() Option    { return func(o *widgetOpts) { o.muted = true } }

// StartTime / EndTime clip Audio/Video playback to a range (seconds).
func StartTime(seconds float64) Option { return func(o *widgetOpts) { o.startTime = seconds } }
func EndTime(seconds float64) Option   { return func(o *widgetOpts) { o.endTime = seconds } }

// Subtitles adds a subtitle track (WebVTT URL) to a Video.
func Subtitles(vttURL string) Option { return func(o *widgetOpts) { o.subtitles = vttURL } }

// AcceptNewOptions lets a MultiSelect accept values typed by the user in
// addition to the predefined options.
func AcceptNewOptions() Option { return func(o *widgetOpts) { o.acceptNew = true } }

// ColumnOrder reorders (and filters) the displayed columns of a DataFrame /
// DataEditor; columns not listed are hidden.
func ColumnOrder(cols ...string) Option { return func(o *widgetOpts) { o.columnOrder = cols } }

// SelectionMode sets what sy.Selectable() selects on a DataFrame:
// "multi-row" (default), "single-row", "multi-column", or "single-column".
// Row modes return selected row indices; column modes return selected column
// indices (into the headers slice). In column modes, clicking a header
// selects it (sorting is disabled).
func SelectionMode(mode string) Option { return func(o *widgetOpts) { o.selectionMode = mode } }

// ShowTime makes a Spinner display the elapsed time next to its label.
func ShowTime() Option { return func(o *widgetOpts) { o.showTime = true } }

// Mono renders a TextInput / TextArea in the code font — for formulas,
// identifiers, or anything typed character-by-character.
func Mono() Option { return func(o *widgetOpts) { o.mono = true } }

// Formula gives a TextInput the formula-bar look: an ƒx marker inside the
// box, code font, and a code-block background — visually distinct from
// ordinary text inputs. Implies Mono.
func Formula() Option { return func(o *widgetOpts) { o.formula = true; o.mono = true } }

// LineNumbers shows a line-number gutter on a Code block; Wrap soft-wraps long
// lines instead of scrolling horizontally.
func LineNumbers() Option { return func(o *widgetOpts) { o.lineNumbers = true } }
func Wrap() Option        { return func(o *widgetOpts) { o.wrap = true } }

// Avatar sets a custom avatar (emoji or image URL) for a ChatMessage.
func Avatar(v string) Option { return func(o *widgetOpts) { o.avatar = v } }

// Zoom sets the initial zoom level (1–18) of a Map.
func Zoom(level int) Option { return func(o *widgetOpts) { o.zoom = level } }

// RunEvery makes a Fragment re-run automatically on the given interval, for live
// dashboards (st.fragment run_every). The fragment re-executes only its own
// function each tick, not the whole app.
func RunEvery(d time.Duration) Option {
	return func(o *widgetOpts) { o.runEvery = int(d.Milliseconds()) }
}

// ClearOnSubmit resets a Form's inputs after a successful submit (st.form
// clear_on_submit). The submit handler still sees the submitted values; the
// inputs reset to type defaults (text→"", number→0, bool→false, select→first
// option). Widgets given an explicit DefaultValue reset to the type default,
// not that custom value.
func ClearOnSubmit() Option { return func(o *widgetOpts) { o.clearOnSubmit = true } }

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
func Caption(text string) { current().add(textNode("caption", text)) }

// Badge renders a small colored label. Color can be "blue", "green", "red",
// "orange", "gray", "violet", or any CSS color string.
func Badge(text string, opts ...Option) {
	o := applyOpts(opts)
	props := map[string]any{"text": text}
	if o.color != "" {
		props["color"] = o.color
	}
	current().add(&Node{Type: "badge", Props: props})
}

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

// Exception renders a Go error in a styled, monospace error box — the
// equivalent of Streamlit's st.exception. A nil error renders nothing.
func Exception(err error) {
	if err == nil {
		return
	}
	current().add(&Node{Type: "exception", Props: map[string]any{"text": err.Error()}})
}

func statusNode(level, text string) *Node {
	return &Node{Type: "status", Props: map[string]any{"level": level, "text": text}}
}

// --- Inputs (SPEC §6.4) ---

func TextInput(label string, opts ...Option) string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("text_input", o.key)
	val, exists := rc.sess.widgetValue(id)
	s, _ := val.(string)
	if !exists {
		// Initial value only: once the user has edited (even to empty), the
		// stored value wins, so the field stays clearable.
		if dv, ok := o.defaultVal.(string); ok {
			s = dv
		}
	}
	props := map[string]any{"label": label, "value": s}
	if o.placeholder != "" {
		props["placeholder"] = o.placeholder
	}
	if o.maxChars > 0 {
		props["max_chars"] = o.maxChars
	}
	if o.mono {
		props["mono"] = true
	}
	if o.formula {
		props["formula"] = true
	}
	applyCommonProps(props, o)
	rc.add(&Node{ID: id, Type: "text_input", Props: props})
	return s
}

// PasswordInput renders a password text input (masked characters).
func PasswordInput(label string, opts ...Option) string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("password_input", o.key)
	val, exists := rc.sess.widgetValue(id)
	s, _ := val.(string)
	if !exists {
		if dv, ok := o.defaultVal.(string); ok {
			s = dv
		}
	}
	props := map[string]any{"label": label, "value": s, "input_type": "password"}
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
	val, ok := rc.sess.widgetValue(id)
	b, _ := val.(bool)
	if !ok {
		if d, isBool := o.defaultVal.(bool); isBool {
			b = d
		}
	}
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
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	applyButtonProps(props, o)
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

// toFloatPair coerces a two-element value (sent as a JSON array, or supplied to
// DefaultValue as []float64 / [2]float64 / []int / [2]int) into an ordered pair.
func toFloatPair(val any) (float64, float64, bool) {
	switch v := val.(type) {
	case []any:
		if len(v) >= 2 {
			return toFloat64(v[0]), toFloat64(v[1]), true
		}
	case []float64:
		if len(v) >= 2 {
			return v[0], v[1], true
		}
	case [2]float64:
		return v[0], v[1], true
	case []int:
		if len(v) >= 2 {
			return float64(v[0]), float64(v[1]), true
		}
	case [2]int:
		return float64(v[0]), float64(v[1]), true
	}
	return 0, 0, false
}

// RangeSlider is a two-handle slider returning the selected (low, high) range —
// the equivalent of Streamlit's st.slider called with a tuple value. The initial
// range defaults to [min, max]; override with sy.DefaultValue([2]float64{lo, hi}).
// Inside a Form it is batched and commits on submit like any other input.
func RangeSlider(label string, min, max float64, opts ...Option) (float64, float64) {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("range_slider", o.key)

	lo, hi := min, max
	if val, exists := rc.sess.widgetValue(id); exists {
		if a, b, ok := toFloatPair(val); ok {
			lo, hi = a, b
		}
	} else if o.defaultVal != nil {
		if a, b, ok := toFloatPair(o.defaultVal); ok {
			lo, hi = a, b
		}
	}
	// Keep the pair ordered and within [min, max].
	if lo > hi {
		lo, hi = hi, lo
	}
	if lo < min {
		lo = min
	}
	if hi > max {
		hi = max
	}

	step := 1.0
	if o.step != nil {
		step = *o.step
	}
	props := map[string]any{"label": label, "low": lo, "high": hi, "min": min, "max": max, "step": step}
	if o.disabled {
		props["disabled"] = true
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	rc.add(&Node{ID: id, Type: "range_slider", Props: props})
	return lo, hi
}

// DateSlider is a slider over a date range, returning the picked date as a
// "YYYY-MM-DD" string — the equivalent of Streamlit's st.slider with date
// values. minDate/maxDate are "YYYY-MM-DD"; the initial value defaults to
// minDate (override with sy.DefaultValue("YYYY-MM-DD")). It is form-batched.
func DateSlider(label, minDate, maxDate string, opts ...Option) string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("date_slider", o.key)
	val, exists := rc.sess.widgetValue(id)
	cur, _ := val.(string)
	if !exists || cur == "" {
		if dv, ok := o.defaultVal.(string); ok && dv != "" {
			cur = dv
		} else {
			cur = minDate
		}
	}
	props := map[string]any{"label": label, "value": cur, "min": minDate, "max": maxDate}
	if o.disabled {
		props["disabled"] = true
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	rc.add(&Node{ID: id, Type: "date_slider", Props: props})
	return cur
}

// TimeSlider is a slider over a time-of-day range, returning "HH:MM" — the
// time-valued form of Streamlit's st.slider. minTime/maxTime are "HH:MM"; the
// initial value defaults to minTime. Use sy.Step (minutes) for the increment
// (default 15). It is form-batched.
func TimeSlider(label, minTime, maxTime string, opts ...Option) string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("time_slider", o.key)
	val, exists := rc.sess.widgetValue(id)
	cur, _ := val.(string)
	if !exists || cur == "" {
		if dv, ok := o.defaultVal.(string); ok && dv != "" {
			cur = dv
		} else {
			cur = minTime
		}
	}
	props := map[string]any{"label": label, "value": cur, "min": minTime, "max": maxTime}
	if o.step != nil {
		props["step"] = *o.step
	}
	if o.disabled {
		props["disabled"] = true
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	rc.add(&Node{ID: id, Type: "time_slider", Props: props})
	return cur
}

func TextArea(label string, opts ...Option) string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("textarea", o.key)
	val, exists := rc.sess.widgetValue(id)
	s, _ := val.(string)
	if !exists {
		if dv, ok := o.defaultVal.(string); ok {
			s = dv
		}
	}
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
	if o.acceptNew {
		props["accept_new"] = true
	}
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
	applyDateBounds(props, o)
	rc.add(&Node{ID: id, Type: "date_input", Props: props})
	return s
}

// DatetimeInput renders a combined date-and-time picker and returns the value
// as "YYYY-MM-DD HH:MM" ("" until the user picks one). Supports MinDate /
// MaxDate ("YYYY-MM-DD" or "YYYY-MM-DDTHH:MM") bounds and DefaultValue.
func DatetimeInput(label string, opts ...Option) string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("datetime_input", o.key)
	val, ok := rc.sess.widgetValue(id)
	s, _ := val.(string)
	if !ok {
		if d, isStr := o.defaultVal.(string); isStr {
			s = d
		}
	}
	// The browser control uses "YYYY-MM-DDTHH:MM"; normalize to a space.
	s = strings.ReplaceAll(s, "T", " ")
	props := map[string]any{"label": label, "value": strings.ReplaceAll(s, " ", "T")}
	if o.disabled {
		props["disabled"] = true
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	applyDateBounds(props, o)
	rc.add(&Node{ID: id, Type: "datetime_input", Props: props})
	return s
}

// MenuButton renders a button that opens a dropdown of options and returns the
// clicked option for exactly one rerun ("" otherwise) — like Button, but with
// a choice attached.
func MenuButton(label string, options []string, opts ...Option) string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("menu_button", o.key)
	choice, _ := rc.sess.takeWidget(id).(string) // one-shot, like a button press
	props := map[string]any{"label": label, "options": options}
	if o.disabled {
		props["disabled"] = true
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	applyButtonProps(props, o)
	rc.add(&Node{ID: id, Type: "menu_button", Props: props})
	return choice
}

// applyDateBounds copies min/max date bounds onto a date widget's props.
func applyDateBounds(props map[string]any, o widgetOpts) {
	if o.minDate != "" {
		props["min"] = o.minDate
	}
	if o.maxDate != "" {
		props["max"] = o.maxDate
	}
}

// DateRangeInput is a pair of date pickers returning the selected (start, end)
// dates as "YYYY-MM-DD" strings (empty until picked) — the equivalent of
// Streamlit's st.date_input called with a (start, end) tuple.
// Inside a Form it is batched and commits on submit like any other input.
func DateRangeInput(label string, opts ...Option) (string, string) {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("date_range_input", o.key)

	var start, end string
	if val, exists := rc.sess.widgetValue(id); exists {
		if a, ok := val.([]any); ok && len(a) >= 2 {
			start, _ = a[0].(string)
			end, _ = a[1].(string)
		}
	} else if o.defaultVal != nil {
		if a, ok := o.defaultVal.([]string); ok && len(a) >= 2 {
			start, end = a[0], a[1]
		}
	}
	props := map[string]any{"label": label, "start": start, "end": end}
	if o.disabled {
		props["disabled"] = true
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	applyDateBounds(props, o)
	rc.add(&Node{ID: id, Type: "date_range_input", Props: props})
	return start, end
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
	val, ok := rc.sess.widgetValue(id)
	b, _ := val.(bool)
	if !ok {
		if d, isBool := o.defaultVal.(bool); isBool {
			b = d
		}
	}
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

	file := decodeUploadedFile(val)

	props := map[string]any{"label": label, "max_size": uploadLimitBytes}
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

// FileUploaderMultiple renders a file upload widget that accepts several files
// at once and returns all of them (empty slice when none are uploaded yet).
func FileUploaderMultiple(label string, opts ...Option) []*UploadedFile {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("file_uploader", o.key)
	val, _ := rc.sess.widgetValue(id)

	var files []*UploadedFile
	if list, ok := val.([]any); ok {
		for _, item := range list {
			if f := decodeUploadedFile(item); f != nil {
				files = append(files, f)
			}
		}
	}

	props := map[string]any{"label": label, "multiple": true, "max_size": uploadLimitBytes}
	if len(files) > 0 {
		names := make([]string, len(files))
		var total int64
		for i, f := range files {
			names[i] = f.Name
			total += f.Size
		}
		props["file_names"] = names
		props["file_size"] = total
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	rc.add(&Node{ID: id, Type: "file_uploader", Props: props})
	return files
}

// decodeUploadedFile converts a browser upload payload into an UploadedFile.
func decodeUploadedFile(val any) *UploadedFile {
	m, ok := val.(map[string]any)
	if !ok {
		return nil
	}
	name, _ := m["name"].(string)
	size := toFloat64(m["size"])
	typ, _ := m["type"].(string)
	dataStr, _ := m["data"].(string)
	data, err := base64.StdEncoding.DecodeString(dataStr)
	if err != nil || name == "" {
		return nil
	}
	return &UploadedFile{Name: name, Size: int64(size), Type: typ, Data: data}
}

// LinkButton renders a button-styled hyperlink that opens in a new tab.
func LinkButton(label, url string, opts ...Option) {
	o := applyOpts(opts)
	props := map[string]any{"label": label, "url": url}
	if o.disabled {
		props["disabled"] = true
	}
	applyButtonProps(props, o)
	current().add(&Node{Type: "link_button", Props: props})
}

// applyButtonProps copies the shared button styling options (icon, type, width)
// onto a node's props, so Button, LinkButton and DownloadButton style alike.
func applyButtonProps(props map[string]any, o widgetOpts) {
	if o.icon != "" {
		props["icon"] = o.icon
	}
	if o.buttonType != "" {
		props["buttonType"] = o.buttonType
	}
	if o.useContainerWidth {
		props["containerWidth"] = true
	}
}

// PageLink renders a navigation link to another page in the app (or an
// external URL). When clicked for an internal page, a page_change is sent.
func PageLink(label string, page string, opts ...Option) {
	o := applyOpts(opts)
	props := map[string]any{"label": label, "page": page}
	if o.disabled {
		props["disabled"] = true
	}
	current().add(&Node{Type: "page_link", Props: props})
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
		case bool:
			Textf("%v", v)
		case int, int8, int16, int32, int64,
			uint, uint8, uint16, uint32, uint64,
			float32, float64:
			Textf("%v", v)
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
func ChatMessage(role string, fn func(), opts ...Option) {
	rc := current()
	o := applyOpts(opts)
	props := map[string]any{"role": role}
	if o.avatar != "" {
		props["avatar"] = o.avatar
	}
	msg := &Node{Type: "chat_message", Props: props}
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
func Spinner(text ...string) { spinnerOpts(nil, text...) }

// SpinnerWith renders a spinner with options, e.g. sy.ShowTime().
func SpinnerWith(opts []Option, text ...string) { spinnerOpts(opts, text...) }

func spinnerOpts(opts []Option, text ...string) {
	label := "Loading..."
	if len(text) > 0 && text[0] != "" {
		label = text[0]
	}
	props := map[string]any{"text": label}
	if applyOpts(opts).showTime {
		props["show_time"] = true
	}
	current().add(&Node{Type: "spinner", Props: props})
}

// Popover renders a button that shows a floating panel with the content
// produced by fn.
func Popover(label string, fn func(), opts ...Option) {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("popover", o.key)
	val, _ := rc.sess.widgetValue(id)
	open, _ := val.(bool)
	props := map[string]any{"label": label, "open": open}
	if o.disabled {
		props["disabled"] = true
	}
	if o.helpText != "" {
		props["help"] = o.helpText
	}
	applyButtonProps(props, o)
	pop := &Node{ID: id, Type: "popover", Props: props}
	rc.add(pop)
	rc.stack = append(rc.stack, pop)
	fn()
	rc.stack = rc.stack[:len(rc.stack)-1]
}

// Feedback renders a thumbs up/down rating widget. Returns "up", "down",
// or "" (no selection yet). Useful for collecting user feedback on AI responses.
func Feedback(opts ...Option) string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("feedback", o.key)
	val, _ := rc.sess.widgetValue(id)
	s, _ := val.(string)
	style := o.feedbackStyle
	if style == "" {
		style = "thumbs"
	}
	props := map[string]any{"value": s, "style": style}
	if o.disabled {
		props["disabled"] = true
	}
	rc.add(&Node{ID: id, Type: "feedback", Props: props})
	return s
}

// SegmentedControl renders a row of mutually exclusive buttons. Returns the
// selected option string. Similar to Radio but rendered as a single bar.
func SegmentedControl(label string, options []string, opts ...Option) string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("segmented_control", o.key)
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
	applyCommonProps(props, o)
	rc.add(&Node{ID: id, Type: "segmented_control", Props: props})
	return s
}

// Pills renders a set of selectable tag-like buttons. Returns the selected
// option string. Use for filter or category selection.
// toStringSlice coerces a stored widget value (a JSON array → []any, or already
// []string) into []string.
func toStringSlice(val any) []string {
	switch v := val.(type) {
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, e := range v {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// PillsMulti is the multi-select form of Pills: the user can toggle any number
// of options, and it returns the selected set (st.pills selection_mode="multi").
func PillsMulti(label string, options []string, opts ...Option) []string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("pills", o.key)
	val, _ := rc.sess.widgetValue(id)
	sel := toStringSlice(val)
	props := map[string]any{"label": label, "options": options, "value": sel, "multi": true}
	applyCommonProps(props, o)
	rc.add(&Node{ID: id, Type: "pills", Props: props})
	return sel
}

// SegmentedControlMulti is the multi-select form of SegmentedControl.
func SegmentedControlMulti(label string, options []string, opts ...Option) []string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("segmented_control", o.key)
	val, _ := rc.sess.widgetValue(id)
	sel := toStringSlice(val)
	props := map[string]any{"label": label, "options": options, "value": sel, "multi": true}
	applyCommonProps(props, o)
	rc.add(&Node{ID: id, Type: "segmented_control", Props: props})
	return sel
}

func Pills(label string, options []string, opts ...Option) string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("pills", o.key)
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
	applyCommonProps(props, o)
	rc.add(&Node{ID: id, Type: "pills", Props: props})
	return s
}

// Pagination renders a page selector for paginating data. Returns the current
// page number (1-based). totalPages is the total number of pages.
func Pagination(totalPages int, opts ...Option) int {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("pagination", o.key)
	val, _ := rc.sess.widgetValue(id)
	page, _ := val.(float64)
	if page < 1 {
		page = 1
	}
	if int(page) > totalPages {
		page = float64(totalPages)
	}
	props := map[string]any{"page": int(page), "total_pages": totalPages}
	if o.disabled {
		props["disabled"] = true
	}
	rc.add(&Node{ID: id, Type: "pagination", Props: props})
	return int(page)
}

// CameraInput renders a webcam capture widget. The user clicks "Take Photo"
// to capture a snapshot, which is returned as a base64-encoded JPEG data URI.
// Returns empty string until a photo is taken.
func CameraInput(label string, opts ...Option) string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("camera_input", o.key)
	val, _ := rc.sess.widgetValue(id)
	s, _ := val.(string)
	props := map[string]any{"label": label}
	applyCommonProps(props, o)
	rc.add(&Node{ID: id, Type: "camera_input", Props: props})
	return s
}

// AudioInput renders a microphone recording widget. Returns the recorded audio
// as a base64 data URI (audio/webm), or "" if nothing recorded.
func AudioInput(label string, opts ...Option) string {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("audio_input", o.key)
	val, _ := rc.sess.widgetValue(id)
	s, _ := val.(string)
	props := map[string]any{"label": label}
	applyCommonProps(props, o)
	rc.add(&Node{ID: id, Type: "audio_input", Props: props})
	return s
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
