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
