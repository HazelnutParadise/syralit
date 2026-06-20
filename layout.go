package syralit

// Column renders children into one column of a Columns layout.
type Column func(fn func())

// Columns creates n side-by-side columns. Call each returned Column with a
// callback to populate it.
func Columns(n int) []Column {
	rc := current()
	colsNode := &Node{Type: "columns", Props: map[string]any{"count": n}}
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
	val, _ := rc.sess.widgetValue(id)
	expanded, _ := val.(bool)
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
func Container(fn func()) {
	rc := current()
	node := &Node{Type: "container"}
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

// Divider renders a horizontal rule.
func Divider() {
	current().add(&Node{Type: "divider"})
}
