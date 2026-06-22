// Package syiplot renders Insyra's go-echarts charts (insyra/plot) inline in a
// Syralit app, unlocking interactive chart types Syralit's built-in Chart.js
// layer doesn't have — Sankey, word cloud, K-line, gauge, funnel, theme-river,
// box plot, radar, and more.
//
// It lives in its own package because insyra/plot pulls in heavy dependencies
// (go-echarts and, transitively, chromedp for its PNG snapshot path). Keeping
// it separate means apps that only use the core Insyra adapter don't pay that
// cost. Import it explicitly when you want native plotting:
//
//	import syiplot "github.com/HazelnutParadise/syralit/integrations/insyra/eplot"
package syiplot

import (
	"bytes"

	sy "github.com/HazelnutParadise/syralit"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/plot"
)

// EChart renders any insyra/plot chart (go-echarts) inline as a fully
// interactive chart, in a sandboxed iframe:
//
//	syiplot.EChart(plot.CreateSankeyChart(cfg, links...))
//	syiplot.EChart(plot.CreateKlineChart(cfg, data...), sy.Height(600))
//
// Pass sy.Height/sy.Width to size the frame (defaults to 520px tall, full
// width). Note: go-echarts loads echarts.min.js from its own CDN, so an EChart
// needs internet at view time, like Syralit's other external chart helpers.
func EChart(chart plot.Renderable, opts ...sy.Option) {
	if chart == nil {
		sy.Warning("nil chart")
		return
	}
	var buf bytes.Buffer
	if err := chart.Render(&buf); err != nil {
		sy.Exception(err)
		return
	}
	// Default to a tall-enough frame; any user-supplied Height overrides it.
	all := append([]sy.Option{sy.Height(520)}, opts...)
	sy.Component(buf.String(), all...)
}

// WordCloud renders an interactive word cloud from a DataList whose values are
// the word weights. There is no Chart.js equivalent. title is optional.
func WordCloud(dl *insyra.DataList, title string, opts ...sy.Option) {
	if dl == nil {
		sy.Warning("nil DataList")
		return
	}
	cfg := plot.WordCloudConfig{Title: title, Width: "100%", Height: "480px"}
	EChart(plot.CreateWordCloud(cfg, dl), opts...)
}
