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

func TestRenderIndexFontTheme(t *testing.T) {
	html := renderIndex("Fonts", Theme{
		Font:               "serif",
		HeadingFont:        `"My Brand", sans-serif`,
		CodeFont:           "monospace",
		BaseFontSize:       18,
		BaseFontWeight:     350,
		HeadingFontSizes:   []string{"2.2rem", "1.6rem"},
		HeadingFontWeights: []int{800},
		CodeFontSize:       "0.875rem",
		CodeFontWeight:     500,
		FontFaces: []FontFace{{
			Family: "My Brand", URL: "https://example.com/brand.woff2",
			Weight: "200 900", Style: "italic", UnicodeRange: "U+0-10FFFF",
		}},
		Sidebar: SidebarTheme{Font: "sans-serif"},
	})
	for _, want := range []string{
		`--sy-font:"Source Serif 4"`,
		`--sy-font-heading:"My Brand", sans-serif`,
		`--sy-font-code:"Source Code Pro"`,
		"html{font-size:18px}",
		"body{font-weight:350}",
		".sy-title, .sy-markdown h1{font-size:2.2rem;font-weight:800}",
		".sy-header, .sy-markdown h2{font-size:1.6rem}",
		"font-size:0.875rem;font-weight:500}",
		`@font-face{font-family:"My Brand";src:url("https://example.com/brand.woff2");font-weight:200 900;font-style:italic;unicode-range:U+0-10FFFF;font-display:swap}`,
		`#syralit-sidebar{--sy-font:"Source Sans 3"`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("rendered index missing %q:\n%s", want, html)
		}
	}
}

func TestRenderIndexRejectsUnsafeFontTheme(t *testing.T) {
	html := renderIndex("X", Theme{
		Font: "sans}</style><script>alert(1)</script>",
		FontFaces: []FontFace{{
			Family: "Evil",
			URL:    `x") format("woff2"); } </style><script>alert(2)</script>`,
		}},
	})
	if strings.Contains(html, "<script>") {
		t.Fatalf("unsafe font value leaked into output:\n%s", html)
	}
	if strings.Contains(html, "--sy-font:") || strings.Contains(html, "@font-face") {
		t.Fatalf("unsafe font theme should have been rejected:\n%s", html)
	}
}

func TestFileConfigFontTheme(t *testing.T) {
	dir := t.TempDir()
	toml := `[theme]
font = "serif"
heading_font = "Inter, sans-serif"
code_font = "monospace"
base_font_size = 18
base_font_weight = 350
heading_font_sizes = ["2rem", "1.5rem"]
heading_font_weights = [700, 600]
code_font_size = "0.875rem"
code_font_weight = 500

[[theme.font_faces]]
family = "Inter"
url = "/fonts/inter.woff2"
weight = "100 900"
style = "normal"
unicode_range = "U+0-10FFFF"

[theme.sidebar]
font = "sans-serif"
`
	if err := os.WriteFile(filepath.Join(dir, ConfigFileName), []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	fc := loadFileConfig(dir)
	if fc == nil {
		t.Fatal("expected config, got nil")
	}
	cfg := Config{}
	fc.applyToConfig(&cfg)
	th := cfg.Theme
	if th.Font != "serif" || th.HeadingFont != "Inter, sans-serif" || th.CodeFont != "monospace" {
		t.Fatalf("fonts not applied: %+v", th)
	}
	if th.BaseFontSize != 18 || th.BaseFontWeight != 350 || th.CodeFontSize != "0.875rem" || th.CodeFontWeight != 500 {
		t.Fatalf("sizes/weights not applied: %+v", th)
	}
	if len(th.HeadingFontSizes) != 2 || th.HeadingFontSizes[0] != "2rem" || len(th.HeadingFontWeights) != 2 {
		t.Fatalf("heading sizes/weights not applied: %+v", th)
	}
	if len(th.FontFaces) != 1 || th.FontFaces[0].Family != "Inter" || th.FontFaces[0].URL != "/fonts/inter.woff2" ||
		th.FontFaces[0].Weight != "100 900" || th.FontFaces[0].UnicodeRange != "U+0-10FFFF" {
		t.Fatalf("font_faces not applied: %+v", th.FontFaces)
	}
	if th.Sidebar.Font != "sans-serif" {
		t.Fatalf("sidebar font not applied: %+v", th.Sidebar)
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
