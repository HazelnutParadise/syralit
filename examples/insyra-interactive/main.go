// Click-to-filter dashboard: an Insyra DataTable drives a selectable grouped
// chart; clicking a bar filters the detail table and metrics to that group —
// the interactive-dashboard pattern enabled by chart selections + Insyra.
package main

import (
	sy "github.com/HazelnutParadise/syralit"
	syi "github.com/HazelnutParadise/syralit/integrations/insyra"

	"github.com/HazelnutParadise/insyra"
)

func salesTable() *insyra.DataTable {
	region := insyra.NewDataList(
		"North", "South", "East", "North", "South", "West", "East", "North", "West", "South").SetName("Region")
	product := insyra.NewDataList(
		"Widget", "Widget", "Gadget", "Gadget", "Gizmo", "Widget", "Gizmo", "Gizmo", "Gadget", "Gadget").SetName("Product")
	revenue := insyra.NewDataList(
		1200.0, 800, 640, 1500, 420, 980, 730, 1100, 560, 900).SetName("Revenue")
	cost := insyra.NewDataList(
		500.0, 350, 300, 620, 200, 410, 330, 470, 260, 380).SetName("Cost")
	return insyra.NewDataTable(region, product, revenue, cost)
}

func main() {
	sy.App(func() {
		sy.Title("Insyra Interactive Dashboard")
		sy.Caption("Click a bar to filter everything below to that region; click another to switch.")

		dt := salesTable()

		// Selectable grouped chart: one bar per region, revenue summed by Insyra.
		sel := syi.GroupedBarChart(dt, "Region", "Revenue", insyra.OpSum,
			sy.Selectable(), sy.Key("by_region"), sy.ChartTitle("Revenue by region (click to filter)"))

		filtered := syi.FilterBySelection(dt, "Region", sel)
		if sel != nil {
			sy.Badge("Region: "+sel.X, sy.Color("violet"))
			if sy.Button("Clear filter", sy.Key("clear"), sy.ButtonType("secondary")) {
				sy.ResetWidget("by_region") // drop the stored chart selection
				sy.SetQueryParam("region", "")
				sy.Rerun()
			}
			sy.SetQueryParam("region", sel.X) // deep-linkable state
		}

		sy.Header("Detail")
		syi.Table(filtered)
		syi.Metrics(filtered, "Revenue")

		sy.Header("Revenue vs Cost")
		syi.MultiBarChart(filtered, "Product", nil) // all numeric columns
	})
}
