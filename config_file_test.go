package syralit

import (
	"io"
	"net/http"
	"net/http/httptest"
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
	html := renderIndex("My App", Theme{Mode: "dark", ThemeStyle: ThemeStyle{Accent: "#7C3AED", Radius: "12px"}})
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
		ThemeStyle: ThemeStyle{
			Font:               "serif",
			HeadingFont:        `"My Brand", sans-serif`,
			CodeFont:           "monospace",
			BaseFontSize:       18,
			BaseFontWeight:     350,
			HeadingFontSizes:   []string{"2.2rem", "1.6rem"},
			HeadingFontWeights: []int{800},
			CodeFontSize:       "0.875rem",
			CodeFontWeight:     500,
		},
		FontFaces: []FontFace{{
			Family: "My Brand", URL: "https://example.com/brand.woff2",
			Weight: "200 900", Style: "italic", UnicodeRange: "U+0-10FFFF",
		}},
		Sidebar: ThemeStyle{Font: "sans-serif"},
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
		ThemeStyle: ThemeStyle{Font: "sans}</style><script>alert(1)</script>"},
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

func boolPtr(b bool) *bool { return &b }

func TestRenderIndexFullTheme(t *testing.T) {
	html := renderIndex("Full", Theme{
		ThemeStyle: ThemeStyle{
			Accent:                         "#0F766E",
			Radius:                         "8px",
			ButtonRadius:                   "999px",
			BackgroundColor:                "#101418",
			SecondaryBackgroundColor:       "#1a1f26",
			TextColor:                      "#e6e8eb",
			LinkColor:                      "#38bdf8",
			LinkUnderline:                  boolPtr(true),
			CodeTextColor:                  "#a5f3fc",
			CodeBackgroundColor:            "#0b1020",
			BorderColor:                    "#334155",
			DataframeBorderColor:           "#475569",
			DataframeHeaderBackgroundColor: "#1e293b",
			ShowWidgetBorder:               boolPtr(false),
			RedColor:                       "#f87171",
			BlueBackgroundColor:            "#0c4a6e",
			GreenTextColor:                 "#bbf7d0",
		},
		ShowSidebarBorder:      boolPtr(false),
		ChartCategoricalColors: []string{"#111111", "#222222"},
		ChartSequentialColors:  []string{"#000000", "#ffffff"},
		Sidebar: ThemeStyle{
			Accent:          "#f59e0b",
			BackgroundColor: "#0a0a0a",
			RedColor:        "#ff0000",
			BaseFontSize:    15,
		},
	})
	for _, want := range []string{
		"--sy-accent:#0F766E", "--sy-radius:8px", "--sy-btn-radius:999px",
		"--sy-bg:#101418", "--sy-fg:#e6e8eb", "--sy-sidebar-bg:#1a1f26",
		"--sy-link:#38bdf8", "--sy-border:#334155",
		"--sy-df-border:#475569", "--sy-df-header-bg:#1e293b",
		"--sy-color-red:#f87171", "--sy-color-blue-bg:#0c4a6e", "--sy-color-green-text:#bbf7d0",
		".sy-link, .sy-markdown a{text-decoration:underline}",
		".sy-input, .sy-select, .sy-textarea{border-color:transparent}",
		"color:#a5f3fc;background:#0b1020}",
		"#syralit-root.has-sidebar #syralit-sidebar{border-right:none}",
		// Sidebar scope re-derives aliases so scoped overrides cascade.
		"--sy-primary:var(--sy-accent)",
		"--sy-color-red-bg:color-mix(in srgb, var(--sy-color-red) 10%, var(--sy-bg))",
		"--sy-error:var(--sy-color-red)",
		"background:#0a0a0a",
		"#syralit-sidebar{font-size:15px}",
		`window.__SY_THEME={"chart_categorical_colors":["#111111","#222222"],"chart_sequential_colors":["#000000","#ffffff"]}`,
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("rendered index missing %q:\n%s", want, html)
		}
	}
}

