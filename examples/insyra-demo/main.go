package main

import (
	sy "github.com/HazelnutParadise/syralit"
	syi "github.com/HazelnutParadise/syralit/integrations/insyra"

	"github.com/HazelnutParadise/insyra"
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

		ltab := sy.Tabs([]string{"List", "Histogram", "Bar", "Line"})
		ltab("List", func() { syi.List(scores, sy.Height(240)) })
		ltab("Histogram", func() { syi.Histogram(scores, 6) })
		ltab("Bar", func() { syi.ListBarChart(scores) })
		ltab("Line", func() { syi.ListLineChart(scores) })

		sy.Divider()

		sy.Header("Editable Table")
		sy.Caption("Edit cells directly — changes are reflected on rerun.")
		syi.EditableTable(dt, sy.Key("edit_dt"))
	})
}
