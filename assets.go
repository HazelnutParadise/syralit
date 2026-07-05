package syralit

import (
	"embed"
	"encoding/json"
	"fmt"
	"mime"
	"strings"
	"sync"
)

// Font MIME types are not registered on every platform (notably Windows);
// register them so embedded and user-served font files get a correct
// Content-Type.
func init() {
	mime.AddExtensionType(".woff2", "font/woff2")
	mime.AddExtensionType(".woff", "font/woff")
	mime.AddExtensionType(".ttf", "font/ttf")
	mime.AddExtensionType(".otf", "font/otf")
}

//go:embed assets
var assetsFS embed.FS

// Third-party library URLs (Chart.js, Leaflet, KaTeX, Plotly, …) can be
// repointed so they load from a self-hosted copy instead of a CDN — the key to
// fully offline / air-gapped / strict-CSP deployments. Drop the files in your
// public/ dir, point the names at them, and `syralit build` bakes everything
// into one binary that needs no internet.
var (
	assetMu        sync.Mutex
	assetOverrides = map[string]string{}
)

// SetAssetURL overrides where a named front-end library is loaded from. Names:
// chartjs, leaflet_js, leaflet_css, katex_js, katex_css, highlight_js,
// highlight_css, highlight_css_dark, viz, vega, vega_lite, vega_embed, plotly,
// bokeh, deckgl, mapbox_js, mapbox_css.
//
//	sy.SetAssetURL("chartjs", "/chart.umd.min.js") // served from public/
func SetAssetURL(name, url string) {
	assetMu.Lock()
	assetOverrides[name] = url
	assetMu.Unlock()
}

// assetOverridesScript renders the override map as an inline script so the
// runtime can resolve library URLs against it.
func assetOverridesScript() string {
	assetMu.Lock()
	defer assetMu.Unlock()
	if len(assetOverrides) == 0 {
		return ""
	}
	b, err := json.Marshal(assetOverrides)
	if err != nil {
		return ""
	}
	return "\n<script>window.__SY_ASSETS=" + string(b) + ";</script>"
}

// indexHTML is the app shell. The two %s slots are the page title and an optional
// inline <style> block carrying theme overrides. The client runtime is loaded
// from /_syralit/assets and opens the WebSocket back to the server.
const indexHTML = `<!doctype html>
<html lang="en"%[1]s>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%[2]s</title>
<link rel="stylesheet" href="%[4]s/_syralit/assets/runtime.css">%[3]s
</head>
<body>
<div id="syralit-root">
<nav id="syralit-sidebar"></nav>
<main id="syralit-app"><div class="sy-loading"><div class="sy-loading-spinner"></div><p>%[6]s</p></div></main>
</div>
%[5]s<script src="%[4]s/_syralit/assets/runtime.js"></script>
</body>
</html>`

// uiStrings holds the [i18n] overrides for built-in UI text; nil means the
// English defaults. Published to the front end as window.__SY_I18N.
var uiStrings map[string]string

// Built-in font stacks selected by the "sans-serif" / "serif" / "monospace"
// theme keywords. The named families are embedded (assets/fonts) and declared
// via @font-face in runtime.css; the rest of each stack is the fallback chain.
var builtinFontStacks = map[string]string{
	"sans-serif": `"Source Sans 3", ui-sans-serif, system-ui, -apple-system, "Segoe UI", Roboto, "Noto Sans TC", sans-serif`,
	"serif":      `"Source Serif 4", Georgia, "Times New Roman", "Noto Serif TC", serif`,
	"monospace":  `"Source Code Pro", ui-monospace, "Cascadia Code", "Consolas", monospace`,
}

// codeFontSelector lists every place runtime.css renders code-like text, so
// CodeFontSize / CodeFontWeight can be applied with a single injected rule.
const codeFontSelector = ".sy-code, .sy-code-gutter, .sy-json, .sy-color-hex, .sy-exception-body, .sy-markdown pre, .sy-markdown code"

// headingSelectors maps heading level 1..6 to the CSS selectors it styles
// (widget classes double as markdown headings).
var headingSelectors = [6]string{
	".sy-title, .sy-markdown h1",
	".sy-header, .sy-markdown h2",
	".sy-subheader, .sy-markdown h3",
	".sy-markdown h4",
	".sy-markdown h5",
	".sy-markdown h6",
}

// resolveFont maps the theme keywords onto the built-in stacks and validates
// anything else as a literal font-family list. Returns "" when unusable.
func resolveFont(v string) string {
	if stack, ok := builtinFontStacks[strings.ToLower(strings.TrimSpace(v))]; ok {
		return stack
	}
	if fontValueSafe(v) {
		return v
	}
	return ""
}

