package syralit

import (
	"log"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// ConfigFileName is the conventional per-project settings file. It is optional;
// when present at the project root it fills in any value not set explicitly in
// code or via CLI flags (precedence: flag > syralit.toml > built-in default).
const ConfigFileName = "syralit.toml"

// Theme controls the front-end look (SPEC §17).
type Theme struct {
	Mode   string // "light" | "dark" | "system"
	Accent string // CSS color, e.g. "#7C3AED"
	Radius string // CSS length, e.g. "12px"

	// Font selects the base font for all text except code. The keywords
	// "sans-serif", "serif" and "monospace" pick the built-in Source Sans 3,
	// Source Serif 4 and Source Code Pro (embedded, served locally); any
	// other value is used verbatim as a CSS font-family list. Families not
	// installed on the visitor's machine can be loaded via FontFaces.
	Font        string
	HeadingFont string // font for titles/headers/markdown headings; defaults to Font
	CodeFont    string // font for code blocks and inline code

	BaseFontSize   int // root font size in px (0 = browser default, 16)
	BaseFontWeight int // body font weight (0 = default)

	// HeadingFontSizes / HeadingFontWeights style h1..h6 in order (and the
	// matching sy.Title / sy.Header / sy.Subheader widgets). Shorter slices
	// leave the remaining levels at their defaults.
	HeadingFontSizes   []string // CSS lengths, e.g. "2rem"
	HeadingFontWeights []int

	CodeFontSize   string // CSS length for code text, e.g. "0.875rem"
	CodeFontWeight int

	FontFaces []FontFace   // custom @font-face declarations
	Sidebar   SidebarTheme // sidebar-scoped font overrides
}

// FontFace declares a custom font to load (rendered as CSS @font-face), so
// Font / HeadingFont / CodeFont can reference families that are not installed
// on the visitor's machine. OTF, TTF, WOFF and WOFF2 sources work; URL may be
// an absolute URL or a path served by the app (e.g. from public/).
type FontFace struct {
	Family       string // font family name to register
	URL          string // font file location
	Weight       string // optional, e.g. "400" or a range "200 900"
	Style        string // optional: "normal" | "italic" | "oblique"
	UnicodeRange string // optional, e.g. "U+0000-00FF"
}

// SidebarTheme overrides fonts inside the sidebar only. Empty fields inherit
// the main theme.
type SidebarTheme struct {
	Font        string
	HeadingFont string
	CodeFont    string
}

// fileConfig mirrors syralit.toml.
type fileTheme struct {
	Mode               string   `toml:"mode"`
	Accent             string   `toml:"accent"`
	Radius             string   `toml:"radius"`
	Font               string   `toml:"font"`
	HeadingFont        string   `toml:"heading_font"`
	CodeFont           string   `toml:"code_font"`
	BaseFontSize       int      `toml:"base_font_size"`
	BaseFontWeight     int      `toml:"base_font_weight"`
	HeadingFontSizes   []string `toml:"heading_font_sizes"`
	HeadingFontWeights []int    `toml:"heading_font_weights"`
	CodeFontSize       string   `toml:"code_font_size"`
	CodeFontWeight     int      `toml:"code_font_weight"`
	FontFaces          []struct {
		Family       string `toml:"family"`
		URL          string `toml:"url"`
		Weight       string `toml:"weight"`
		Style        string `toml:"style"`
		UnicodeRange string `toml:"unicode_range"`
	} `toml:"font_faces"`
	Sidebar struct {
		Font        string `toml:"font"`
		HeadingFont string `toml:"heading_font"`
		CodeFont    string `toml:"code_font"`
	} `toml:"sidebar"`
}

type fileConfig struct {
	Title   string            `toml:"title"`
	Host    string            `toml:"host"`
	Port    int               `toml:"port"`
	Secrets map[string]string `toml:"secrets"`
	Theme   fileTheme         `toml:"theme"`
}

// loadFileConfig reads dir/syralit.toml if present. A missing file is not an
// error (returns nil); a malformed file is logged and ignored.
func loadFileConfig(dir string) *fileConfig {
	path := filepath.Join(dir, ConfigFileName)
	if _, err := os.Stat(path); err != nil {
		return nil
	}
	var fc fileConfig
	if _, err := toml.DecodeFile(path, &fc); err != nil {
		log.Printf("syralit: ignoring %s: %v", path, err)
		return nil
	}
	return &fc
}

var loadedSecrets map[string]string

func (fc *fileConfig) applyToConfig(cfg *Config) {
	if fc != nil && fc.Secrets != nil {
		loadedSecrets = fc.Secrets
	}
	if fc == nil {
		return
	}
	if cfg.Title == "" {
		cfg.Title = fc.Title
	}
	if cfg.Host == "" {
		cfg.Host = fc.Host
	}
	if cfg.Port == 0 {
		cfg.Port = fc.Port
	}
	fc.applyTheme(&cfg.Theme)
}

func (fc *fileConfig) applyToDev(o *DevOptions) {
	if fc == nil {
		return
	}
	if o.Title == "" {
		o.Title = fc.Title
	}
	if o.Host == "" {
		o.Host = fc.Host
	}
	if o.Port == 0 {
		o.Port = fc.Port
	}
	fc.applyTheme(&o.Theme)
}

func (fc *fileConfig) applyTheme(t *Theme) {
	setIfEmpty := func(dst *string, v string) {
		if *dst == "" {
			*dst = v
		}
	}
	setIfEmpty(&t.Mode, fc.Theme.Mode)
	setIfEmpty(&t.Accent, fc.Theme.Accent)
	setIfEmpty(&t.Radius, fc.Theme.Radius)
	setIfEmpty(&t.Font, fc.Theme.Font)
	setIfEmpty(&t.HeadingFont, fc.Theme.HeadingFont)
	setIfEmpty(&t.CodeFont, fc.Theme.CodeFont)
	setIfEmpty(&t.CodeFontSize, fc.Theme.CodeFontSize)
	setIfEmpty(&t.Sidebar.Font, fc.Theme.Sidebar.Font)
	setIfEmpty(&t.Sidebar.HeadingFont, fc.Theme.Sidebar.HeadingFont)
	setIfEmpty(&t.Sidebar.CodeFont, fc.Theme.Sidebar.CodeFont)
	if t.BaseFontSize == 0 {
		t.BaseFontSize = fc.Theme.BaseFontSize
	}
	if t.BaseFontWeight == 0 {
		t.BaseFontWeight = fc.Theme.BaseFontWeight
	}
	if t.CodeFontWeight == 0 {
		t.CodeFontWeight = fc.Theme.CodeFontWeight
	}
	if len(t.HeadingFontSizes) == 0 {
		t.HeadingFontSizes = fc.Theme.HeadingFontSizes
	}
	if len(t.HeadingFontWeights) == 0 {
		t.HeadingFontWeights = fc.Theme.HeadingFontWeights
	}
	if len(t.FontFaces) == 0 {
		for _, f := range fc.Theme.FontFaces {
			t.FontFaces = append(t.FontFaces, FontFace{
				Family: f.Family, URL: f.URL, Weight: f.Weight,
				Style: f.Style, UnicodeRange: f.UnicodeRange,
			})
		}
	}
}
