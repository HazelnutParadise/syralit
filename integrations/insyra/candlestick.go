package syinsyra

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"time"

	sy "github.com/HazelnutParadise/syralit"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/quant/ta"
)

// OHLCV names the DataTable columns CandlestickChart reads. Date, Open, High,
// Low and Close are required; Volume is optional (empty = no volume pane).
type OHLCV struct {
	Date, Open, High, Low, Close, Volume string
}

// BollingerSpec configures Bollinger bands: an N-row average with bands K
// standard deviations away.
type BollingerSpec struct {
	N int
	K float64
}

// MACDSpec configures the MACD sub-pane (typically 12, 26, 9).
type MACDSpec struct {
	Fast, Slow, Signal int
}

// Indicators selects what CandlestickChart draws on top of the candles. The
// zero value draws only the candles (and the volume pane when OHLCV.Volume is
// set). SMA, EMA and Bollinger overlay the price pane; Bias, MACD, RSI and KD
// get a sub-pane each.
type Indicators struct {
	SMA, EMA  []int          // moving-average periods, e.g. []int{5, 20}
	Bollinger *BollingerSpec // nil = off
	Bias      []int          // 乖離率 periods in percent, e.g. []int{10}
	MACD      *MACDSpec      // nil = off
	RSI       int            // period (Wilder); 0 = off
	KD        int            // stochastic KD period (Taiwan-style recursive); 0 = off

	// UpColor and DownColor default to the Taiwan convention: red for a rise,
	// green for a fall. Set them to swap to the international convention.
	UpColor, DownColor string
}

const (
	defaultUpColor   = "#e53935"
	defaultDownColor = "#2e9e5b"
)

const gridColor = "rgba(128,128,128,0.25)"

var overlayPalette = []string{"#f5a623", "#4a90d9", "#9b59b6", "#16a085", "#d35400", "#7f8c8d"}

// CandlestickChart renders an interactive candlestick chart from the OHLCV
// columns of dt, with optional technical indicators from Insyra's quant/ta.
// Rows are sorted by date first, because indicators depend on row order.
//
//	syi.CandlestickChart(dt,
//	    syi.OHLCV{Date: "date", Open: "open", High: "high", Low: "low", Close: "close", Volume: "volume"},
//	    syi.Indicators{SMA: []int{5, 20}, MACD: &syi.MACDSpec{Fast: 12, Slow: 26, Signal: 9}, KD: 9})
//
// An indicator that cannot be computed (for example because the close column
// has empty cells) is skipped with a warning; the candles are still drawn.
// Extra sy.Option values (sy.Height, sy.Width) apply to the chart. The chart
// is not selectable.
func CandlestickChart(dt *insyra.DataTable, cols OHLCV, ind Indicators, opts ...sy.Option) {
	if dt == nil {
		sy.Warning("nil DataTable")
		return
	}
	b, err := newCandleBuild(dt, cols, ind)
	if err != nil {
		sy.Exception(err)
		return
	}
	spec, warnings := b.spec()
	for _, w := range warnings {
		sy.Warning(w)
	}
	height := 420 + 130*b.subPanes()
	sy.PlotlyChart(spec, append([]sy.Option{sy.Height(height)}, opts...)...)
}

type candleBuild struct {
	dates                   []string
	open, high, low, closeP []float64
	volume                  []float64
	hasVolume               bool
	ind                     Indicators
	up, down                string
	traces                  []map[string]any
	warnings                []string
}

func newCandleBuild(dt *insyra.DataTable, cols OHLCV, ind Indicators) (*candleBuild, error) {
	b := &candleBuild{ind: ind, up: ind.UpColor, down: ind.DownColor}
	if b.up == "" {
		b.up = defaultUpColor
	}
	if b.down == "" {
		b.down = defaultDownColor
	}
	dateList := dt.GetColByName(cols.Date)
	if dateList == nil {
		return nil, fmt.Errorf("column %q not found", cols.Date)
	}
	raw := dateList.Data()
	b.dates = make([]string, len(raw))
	for i, v := range raw {
		b.dates[i] = dateLabel(v)
	}
	var err error
	for _, c := range []struct {
		name string
		dst  *[]float64
	}{{cols.Open, &b.open}, {cols.High, &b.high}, {cols.Low, &b.low}, {cols.Close, &b.closeP}} {
		if *c.dst, err = floatColumn(dt, c.name, len(raw)); err != nil {
			return nil, err
		}
	}
	if cols.Volume != "" {
		if b.volume, err = floatColumn(dt, cols.Volume, len(raw)); err != nil {
			return nil, err
		}
		b.hasVolume = true
	}
	b.sortByDate()
	return b, nil
}

