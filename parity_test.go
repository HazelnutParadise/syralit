package syralit

import (
	"encoding/base64"
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