// styleVars renders a ThemeStyle's variable-backed options as CSS custom
// property declarations. sidebar=true additionally re-emits derived variables
// (aliases and color-mix tints normally computed at :root) so scoped
// overrides actually cascade — custom properties substitute var() where they
// are declared, so a sidebar-scoped base color would otherwise never reach an
// alias declared at :root.
func styleVars(s *ThemeStyle, sidebar bool) []string {
	var vars []string
	add := func(name, val string) {
		if cssValueSafe(val) {
			vars = append(vars, name+":"+val)
		}
	}
	add("--sy-accent", s.Accent)
	add("--sy-radius", s.Radius)
	add("--sy-btn-radius", s.ButtonRadius)
	add("--sy-bg", s.BackgroundColor)
	add("--sy-fg", s.TextColor)
	add("--sy-sidebar-bg", s.SecondaryBackgroundColor)
	add("--sy-link", s.LinkColor)
	add("--sy-border", s.BorderColor)
	add("--sy-df-border", s.DataframeBorderColor)
	add("--sy-df-header-bg", s.DataframeHeaderBackgroundColor)
	if sidebar && s.Accent != "" && cssValueSafe(s.Accent) {
		vars = append(vars, "--sy-primary:var(--sy-accent)",
			"--sy-primary-alpha:color-mix(in srgb, var(--sy-accent) 12%, transparent)")
		if s.LinkColor == "" {
			vars = append(vars, "--sy-link:var(--sy-accent)")
		}
	}

	statusAlias := map[string]string{"blue": "--sy-info", "green": "--sy-success", "orange": "--sy-warning", "red": "--sy-error"}
	palette := []struct{ name, base, bg, text string }{
		{"red", s.RedColor, s.RedBackgroundColor, s.RedTextColor},
		{"orange", s.OrangeColor, s.OrangeBackgroundColor, s.OrangeTextColor},
		{"yellow", s.YellowColor, s.YellowBackgroundColor, s.YellowTextColor},
		{"blue", s.BlueColor, s.BlueBackgroundColor, s.BlueTextColor},
		{"green", s.GreenColor, s.GreenBackgroundColor, s.GreenTextColor},
		{"violet", s.VioletColor, s.VioletBackgroundColor, s.VioletTextColor},
		{"gray", s.GrayColor, s.GrayBackgroundColor, s.GrayTextColor},
	}
	for _, c := range palette {
		v := "--sy-color-" + c.name
		add(v, c.base)
		add(v+"-bg", c.bg)
		add(v+"-text", c.text)
		if sidebar && c.base != "" && cssValueSafe(c.base) {
			// Re-derive the tints/aliases at sidebar scope.
			if c.bg == "" {
				vars = append(vars, v+"-bg:color-mix(in srgb, var("+v+") 10%, var(--sy-bg))")
			}
			if c.text == "" {
				vars = append(vars, v+"-text:var("+v+")")
			}
			if alias, ok := statusAlias[c.name]; ok {
				vars = append(vars, alias+":var("+v+")")
			}
		}
	}

	if f := resolveFont(s.Font); s.Font != "" && f != "" {
		vars = append(vars, "--sy-font:"+f)
		if sidebar && s.HeadingFont == "" {
			vars = append(vars, "--sy-font-heading:"+f)
		}
	}
	if f := resolveFont(s.HeadingFont); s.HeadingFont != "" && f != "" {
		vars = append(vars, "--sy-font-heading:"+f)
	}
	if f := resolveFont(s.CodeFont); s.CodeFont != "" && f != "" {
		vars = append(vars, "--sy-font-code:"+f)
	}
	return vars
}

// prefixSelector scopes a comma-separated selector list under prefix.
func prefixSelector(sel, prefix string) string {
	if prefix == "" {
		return sel
	}
	parts := strings.Split(sel, ", ")
	for i, p := range parts {
		parts[i] = prefix + " " + p
	}
	return strings.Join(parts, ", ")
}