// floatColumn reads a column positionally; cells that are not numbers become NaN.
func floatColumn(dt *insyra.DataTable, name string, n int) ([]float64, error) {
	dl := dt.GetColByName(name)
	if dl == nil {
		return nil, fmt.Errorf("column %q not found", name)
	}
	raw := dl.Data()
	if len(raw) != n {
		return nil, fmt.Errorf("column %q has %d rows, expected %d", name, len(raw), n)
	}
	out := make([]float64, n)
	for i, v := range raw {
		f, ok := insyra.ToFloat64Safe(v)
		if !ok || math.IsInf(f, 0) {
			f = math.NaN()
		}
		out[i] = f
	}
	return out, nil
}

var compactDate = regexp.MustCompile(`^(\d{4})(\d{2})(\d{2})$`)

// dateLabel turns a date cell into an ISO-like string Plotly reads as a date
// axis; a number such as 20240102 becomes 2024-01-02.
func dateLabel(v any) string {
	if t, ok := v.(time.Time); ok {
		if t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 {
			return t.Format("2006-01-02")
		}
		return t.Format("2006-01-02 15:04:05") // keep intraday bars apart and ordered
	}
	if f, ok := v.(float64); ok && f == math.Trunc(f) && math.Abs(f) < 1e15 {
		v = int64(f) // 2.0240102e+07 would otherwise miss the compact-date match
	}
	s := fmt.Sprint(v)
	if m := compactDate.FindStringSubmatch(s); m != nil {
		return m[1] + "-" + m[2] + "-" + m[3]
	}
	return s
}

func (b *candleBuild) sortByDate() {
	idx := make([]int, len(b.dates))
	for i := range idx {
		idx[i] = i
	}
	if sort.SliceIsSorted(idx, func(i, j int) bool { return b.dates[idx[i]] < b.dates[idx[j]] }) {
		return
	}
	sort.SliceStable(idx, func(i, j int) bool { return b.dates[idx[i]] < b.dates[idx[j]] })
	pick := func(s []float64) []float64 {
		if s == nil {
			return nil
		}
		out := make([]float64, len(idx))
		for i, k := range idx {
			out[i] = s[k]
		}
		return out
	}
	dates := make([]string, len(idx))
	for i, k := range idx {
		dates[i] = b.dates[k]
	}
	b.dates = dates
	b.open, b.high, b.low, b.closeP, b.volume = pick(b.open), pick(b.high), pick(b.low), pick(b.closeP), pick(b.volume)
}

// subPanes is the number of sub-panes below the price pane.
func (b *candleBuild) subPanes() int {
	n := len(b.ind.Bias)
	if b.hasVolume {
		n++
	}
	if b.ind.MACD != nil {
		n++
	}
	if b.ind.RSI > 0 {
		n++
	}
	if b.ind.KD > 0 {
		n++
	}
	return n
}

// jsonSeries converts values to a JSON-safe slice: NaN and Inf become nil, as
// encoding/json refuses them and a failed marshal drops the whole connection.
func jsonSeries(v []float64) []any {
	out := make([]any, len(v))
	for i, f := range v {
		if math.IsNaN(f) || math.IsInf(f, 0) {
			continue
		}
		out[i] = f
	}
	return out
}

func listSeries(dl *insyra.DataList) []float64 {
	raw := dl.Data()
	out := make([]float64, len(raw))
	for i, v := range raw {
		f, ok := insyra.ToFloat64Safe(v)
		if !ok {
			f = math.NaN()
		}
		out[i] = f
	}
	return out
}

