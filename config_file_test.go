package syralit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFileConfigPrecedence(t *testing.T) {
	dir := t.TempDir()
	toml := `title = "From File"
host = "0.0.0.0"
port = 9000

[theme]
mode = "dark"
accent = "#FF0000"
radius = "8px"
`
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}

	fc := loadFileConfig(dir)
	if fc == nil {
		t.Fatal("expected config, got nil")
	}

	// Unset fields are filled from the file.
	cfg := Config{}
	fc.applyToConfig(&cfg)
	if cfg.Title != "From File" || cfg.Port != 9000 || cfg.Host != "0.0.0.0" {
		t.Fatalf("file values not applied: %+v", cfg)
	}
	if cfg.Theme.Mode != "dark" || cfg.Theme.Accent != "#FF0000" || cfg.Theme.Radius != "8px" {
		t.Fatalf("theme not applied: %+v", cfg.Theme)
	}

	// Explicit code values win over the file.
	cfg2 := Config{Title: "Explicit", Port: 1234}
	fc.applyToConfig(&cfg2)
	if cfg2.Title != "Explicit" || cfg2.Port != 1234 {
		t.Fatalf("explicit values overridden by file: %+v", cfg2)
	}
	if cfg2.Host != "0.0.0.0" { // unset field still filled
		t.Fatalf("expected host from file, got %q", cfg2.Host)
	}
}

func TestLoadFileConfigMissing(t *testing.T) {
	if fc := loadFileConfig(t.TempDir()); fc != nil {
		t.Fatalf("expected nil for missing config, got %+v", fc)
	}
}

func TestRenderIndexTheme(t *testing.T) {
	html := renderIndex("My App", Theme{Mode: "dark", Accent: "#7C3AED", Radius: "12px"})
	for _, want := range []string{
		"<title>My App</title>",
		`data-theme="dark"`,
		"--sy-accent:#7C3AED",
		"--sy-radius:12px",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("rendered index missing %q:\n%s", want, html)
		}
	}
}

func TestRenderIndexRejectsUnsafeTheme(t *testing.T) {
	// A value trying to break out of the CSS context must be dropped.
	html := renderIndex("X", Theme{Accent: "#fff}</style><script>alert(1)</script>"})
	if strings.Contains(html, "<script>") {
		t.Fatalf("unsafe theme value leaked into output:\n%s", html)
	}
	if strings.Contains(html, "--sy-accent") {
		t.Fatalf("unsafe accent should have been rejected:\n%s", html)
	}
}
