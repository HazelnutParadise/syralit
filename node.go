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