func (b *candleBuild) line(name, color string, y []float64, axis int, extra map[string]any) {
	tr := map[string]any{
		"type": "scatter", "mode": "lines", "name": name,
		"x": b.dates, "y": jsonSeries(y),
		"line": map[string]any{"color": color, "width": 1.3},
	}
	b.setAxis(tr, axis)
	for k, v := range extra {
		tr[k] = v
	}
	b.traces = append(b.traces, tr)
}

func (b *candleBuild) setAxis(tr map[string]any, axis int) {
	tr["xaxis"] = "x"
	if axis == 1 {
		tr["yaxis"] = "y"
	} else {
		tr["yaxis"] = fmt.Sprintf("y%d", axis)
	}
}

// try runs one indicator computation, recording a warning when it fails.
func (b *candleBuild) try(name string, fn func() error) {
	if err := fn(); err != nil {
		b.warnings = append(b.warnings, fmt.Sprintf("%s skipped: %v", name, err))
	}
}

func (b *candleBuild) spec() (map[string]any, []string) {
	closeList := insyra.NewDataList(toAnySlice(b.closeP)...)
	candle := map[string]any{
		"type": "candlestick", "name": "價格", "xaxis": "x", "yaxis": "y",
		"x": b.dates, "open": jsonSeries(b.open), "high": jsonSeries(b.high),
		"low": jsonSeries(b.low), "close": jsonSeries(b.closeP),
		"increasing": map[string]any{"line": map[string]any{"color": b.up}, "fillcolor": b.up},
		"decreasing": map[string]any{"line": map[string]any{"color": b.down}, "fillcolor": b.down},
	}
	b.traces = append(b.traces, candle)

	ci := 0
	nextColor := func() string { c := overlayPalette[ci%len(overlayPalette)]; ci++; return c }
	for _, n := range b.ind.SMA {
		n := n
		b.try(fmt.Sprintf("SMA(%d)", n), func() error {
			r, err := ta.SMA(closeList, n)
			if err == nil {
				b.line(fmt.Sprintf("SMA%d", n), nextColor(), listSeries(r), 1, nil)
			}
			return err
		})
	}
	for _, n := range b.ind.EMA {
		n := n
		b.try(fmt.Sprintf("EMA(%d)", n), func() error {
			r, err := ta.EMA(closeList, n)
			if err == nil {
				b.line(fmt.Sprintf("EMA%d", n), nextColor(), listSeries(r), 1, nil)
			}
			return err
		})
	}
	if bb := b.ind.Bollinger; bb != nil {
		b.try("Bollinger", func() error {
			r, err := ta.Bollinger(closeList, bb.N, bb.K)
			if err != nil {
				return err
			}
			color := "#95a5a6"
			b.line("布林上軌", color, listSeries(r.Upper), 1, map[string]any{"line": map[string]any{"color": color, "width": 1, "dash": "dot"}})
			b.line("布林中軌", color, listSeries(r.Middle), 1, map[string]any{"line": map[string]any{"color": color, "width": 1}})
			b.line("布林下軌", color, listSeries(r.Lower), 1, map[string]any{"line": map[string]any{"color": color, "width": 1, "dash": "dot"}})
			return nil
		})
	}

	axis := 1
	titles := map[int]string{}
	nextPane := func(title string) int { axis++; titles[axis] = title; return axis }

	if b.hasVolume {
		ax := nextPane("成交量")
		colors := make([]string, len(b.closeP))
		for i := range colors {
			colors[i] = b.up
			if b.closeP[i] < b.open[i] {
				colors[i] = b.down
			}
		}
		tr := map[string]any{"type": "bar", "name": "成交量", "x": b.dates, "y": jsonSeries(b.volume), "marker": map[string]any{"color": colors}}
		b.setAxis(tr, ax)
		b.traces = append(b.traces, tr)
	}
	if m := b.ind.MACD; m != nil {
		b.try("MACD", func() error {
			r, err := ta.MACD(closeList, m.Fast, m.Slow, m.Signal)
			if err != nil {
				return err
			}
			ax := nextPane("MACD")
			hist := listSeries(r.Histogram)
			hc := make([]string, len(hist))
			for i, h := range hist {
				hc[i] = b.up
				if h < 0 {
					hc[i] = b.down
				}
			}
			bar := map[string]any{"type": "bar", "name": "MACD柱", "x": b.dates, "y": jsonSeries(hist), "marker": map[string]any{"color": hc}}
			b.setAxis(bar, ax)
			b.traces = append(b.traces, bar)
			b.line("MACD", "#4a90d9", listSeries(r.MACD), ax, nil)
			b.line("Signal", "#f5a623", listSeries(r.Signal), ax, nil)
			return nil
		})
	}
	if n := b.ind.RSI; n > 0 {
		b.try("RSI", func() error {
			r, err := ta.RSI(closeList, n)
			if err != nil {
				return err
			}
			b.line(fmt.Sprintf("RSI%d", n), "#9b59b6", listSeries(r), nextPane("RSI"), nil)
			return nil
		})
	}
	if n := b.ind.KD; n > 0 {
		b.try("KD", func() error {
			high, low := insyra.NewDataList(toAnySlice(b.high)...), insyra.NewDataList(toAnySlice(b.low)...)
			r, err := ta.KD(high, low, closeList, n)
			if err != nil {
				return err
			}
			ax := nextPane("KD")
			b.line("K", "#f5a623", listSeries(r.K), ax, nil)
			b.line("D", "#4a90d9", listSeries(r.D), ax, nil)
			return nil
		})
	}
	for _, n := range b.ind.Bias {
		n := n
		b.try(fmt.Sprintf("Bias(%d)", n), func() error {
			r, err := ta.Bias(closeList, n)
			if err != nil {
				return err
			}
			b.line(fmt.Sprintf("乖離%d", n), "#d35400", listSeries(r), nextPane(fmt.Sprintf("乖離率(%d)", n)), nil)
			return nil
		})
	}
	return map[string]any{"data": b.traces, "layout": b.layout(axis, titles)}, b.warnings
}

