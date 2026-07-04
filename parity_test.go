package syralit

import (
	"encoding/base64"
	"net/http"
	"os"
	"path/filepath"
	"testing"
)

func TestDecodeUploadedFile(t *testing.T) {
	data := base64.StdEncoding.EncodeToString([]byte("hello"))
	f := decodeUploadedFile(map[string]any{
		"name": "a.txt", "size": float64(5), "type": "text/plain", "data": data,
	})
	if f == nil || f.Name != "a.txt" || string(f.Data) != "hello" || f.Size != 5 {
		t.Fatalf("decode failed: %+v", f)
	}
	if decodeUploadedFile("nope") != nil {
		t.Fatal("expected nil for non-map value")
	}
	if decodeUploadedFile(map[string]any{"name": "", "data": data}) != nil {
		t.Fatal("expected nil for empty name")
	}
	if decodeUploadedFile(map[string]any{"name": "x", "data": "!!not-base64!!"}) != nil {
		t.Fatal("expected nil for invalid base64")
	}
}

func TestChartSelection(t *testing.T) {
	at := NewAppTest(func() {
		if sel := LineChart(map[string][]float64{"A": {1, 2, 3}}, Selectable(), Key("chart")); sel != nil {
			Textf("%s[%d]=%v@%s", sel.Series, sel.Index, sel.Value, sel.X)
		}
	})
	at.Run()
	if len(at.Texts("text")) != 0 {
		t.Fatal("selection should be nil before any click")
	}
	n := at.FindAll("line_chart")
	if len(n) != 1 || n[0].ID != "chart" || n[0].Props["selectable"] != true {
		t.Fatalf("chart node not selectable: %+v", n)
	}
	at.SetValue("chart", map[string]any{"series": "A", "index": float64(2), "x": "3", "value": float64(3)})
	at.Run()
	if got := at.Texts("text"); len(got) != 1 || got[0] != "A[2]=3@3" {
		t.Fatalf("selection = %v", got)
	}
	// Selection persists across reruns (unlike button presses).
	at.Run()
	if len(at.Texts("text")) != 1 {
		t.Fatal("selection should persist")
	}
}

func TestSetQueryParam(t *testing.T) {
	at := NewAppTest(func() {
		SetQueryParam("tab", "sales")
		Text("q=" + QueryParam("tab"))
	})
	at.Run()
	if got := at.Texts("text"); len(got) != 1 || got[0] != "q=sales" {
		t.Fatalf("query param = %v", got)
	}
	at.sess.mu.Lock()
	dirty := at.sess.queryDirty
	at.sess.mu.Unlock()
	if !dirty {
		t.Fatal("queryDirty should be set for the client URL update")
	}
}

func TestRequestBasePath(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.RequestURI = "/dash/"
	r.URL.Path = "/"
	if got := requestBasePath(r); got != "/dash" {
		t.Fatalf("base = %q, want /dash", got)
	}
	r.RequestURI = "/"
	if got := requestBasePath(r); got != "" {
		t.Fatalf("root base = %q, want empty", got)
	}
}

func TestSSLConfig(t *testing.T) {
	dir := t.TempDir()
	toml := "[server]\nssl_cert_file = \"cert.pem\"\nssl_key_file = \"key.pem\"\n"
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg := Config{}
	loadFileConfig(dir).applyToConfig(&cfg)
	if cfg.SSLCertFile != "cert.pem" || cfg.SSLKeyFile != "key.pem" {
		t.Fatalf("ssl config not applied: %+v", cfg)
	}
}

func TestMaxUploadSizeConfig(t *testing.T) {
	dir := t.TempDir()
	toml := "[server]\nmax_upload_size_mb = 50\n"
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	fc := loadFileConfig(dir)
	cfg := Config{}
	fc.applyToConfig(&cfg)
	if cfg.MaxUploadSizeMB != 50 {
		t.Fatalf("max_upload_size_mb not applied: %+v", cfg)
	}
	if got := cfg.uploadLimit(); got != 50<<20 {
		t.Fatalf("uploadLimit = %d, want %d", got, 50<<20)
	}
	if got := (&Config{}).uploadLimit(); got != 10<<20 {
		t.Fatalf("default uploadLimit = %d, want %d", got, 10<<20)
	}
}
