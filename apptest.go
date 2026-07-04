package syralit

import "fmt"

// AppTest runs an app function headlessly for testing — the Go counterpart of
// Streamlit's st.testing.v1.AppTest. It renders the UI tree without a server
// or browser, lets the test set widget values and click buttons, and exposes
// the resulting tree for assertions.
//
//	at := sy.NewAppTest(func() {
//	    name := sy.TextInput("Name", sy.Key("name"))
//	    if sy.Button("Greet", sy.Key("greet")) {
//	        sy.Success("Hello, " + name)
//	    }
//	})
//	at.Run()
//	at.SetValue("name", "Ada")
//	at.Click("greet")
//	at.Run()
//	if len(at.FindAll("status")) == 0 { t.Fatal("no greeting") }
//
// Widgets addressed by test code should use sy.Key so their IDs are stable.
type AppTest struct {
	sess *session
	// Root is the UI tree produced by the most recent Run (nil before).
	Root *Node
}

// NewAppTest wraps an app function for headless testing. In multi-page apps
// pass nil and use SwitchToPage.
func NewAppTest(fn func()) *AppTest {
	return &AppTest{sess: newSession(fn)}
}

// Run executes one rerun (exactly like a browser-triggered rerun, including
// one-shot button semantics) and stores the produced tree in Root.
func (t *AppTest) Run() *Node {
	t.Root = runRerun(t.sess)
	return t.Root
}

// SetValue sets a widget's value by its ID (the sy.Key of the widget). The
// change takes effect on the next Run, like a browser interaction.
func (t *AppTest) SetValue(key string, value any) *AppTest {
	t.sess.setWidget(key, value)
	return t
}

// Value returns the currently stored value for a widget ID.
func (t *AppTest) Value(key string) any {
	v, _ := t.sess.widgetValue(key)
	return v
}

// Click registers a button press by widget ID; the button returns true during
// the next Run only.
func (t *AppTest) Click(key string) *AppTest {
	t.sess.pressButton(key)
	return t
}

// ClickLabel finds a button by its visible label in the last rendered tree and
// registers a press. Run must have been called first.
func (t *AppTest) ClickLabel(label string) error {
	n := t.FindByLabel("button", label)
	if n == nil || n.ID == "" {
		return fmt.Errorf("apptest: no button labeled %q in last render", label)
	}
	t.sess.pressButton(n.ID)
	return nil
}

// SwitchToPage sets the active page (multi-page apps) for subsequent Runs.
func (t *AppTest) SwitchToPage(title string) *AppTest {
	t.sess.setCurrentPage(title)
	return t
}

// FindAll returns every node of the given type in the last rendered tree,
// depth-first.
func (t *AppTest) FindAll(nodeType string) []*Node {
	var out []*Node
	walkNodes(t.Root, func(n *Node) {
		if n.Type == nodeType {
			out = append(out, n)
		}
	})
	return out
}

// FindByLabel returns the first node of the given type whose "label" prop
// equals label, or nil.
func (t *AppTest) FindByLabel(nodeType, label string) *Node {
	var found *Node
	walkNodes(t.Root, func(n *Node) {
		if found == nil && n.Type == nodeType {
			if l, _ := n.Props["label"].(string); l == label {
				found = n
			}
		}
	})
	return found
}

// Texts returns the "text" prop of every node of the given type, in render
// order — convenient for asserting on Title/Text/Success/Error output.
func (t *AppTest) Texts(nodeType string) []string {
	var out []string
	for _, n := range t.FindAll(nodeType) {
		if s, ok := n.Props["text"].(string); ok {
			out = append(out, s)
		}
	}
	return out
}

// walkNodes visits every node depth-first.
func walkNodes(n *Node, visit func(*Node)) {
	if n == nil {
		return
	}
	visit(n)
	for _, c := range n.Children {
		walkNodes(c, visit)
	}
}
