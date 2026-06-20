package syralit

import (
	"fmt"
	"runtime/debug"
	"sync"
)

// renderContext is the scratch state for a single rerun. Widget functions
// (Title, TextInput, Button, ...) read it through current() to know which
// session they belong to and where to append their Node.
type renderContext struct {
	sess    *session
	root    *Node
	stack   []*Node        // container stack; root sits at the bottom
	autoSeq map[string]int // widget type -> counter, for auto-generated IDs
}

// The rerun model lets users write `sy.App(func(){ sy.TextInput("Name") })` with
// no context parameter, so the global widget functions must discover "which rerun
// am I inside". Go has no goroutine-local storage, so instead of a gid hack we
// serialize reruns with rerunMu: at any instant only one rerun runs, so a single
// package-level pointer is safe to read from the widget functions.
//
// Cost: reruns across different sessions cannot run in parallel. That is fine for
// Syralit's single-machine, local-app positioning. Multi-session concurrency is a
// later concern (see SPEC §3 — multi-user collaboration is a non-goal).
var (
	rerunMu sync.Mutex
	cur     *renderContext
)

func current() *renderContext {
	if cur == nil {
		panic("syralit: widget called outside of a rerun (did you call it inside sy.App's function?)")
	}
	return cur
}

// add appends n to the current container (top of the stack).
func (rc *renderContext) add(n *Node) {
	parent := rc.stack[len(rc.stack)-1]
	parent.Children = append(parent.Children, n)
}

// widgetID returns the stable ID for a widget. An explicit key always wins;
// otherwise we fall back to call-order numbering, which is unstable under
// conditional UI — see SPEC §6.10, hence Key is encouraged everywhere.
func (rc *renderContext) widgetID(typ, key string) string {
	if key != "" {
		return key
	}
	rc.autoSeq[typ]++
	if hasPages() {
		return fmt.Sprintf("__%s:%s_%d", rc.sess.activePage(), typ, rc.autoSeq[typ])
	}
	return fmt.Sprintf("__%s_%d", typ, rc.autoSeq[typ])
}

type stopSentinel struct{}

// Stop halts the current rerun early. Widgets rendered so far are kept.
// Useful for conditional guards:
//
//	if !loggedIn { sy.Warning("Please log in"); sy.Stop() }
func Stop() { panic(stopSentinel{}) }

// runRerun executes the user's App function once and returns the produced UI
// tree. A panic in user code is caught and rendered as an error node so a single
// rerun cannot crash the server (SPEC §14).
func runRerun(sess *session) *Node {
	rerunMu.Lock()
	defer rerunMu.Unlock()

	rc := &renderContext{
		sess:    sess,
		root:    &Node{Type: "root"},
		autoSeq: map[string]int{},
	}
	rc.stack = []*Node{rc.root}
	cur = rc
	defer func() { cur = nil }()

	func() {
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(stopSentinel); !ok {
					stack := string(debug.Stack())
					rc.add(&Node{Type: "status", Props: map[string]any{
						"level": "error",
						"text":  fmt.Sprintf("App panic: %v", r),
					}})
					rc.add(&Node{Type: "code", Props: map[string]any{"code": stack}})
				}
			}
		}()
		fn := sess.appFn
		if hasPages() {
			if pageFn := pageByTitle(sess.activePage()); pageFn != nil {
				fn = pageFn
			}
		}
		if fn != nil {
			fn()
		}
	}()

	// Buttons are transient: a click is true for exactly one rerun, then reset.
	sess.resetTransient()
	return rc.root
}
