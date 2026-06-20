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
		sy.Markdown("Demonstrating first-class **Insyra DataTable** integration with Syralit.")

		dt := insyra.NewDataTable(
			insyra.NewDataList("Name", "Alice", "Bob", "Carol", "Dave", "Eve"),
			insyra.NewDataList("Sales", 120, 85, 200, 150, 95),
			insyra.NewDataList("Revenue", 2400.0, 1700.0, 4000.0, 3000.0, 1900.0),
			insyra.NewDataList("Region", "North", "South", "North", "East", "South"),
		)

		sy.Header("Full Table")
		syi.Table(dt)

		sy.Divider()

		sy.Header("Column Metrics")
		col := syi.ColumnSelect("Select column", dt, sy.Key("metric_col"))
		if col != "" {
			syi.Metrics(dt, col)
		}

		sy.Divider()

		sy.Header("Charts")
		tab := sy.Tabs([]string{"Bar", "Line", "Scatter"})
		tab("Bar", func() {
			syi.BarChart(dt, "Name", "Sales")
		})
		tab("Line", func() {
			syi.LineChart(dt, "Name", "Revenue")
		})
		tab("Scatter", func() {
			syi.ScatterChart(dt, "Sales", "Revenue")
		})

		sy.Divider()

		sy.Header("Editable Table")
		sy.Caption("Edit cells directly — changes are reflected on rerun.")
		syi.EditableTable(dt, sy.Key("edit_dt"))
	})
}
