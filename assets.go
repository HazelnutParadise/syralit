package syralit

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
)

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

// renderIndex builds the app shell with the title and theme applied. Theme
// values are validated before being inlined as CSS variables.
func renderIndex(title string, th Theme) string {
	htmlAttr := ""
	if th.Mode == "light" || th.Mode == "dark" {
		htmlAttr = fmt.Sprintf(` data-theme=%q`, th.Mode)
	}

	var vars []string
	if cssValueSafe(th.Accent) {
		vars = append(vars, "--sy-accent:"+th.Accent)
	}
	if cssValueSafe(th.Radius) {
		vars = append(vars, "--sy-radius:"+th.Radius)
	}
	style := ""
	if len(vars) > 0 {
		style = "\n<style>:root{" + strings.Join(vars, ";") + "}</style>"
	}

	return fmt.Sprintf(indexHTML, htmlAttr, htmlEscape(title), style+assetOverridesScript())
}

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