// styleRules renders the ThemeStyle options that need whole CSS rules rather
// than variables. prefix is "" for the main area or a scoping selector.
func styleRules(css *strings.Builder, s *ThemeStyle, prefix string) {
	if s.BaseFontSize > 0 {
		if prefix == "" {
			fmt.Fprintf(css, "html{font-size:%dpx}", s.BaseFontSize)
		} else {
			fmt.Fprintf(css, "%s{font-size:%dpx}", prefix, s.BaseFontSize)
		}
	}
	if fontWeightValid(s.BaseFontWeight) {
		sel := prefix
		if sel == "" {
			sel = "body"
		}
		fmt.Fprintf(css, "%s{font-weight:%d}", sel, s.BaseFontWeight)
	}

	for i, sel := range headingSelectors {
		var decls []string
		if i < len(s.HeadingFontSizes) && cssValueSafe(s.HeadingFontSizes[i]) {
			decls = append(decls, "font-size:"+s.HeadingFontSizes[i])
		}
		if i < len(s.HeadingFontWeights) && fontWeightValid(s.HeadingFontWeights[i]) {
			decls = append(decls, fmt.Sprintf("font-weight:%d", s.HeadingFontWeights[i]))
		}
		if len(decls) > 0 {
			css.WriteString(prefixSelector(sel, prefix) + "{" + strings.Join(decls, ";") + "}")
		}
	}

	var codeDecls []string
	if cssValueSafe(s.CodeFontSize) {
		codeDecls = append(codeDecls, "font-size:"+s.CodeFontSize)
	}
	if fontWeightValid(s.CodeFontWeight) {
		codeDecls = append(codeDecls, fmt.Sprintf("font-weight:%d", s.CodeFontWeight))
	}
	if cssValueSafe(s.CodeTextColor) {
		codeDecls = append(codeDecls, "color:"+s.CodeTextColor)
	}
	if cssValueSafe(s.CodeBackgroundColor) {
		codeDecls = append(codeDecls, "background:"+s.CodeBackgroundColor)
	}
	if len(codeDecls) > 0 {
		css.WriteString(prefixSelector(codeFontSelector, prefix) + "{" + strings.Join(codeDecls, ";") + "}")
	}

	if s.LinkUnderline != nil {
		linkSel := prefixSelector(".sy-link, .sy-markdown a", prefix)
		hoverSel := prefixSelector(".sy-link:hover, .sy-markdown a:hover", prefix)
		if *s.LinkUnderline {
			css.WriteString(linkSel + "{text-decoration:underline}")
		} else {
			css.WriteString(linkSel + "{text-decoration:none}" + hoverSel + "{text-decoration:none}")
		}
	}

	if s.ShowWidgetBorder != nil && !*s.ShowWidgetBorder {
		css.WriteString(prefixSelector(".sy-input, .sy-select, .sy-textarea", prefix) + "{border-color:transparent}")
	}
}

// chartColorsJSON validates a palette and returns it, or nil.
func chartColorsJSON(colors []string) []string {
	if len(colors) == 0 {
		return nil
	}
	for _, c := range colors {
		if !cssValueSafe(c) {
			return nil
		}
	}
	return colors
}

