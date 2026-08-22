package syralit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/coder/websocket"
)

func TestPageSlugs(t *testing.T) {
	defer resetPages()

	AddPage("Home", func() {})
	AddPage("Data Explorer", func() {})
	AddPage("設定", func() {})
	AddPage("Reports / 2026", func() {})
	AddPage("Billing", func() {}, PageSlug("pay"))
	AddPage("data explorer", func() {}) // collides with "Data Explorer"

	got := map[string]string{}
	for _, p := range resolvedPages() {
		got[p.title] = p.slug
	}
	want := map[string]string{
		"Home":           "home",
		"Data Explorer":  "data_explorer",
		"設定":             "設定",
		"Reports / 2026": "reports_2026",
		"Billing":        "pay",
		"data explorer":  "data_explorer-2",
	}
	for title, slug := range want {
		if got[title] != slug {
			t.Errorf("slug for %q = %q, want %q", title, got[title], slug)
		}
	}

	if p, ok := pageBySlug("data_explorer"); !ok || p.title != "Data Explorer" {
		t.Fatalf("pageBySlug(data_explorer) = %+v, %v", p, ok)
	}
	if _, ok := pageBySlug("nope"); ok {
		t.Fatal("pageBySlug matched an unknown slug")
	}
}

func TestPageInfosCarrySlug(t *testing.T) {
	defer resetPages()
	AddPage("Data Explorer", func() {})

	infos := pageInfos()
	if len(infos) != 1 || infos[0]["slug"] != "data_explorer" {
		t.Fatalf("page infos missing slug: %+v", infos)
	}
}

func TestIndexServesPagePath(t *testing.T) {
	defer resetPages()
	AddPage("Home", func() { Title("home") })
	AddPage("Data Explorer", func() { Title("explorer") })

	srv := httptest.NewServer((&server{cfg: Config{Title: "My App"}, appFn: nil}).handler())
	defer srv.Close()

	get := func(path string) (int, string) {
		t.Helper()
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("get %s: %v", path, err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}

	// The app root keeps the app-wide title.
	if code, body := get("/"); code != 200 || !strings.Contains(body, "<title>My App</title>") {
		t.Fatalf("root: %d\n%s", code, body)
	}
	// A page path serves the shell with that page's title, so a link preview
	// or View Source sees the right page without running the client.
	if code, body := get("/data_explorer"); code != 200 || !strings.Contains(body, "<title>Data Explorer</title>") {
		t.Fatalf("page path: %d\n%s", code, body)
	}
	// An unknown path is still a 404, not a silent fallback to the app.
	if code, _ := get("/nope"); code != http.StatusNotFound {
		t.Fatalf("unknown path returned %d, want 404", code)
	}
}

func TestStaticFileWinsOverPageSlug(t *testing.T) {
	defer resetPages()
	resetStatic()
	defer resetStatic()
	AddPage("Robots", func() {})
	Static(fstest.MapFS{"robots.txt": {Data: []byte("User-agent: *")}})

	srv := httptest.NewServer((&server{cfg: Config{}, appFn: nil}).handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/robots.txt")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(b), "User-agent") {
		t.Fatalf("static file should win over a page slug, got:\n%s", b)
	}
}

// TestInitialPageFromURL is the point of the whole change: opening a page URL
// directly must render that page, not the first one in the sidebar.
func TestInitialPageFromURL(t *testing.T) {
	defer resetPages()

	AddPage("Home", func() { Title("home page") }, PageOrder(1))
	AddPage("Data Explorer", func() {
		Title("explorer page")
		if _, ok := QueryParams()[pageQueryKey]; ok {
			Text("LEAKED")
		}
	}, PageOrder(2))

	srv := httptest.NewServer((&server{cfg: Config{}, appFn: nil}).handler())
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/_syralit/ws?" + pageQueryKey + "=data_explorer"
	c, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.CloseNow()

	_, data, err := c.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var msg struct {
		ActivePage string `json:"active_page"`
	}
	if err := json.Unmarshal(data, &msg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if msg.ActivePage != "Data Explorer" {
		t.Fatalf("active page = %q, want Data Explorer\n%s", msg.ActivePage, data)
	}
	// The marker is transport-level; it must not surface in sy.QueryParams().
	if strings.Contains(string(data), "LEAKED") || strings.Contains(string(data), pageQueryKey) {
		t.Fatalf("page marker leaked into the session:\n%s", data)
	}
}

// TestPageChangeAcceptsSlug covers the dev path: the supervisor has no page
// registry, so it forwards the slug it read off the browser URL and the child
// has to resolve it.
func TestPageChangeAcceptsSlug(t *testing.T) {
	defer resetPages()
	AddPage("Home", func() { Title("home") }, PageOrder(1))
	AddPage("Data Explorer", func() { Title("explorer") }, PageOrder(2))

	srv := httptest.NewServer((&server{cfg: Config{}, appFn: nil}).handler())
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/_syralit/ws", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.CloseNow()

	readActive := func() string {
		t.Helper()
		_, data, err := c.Read(ctx)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		var msg struct {
			ActivePage string `json:"active_page"`
		}
		if err := json.Unmarshal(data, &msg); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		return msg.ActivePage
	}

	if got := readActive(); got != "Home" {
		t.Fatalf("initial page = %q, want Home", got)
	}
	if err := c.Write(ctx, websocket.MessageText, []byte(`{"type":"page_change","page":"data_explorer"}`)); err != nil {
		t.Fatalf("write: %v", err)
	}
	if got := readActive(); got != "Data Explorer" {
		t.Fatalf("page_change by slug landed on %q, want Data Explorer", got)
	}
}

func TestDevSupervisorServesPageURL(t *testing.T) {
	appDir := "_devpage_app"
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(appDir)

	s, mux, err := startSupervisor(DevOptions{Dir: appDir, Target: "."})
	if err != nil {
		t.Fatalf("startSupervisor: %v", err)
	}
	defer s.shutdown()

	srv := httptest.NewServer(mux)
	defer srv.Close()

	get := func(path string) (int, string) {
		t.Helper()
		resp, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("get %s: %v", path, err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}

	// The page registry lives in the child, so the supervisor serves the shell
	// for any plain path — otherwise reloading /reports would 404 in dev only.
	if code, body := get("/reports"); code != 200 || !strings.Contains(body, "syralit-root") {
		t.Fatalf("dev page URL: %d\n%s", code, body)
	}
	// A missing asset must still be a 404 rather than a page of HTML.
	if code, _ := get("/logo.png"); code != http.StatusNotFound {
		t.Fatalf("missing asset returned %d, want 404", code)
	}
	if code, _ := get("/a/b"); code != http.StatusNotFound {
		t.Fatalf("nested path returned %d, want 404", code)
	}
}
