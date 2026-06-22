// insyra-charts demonstrates Insyra's native go-echarts charts inside Syralit
// via the opt-in eplot subpackage — interactive chart types the built-in
// Chart.js layer doesn't have (Sankey, gauge, funnel, word cloud, …) — plus
// offline mode, which inlines the echarts JavaScript so the app needs no CDN
// at view time (air-gapped / strict-CSP / `syralit build` single binary).
package main

import (
	sy "github.com/HazelnutParadise/syralit"
	syiplot "github.com/HazelnutParadise/syralit/integrations/insyra/eplot"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/plot"
)

func main() {
	sy.App(func() {
		sy.SetPageConfig(sy.PageTitle("Insyra Native Charts"), sy.ConfigIcon("📈"))
		sy.Title("Insyra Native Charts")
		sy.Markdown("Interactive **go-echarts** charts beyond the built-in Chart.js layer, via `syiplot.EChart`.")

		offline := sy.Toggle("Offline mode (inline echarts JS — no CDN)", sy.DefaultValue(true))
		syiplot.SetOffline(offline)
		if offline {
			sy.Caption("Each chart's echarts JavaScript is embedded directly in the page — works fully offline.")
		} else {
			sy.Caption("Charts load echarts from go-echarts' CDN (needs internet).")
		}

		sy.Divider()

		tabs := sy.Tabs([]string{"Sankey", "Gauge", "Funnel", "Pie", "Word Cloud"})

		tabs("Sankey", func() {
			sy.Subheader("Sales flow: Region → Product")
			cfg := plot.SankeyChartConfig{
				Title: "Region → Product",
				Nodes: []string{"North", "South", "East", "Laptop", "Phone", "Tablet"},
			}
			links := []plot.SankeyLink{
				{Source: "North", Target: "Laptop", Value: 120},
				{Source: "North", Target: "Phone", Value: 80},
				{Source: "South", Target: "Phone", Value: 95},
				{Source: "South", Target: "Tablet", Value: 60},
				{Source: "East", Target: "Laptop", Value: 70},
				{Source: "East", Target: "Tablet", Value: 40},
			}
			syiplot.EChart(plot.CreateSankeyChart(cfg, links...), sy.Height(560))
		})

		tabs("Gauge", func() {
			sy.Subheader("Goal completion")
			cfg := plot.GaugeChartConfig{Title: "Quarterly target", SeriesName: "Progress"}
			syiplot.EChart(plot.CreateGaugeChart(cfg, 72))
		})

		tabs("Funnel", func() {
			sy.Subheader("Conversion funnel")
			cfg := plot.FunnelChartConfig{Title: "Signup funnel", ShowLabels: true}
			data := map[string]float64{"Visits": 100, "Signups": 62, "Trials": 38, "Paid": 19}
			syiplot.EChart(plot.CreateFunnelChart(cfg, data))
		})

		tabs("Pie", func() {
			sy.Subheader("Revenue by region")
			cfg := plot.PieChartConfig{Title: "Revenue share"}
			items := []plot.PieItem{
				{Name: "North", Value: 4200},
				{Name: "South", Value: 3100},
				{Name: "East", Value: 2600},
				{Name: "West", Value: 1800},
			}
			syiplot.EChart(plot.CreatePieChart(cfg, items...))
		})

		tabs("Word Cloud", func() {
			sy.Subheader("Tag frequency")
			words := insyra.NewDataList(
				"Go", "Go", "Go", "Go", "Go", "Data", "Data", "Data", "Data",
				"Syralit", "Syralit", "Syralit", "Insyra", "Insyra",
				"Chart", "Chart", "Stats", "Offline", "Echarts", "Sankey",
			).SetName("Tags")
			syiplot.WordCloud(words, "Tags")
		})
	})
}
