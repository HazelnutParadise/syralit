package uitest

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"

	sy "github.com/HazelnutParadise/syralit"
	syi "github.com/HazelnutParadise/syralit/integrations/insyra"

	"github.com/HazelnutParadise/insyra"
)

// rangeApp exposes a range-selectable line chart.
func rangeApp() {
	sy.Title("Range")
	sel := sy.LineChart(map[string][]float64{"S": {1, 5, 3, 8, 2, 7}},
		sy.XLabels([]string{"a", "b", "c", "d", "e", "f"}),
		sy.RangeSelectable(), sy.Key("range_chart"))
	if sel != nil && sel.Range {
		sy.Textf("picked:%s..%s", sel.X, sel.EndX)
	}
}

func TestChartRangeDragSelection(t *testing.T) {
	srv := startAppFn(t, rangeApp)
	ctx, cancel := browser(t)
	defer cancel()

	var coords []float64
	var picked string
	err := chromedp.Run(ctx,
		waitApp(srv.URL),
		chromedp.WaitVisible(".sy-chart-wrap canvas", chromedp.ByQuery),
		chromedp.ScrollIntoView(".sy-chart-wrap", chromedp.ByQuery),
		chromedp.WaitVisible(`.sy-chart-wrap[data-chart-state="settled"]`, chromedp.ByQuery),
		chromedp.Poll(`(function(){
			var c = document.querySelector(".sy-chart-wrap canvas");
			if (!window.Chart || !Chart.getChart(c)) return null;
			var chart = Chart.getChart(c);
			var x = chart.scales.x;
			var r = c.getBoundingClientRect();
			// drag from index 1 to index 4, vertically mid-chart
			var y = r.top + (chart.chartArea.top + chart.chartArea.bottom) / 2;
			return [r.left + x.getPixelForValue(1), r.left + x.getPixelForValue(4), y];
		})()`, &coords, chromedp.WithPollingTimeout(15*time.Second)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			if len(coords) != 3 {
				return fmt.Errorf("no drag coordinates: %v", coords)
			}
			x0, x1, y := coords[0], coords[1], coords[2]
			if err := input.DispatchMouseEvent(input.MousePressed, x0, y).
				WithButton(input.Left).WithClickCount(1).Do(ctx); err != nil {
				return err
			}
			for _, fx := range []float64{0.25, 0.5, 0.75, 1} {
				if err := input.DispatchMouseEvent(input.MouseMoved, x0+(x1-x0)*fx, y).
					WithButton(input.Left).Do(ctx); err != nil {
					return err
				}
			}
			return input.DispatchMouseEvent(input.MouseReleased, x1, y).
				WithButton(input.Left).WithClickCount(1).Do(ctx)
		}),
		chromedp.WaitVisible(`//p[contains(., "picked:")]`),
		chromedp.Text(`//p[contains(., "picked:")]`, &picked),
	)
	if err != nil {
		t.Fatal(err)
	}
	if picked != "picked:b..e" {
		t.Fatalf("range = %q, want picked:b..e", picked)
	}
}

// insyraApp reproduces the click-to-filter dashboard flow.
func insyraApp() {
	sy.Title("Filter")
	region := insyra.NewDataList("north", "south", "north", "east").SetName("region")
	revenue := insyra.NewDataList(100.0, 50, 200, 70).SetName("revenue")
	dt := insyra.NewDataTable(region, revenue)

	sel := syi.GroupedBarChart(dt, "region", "revenue", insyra.OpSum,
		sy.Selectable(), sy.Key("by_region"))
	filtered := syi.FilterBySelection(dt, "region", sel)
	syi.Table(filtered)
	syi.DownloadCSV("Export CSV", filtered, "filtered.csv", sy.Key("dl"))
}

func TestInsyraClickToFilterFlow(t *testing.T) {
	srv := startAppFn(t, insyraApp)
	ctx, cancel := browser(t)
	defer cancel()

	var coords []float64
	var rowsBefore, rowsAfter int
	err := chromedp.Run(ctx,
		waitApp(srv.URL),
		chromedp.WaitVisible(".sy-dataframe", chromedp.ByQuery),
		chromedp.Evaluate(`document.querySelectorAll(".sy-dataframe tbody tr").length`, &rowsBefore),
		chromedp.ScrollIntoView(".sy-chart-wrap", chromedp.ByQuery),
		chromedp.Poll(`(function(){
			var c = document.querySelector(".sy-chart-wrap canvas");
			if (!window.Chart || !Chart.getChart(c)) return null;
			var chart = Chart.getChart(c);
			var idx = chart.data.labels.indexOf("north");
			var el = chart.getDatasetMeta(0).data[idx];
			if (!el) return null;
			var r = c.getBoundingClientRect();
			return [r.left + el.x, r.top + (el.y + chart.chartArea.bottom) / 2];
		})()`, &coords, chromedp.WithPollingTimeout(15*time.Second)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			return chromedp.MouseClickXY(coords[0], coords[1]).Do(ctx)
		}),
		chromedp.Poll(`document.querySelectorAll(".sy-dataframe tbody tr").length === 2`,
			nil, chromedp.WithPollingTimeout(10*time.Second)),
		chromedp.Evaluate(`document.querySelectorAll(".sy-dataframe tbody tr").length`, &rowsAfter),
	)
	if err != nil {
		t.Fatal(err)
	}
	if rowsBefore != 4 || rowsAfter != 2 {
		t.Fatalf("rows before/after = %d/%d, want 4/2", rowsBefore, rowsAfter)
	}
}

// startAppFn is startApp for an arbitrary app function.
func startAppFn(t *testing.T, fn func()) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(sy.Handler(sy.Config{Title: "UITest"}, fn))
	t.Cleanup(srv.Close)
	return srv
}
