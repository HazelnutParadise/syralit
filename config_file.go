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

// Theme controls the front-end look (SPEC §17). Visual fields live in the
// embedded ThemeStyle (so they are addressed as Theme.Accent etc.); the same
// set can be overridden for the sidebar only via Theme.Sidebar.
type Theme struct {
	Mode string // "light" | "dark" | "system"

	ThemeStyle // main-area styles (also the defaults the sidebar inherits)

	// ShowSidebarBorder toggles the border between sidebar and main area.
	// nil keeps the default (shown).
	ShowSidebarBorder *bool

	// Chart color palettes. Categorical drives the default series colors of
	// the built-in Chart.js charts; sequential/diverging are published to the
	// front end (window.__SY_THEME) for custom components and future charts.
	ChartCategoricalColors []string
	ChartSequentialColors  []string
	ChartDivergingColors   []string

	FontFaces []FontFace // custom @font-face declarations
	Sidebar   ThemeStyle // sidebar-scoped overrides (empty fields inherit)
}

// ThemeStyle is the set of visual options that exist for both the main area
// and (independently) the sidebar. All fields are optional; empty means "keep
// the built-in default".
type ThemeStyle struct {
	Accent       string // primary/accent color, e.g. "#7C3AED"
	Radius       string // base corner radius, e.g. "12px"
	ButtonRadius string // button corner radius; defaults to Radius

	BackgroundColor          string // app background
	SecondaryBackgroundColor string // widget/code-block/sidebar surface color
	TextColor                string // main text color
	LinkColor                string // link color; defaults to Accent
	LinkUnderline            *bool  // true: always underline links; false: never; nil: underline on hover

	CodeTextColor       string // text color for code blocks / inline code
	CodeBackgroundColor string // background for code blocks / inline code

	BorderColor                    string // general element/widget border color
	DataframeBorderColor           string // table/dataframe borders; defaults to BorderColor
	DataframeHeaderBackgroundColor string // table/dataframe header background
	ShowWidgetBorder               *bool  // false: hide input widget borders; nil/true: show

	// Basic color palette. The base color is used by badges and status
	// elements, the *BackgroundColor tint by alert surfaces, the *TextColor
	// shade by text on those surfaces (each defaults sensibly when unset).
	RedColor, OrangeColor, YellowColor, BlueColor, GreenColor, VioletColor, GrayColor                                                                       string
	RedBackgroundColor, OrangeBackgroundColor, YellowBackgroundColor, BlueBackgroundColor, GreenBackgroundColor, VioletBackgroundColor, GrayBackgroundColor string
	RedTextColor, OrangeTextColor, YellowTextColor, BlueTextColor, GreenTextColor, VioletTextColor, GrayTextColor                                           string

	// Font selects the base font for all text except code. The keywords
	// "sans-serif", "serif" and "monospace" pick the built-in Source Sans 3,
	// Source Serif 4 and Source Code Pro (embedded, served locally); any
	// other value is used verbatim as a CSS font-family list. Families not
	// installed on the visitor's machine can be loaded via Theme.FontFaces.
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

// fileThemeStyle mirrors the ThemeStyle fields in syralit.toml (used both at
// [theme] level and under [theme.sidebar]).
type fileThemeStyle struct {
	Accent       string `toml:"accent"`
	Radius       string `toml:"radius"`
	ButtonRadius string `toml:"button_radius"`

	BackgroundColor          string `toml:"background_color"`
	SecondaryBackgroundColor string `toml:"secondary_background_color"`
	TextColor                string `toml:"text_color"`
	LinkColor                string `toml:"link_color"`
	LinkUnderline            *bool  `toml:"link_underline"`

	CodeTextColor       string `toml:"code_text_color"`
	CodeBackgroundColor string `toml:"code_background_color"`

	BorderColor                    string `toml:"border_color"`
	DataframeBorderColor           string `toml:"dataframe_border_color"`
	DataframeHeaderBackgroundColor string `toml:"dataframe_header_background_color"`
	ShowWidgetBorder               *bool  `toml:"show_widget_border"`

	RedColor    string `toml:"red_color"`
	OrangeColor string `toml:"orange_color"`
	YellowColor string `toml:"yellow_color"`
	BlueColor   string `toml:"blue_color"`
	GreenColor  string `toml:"green_color"`
	VioletColor string `toml:"violet_color"`
	GrayColor   string `toml:"gray_color"`

	RedBackgroundColor    string `toml:"red_background_color"`
	OrangeBackgroundColor string `toml:"orange_background_color"`
	YellowBackgroundColor string `toml:"yellow_background_color"`
	BlueBackgroundColor   string `toml:"blue_background_color"`
	GreenBackgroundColor  string `toml:"green_background_color"`
	VioletBackgroundColor string `toml:"violet_background_color"`
	GrayBackgroundColor   string `toml:"gray_background_color"`

	RedTextColor    string `toml:"red_text_color"`
	OrangeTextColor string `toml:"orange_text_color"`
	YellowTextColor string `toml:"yellow_text_color"`
	BlueTextColor   string `toml:"blue_text_color"`
	GreenTextColor  string `toml:"green_text_color"`
	VioletTextColor string `toml:"violet_text_color"`
	GrayTextColor   string `toml:"gray_text_color"`

	Font               string   `toml:"font"`
	HeadingFont        string   `toml:"heading_font"`
	CodeFont           string   `toml:"code_font"`
	BaseFontSize       int      `toml:"base_font_size"`
	BaseFontWeight     int      `toml:"base_font_weight"`
	HeadingFontSizes   []string `toml:"heading_font_sizes"`
	HeadingFontWeights []int    `toml:"heading_font_weights"`
	CodeFontSize       string   `toml:"code_font_size"`
	CodeFontWeight     int      `toml:"code_font_weight"`
}

// fileConfig mirrors syralit.toml.
type fileTheme struct {
	fileThemeStyle

	Mode              string `toml:"mode"`
	ShowSidebarBorder *bool  `toml:"show_sidebar_border"`

	ChartCategoricalColors []string `toml:"chart_categorical_colors"`
	ChartSequentialColors  []string `toml:"chart_sequential_colors"`
	ChartDivergingColors   []string `toml:"chart_diverging_colors"`

	FontFaces []struct {
		Family       string `toml:"family"`
		URL          string `toml:"url"`
		Weight       string `toml:"weight"`
		Style        string `toml:"style"`
		UnicodeRange string `toml:"unicode_range"`
	} `toml:"font_faces"`
	Sidebar fileThemeStyle `toml:"sidebar"`
}

type fileConfig struct {
	Title   string            `toml:"title"`
	Host    string            `toml:"host"`
	Port    int               `toml:"port"`
	Secrets map[string]string `toml:"secrets"`
	Theme   fileTheme         `toml:"theme"`
	Server  struct {
		MaxUploadSizeMB int    `toml:"max_upload_size_mb"`
		SSLCertFile     string `toml:"ssl_cert_file"`
		SSLKeyFile      string `toml:"ssl_key_file"`
	} `toml:"server"`
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
	if cfg.MaxUploadSizeMB == 0 {
		cfg.MaxUploadSizeMB = fc.Server.MaxUploadSizeMB
	}
	if cfg.SSLCertFile == "" {
		cfg.SSLCertFile = fc.Server.SSLCertFile
	}
	if cfg.SSLKeyFile == "" {
		cfg.SSLKeyFile = fc.Server.SSLKeyFile
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

func setIfEmpty(dst *string, v string) {
	if *dst == "" {
		*dst = v
	}
}

func setIfZero(dst *int, v int) {
	if *dst == 0 {
		*dst = v
	}
}

func setIfNil(dst **bool, v *bool) {
	if *dst == nil {
		*dst = v
	}
}

// applyStyle fills every unset field of dst from src.
func applyStyle(dst *ThemeStyle, src *fileThemeStyle) {
	strs := [][2]*string{
		{&dst.Accent, &src.Accent}, {&dst.Radius, &src.Radius}, {&dst.ButtonRadius, &src.ButtonRadius},
		{&dst.BackgroundColor, &src.BackgroundColor}, {&dst.SecondaryBackgroundColor, &src.SecondaryBackgroundColor},
		{&dst.TextColor, &src.TextColor}, {&dst.LinkColor, &src.LinkColor},
		{&dst.CodeTextColor, &src.CodeTextColor}, {&dst.CodeBackgroundColor, &src.CodeBackgroundColor},
		{&dst.BorderColor, &src.BorderColor}, {&dst.DataframeBorderColor, &src.DataframeBorderColor},
		{&dst.DataframeHeaderBackgroundColor, &src.DataframeHeaderBackgroundColor},
		{&dst.RedColor, &src.RedColor}, {&dst.OrangeColor, &src.OrangeColor}, {&dst.YellowColor, &src.YellowColor},
		{&dst.BlueColor, &src.BlueColor}, {&dst.GreenColor, &src.GreenColor}, {&dst.VioletColor, &src.VioletColor},
		{&dst.GrayColor, &src.GrayColor},
		{&dst.RedBackgroundColor, &src.RedBackgroundColor}, {&dst.OrangeBackgroundColor, &src.OrangeBackgroundColor},
		{&dst.YellowBackgroundColor, &src.YellowBackgroundColor}, {&dst.BlueBackgroundColor, &src.BlueBackgroundColor},
		{&dst.GreenBackgroundColor, &src.GreenBackgroundColor}, {&dst.VioletBackgroundColor, &src.VioletBackgroundColor},
		{&dst.GrayBackgroundColor, &src.GrayBackgroundColor},
		{&dst.RedTextColor, &src.RedTextColor}, {&dst.OrangeTextColor, &src.OrangeTextColor},
		{&dst.YellowTextColor, &src.YellowTextColor}, {&dst.BlueTextColor, &src.BlueTextColor},
		{&dst.GreenTextColor, &src.GreenTextColor}, {&dst.VioletTextColor, &src.VioletTextColor},
		{&dst.GrayTextColor, &src.GrayTextColor},
		{&dst.Font, &src.Font}, {&dst.HeadingFont, &src.HeadingFont}, {&dst.CodeFont, &src.CodeFont},
		{&dst.CodeFontSize, &src.CodeFontSize},
	}
	for _, p := range strs {
		setIfEmpty(p[0], *p[1])
	}
	setIfZero(&dst.BaseFontSize, src.BaseFontSize)
	setIfZero(&dst.BaseFontWeight, src.BaseFontWeight)
	setIfZero(&dst.CodeFontWeight, src.CodeFontWeight)
	setIfNil(&dst.LinkUnderline, src.LinkUnderline)
	setIfNil(&dst.ShowWidgetBorder, src.ShowWidgetBorder)
	if len(dst.HeadingFontSizes) == 0 {
		dst.HeadingFontSizes = src.HeadingFontSizes
	}
	if len(dst.HeadingFontWeights) == 0 {
		dst.HeadingFontWeights = src.HeadingFontWeights
	}
}

func (fc *fileConfig) applyTheme(t *Theme) {
	setIfEmpty(&t.Mode, fc.Theme.Mode)
	setIfNil(&t.ShowSidebarBorder, fc.Theme.ShowSidebarBorder)
	applyStyle(&t.ThemeStyle, &fc.Theme.fileThemeStyle)
	applyStyle(&t.Sidebar, &fc.Theme.Sidebar)
	if len(t.ChartCategoricalColors) == 0 {
		t.ChartCategoricalColors = fc.Theme.ChartCategoricalColors
	}
	if len(t.ChartSequentialColors) == 0 {
		t.ChartSequentialColors = fc.Theme.ChartSequentialColors
	}
	if len(t.ChartDivergingColors) == 0 {
		t.ChartDivergingColors = fc.Theme.ChartDivergingColors
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
