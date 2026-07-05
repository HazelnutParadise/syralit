package syinsyra

import (
	"fmt"
	"testing"

	sy "github.com/HazelnutParadise/syralit"

	"github.com/HazelnutParadise/insyra"
)

func salesData() *insyra.DataTable {
	region := insyra.NewDataList("north", "south", "north", "south", "east").SetName("region")
	revenue := insyra.NewDataList(100.0, 50, 200, 70, 30).SetName("revenue")
	cost := insyra.NewDataList(40.0, 20, 90, 30, 10).SetName("cost")
	return insyra.NewDataTable(region, revenue, cost)
}

func TestChartUsesXLabels(t *testing.T) {
	tree := sy.RenderOnce(func() { BarChart(salesData(), "region", "revenue") })
	charts := tree.Find("bar_chart")
	if len(charts) != 1 {
		t.Fatalf("expected 1 bar_chart, got %d", len(charts))
	}
	labels, _ := charts[0].Props["x_labels"].([]string)
	if len(labels) != 5 || labels[0] != "north" || labels[4] != "east" {
		t.Fatalf("x_labels = %v", labels)
	}
}

func TestMultiLineChartAutoColumns(t *testing.T) {
	tree := sy.RenderOnce(func() { MultiLineChart(salesData(), "region", nil) })
	charts := tree.Find("line_chart")
	if len(charts) != 1 {
		t.Fatalf("expected 1 line_chart, got %d", len(charts))
	}
	series, _ := charts[0].Props["series"].(map[string][]float64)
	if len(series) != 2 || series["revenue"] == nil || series["cost"] == nil {
		t.Fatalf("series = %v", series)
	}
}

func TestGroupedBarChartAndSelectionFilter(t *testing.T) {
	at := sy.NewAppTest(func() {
		dt := salesData()
		sel := GroupedBarChart(dt, "region", "revenue", insyra.OpSum,
			sy.Selectable(), sy.Key("by_region"))
		filtered := FilterBySelection(dt, "region", sel)
		Table(filtered)
		if sel != nil {
			sy.Textf("picked:%s=%v", sel.X, sel.Value)
		}
	})

	at.Run()
	charts := at.FindAll("bar_chart")
	if len(charts) != 1 || charts[0].ID != "by_region" {
		t.Fatalf("grouped chart missing: %+v", charts)
	}
	series, _ := charts[0].Props["series"].(map[string][]float64)
	sums := series["revenue_sum"]
	if len(sums) != 3 {
		t.Fatalf("expected 3 groups, got %v", series)
	}
	labels, _ := charts[0].Props["x_labels"].([]string)
	// Find north's aggregated value: 100+200=300.
	found := false
	for i, l := range labels {
		if l == "north" && sums[i] == 300 {
			found = true
		}
	}
	if !found {
		t.Fatalf("north sum != 300: labels=%v sums=%v", labels, sums)
	}
	// No selection yet — the table shows all 5 rows.
	if rows := dataframeRows(at.Root); rows != 5 {
		t.Fatalf("unfiltered rows = %d, want 5", rows)
	}

	// Click the "north" bar → table filters to north's 2 rows.
	idx := 0
	for i, l := range labels {
		if l == "north" {
			idx = i
		}
	}
	at.SetValue("by_region", map[string]any{
		"series": "revenue_sum", "index": float64(idx), "x": "north", "value": float64(300),
	})
	at.Run()
	if rows := dataframeRows(at.Root); rows != 2 {
		t.Fatalf("filtered rows = %d, want 2", rows)
	}
	if got := at.Texts("text"); len(got) != 1 || got[0] != "picked:north=300" {
		t.Fatalf("selection echo = %v", got)
	}
}

func dataframeRows(tree *sy.Node) int {
	dfs := tree.Find("dataframe")
	if len(dfs) == 0 {
		return -1
	}
	rows, _ := dfs[0].Props["rows"].([][]any)
	return len(rows)
}

func TestFilterEquals(t *testing.T) {
	dt := salesData()
	got := FilterEquals(dt, "region", "south")
	if rows := got.To2DSlice(); len(rows) != 2 {
		t.Fatalf("filtered rows = %d (%v), want 2", len(rows), rows)
	}
	for _, row := range got.To2DSlice() {
		if fmt.Sprint(row[0]) != "south" {
			t.Fatalf("unexpected row: %v", row)
		}
	}
	// Source table untouched.
	if len(dt.To2DSlice()) != 5 {
		t.Fatal("source table was mutated")
	}
	// Unknown column falls back to the original table with a warning.
	if FilterEquals(dt, "nope", "x") != dt {
		t.Fatal("unknown column should return the input table")
	}
}

func TestEditableDataTableRoundTrip(t *testing.T) {
	var out *insyra.DataTable
	sy.RenderOnce(func() { out = EditableDataTable(salesData()) })
	if out == nil {
		t.Fatal("nil result")
	}
	if got := out.Headers(); len(got) != 3 || got[0] != "region" {
		t.Fatalf("headers = %v", got)
	}
	if rows := out.To2DSlice(); len(rows) != 5 {
		t.Fatalf("rows = %d, want 5", len(rows))
	}
}
