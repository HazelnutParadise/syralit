package syralit

import (
	"context"
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

func main() {
	sy.App(func() {
		sy.Title("V1")
		c := sy.State("count", 0)
		if sy.Button("Add", sy.Key("add")) {
			c.Set(c.Get() + 1)
		}
		sy.Textf("Count: %d", c.Get())
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

	// 2. Click Add twice -> count 2.
	addClick := `{"type":"widget_change","widget_id":"add","value":true,"is_button":true}`
	sendWS(t, c, addClick)
	readUntil(t, c, 10*time.Second, "Count: 1")
	sendWS(t, c, addClick)
	readUntil(t, c, 10*time.Second, "Count: 2")

	// 3. Edit source -> rebuild -> V2 appears AND count is preserved at 2
	//    (both must be in the same frame: the post-restore render).
	writeApp(strings.Replace(devAppV1, `"V1"`, `"V2"`, 1))
	readUntil(t, c, 30*time.Second, "V2", "Count: 2")

	// 4. Introduce a compile error -> build_error overlay, old app keeps running.
	writeApp(devAppV1 + "\nfunc broken() { this is not go }\n")
	readUntil(t, c, 30*time.Second, "__dev_build_error")

	// 5. Fix it (V3) -> recovers, state still preserved at 2.
	writeApp(strings.Replace(devAppV1, `"V1"`, `"V3"`, 1))
	readUntil(t, c, 30*time.Second, "V3", "Count: 2")
}
