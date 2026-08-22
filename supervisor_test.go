package syralit

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

const devAppV1 = `package main

import sy "github.com/HazelnutParadise/syralit"

var board = sy.NewArtifactStore("main", sy.ArtifactSpec{
	Version: "v1",
	Nodes: []sy.ArtifactNode{{
		ID: "message", Component: "text",
		Props: map[string]any{"text": "Dev artifact"},
	}},
})

func init() {
	sy.HandleArtifactAPI(
		"/api/agent/artifacts",
		sy.StaticAgentKey("dev-test", "secret"),
		board,
	)
}

func main() {
	sy.App(func() {
		sy.Title("V1")
		c := sy.State("count", 0)
		if sy.Button("Add", sy.Key("add")) {
			c.Set(c.Get() + 1)
		}
		sy.Textf("Count: %d", c.Get())
		sy.ArtifactCanvas(board)
	})
}
`

// readUntil reads frames until all wants appear in a single frame, or it times
// out. Returns the matching frame text. Requiring all tokens in one frame avoids
// races with the new child's pre-restore initial render.
func readUntil(t *testing.T, c *websocket.Conn, timeout time.Duration, wants ...string) string {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		left := time.Until(deadline)
		if left <= 0 {
			t.Fatalf("timed out waiting for %v", wants)
		}
		ctx, cancel := context.WithTimeout(context.Background(), left)
		_, data, err := c.Read(ctx)
		cancel()
		if err != nil {
			t.Fatalf("read waiting for %v: %v", wants, err)
		}
		s := string(data)
		all := true
		for _, w := range wants {
			if !strings.Contains(s, w) {
				all = false
				break
			}
		}
		if all {
			return s
		}
	}
}

func sendWS(t *testing.T, c *websocket.Conn, frame string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := c.Write(ctx, websocket.MessageText, []byte(frame)); err != nil {
		t.Fatalf("write: %v", err)
	}
}

