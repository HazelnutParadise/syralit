package syralit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// findNode walks a node list and returns the first node of the given type whose
// "text" prop contains want (or any text node if want == "").
func findText(nodes []map[string]any, want string) (map[string]any, bool) {
	for _, n := range nodes {
		props, _ := n["props"].(map[string]any)
		if props != nil {
			if txt, _ := props["text"].(string); want == "" || strings.Contains(txt, want) {
				if want != "" && strings.Contains(txt, want) {
					return n, true
				}
			}
		}
	}
	return nil, false
}

func nodeByID(nodes []map[string]any, id string) (map[string]any, bool) {
	for _, n := range nodes {
		if n["id"] == id {
			return n, true
		}
	}
	return nil, false
}

// readPatch reads one ui_patch frame and returns its nodes.
func readPatch(t *testing.T, ctx context.Context, c *websocket.Conn) []map[string]any {
	t.Helper()
	_, data, err := c.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var msg struct {
		Type  string           `json:"type"`
		Nodes []map[string]any `json:"nodes"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.Type != "ui_patch" {
		t.Fatalf("expected ui_patch, got %q", msg.Type)
	}
	return msg.Nodes
}

func sendChange(t *testing.T, ctx context.Context, c *websocket.Conn, id string, value any, isButton bool) {
	t.Helper()
	b, _ := json.Marshal(map[string]any{
		"type": "widget_change", "widget_id": id, "value": value, "is_button": isButton,
	})
	if err := c.Write(ctx, websocket.MessageText, b); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// TestRerunSpine drives the full loop over a real WebSocket: initial render,
// text input rerun, and button+state rerun.
func TestRerunSpine(t *testing.T) {
	app := func() {
		Title("Demo")
		name := TextInput("Your name", Key("name"))
		if name != "" {
			Text("Hello, " + name + "!")
		}
		count := State("count", 0)
		if Button("Add", Key("add")) {
			count.Set(count.Get() + 1)
		}
		Textf("Count: %d", count.Get())
	}

	srv := httptest.NewServer((&server{cfg: Config{}, appFn: app}).handler())
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/_syralit/ws"
	c, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.CloseNow()

	// 1. Initial render: title present, no greeting yet, count 0.
	nodes := readPatch(t, ctx, c)
	if _, ok := nodeByID(nodes, "name"); !ok {
		t.Fatal("initial render missing text_input 'name'")
	}
	if _, ok := findText(nodes, "Hello, "); ok {
		t.Fatal("greeting should not appear before typing")
	}
	if _, ok := findText(nodes, "Count: 0"); !ok {
		t.Fatal("expected 'Count: 0' initially")
	}

	// 2. Type a name -> rerun -> greeting appears.
	sendChange(t, ctx, c, "name", "Tim", false)
	nodes = readPatch(t, ctx, c)
	if _, ok := findText(nodes, "Hello, Tim!"); !ok {
		t.Fatal("expected greeting 'Hello, Tim!' after typing")
	}

	// 3. Click Add twice -> count increments and persists across reruns.
	sendChange(t, ctx, c, "add", true, true)
	nodes = readPatch(t, ctx, c)
	if _, ok := findText(nodes, "Count: 1"); !ok {
		t.Fatal("expected 'Count: 1' after one click")
	}
	sendChange(t, ctx, c, "add", true, true)
	nodes = readPatch(t, ctx, c)
	if _, ok := findText(nodes, "Count: 2"); !ok {
		t.Fatal("expected 'Count: 2' after two clicks")
	}

	// 4. Button is transient: a non-button rerun must not re-increment.
	sendChange(t, ctx, c, "name", "Tim2", false)
	nodes = readPatch(t, ctx, c)
	if _, ok := findText(nodes, "Count: 2"); !ok {
		t.Fatal("count should stay at 2 when no button pressed (transient semantics)")
	}
}

// TestMultiPage verifies sidebar metadata, page switching, and per-page widget
// ID isolation.
func TestMultiPage(t *testing.T) {
	defer resetPages()

	AddPage("Home", func() {
		Title("Home Page")
		TextInput("Name", Key("name"))
	}, PageIcon("🏠"), PageOrder(1))
	AddPage("About", func() {
		Title("About Page")
		TextInput("Bio", Key("bio"))
	}, PageIcon("ℹ️"), PageOrder(2))

	srv := httptest.NewServer((&server{cfg: Config{}, appFn: nil}).handler())
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/_syralit/ws"
	c, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.CloseNow()

	// Helper to read a ui_patch and also capture pages/active_page.
	type patchMsg struct {
		Type       string           `json:"type"`
		Nodes      []map[string]any `json:"nodes"`
		Pages      []map[string]any `json:"pages"`
		ActivePage string           `json:"active_page"`
	}
	readMultiPatch := func() patchMsg {
		t.Helper()
		_, data, err := c.Read(ctx)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var msg patchMsg
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if msg.Type != "ui_patch" {
			t.Fatalf("expected ui_patch, got %q", msg.Type)
		}
		return msg
	}

	// 1. Initial render: default page is Home (lowest order).
	msg := readMultiPatch()
	if len(msg.Pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(msg.Pages))
	}
	if msg.ActivePage != "Home" {
		t.Fatalf("expected active page 'Home', got %q", msg.ActivePage)
	}
	if _, ok := findText(msg.Nodes, "Home Page"); !ok {
		t.Fatal("Home page content not rendered")
	}
	if _, ok := nodeByID(msg.Nodes, "name"); !ok {
		t.Fatal("Home page should have 'name' widget")
	}

	// 2. Switch to About page.
	b, _ := json.Marshal(map[string]any{"type": "page_change", "page": "About"})
	if err := c.Write(ctx, websocket.MessageText, b); err != nil {
		t.Fatalf("write page_change: %v", err)
	}
	msg = readMultiPatch()
	if msg.ActivePage != "About" {
		t.Fatalf("expected active page 'About', got %q", msg.ActivePage)
	}
	if _, ok := findText(msg.Nodes, "About Page"); !ok {
		t.Fatal("About page content not rendered")
	}
	if _, ok := nodeByID(msg.Nodes, "bio"); !ok {
		t.Fatal("About page should have 'bio' widget")
	}

	// 3. Switch back to Home — state should persist (widget still has value).
	sendChange(t, ctx, c, "name", "Tim", false)
	_ = readMultiPatch() // consume the rerun from widget_change

	b, _ = json.Marshal(map[string]any{"type": "page_change", "page": "Home"})
	if err := c.Write(ctx, websocket.MessageText, b); err != nil {
		t.Fatalf("write page_change: %v", err)
	}
	msg = readMultiPatch()
	nameNode, ok := nodeByID(msg.Nodes, "name")
	if !ok {
		t.Fatal("name widget missing after switching back to Home")
	}
	props, _ := nameNode["props"].(map[string]any)
	if v, _ := props["value"].(string); v != "Tim" {
		t.Fatalf("expected name widget value 'Tim', got %q", v)
	}
}

func TestIndexServed(t *testing.T) {
	srv := httptest.NewServer((&server{cfg: Config{Title: "T"}, appFn: func() {}}).handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("index status %d", resp.StatusCode)
	}

	js, err := http.Get(srv.URL + "/_syralit/assets/runtime.js")
	if err != nil {
		t.Fatal(err)
	}
	defer js.Body.Close()
	if js.StatusCode != 200 {
		t.Fatalf("runtime.js status %d", js.StatusCode)
	}
}