// renderIndex builds the app shell with the title and theme applied. Theme
// values are validated before being inlined as CSS. basePath is the URL
// prefix the app is mounted under ("" at the root; e.g. "/dash" behind
// sy.Handler + http.StripPrefix) so assets and endpoints resolve correctly.
func renderIndex(title string, th Theme, basePath ...string) string {
	base := ""
	if len(basePath) > 0 {
		base = strings.TrimSuffix(basePath[0], "/")
	}
	// Only emitted when actually mounted under a prefix; the runtime defaults
	// to "" otherwise.
	baseScript := ""
	if base != "" {
		if b, err := json.Marshal(base); err == nil {
			baseScript = "<script>window.__SY_BASE=" + string(b) + ";</script>\n"
		}
	}
	// [i18n] overrides for built-in UI text. json.Marshal escapes <, > and &,
	// so the payload cannot break out of the script element.
	if len(uiStrings) > 0 {
		if b, err := json.Marshal(uiStrings); err == nil {
			baseScript += "<script>window.__SY_I18N=" + string(b) + ";</script>\n"
		}
	}
	connecting := "Connecting…"
	if v := uiStrings["connecting"]; v != "" {
		connecting = v
	}
	htmlAttr := ""
	if th.Mode == "light" || th.Mode == "dark" {
		htmlAttr = fmt.Sprintf(` data-theme=%q`, th.Mode)
	}

	var css strings.Builder

	// Custom @font-face declarations come first so the families they
	// register are resolvable by everything below.
	for _, ff := range th.FontFaces {
		if !fontValueSafe(ff.Family) || !cssURLSafe(ff.URL) {
			continue
		}
		css.WriteString("@font-face{font-family:" + cssQuote(ff.Family) + ";src:url(" + cssQuote(ff.URL) + ")")
		if fontValueSafe(ff.Weight) {
			css.WriteString(";font-weight:" + ff.Weight)
		}
		switch ff.Style {
		case "normal", "italic", "oblique":
			css.WriteString(";font-style:" + ff.Style)
		}
		if unicodeRangeSafe(ff.UnicodeRange) {
			css.WriteString(";unicode-range:" + ff.UnicodeRange)
		}
		css.WriteString(";font-display:swap}")
	}

	if vars := styleVars(&th.ThemeStyle, false); len(vars) > 0 {
		css.WriteString(":root{" + strings.Join(vars, ";") + "}")
	}
	styleRules(&css, &th.ThemeStyle, "")

	sbVars := styleVars(&th.Sidebar, true)
	if cssValueSafe(th.Sidebar.BackgroundColor) {
		// The sidebar surface is painted with --sy-sidebar-bg, so a sidebar
		// background override needs an explicit background declaration too.
		sbVars = append(sbVars, "background:"+th.Sidebar.BackgroundColor)
	}
	if cssValueSafe(th.Sidebar.TextColor) {
		sbVars = append(sbVars, "color:"+th.Sidebar.TextColor)
	}
	if len(sbVars) > 0 {
		css.WriteString("#syralit-sidebar{" + strings.Join(sbVars, ";") + "}")
	}
	styleRules(&css, &th.Sidebar, "#syralit-sidebar")

	if th.ShowSidebarBorder != nil && !*th.ShowSidebarBorder {
		css.WriteString("#syralit-root.has-sidebar #syralit-sidebar{border-right:none}")
	}

	style := ""
	if css.Len() > 0 {
		style = "\n<style>" + css.String() + "</style>"
	}

	// Chart palettes ride to the front end as JSON; the runtime resolves the
	// default Chart.js series colors against window.__SY_THEME.
	themeJS := map[string][]string{}
	if c := chartColorsJSON(th.ChartCategoricalColors); c != nil {
		themeJS["chart_categorical_colors"] = c
	}
	if c := chartColorsJSON(th.ChartSequentialColors); c != nil {
		themeJS["chart_sequential_colors"] = c
	}
	if c := chartColorsJSON(th.ChartDivergingColors); c != nil {
		themeJS["chart_diverging_colors"] = c
	}
	themeScript := ""
	if len(themeJS) > 0 {
		if b, err := json.Marshal(themeJS); err == nil {
			themeScript = "\n<script>window.__SY_THEME=" + string(b) + ";</script>"
		}
	}

	return fmt.Sprintf(indexHTML, htmlAttr, htmlEscape(title), style+themeScript+assetOverridesScript(), base, baseScript, htmlEscape(connecting))
}

// fontValueSafe allows a CSS font-family list (quoted names, commas) while
// still blocking anything that could break out of the inline style context
// (angle brackets, braces, semicolons, backslashes).
func fontValueSafe(v string) bool {
	if v == "" {
		return false
	}
	for _, r := range v {
		ok := r == '"' || r == '\'' || r == ',' || r == '-' || r == ' ' || r == '.' ||
			(r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		if !ok {
			return false
		}
	}
	return true
}

// cssURLSafe validates a font source location for use inside url("…").
func cssURLSafe(v string) bool {
	if v == "" {
		return false
	}
	for _, r := range v {
		if r == '"' || r == '\'' || r == '\\' || r == '<' || r == '>' ||
			r == '(' || r == ')' || r < 0x20 || r > 0x7e {
			return false
		}
	}
	return true
}

// unicodeRangeSafe validates a CSS unicode-range value, e.g. "U+0-10FFFF" or
// "U+4E00-9FFF, U+3000-303F".
func unicodeRangeSafe(v string) bool {
	if v == "" {
		return false
	}
	for _, r := range v {
		ok := r == 'U' || r == 'u' || r == '+' || r == '-' || r == ',' ||
			r == '?' || r == ' ' ||
			(r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
		if !ok {
			return false
		}
	}
	return true
}

// cssQuote wraps v in double quotes; callers must have validated v (no `"`
// or `\` inside) via fontValueSafe / cssURLSafe first.
func cssQuote(v string) string { return `"` + strings.ReplaceAll(v, `"`, ``) + `"` }

func fontWeightValid(w int) bool { return w >= 1 && w <= 1000 }

// cssValueSafe allows only characters expected in a color/length value, to keep
// theme strings from breaking out of the inline style context.
func cssValueSafe(v string) bool {
	if v == "" {
		return false
	}
	for _, r := range v {
		ok := r == '#' || r == '(' || r == ')' || r == ',' || r == '.' ||
			r == '%' || r == '-' || r == ' ' ||
			(r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
		if !ok {
			return false
		}
	}
	return true
}
