package syinsyra

import (
	"fmt"
	"testing"
	"time"

	sy "github.com/HazelnutParadise/syralit"

	"github.com/HazelnutParadise/insyra"
)

func tsData() *insyra.DataTable {
	month := insyra.NewDataList("Jan", "Feb", "Mar", "Apr").SetName("Month")
	rev := insyra.NewDataList(10.0, 20, 30, 40).SetName("Revenue")
	return insyra.NewDataTable(month, rev)
}

func TestDownloadCSV(t *testing.T) {
	tree := sy.RenderOnce(func() { DownloadCSV("Export", tsData(), "out.csv") })
	btns := tree.Find("download_button")
	if len(btns) != 1 {
		t.Fatalf("expected download button, got %d", len(btns))
	}
	if btns[0].Props["mime"] != "text/csv" || btns[0].Props["filename"] != "out.csv" {
		t.Fatalf("props = %v", btns[0].Props)
	}
	data, _ := btns[0].Props["data"].(string)
	if data == "" {
		t.Fatal("empty CSV payload")
	}
}

func TestRollingMeanChart(t *testing.T) {
	tree := sy.RenderOnce(func() { RollingMeanChart(tsData(), "Month", "Revenue", 2) })
	charts := tree.Find("line_chart")
	if len(charts) != 1 {
		t.Fatalf("expected line chart, got %d", len(charts))
	}
	series, _ := charts[0].Props["series"].(map[string][]float64)
	if len(series) != 2 || series["Revenue"] == nil || series["Revenue (rolling 2)"] == nil {
		t.Fatalf("series = %v", series)
	}
	rolled := series["Revenue (rolling 2)"]
	// windows: [10,20]=15, [20,30]=25, [30,40]=35 (first window may be dropped)
	if rolled[len(rolled)-1] != 35 {
		t.Fatalf("rolling mean = %v", rolled)
	}
}

func TestCumSumAndPctChangeCharts(t *testing.T) {
	tree := sy.RenderOnce(func() {
		CumSumChart(tsData(), "Month", "Revenue")
		PctChangeChart(tsData(), "Month", "Revenue", 1)
	})
	if n := len(tree.Find("line_chart")); n != 1 {
		t.Fatalf("cumsum chart missing: %d", n)
	}
	if n := len(tree.Find("bar_chart")); n != 1 {
		t.Fatalf("pctchange chart missing: %d", n)
	}
}

func TestEvalCCL(t *testing.T) {
	out, err := evalCCL(tsData(), "double", "B * 2", 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if got := out.Headers(); len(got) != 3 || got[2] != "double" {
		t.Fatalf("headers = %v", got)
	}
	rows := out.To2DSlice()
	if v := rows[0][2]; toStr(v) != "20" {
		t.Fatalf("computed value = %v (%T)", v, v)
	}
	// Source untouched.
	if len(tsData().Headers()) != 2 {
		t.Fatal("source mutated")
	}
}

func TestAddFormulaColumnWidget(t *testing.T) {
	at := sy.NewAppTest(func() {
		out := AddFormulaColumn(tsData(), "ccl")
		Table(out)
	})
	at.Run()
	if n := len(at.FindAll("text_input")); n != 2 {
		t.Fatalf("expected 2 text inputs, got %d", n)
	}
	at.SetValue("ccl_formula", "B * 2")
	at.SetValue("ccl_name", "double")
	at.Run()
	dfs := at.FindAll("dataframe")
	headers, _ := dfs[0].Props["headers"].([]string)
	if len(headers) != 3 || headers[2] != "double" {
		t.Fatalf("headers after formula = %v", headers)
	}
}

func toStr(v any) string {
	return fmt.Sprint(v)
}