func TestFileConfigFullTheme(t *testing.T) {
	dir := t.TempDir()
	toml := `[theme]
background_color = "#101418"
secondary_background_color = "#1a1f26"
text_color = "#e6e8eb"
link_color = "#38bdf8"
link_underline = true
code_text_color = "#a5f3fc"
code_background_color = "#0b1020"
border_color = "#334155"
dataframe_border_color = "#475569"
dataframe_header_background_color = "#1e293b"
show_widget_border = false
show_sidebar_border = false
button_radius = "999px"
red_color = "#f87171"
blue_background_color = "#0c4a6e"
green_text_color = "#bbf7d0"
chart_categorical_colors = ["#111111", "#222222"]
chart_sequential_colors = ["#000000", "#ffffff"]
chart_diverging_colors = ["#ff0000", "#0000ff"]

[theme.sidebar]
accent = "#f59e0b"
background_color = "#0a0a0a"
red_color = "#ff0000"
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
	if th.BackgroundColor != "#101418" || th.SecondaryBackgroundColor != "#1a1f26" ||
		th.TextColor != "#e6e8eb" || th.LinkColor != "#38bdf8" {
		t.Fatalf("base colors not applied: %+v", th.ThemeStyle)
	}
	if th.LinkUnderline == nil || !*th.LinkUnderline ||
		th.ShowWidgetBorder == nil || *th.ShowWidgetBorder ||
		th.ShowSidebarBorder == nil || *th.ShowSidebarBorder {
		t.Fatalf("bool toggles not applied: %+v", th)
	}
	if th.CodeTextColor != "#a5f3fc" || th.CodeBackgroundColor != "#0b1020" ||
		th.BorderColor != "#334155" || th.DataframeBorderColor != "#475569" ||
		th.DataframeHeaderBackgroundColor != "#1e293b" || th.ButtonRadius != "999px" {
		t.Fatalf("borders/code/radius not applied: %+v", th.ThemeStyle)
	}
	if th.RedColor != "#f87171" || th.BlueBackgroundColor != "#0c4a6e" || th.GreenTextColor != "#bbf7d0" {
		t.Fatalf("palette not applied: %+v", th.ThemeStyle)
	}
	if len(th.ChartCategoricalColors) != 2 || len(th.ChartSequentialColors) != 2 || len(th.ChartDivergingColors) != 2 {
		t.Fatalf("chart colors not applied: %+v", th)
	}
	if th.Sidebar.Accent != "#f59e0b" || th.Sidebar.BackgroundColor != "#0a0a0a" || th.Sidebar.RedColor != "#ff0000" {
		t.Fatalf("sidebar not applied: %+v", th.Sidebar)
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
	html := renderIndex("X", Theme{ThemeStyle: ThemeStyle{Accent: "#fff}</style><script>alert(1)</script>"}})
	if strings.Contains(html, "<script>") {
		t.Fatalf("unsafe theme value leaked into output:\n%s", html)
	}
	if strings.Contains(html, "--sy-accent") {
		t.Fatalf("unsafe accent should have been rejected:\n%s", html)
	}
}

func TestRenderIndexShellDefaults(t *testing.T) {
	// An app that sets none of the shell options must render exactly what it
	// rendered before Lang/Dir/HeadHTML existed.
	defer restoreShell(saveShell())
	setShellConfig("", "", "")

	html := renderIndex("X", Theme{})
	if !strings.Contains(html, `<html lang="en">`) {
		t.Fatalf("default shell lost its lang attribute:\n%s", html)
	}
	if strings.Contains(html, " dir=") {
		t.Fatalf("unset Dir must not emit the attribute:\n%s", html)
	}
	if !strings.Contains(html, "runtime.css\">\n</head>") {
		t.Fatalf("default head gained unexpected markup:\n%s", html)
	}
}

func TestRenderIndexLangAndDir(t *testing.T) {
	defer restoreShell(saveShell())
	setShellConfig("ar-EG", "rtl", "")

	html := renderIndex("X", Theme{Mode: "dark"})
	if !strings.Contains(html, `<html lang="ar-EG" dir="rtl" data-theme="dark">`) {
		t.Fatalf("lang/dir/theme attributes wrong:\n%s", html)
	}
}

func TestRenderIndexRejectsUnsafeLangAndDir(t *testing.T) {
	defer restoreShell(saveShell())
	setShellConfig(`en" onload="alert(1)`, `rtl"><script>alert(2)</script>`, "")

	html := renderIndex("X", Theme{})
	if !strings.Contains(html, `<html lang="en">`) {
		t.Fatalf("invalid lang/dir should fall back to the default:\n%s", html)
	}
	if strings.Contains(html, "onload") || strings.Contains(html, "alert(") {
		t.Fatalf("unsafe attribute value leaked into output:\n%s", html)
	}

	// "sideways" is a real CSS writing direction but not a valid dir value.
	setShellConfig("en", "sideways", "")
	if html := renderIndex("X", Theme{}); strings.Contains(html, " dir=") {
		t.Fatalf("unknown dir value should have been dropped:\n%s", html)
	}
}

