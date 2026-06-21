package syralit

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

// resetStatic clears the global static registries so a test starts clean and
// doesn't leak registrations into others.
func resetStatic() {
	staticMu.Lock()
	rootStatics, assetOverlays = nil, nil
	staticMu.Unlock()
}

func TestStaticServing(t *testing.T) {
	resetStatic()
	defer resetStatic()

	Static(fstest.MapFS{"img/logo.svg": {Data: []byte("<svg></svg>")}})
	StaticAssets(fstest.MapFS{"runtime.css": {Data: []byte("/* custom */")}})

	srv := httptest.NewServer((&server{cfg: Config{Title: "t"}, appFn: func() {}}).handler())
	defer srv.Close()

	get := func(path string) (int, string) {
		res, err := http.Get(srv.URL + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return res.StatusCode, string(b)
	}

	// public/ file served at the site root.
	if code, body := get("/img/logo.svg"); code != 200 || body != "<svg></svg>" {
		t.Fatalf("public file: got %d %q", code, body)
	}

	// user asset override shadows the framework copy.
	if code, body := get("/_syralit/assets/runtime.css"); code != 200 || body != "/* custom */" {
		t.Fatalf("asset override: got %d %q", code, body)
	}

	// a framework asset with no override is still served from the embed.
	if code, body := get("/_syralit/assets/runtime.js"); code != 200 || len(body) == 0 {
		t.Fatalf("framework asset: got %d len=%d", code, len(body))
	}

	// unknown path falls through to 404.
	if code, _ := get("/does-not-exist"); code != 404 {
		t.Fatalf("missing file: expected 404, got %d", code)
	}

	// "/" still renders the app shell, not a static file.
	if code, body := get("/"); code != 200 || !strings.Contains(body, "syralit-root") {
		t.Fatalf("app shell: got %d, contains-root=%v", code, strings.Contains(body, "syralit-root"))
	}
}

func TestSetAssetURL(t *testing.T) {
	SetAssetURL("chartjs", "/vendor/chart.umd.min.js")
	defer func() {
		assetMu.Lock()
		delete(assetOverrides, "chartjs")
		assetMu.Unlock()
	}()

	srv := httptest.NewServer((&server{cfg: Config{Title: "t"}, appFn: func() {}}).handler())
	defer srv.Close()

	res, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	s := string(body)
	if !strings.Contains(s, "window.__SY_ASSETS") || !strings.Contains(s, "/vendor/chart.umd.min.js") {
		t.Fatalf("asset override not injected into index: %s", s)
	}
}

func TestStaticPathTraversalRejected(t *testing.T) {
	resetStatic()
	defer resetStatic()

	Static(fstest.MapFS{"ok.txt": {Data: []byte("ok")}})

	srv := httptest.NewServer((&server{cfg: Config{}, appFn: func() {}}).handler())
	defer srv.Close()

	// A traversal attempt must not escape the static FS.
	res, err := http.Get(srv.URL + "/../server.go")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode == 200 {
		b, _ := io.ReadAll(res.Body)
		if strings.Contains(string(b), "package syralit") {
			t.Fatal("path traversal served a source file")
		}
	}
}
