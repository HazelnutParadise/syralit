// Data Studio: the full upload → explore workflow in one app. Upload a
// CSV/Excel/JSON file (or use the built-in demo data), pick columns, and
// explore through a selectable GroupBy chart that filters the detail view —
// Insyra does the data work, Syralit the UI, with zero conversion between.
package main

import (
	"strings"

	sy "github.com/HazelnutParadise/syralit"
	syi "github.com/HazelnutParadise/syralit/integrations/insyra"

	"github.com/HazelnutParadise/insyra"
)

func demoData() *insyra.DataTable {
	region := insyra.NewDataList(
		"North", "South", "East", "North", "South", "West", "East", "North", "West", "South",
		"East", "West", "North", "South", "East").SetName("Region")
	product := insyra.NewDataList(
		"Widget", "Widget", "Gadget", "Gadget", "Gizmo", "Widget", "Gizmo", "Gizmo", "Gadget", "Gadget",
		"Widget", "Gizmo", "Widget", "Gizmo", "Gadget").SetName("Product")
	revenue := insyra.NewDataList(
		1200.0, 800, 640, 1500, 420, 980, 730, 1100, 560, 900,
		760, 1250, 990, 610, 880).SetName("Revenue")
	cost := insyra.NewDataList(
		500.0, 350, 300, 620, 200, 410, 330, 470, 260, 380,
		320, 540, 400, 280, 360).SetName("Cost")
	return insyra.NewDataTable(region, product, revenue, cost)
}

func main() {
	sy.App(func() {
		sy.SetPageConfig(sy.PageTitle("Data Studio"), sy.ConfigIcon("📊"))

		sy.Sidebar(func() {
			sy.Header("Data Studio")
			sy.Caption("Upload → group → click → drill down")
		})

		sy.Title("Data Studio")

		dt := syi.UploadTable("Upload CSV / Excel / JSON (or explore the demo data)")
		if dt == nil {
			dt = demoData()
			sy.Caption("Using built-in demo data — upload a file to explore your own.")
		}

		sy.Header("Explore")
		cols := sy.Columns(2)
		var groupCol, valueCol string
		cols[0](func() { groupCol = syi.ColumnSelect("Group by", dt, sy.Key("group_col")) })
		cols[1](func() { valueCol = syi.ColumnSelect("Value column", dt, sy.Key("value_col")) })
		if groupCol == "" || valueCol == "" || groupCol == valueCol {
			sy.Info("Pick a group column and a different numeric value column.")
			sy.Stop()
		}
		agg := sy.SegmentedControl("Aggregate", []string{"sum", "mean", "count"},
			sy.Key("agg"), sy.DefaultValue("sum"))
		op := map[string]insyra.AggregateOp{
			"sum": insyra.OpSum, "mean": insyra.OpMean, "count": insyra.OpCount,
		}[strings.ToLower(agg)]

		sel := syi.GroupedBarChart(dt, groupCol, valueCol, op,
			sy.Selectable(), sy.Key("explore_chart"),
			sy.ChartTitle(valueCol+" by "+groupCol+" — click a bar to drill down"))

		filtered := syi.FilterBySelection(dt, groupCol, sel)
		if sel != nil {
			sy.Badge(groupCol+": "+sel.X, sy.Color("violet"))
			if sy.Button("Clear", sy.Key("clear"), sy.ButtonType("secondary")) {
				sy.ResetWidget("explore_chart")
				sy.Rerun()
			}
		}

		sy.Header("Detail")
		syi.Metrics(filtered, valueCol)
		syi.Table(filtered, sy.Height(320))

		sy.Expander("Column statistics", func() {
			syi.Describe(filtered)
		})
	})
}