func TestRenderIndexHeadHTML(t *testing.T) {
	defer restoreShell(saveShell())
	head := `<meta name="description" content="A & B">` + "\n" + `<link rel="icon" href="/icon.svg">`
	setShellConfig("", "", head)

	html := renderIndex("X", Theme{ThemeStyle: ThemeStyle{Accent: "#0F766E"}})
	if !strings.Contains(html, head+"\n</head>") {
		t.Fatalf("head html not emitted verbatim at the end of <head>:\n%s", html)
	}
	// It must land after the theme block so an app can override the CSS vars.
	if strings.Index(html, "--sy-accent") > strings.Index(html, "rel=\"icon\"") {
		t.Fatalf("head html must follow the theme style block:\n%s", html)
	}
}

func TestShellConfigFromFile(t *testing.T) {
	fc := &fileConfig{
		Lang:     "zh-Hant-TW",
		Dir:      "ltr",
		HeadHTML: `<meta name="robots" content="noindex">`,
		I18n:     map[string]string{"connecting": "連線中…"},
	}

	var cfg Config
	fc.applyToConfig(&cfg)
	if cfg.Lang != "zh-Hant-TW" || cfg.Dir != "ltr" || cfg.HeadHTML != fc.HeadHTML {
		t.Fatalf("shell keys not applied to Config: %+v", cfg)
	}

	// Explicit code values win over the file.
	cfg2 := Config{Lang: "ja", Dir: "auto", HeadHTML: "<meta name=\"x\">"}
	fc.applyToConfig(&cfg2)
	if cfg2.Lang != "ja" || cfg2.Dir != "auto" || cfg2.HeadHTML != "<meta name=\"x\">" {
		t.Fatalf("file overrode explicit shell values: %+v", cfg2)
	}

	// The dev supervisor renders the same shell, so it needs the same values —
	// including the [i18n] table, which it used to drop.
	var dev DevOptions
	fc.applyToDev(&dev)
	if dev.Lang != "zh-Hant-TW" || dev.TextDir != "ltr" || dev.HeadHTML != fc.HeadHTML {
		t.Fatalf("shell keys not applied to DevOptions: %+v", dev)
	}
	if dev.UIStrings["connecting"] != "連線中…" {
		t.Fatalf("[i18n] not applied to DevOptions: %+v", dev.UIStrings)
	}
}

func saveShell() (string, string, string) { return docLang, docDir, headHTML }

func restoreShell(lang, dir, head string) { docLang, docDir, headHTML = lang, dir, head }

func TestHandlerServesShellConfig(t *testing.T) {
	// End to end: the values on Config must reach the very first HTML response,
	// which is the whole point of the feature (crawlers never open a socket).
	defer restoreShell(saveShell())

	h := Handler(Config{
		Title:    "My App",
		Lang:     "ar-EG",
		Dir:      "rtl",
		HeadHTML: `<meta name="description" content="hi">`,
	}, func() { Text("hi") })
	srv := httptest.NewServer(h)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("get index: %v", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	for _, want := range []string{
		`<html lang="ar-EG" dir="rtl">`,
		"<title>My App</title>",
		`<meta name="description" content="hi">` + "\n</head>",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("first response missing %q:\n%s", want, html)
		}
	}
}
