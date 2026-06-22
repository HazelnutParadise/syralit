package syiplot

import (
	"embed"
	"regexp"
	"strings"
	"sync/atomic"
)

//go:embed assets/*.js
var assetsFS embed.FS

var offline atomic.Bool

// SetOffline controls whether EChart inlines the echarts JavaScript into the
// chart HTML (true) or references go-echarts' CDN (false, the default).
//
// Enable it for air-gapped / offline deployments, strict-CSP environments, and
// `syralit build` single-binary apps: the rendered chart then needs no network
// at view time because its scripts are embedded directly in the iframe.
//
// The bundled scripts (echarts core, the v4 build, and the word-cloud
// extension) cover every chart this package can produce. A chart that pulls in
// some other echarts extension still references that one from the CDN.
//
// Trade-off: inlining repeats the (~1 MB) echarts source in each chart's
// iframe, so prefer it when a page has only a few charts.
func SetOffline(v bool) { offline.Store(v) }

// Offline reports whether offline asset inlining is currently enabled.
func Offline() bool { return offline.Load() }

var scriptSrcRE = regexp.MustCompile(`<script src="([^"]+)"></script>`)

// inlineAssets replaces each <script src=".../name.js"> whose basename is
// bundled under assets/ with an inline <script> carrying that file's contents.
// Unbundled references are left untouched (they keep pointing at the CDN).
func inlineAssets(html string) string {
	return scriptSrcRE.ReplaceAllStringFunc(html, func(tag string) string {
		m := scriptSrcRE.FindStringSubmatch(tag)
		if len(m) < 2 {
			return tag
		}
		url := m[1]
		name := url[strings.LastIndex(url, "/")+1:]
		data, err := assetsFS.ReadFile("assets/" + name)
		if err != nil {
			return tag
		}
		return "<script>" + string(data) + "</script>"
	})
}
