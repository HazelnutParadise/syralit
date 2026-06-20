package syralit

import (
	"strconv"
	"strings"
)

// Column renders children into one column of a Columns layout.
type Column func(fn func())

// Columns creates n side-by-side columns. Call each returned Column with a
// callback to populate it.
func Columns(n int, opts ...Option) []Column {
	rc := current()
	o := applyOpts(opts)
	props := map[string]any{"count": n}
	if o.gap > 0 {
		props["gap"] = o.gap
	}
	colsNode := &Node{Type: "columns", Props: props}
	rc.add(colsNode)

	cols := make([]Column, n)
	for i := range n {
		colNode := &Node{Type: "column"}
		colsNode.Children = append(colsNode.Children, colNode)
		cols[i] = func(fn func()) {
			rc.stack = append(rc.stack, colNode)
			fn()
			rc.stack = rc.stack[:len(rc.stack)-1]
		}
	}
	return cols
}

// Expander renders a collapsible section. The expanded/collapsed state is
// tracked server-side and persists across reruns.
func Expander(label string, fn func(), opts ...Option) {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("expander", o.key)
	val, hasVal := rc.sess.widgetValue(id)
	expanded, _ := val.(bool)
	if !hasVal {
		if dv, ok := o.defaultVal.(bool); ok {
			expanded = dv
		}
	}
	node := &Node{ID: id, Type: "expander", Props: map[string]any{
		"label":    label,
		"expanded": expanded,
	}}
	rc.add(node)
	rc.stack = append(rc.stack, node)
	fn()
	rc.stack = rc.stack[:len(rc.stack)-1]
}

// Tabs creates a tabbed container. It returns a function to define each tab's
// content. The active tab is tracked server-side. All tab content is rendered
// and sent; the client shows only the active tab.
func Tabs(labels []string, opts ...Option) func(string, func()) {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("tabs", o.key)
	val, _ := rc.sess.widgetValue(id)
	active, _ := val.(string)
	if active == "" && len(labels) > 0 {
		active = labels[0]
	}
	tabsNode := &Node{ID: id, Type: "tabs", Props: map[string]any{
		"labels": labels,
		"active": active,
	}}
	rc.add(tabsNode)

	return func(label string, fn func()) {
		tabNode := &Node{Type: "tab_panel", Props: map[string]any{"label": label}}
		tabsNode.Children = append(tabsNode.Children, tabNode)
		rc.stack = append(rc.stack, tabNode)
		fn()
		rc.stack = rc.stack[:len(rc.stack)-1]
	}
}

// Sidebar adds user-defined content to the sidebar (below the page links).
// In single-page mode this also causes the sidebar to appear.
func Sidebar(fn func()) {
	rc := current()
	node := &Node{Type: "sidebar_content"}
	rc.add(node)
	rc.stack = append(rc.stack, node)
	fn()
	rc.stack = rc.stack[:len(rc.stack)-1]
}

// Container groups widgets into a plain wrapper (useful for conditional blocks).
// Use Border() to render a visible border and Height(px) for a scrollable area.
func Container(fn func(), opts ...Option) {
	rc := current()
	o := applyOpts(opts)
	props := map[string]any{}
	if o.border {
		props["border"] = true
	}
	if o.height > 0 {
		props["height"] = o.height
	}
	node := &Node{Type: "container", Props: props}
	rc.add(node)
	rc.stack = append(rc.stack, node)
	fn()
	rc.stack = rc.stack[:len(rc.stack)-1]
}

// Form groups widgets so their changes are batched and only sent when the user
// clicks FormSubmitButton. Inside a Form, widget changes do not trigger reruns.
func Form(key string, fn func()) {
	rc := current()
	id := rc.widgetID("form", key)
	node := &Node{ID: id, Type: "form"}
	rc.add(node)
	rc.stack = append(rc.stack, node)
	fn()
	rc.stack = rc.stack[:len(rc.stack)-1]
}

// FormSubmitButton renders a submit button inside a Form. Returns true on the
// single rerun triggered by form submission.
func FormSubmitButton(label string, opts ...Option) bool {
	rc := current()
	o := applyOpts(opts)
	id := rc.widgetID("form_submit", o.key)
	pressed := rc.sess.buttonPressed(id)
	rc.add(&Node{ID: id, Type: "form_submit", Props: map[string]any{"label": label}})
	return pressed
}

// WeightedColumns creates columns with custom widths specified as fractions.
// Example: WeightedColumns(2, 1) creates a 2:1 split (66%/33%).
func WeightedColumns(weights ...float64) []Column {
	rc := current()
	parts := make([]string, len(weights))
	for i, w := range weights {
		parts[i] = strconv.FormatFloat(w, 'f', -1, 64) + "fr"
	}
	colsNode := &Node{Type: "columns", Props: map[string]any{
		"template": strings.Join(parts, " "),
	}}
	rc.add(colsNode)

	cols := make([]Column, len(weights))
	for i := range weights {
		colNode := &Node{Type: "column"}
		colsNode.Children = append(colsNode.Children, colNode)
		cols[i] = func(fn func()) {
			rc.stack = append(rc.stack, colNode)
			fn()
			rc.stack = rc.stack[:len(rc.stack)-1]
		}
	}
	return cols
}

// Status renders a collapsible status container with a state indicator.
// state is "running", "complete", or "error". While "running", a spinner is
// shown next to the label.
//
//	sy.Status("Loading data", "running", func() {
//	    sy.Text("Fetching from API...")
//	    sy.Progress(0.5)
//	})
func Status(label, state string, fn func()) {
	rc := current()
	node := &Node{Type: "status_container", Props: map[string]any{
		"label": label,
		"state": state,
	}}
	rc.add(node)
	rc.stack = append(rc.stack, node)
	fn()
	rc.stack = rc.stack[:len(rc.stack)-1]
}

// Empty renders nothing by default but occupies a slot in the layout. Call
// the returned function with a callback to populate it. Useful for replacing
// content across reruns.
//
//	placeholder := sy.Empty()
//	if ready {
//	    placeholder(func() { sy.Text("Data loaded") })
//	}
func Empty() func(func()) {
	rc := current()
	node := &Node{Type: "container"}
	rc.add(node)
	return func(fn func()) {
		rc.stack = append(rc.stack, node)
		fn()
		rc.stack = rc.stack[:len(rc.stack)-1]
	}
}

// Divider renders a horizontal rule.
func Divider() {
	current().add(&Node{Type: "divider"})
}
