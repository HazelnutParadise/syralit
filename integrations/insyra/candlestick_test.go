package syinsyra

import (
	"encoding/json"
	"math"
	"testing"

	sy "github.com/HazelnutParadise/syralit"

	"github.com/HazelnutParadise/insyra"
)

func ohlcvTable(n int, dates func(i int) any) *insyra.DataTable {
	d, o, h, l, c, v := []any{}, []any{}, []any{}, []any{}, []any{}, []any{}
	for i := 0; i < n; i++ {
		base := 100 + float64(i%7) + float64(i)*0.3
		d = append(d, dates(i))
		o = append(o, base)
		h = append(h, base+2)
		l = append(l, base-2)
		c = append(c, base+math.Sin(float64(i)))
		v = append(v, 1000+i*10)
	}
	return insyra.NewDataTable(
		insyra.NewDataList(d...).SetName("date"), insyra.NewDataList(o...).SetName("open"),
		insyra.NewDataList(h...).SetName("high"), insyra.NewDataList(l...).SetName("low"),
		insyra.NewDataList(c...).SetName("close"), insyra.NewDataList(v...).SetName("volume"))
}

var testCols = OHLCV{Date: "date", Open: "open", High: "high", Low: "low", Close: "close", Volume: "volume"}

func isoDate(i int) any { return "2024-01-" + string(rune('0'+(i+1)/10)) + string(rune('0'+(i+1)%10)) }

func plotlySpec(t *testing.T, tree *sy.Node) map[string]any {
	t.Helper()
	nodes := tree.Find("plotly_chart")
	if len(nodes) != 1 {
		t.Fatalf("want 1 plotly_chart, got %d (tree: %v)", len(nodes), tree.Find("status"))
	}
	if _, err := json.Marshal(nodes[0].Props); err != nil {
		t.Fatalf("props must be JSON-safe (NaN breaks the websocket): %v", err)
	}
	return nodes[0].Props["spec"].(map[string]any)
}

func traceNames(spec map[string]any) []string {
	var names []string
	for _, tr := range spec["data"].([]map[string]any) {
		names = append(names, tr["name"].(string))
	}
	return names
}

func TestCandlestickAllIndicators(t *testing.T) {
	dt := ohlcvTable(30, isoDate)
	tree := sy.RenderOnce(func() {
		CandlestickChart(dt, testCols, Indicators{
			SMA: []int{5}, EMA: []int{10}, Bollinger: &BollingerSpec{N: 10, K: 2}, Bias: []int{10},
			MACD: &MACDSpec{Fast: 5, Slow: 10, Signal: 3}, RSI: 6, KD: 9,
		})
	})
	spec := plotlySpec(t, tree)
	got := map[string]bool{}
	for _, n := range traceNames(spec) {
		got[n] = true
	}
	for _, want := range []string{"價格", "SMA5", "EMA10", "布林上軌", "成交量", "MACD", "Signal", "RSI6", "K", "D", "乖離10"} {
		if !got[want] {
			t.Errorf("missing trace %q; have %v", want, traceNames(spec))
		}
	}
	lay := spec["layout"].(map[string]any)
	for _, k := range []string{"yaxis", "yaxis2", "yaxis3", "yaxis4", "yaxis5", "yaxis6"} {
		if _, ok := lay[k]; !ok {
			t.Errorf("layout missing %s", k)
		}
	}
	if len(tree.Find("status")) != 0 {
		t.Errorf("unexpected warnings: %v", tree.Find("status"))
	}
	// warm-up rows must be null, not NaN.
	sma := spec["data"].([]map[string]any)[1]["y"].([]any)
	if sma[0] != nil || sma[len(sma)-1] == nil {
		t.Errorf("SMA warm-up should be nil then values: %v", sma[:6])
	}
}

func TestCandlestickDefaultsAndColors(t *testing.T) {
	dt := ohlcvTable(10, isoDate)
	cols := testCols
	cols.Volume = ""
	spec := plotlySpec(t, sy.RenderOnce(func() { CandlestickChart(dt, cols, Indicators{}) }))
	candle := spec["data"].([]map[string]any)[0]
	up := candle["increasing"].(map[string]any)["line"].(map[string]any)["color"]
	if up != defaultUpColor {
		t.Errorf("default up color = %v, want Taiwan red %s", up, defaultUpColor)
	}
	if n := len(spec["data"].([]map[string]any)); n != 1 {
		t.Errorf("no indicators/volume should leave only the candle trace, got %d", n)
	}
	spec = plotlySpec(t, sy.RenderOnce(func() { CandlestickChart(dt, cols, Indicators{UpColor: "#0f0", DownColor: "#f00"}) }))
	candle = spec["data"].([]map[string]any)[0]
	if candle["increasing"].(map[string]any)["line"].(map[string]any)["color"] != "#0f0" {
		t.Error("UpColor not applied")
	}
}

func TestCandlestickMissingValueSkipsIndicatorButKeepsCandles(t *testing.T) {
	dt := ohlcvTable(20, isoDate)
	dt.UpdateElement(5, "E", nil) // close column, row 5 empty
	tree := sy.RenderOnce(func() { CandlestickChart(dt, testCols, Indicators{SMA: []int{5}}) })
	spec := plotlySpec(t, tree)
	names := traceNames(spec)
	if names[0] != "價格" {
		t.Fatalf("candles must still be drawn: %v", names)
	}
	for _, n := range names {
		if n == "SMA5" {
			t.Error("SMA should have been skipped")
		}
	}
	if len(tree.Find("status")) == 0 {
		t.Error("expected a warning for the skipped indicator")
	}
	cl := spec["data"].([]map[string]any)[0]["close"].([]any)
	if cl[5] != nil {
		t.Errorf("empty close should be nil, got %v", cl[5])
	}
}

func TestCandlestickSortsUnorderedDatesAndFixesCompactDates(t *testing.T) {
	dt := ohlcvTable(8, func(i int) any { return 20240108 - i }) // descending ints
	spec := plotlySpec(t, sy.RenderOnce(func() { CandlestickChart(dt, testCols, Indicators{}) }))
	x := spec["data"].([]map[string]any)[0]["x"].([]string)
	if x[0] != "2024-01-01" || x[7] != "2024-01-08" {
		t.Errorf("dates not sorted/converted: %v", x)
	}
}

func TestCandlestickBadColumn(t *testing.T) {
	dt := ohlcvTable(5, isoDate)
	cols := testCols
	cols.High = "nope"
	tree := sy.RenderOnce(func() { CandlestickChart(dt, cols, Indicators{}) })
	if len(tree.Find("plotly_chart")) != 0 || len(tree.Find("exception")) == 0 {
		t.Error("a missing column should show an error and no chart")
	}
}

func TestCandlestickNilTable(t *testing.T) {
	tree := sy.RenderOnce(func() { CandlestickChart(nil, testCols, Indicators{}) })
	if len(tree.Find("plotly_chart")) != 0 {
		t.Error("nil table should draw nothing")
	}
}
