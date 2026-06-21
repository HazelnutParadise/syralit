package syralit

// Node is one element in the UI tree that a rerun produces. The whole tree is
// serialized to JSON and sent to the browser, where the client runtime renders
// and reconciles it. See docs/event-protocol — Type drives which DOM element the
// client builds, Props carries its attributes, and ID is the stable widget key.
type Node struct {
	ID       string         `json:"id,omitempty"`
	Type     string         `json:"type"`
	Props    map[string]any `json:"props,omitempty"`
	Children []*Node        `json:"children,omitempty"`
}

// findNodeByID returns the first node (depth-first) with the given ID, or nil.
func findNodeByID(n *Node, id string) *Node {
	if n == nil {
		return nil
	}
	if n.ID == id {
		return n
	}
	for _, c := range n.Children {
		if found := findNodeByID(c, id); found != nil {
			return found
		}
	}
	return nil
}

// clearFormInputs resets the value props of input nodes in a subtree to type
// defaults — used by Form's clear_on_submit. Selects reset to their first
// option; sliders/ranges to their bounds; text/number/bool to empty/zero/false.
func clearFormInputs(n *Node) {
	if n == nil {
		return
	}
	switch n.Type {
	case "text_input", "textarea", "password_input", "date_input", "time_input",
		"color_picker", "chat_input", "date_slider", "time_slider", "feedback":
		n.Props["value"] = ""
	case "number_input":
		n.Props["value"] = 0
	case "checkbox", "toggle":
		n.Props["value"] = false
	case "multi_select":
		n.Props["value"] = []string{}
	case "select", "radio", "select_slider", "segmented_control", "pills":
		if multi, _ := n.Props["multi"].(bool); multi {
			n.Props["value"] = []string{}
		} else if opts, ok := n.Props["options"].([]string); ok && len(opts) > 0 {
			n.Props["value"] = opts[0]
		} else {
			n.Props["value"] = ""
		}
	case "slider":
		if min, ok := n.Props["min"]; ok {
			n.Props["value"] = min
		} else {
			n.Props["value"] = 0
		}
	case "range_slider":
		n.Props["low"] = n.Props["min"]
		n.Props["high"] = n.Props["max"]
	case "date_range_input":
		n.Props["start"] = ""
		n.Props["end"] = ""
	}
	for _, c := range n.Children {
		clearFormInputs(c)
	}
}

// Find returns every descendant of the given type (depth-first, including n
// itself if it matches). It's a convenience for tests built on RenderOnce.
func (n *Node) Find(typ string) []*Node {
	var out []*Node
	var walk func(*Node)
	walk = func(x *Node) {
		if x == nil {
			return
		}
		if x.Type == typ {
			out = append(out, x)
		}
		for _, c := range x.Children {
			walk(c)
		}
	}
	walk(n)
	return out
}
