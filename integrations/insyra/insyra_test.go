package syinsyra

import (
	"testing"

	sy "github.com/HazelnutParadise/syralit"

	"github.com/HazelnutParadise/insyra"
)

func scores() *insyra.DataList {
	return insyra.NewDataList(72, 88, 91, 65, 79, 84, 95, 60, 77, 83).SetName("Scores")
}

func TestList(t *testing.T) {
	tree := sy.RenderOnce(func() { List(scores()) })
	dfs := tree.Find("dataframe")
	if len(dfs) != 1 {
		t.Fatalf("expected 1 dataframe, got %d", len(dfs))
	}
	headers, _ := dfs[0].Props["headers"].([]string)
	if len(headers) != 1 || headers[0] != "Scores" {
		t.Fatalf("expected header [Scores], got %v", headers)
	}
	rows, _ := dfs[0].Props["rows"].([][]any)
	if len(rows) != 10 {
		t.Fatalf("expected 10 rows, got %d", len(rows))
	}
}

func TestListMetrics(t *testing.T) {
	tree := sy.RenderOnce(func() { ListMetrics(scores()) })
	metrics := tree.Find("metric")
	if len(metrics) != 4 {
		t.Fatalf("expected 4 metrics, got %d", len(metrics))
	}
	got := map[string]string{}
	for _, m := range metrics {
		got[m.Props["label"].(string)] = m.Props["value"].(string)
	}
	if got["Count"] != "10" || got["Mean"] != "79.40" || got["Min"] != "60" || got["Max"] != "95" {
		t.Fatalf("unexpected metric values: %v", got)
	}
}

func TestListMetricsNonNumeric(t *testing.T) {
	dl := insyra.NewDataList("a", "b", "c").SetName("Letters")
	tree := sy.RenderOnce(func() { ListMetrics(dl) })
	got := map[string]string{}
	for _, m := range tree.Find("metric") {
		got[m.Props["label"].(string)] = m.Props["value"].(string)
	}
	// A non-numeric column has no mean/min/max — they must render as "—", not "NaN".
	if got["Min"] != "—" || got["Max"] != "—" || got["Mean"] != "—" {
		t.Fatalf("expected dashes for non-numeric stats, got %v", got)
	}
	if got["Count"] != "3" {
		t.Fatalf("expected Count 3, got %v", got["Count"])
	}
}

func TestListDescribe(t *testing.T) {
	tree := sy.RenderOnce(func() { ListDescribe(scores()) })
	tables := tree.Find("table")
	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	rows, _ := tables[0].Props["rows"].([][]string)
	if len(rows) != 8 {
		t.Fatalf("expected 8 stat rows, got %d", len(rows))
	}
	stat := map[string]string{}
	for _, r := range rows {
		if len(r) == 2 {
			stat[r[0]] = r[1]
		}
	}
	if stat["count"] != "10" || stat["mean"] != "79.40" || stat["50%"] != "81.00" {
		t.Fatalf("unexpected describe stats: %v", stat)
	}
}

func TestHistogram(t *testing.T) {
	tree := sy.RenderOnce(func() { Histogram(scores(), 6) })
	if len(tree.Find("histogram_chart")) != 1 {
		t.Fatal("expected a histogram_chart node")
	}
}

func TestTableAndMetricsFromDataTable(t *testing.T) {
	dt := insyra.NewDataTable(
		insyra.NewDataList("Alice", "Bob", "Carol").SetName("Name"),
		insyra.NewDataList(10, 20, 30).SetName("Sales"),
	)
	tree := sy.RenderOnce(func() {
		Table(dt)
		Metrics(dt, "Sales")
	})
	if len(tree.Find("dataframe")) != 1 {
		t.Fatal("expected Table to render a dataframe")
	}
	got := map[string]string{}
	for _, m := range tree.Find("metric") {
		got[m.Props["label"].(string)] = m.Props["value"].(string)
	}
	if got["Count"] != "3" || got["Mean"] != "20.00" || got["Min"] != "10" || got["Max"] != "30" {
		t.Fatalf("unexpected Sales metrics: %v", got)
	}
}
