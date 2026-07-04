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
<html lang="en"%s>
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>%s</title>
<link rel="stylesheet" href="/_syralit/assets/runtime.css">%s
</head>
<body>
<div id="syralit-root">
<nav id="syralit-sidebar"></nav>
<main id="syralit-app"><div class="sy-loading"><div class="sy-loading-spinner"></div><p>Connecting…</p></div></main>
</div>
<script src="/_syralit/assets/runtime.js"></script>
</body>
</html>`

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

// renderIndex builds the app shell with the title and theme applied. Theme
// values are validated before being inlined as CSS.
func renderIndex(title string, th Theme) string {
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

	var vars []string
	if cssValueSafe(th.Accent) {
		vars = append(vars, "--sy-accent:"+th.Accent)
	}
	if cssValueSafe(th.Radius) {
		vars = append(vars, "--sy-radius:"+th.Radius)
	}
	if f := resolveFont(th.Font); th.Font != "" && f != "" {
		vars = append(vars, "--sy-font:"+f)
	}
	if f := resolveFont(th.HeadingFont); th.HeadingFont != "" && f != "" {
		vars = append(vars, "--sy-font-heading:"+f)
	}
	if f := resolveFont(th.CodeFont); th.CodeFont != "" && f != "" {
		vars = append(vars, "--sy-font-code:"+f)
	}
	if len(vars) > 0 {
		css.WriteString(":root{" + strings.Join(vars, ";") + "}")
	}

	// Sidebar-scoped font overrides: the runtime stylesheet resolves all
	// font-family declarations through these variables, so redefining them
	// on the sidebar element cascades to its descendants only.
	var sbVars []string
	if f := resolveFont(th.Sidebar.Font); th.Sidebar.Font != "" && f != "" {
		sbVars = append(sbVars, "--sy-font:"+f, "--sy-font-heading:"+f)
	}
	if f := resolveFont(th.Sidebar.HeadingFont); th.Sidebar.HeadingFont != "" && f != "" {
		sbVars = append(sbVars, "--sy-font-heading:"+f)
	}
	if f := resolveFont(th.Sidebar.CodeFont); th.Sidebar.CodeFont != "" && f != "" {
		sbVars = append(sbVars, "--sy-font-code:"+f)
	}
	if len(sbVars) > 0 {
		css.WriteString("#syralit-sidebar{" + strings.Join(sbVars, ";") + "}")
	}

	if th.BaseFontSize > 0 {
		fmt.Fprintf(&css, "html{font-size:%dpx}", th.BaseFontSize)
	}
	if fontWeightValid(th.BaseFontWeight) {
		fmt.Fprintf(&css, "body{font-weight:%d}", th.BaseFontWeight)
	}

	for i, sel := range headingSelectors {
		var decls []string
		if i < len(th.HeadingFontSizes) && cssValueSafe(th.HeadingFontSizes[i]) {
			decls = append(decls, "font-size:"+th.HeadingFontSizes[i])
		}
		if i < len(th.HeadingFontWeights) && fontWeightValid(th.HeadingFontWeights[i]) {
			decls = append(decls, fmt.Sprintf("font-weight:%d", th.HeadingFontWeights[i]))
		}
		if len(decls) > 0 {
			css.WriteString(sel + "{" + strings.Join(decls, ";") + "}")
		}
	}

	var codeDecls []string
	if cssValueSafe(th.CodeFontSize) {
		codeDecls = append(codeDecls, "font-size:"+th.CodeFontSize)
	}
	if fontWeightValid(th.CodeFontWeight) {
		codeDecls = append(codeDecls, fmt.Sprintf("font-weight:%d", th.CodeFontWeight))
	}
	if len(codeDecls) > 0 {
		css.WriteString(codeFontSelector + "{" + strings.Join(codeDecls, ";") + "}")
	}

	style := ""
	if css.Len() > 0 {
		style = "\n<style>" + css.String() + "</style>"
	}

	return fmt.Sprintf(indexHTML, htmlAttr, htmlEscape(title), style+assetOverridesScript())
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
