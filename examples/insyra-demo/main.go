package main

import (
	sy "github.com/HazelnutParadise/syralit"
	syi "github.com/HazelnutParadise/syralit/integrations/insyra"
	syiplot "github.com/HazelnutParadise/syralit/integrations/insyra/eplot"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/plot"
)

func main() {
	sy.App(func() {
		sy.SetPageConfig(sy.PageTitle("Insyra Demo"), sy.ConfigIcon("📊"))
		sy.Title("Insyra Integration Demo")
		sy.Markdown("Demonstrating first-class **Insyra** integration with Syralit — both `DataTable` and `DataList`.")

		// Columns are named via SetName so GetColByName / Headers work.
		dt := insyra.NewDataTable(
			insyra.NewDataList("Alice", "Bob", "Carol", "Dave", "Eve").SetName("Name"),
			insyra.NewDataList(120, 85, 200, 150, 95).SetName("Sales"),
			insyra.NewDataList(2400.0, 1700.0, 4000.0, 3000.0, 1900.0).SetName("Revenue"),
			insyra.NewDataList("North", "South", "North", "East", "South").SetName("Region"),
		)

		sy.Header("DataTable")
		syi.Table(dt)

		sy.Divider()

		sy.Header("Column Metrics")
		col := syi.ColumnSelect("Select column", dt, sy.Key("metric_col"))
		if col != "" {
			syi.Metrics(dt, col)
		}

		sy.Divider()

		sy.Header("DataTable Charts")
		tab := sy.Tabs([]string{"Bar", "Line", "Scatter"})
		tab("Bar", func() { syi.BarChart(dt, "Name", "Sales") })
		tab("Line", func() { syi.LineChart(dt, "Name", "Revenue") })
		tab("Scatter", func() { syi.ScatterChart(dt, "Sales", "Revenue") })

		sy.Divider()

		// --- DataList: the single-series counterpart to the DataTable set ---
		sy.Header("DataList (single series)")
		sy.Caption("A DataList is one labeled column. Here are the symmetric syi helpers.")

		scores := insyra.NewDataList(72, 88, 91, 65, 79, 84, 95, 60, 77, 83).SetName("Scores")

		syi.ListMetrics(scores)

		ltab := sy.Tabs([]string{"List", "Describe", "Histogram", "Bar", "Line"})
		ltab("List", func() { syi.List(scores, sy.Height(240)) })
		ltab("Describe", func() { syi.ListDescribe(scores) })
		ltab("Histogram", func() { syi.Histogram(scores, 6) })
		ltab("Bar", func() { syi.ListBarChart(scores) })
		ltab("Line", func() { syi.ListLineChart(scores) })

		sy.Divider()

		sy.Header("Editable Table")
		sy.Caption("Edit cells directly — changes are reflected on rerun.")
		syi.EditableTable(dt, sy.Key("edit_dt"))

		sy.Divider()

		// --- Statistical analysis (insyra/stats) -------------------------
		sy.Header("Statistical Analysis")
		sy.Caption("Insyra's stats package, surfaced directly in the UI.")
		stab := sy.Tabs([]string{"Describe", "Correlation", "Regression"})
		stab("Describe", func() { syi.Describe(dt) })
		stab("Correlation", func() {
			syi.Correlation(dt, "Sales", "Revenue", "pearson")
			syi.CorrelationMatrix(dt, "pearson")
		})
		stab("Regression", func() { syi.LinearRegression(dt, "Revenue", "Sales") })

		sy.Divider()

		// --- Interactive transforms (filter / CCL) -----------------------
		sy.Header("Interactive Transforms")
		ttab := sy.Tabs([]string{"Filter", "CCL formula"})
		ttab("Filter", func() {
			sy.Caption("Filter rows by a column condition; the source table is untouched.")
			syi.FilterBuilder(dt)
		})
		ttab("CCL formula", func() {
			sy.Caption("Add a computed column with an Excel-like formula (e.g. `Revenue / Sales`), then press Apply.")
			syi.CCLBuilder(dt)
		})

		sy.Divider()

		// --- Upload your own data ----------------------------------------
		sy.Header("Upload Your Data")
		sy.Caption("Drop a CSV, Excel or JSON file to turn it into a DataTable.")
		if up := syi.UploadTable("Upload a data file", sy.Key("upload_dt")); up != nil {
			syi.Describe(up)
			syi.Table(up, sy.Height(280))
		}

		sy.Divider()

		// --- Native go-echarts charts (beyond Chart.js) ------------------
		sy.Header("Native go-echarts Charts")
		sy.Caption("Interactive chart types the built-in Chart.js layer doesn't have, via syiplot.EChart.")
		etab := sy.Tabs([]string{"Word Cloud", "Sankey"})
		etab("Word Cloud", func() {
			words := insyra.NewDataList(
				"Go", "Go", "Go", "Go", "Data", "Data", "Data",
				"Syralit", "Syralit", "Insyra", "Chart", "Stats", "Go", "Data",
			).SetName("Tags")
			syiplot.WordCloud(words, "Tag frequency")
		})
		etab("Sankey", func() {
			links := []plot.SankeyLink{
				{Source: "North", Target: "Alice", Value: 120},
				{Source: "South", Target: "Bob", Value: 85},
				{Source: "North", Target: "Carol", Value: 200},
				{Source: "East", Target: "Dave", Value: 150},
				{Source: "South", Target: "Eve", Value: 95},
			}
			cfg := plot.SankeyChartConfig{
				Title: "Sales flow: Region → Person",
				Nodes: []string{"North", "South", "East", "Alice", "Bob", "Carol", "Dave", "Eve"},
			}
			syiplot.EChart(plot.CreateSankeyChart(cfg, links...), sy.Height(560))
		})
	})
}