// TestHotReload is the end-to-end proof of the dev supervisor: state survives a
// rebuild, a compile error shows up without killing the running app, and a fix
// recovers — all while the outward connection stays open.
func TestHotReload(t *testing.T) {
	if testing.Short() {
		t.Skip("hot reload e2e shells out to `go build`; skipped in -short")
	}

	// App lives in a leading-underscore dir so it's ignored by ./... globs.
	appDir := "_devtest_app"
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(appDir)
	mainFile := filepath.Join(appDir, "main.go")
	writeApp := func(content string) {
		if err := os.WriteFile(mainFile, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeApp(devAppV1)

	s, mux, err := startSupervisor(DevOptions{Dir: appDir, Target: "."})
	if err != nil {
		t.Fatalf("startSupervisor: %v", err)
	}
	defer s.shutdown()

	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/_syralit/ws", nil)
	cancel()
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	c.SetReadLimit(32 << 20)
	defer c.CloseNow()

	// 1. Initial render: V1, count 0.
	readUntil(t, c, 15*time.Second, "V1")

	// 2. The outward dev server proxies artifact APIs to the child.
	req, err := http.NewRequest(http.MethodGet, srv.URL+"/api/agent/artifacts", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer secret")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("artifact API through supervisor: %v", err)
	}
	body, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusOK || !strings.Contains(string(body), `"id":"main"`) {
		t.Fatalf("artifact API through supervisor: status=%d body=%s", res.StatusCode, body)
	}

	// 3. Click Add twice -> count 2.
	addClick := `{"type":"widget_change","widget_id":"add","value":true,"is_button":true}`
	sendWS(t, c, addClick)
	readUntil(t, c, 10*time.Second, "Count: 1")
	sendWS(t, c, addClick)
	readUntil(t, c, 10*time.Second, "Count: 2")

	// 4. Edit source -> rebuild -> V2 appears AND count is preserved at 2
	//    (both must be in the same frame: the post-restore render).
	writeApp(strings.Replace(devAppV1, `"V1"`, `"V2"`, 1))
	readUntil(t, c, 30*time.Second, "V2", "Count: 2")

	// 5. Introduce a compile error -> build_error overlay, old app keeps running.
	writeApp(devAppV1 + "\nfunc broken() { this is not go }\n")
	readUntil(t, c, 30*time.Second, "__dev_build_error")

	// 6. Fix it (V3) -> recovers, state still preserved at 2.
	writeApp(strings.Replace(devAppV1, `"V1"`, `"V3"`, 1))
	readUntil(t, c, 30*time.Second, "V3", "Count: 2")
}

// TestDevSupervisorShell pins the shell the dev supervisor serves. It owns the
// outward port and renders index.html itself, so every syralit.toml key that
// shapes the document has to reach it too — otherwise dev and production show
// different pages. The child is never built here: the supervisor serves its
// shell whether or not the app compiles, which is the whole point.
func TestDevSupervisorShell(t *testing.T) {
	appDir := "_devshell_app"
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(appDir)
	toml := `title = "Dev Shell"
lang = "he-IL"
dir = "rtl"
head_html = "<meta name=\"robots\" content=\"noindex\">"

[i18n]
connecting = "מתחבר…"
`
	if err := os.WriteFile(filepath.Join(appDir, ConfigFileName), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}

	s, mux, err := startSupervisor(DevOptions{Dir: appDir, Target: "."})
	if err != nil {
		t.Fatalf("startSupervisor: %v", err)
	}
	defer s.shutdown()

	srv := httptest.NewServer(mux)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("get index: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	for _, want := range []string{
		`<html lang="he-IL" dir="rtl">`,
		"<title>Dev Shell</title>",
		`<meta name="robots" content="noindex">` + "\n</head>",
		"מתחבר…", // the [i18n] table used to be dropped on this path
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("dev shell missing %q:\n%s", want, html)
		}
	}
}

const devAppDocument = `package main

import (
	"net/http"

	sy "github.com/HazelnutParadise/syralit"
)

func main() {
	sy.Run(sy.Config{
		Title: "Child Title",
		DocumentFunc: func(r *http.Request) sy.Document {
			if q := r.URL.Query().Get("q"); q != "" {
				return sy.Document{Title: "Doc " + q, HeadHTML: "<meta name=\"from-child\">"}
			}
			return sy.Document{}
		},
	}, func() { sy.Title("V1") })
}
`

// TestDevDocumentFromChild is the dev half of issue #3: the supervisor owns
// the outward port, but only the child can run Config.DocumentFunc, so
// document requests go to the child while it is up and fall back to the
// supervisor's own shell when it is not.
func TestDevDocumentFromChild(t *testing.T) {
	if testing.Short() {
		t.Skip("shells out to `go build`; skipped in -short")
	}
	appDir := "_devdoc_app"
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(appDir)
	if err := os.WriteFile(filepath.Join(appDir, "main.go"), []byte(devAppDocument), 0o644); err != nil {
		t.Fatal(err)
	}

	s, mux, err := startSupervisor(DevOptions{Dir: appDir, Target: ".", Title: "Supervisor Title"})
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

	s.mu.Lock()
	up := s.childConn != nil
	s.mu.Unlock()
	if !up {
		t.Fatalf("child did not start; build error: %q", s.buildErr)
	}

	// Child is up: the document comes from it, DocumentFunc included.
	if code, body := get("/?q=hello"); code != 200 ||
		!strings.Contains(body, "<title>Doc hello</title>") ||
		!strings.Contains(body, `<meta name="from-child">`) {
		t.Fatalf("document not from child: %d\n%s", code, body)
	}
	if _, body := get("/"); !strings.Contains(body, "<title>Child Title</title>") {
		t.Fatalf("empty Document should fall back to the child's Config.Title:\n%s", body)
	}
	// The child also knows which page paths exist, so dev gets real 404s now.
	if code, _ := get("/no-such-page"); code != http.StatusNotFound {
		t.Fatalf("unknown page path via child returned %d, want 404", code)
	}

	// Child gone: the supervisor answers with its own shell so the browser can
	// still connect and show the build-error overlay.
	s.stopChild()
	if code, body := get("/?q=hello"); code != 200 ||
		!strings.Contains(body, "<title>Supervisor Title</title>") ||
		strings.Contains(body, "from-child") {
		t.Fatalf("fallback shell wrong: %d\n%s", code, body)
	}
}

const devAppReadsConfig = `package main

import sy "github.com/HazelnutParadise/syralit"

func main() {
	sy.App(func() { sy.Textf("title=%v", sy.GetOption("title")) })
}
`

// TestDevChildRunsInProjectDir pins the child's working directory: it must be
// the project directory, not wherever `syralit dev` happened to be launched
// from, or the child never finds the project's syralit.toml (and any
// os.DirFS("public") the app opens points at the wrong place).
func TestDevChildRunsInProjectDir(t *testing.T) {
	if testing.Short() {
		t.Skip("shells out to `go build`; skipped in -short")
	}
	appDir := "_devcwd_app"
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(appDir)
	if err := os.WriteFile(filepath.Join(appDir, "main.go"), []byte(devAppReadsConfig), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(appDir, ConfigFileName), []byte("title = \"From Project Toml\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s, mux, err := startSupervisor(DevOptions{Dir: appDir, Target: "."})
	if err != nil {
		t.Fatalf("startSupervisor: %v", err)
	}
	defer s.shutdown()
	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/_syralit/ws", nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	c.SetReadLimit(32 << 20)
	defer c.CloseNow()

	// The child resolves its Config from syralit.toml in its own cwd; only the
	// project dir has one, so the title proves where the child is running.
	readUntil(t, c, 10*time.Second, "title=From Project Toml")
}