func toAnySlice(v []float64) []any {
	out := make([]any, len(v))
	for i, f := range v {
		out[i] = f
	}
	return out
}

// layout stacks the price pane on top and gives every sub-pane an equal strip.
func (b *candleBuild) layout(axes int, titles map[int]string) map[string]any {
	sub := axes - 1
	priceFrac := 1.0
	if sub > 0 {
		priceFrac = math.Max(0.4, 1-0.2*float64(sub))
	}
	lay := map[string]any{
		"paper_bgcolor": "rgba(0,0,0,0)", "plot_bgcolor": "rgba(0,0,0,0)",
		"margin":     map[string]any{"l": 50, "r": 20, "t": 20, "b": 30},
		"font":       map[string]any{"color": "rgba(128,128,128,1)"}, // readable on light and dark themes
		"showlegend": true,
		"legend":     map[string]any{"orientation": "h", "y": 1.08},
		"xaxis":      map[string]any{"gridcolor": gridColor, "rangeslider": map[string]any{"visible": false}, "anchor": "y" + axisSuffix(axes), "type": "date"},
		"yaxis":      map[string]any{"domain": []float64{1 - priceFrac, 1}, "anchor": "x", "gridcolor": gridColor, "zerolinecolor": gridColor},
	}
	if sub > 0 {
		h := (1 - priceFrac - 0.02*float64(sub)) / float64(sub)
		top := 1 - priceFrac - 0.02
		for ax := 2; ax <= axes; ax++ {
			lay[fmt.Sprintf("yaxis%d", ax)] = map[string]any{
				"domain": []float64{math.Max(0, top-h), top}, "anchor": "x", "gridcolor": gridColor, "zerolinecolor": gridColor, "title": map[string]any{"text": titles[ax]},
			}
			top -= h + 0.02
		}
	}
	return lay
}

func axisSuffix(axes int) string {
	if axes <= 1 {
		return ""
	}
	return fmt.Sprint(axes)
}
