package syralit

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed assets
var assetsFS embed.FS

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
<main id="syralit-app"></main>
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

	return fmt.Sprintf(indexHTML, htmlAttr, htmlEscape(title), style)
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
