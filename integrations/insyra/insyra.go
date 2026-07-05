// Package syinsyra provides first-class Insyra integration for Syralit.
// Import with alias syi:
//
//	import syi "github.com/HazelnutParadise/syralit/integrations/insyra"
package syinsyra

import (
	"fmt"
	"math"
	"strconv"

	sy "github.com/HazelnutParadise/syralit"

	"github.com/HazelnutParadise/insyra"
)

// Table renders an Insyra DataTable as a Syralit DataFrame (sortable table).
func Table(dt *insyra.DataTable, opts ...sy.Option) {
	if dt == nil {
		sy.Warning("nil DataTable")
		return
	}
	headers := dt.Headers()
	rows := dt.To2DSlice()
	sy.DataFrame(headers, rows, opts...)
}

// Preview renders the first n rows of a DataTable as a DataFrame.
func Preview(dt *insyra.DataTable, n int, opts ...sy.Option) {
	if dt == nil {
		sy.Warning("nil DataTable")
		return
	}
	headers := dt.Headers()
	all := dt.To2DSlice()
	if n > len(all) {
		n = len(all)
	}
	sy.DataFrame(headers, all[:n], opts...)
}

// EditableTable renders an Insyra DataTable as an editable table.
// Returns the current rows reflecting user edits.
func EditableTable(dt *insyra.DataTable, opts ...sy.Option) [][]any {
	if dt == nil {
		sy.Warning("nil DataTable")
		return nil
	}
	headers := dt.Headers()
	rows := dt.To2DSlice()
	return sy.DataEditor(headers, rows, opts...)
}

// ColumnSelect renders a SelectBox populated with the DataTable's column names.
func ColumnSelect(label string, dt *insyra.DataTable, opts ...sy.Option) string {
	if dt == nil {
		sy.Warning("nil DataTable")
		return ""
	}
	return sy.SelectBox(label, dt.Headers(), opts...)
}

// LineChart extracts two columns from a DataTable and renders a line chart:
// xCol provides x-axis labels, yCol the numeric y values. Options pass
// through to the underlying chart — with sy.Selectable() the clicked point is
// returned (nil until then).
func LineChart(dt *insyra.DataTable, xCol, yCol string, opts ...sy.Option) *sy.ChartSelection {
	ys := extractNumericCol(dt, yCol)
	if ys == nil {
		return nil
	}
	return sy.LineChart(map[string][]float64{yCol: ys}, withXLabels(dt, xCol, opts)...)
}

// BarChart extracts a numeric column and renders a bar chart with xCol as the
// axis labels. Supports sy.Selectable() like LineChart.
func BarChart(dt *insyra.DataTable, xCol, yCol string, opts ...sy.Option) *sy.ChartSelection {
	ys := extractNumericCol(dt, yCol)
	if ys == nil {
		return nil
	}
	return sy.BarChart(map[string][]float64{yCol: ys}, withXLabels(dt, xCol, opts)...)
}

// AreaChart extracts a numeric column and renders an area chart with xCol as
// the axis labels. Supports sy.Selectable() like LineChart.
func AreaChart(dt *insyra.DataTable, xCol, yCol string, opts ...sy.Option) *sy.ChartSelection {
	ys := extractNumericCol(dt, yCol)
	if ys == nil {
		return nil
	}
	return sy.AreaChart(map[string][]float64{yCol: ys}, withXLabels(dt, xCol, opts)...)
}

// ScatterChart extracts x and y columns and renders a scatter chart.
// Supports sy.Selectable() like LineChart.
func ScatterChart(dt *insyra.DataTable, xCol, yCol string, opts ...sy.Option) *sy.ChartSelection {
	x := extractNumericCol(dt, xCol)
	y := extractNumericCol(dt, yCol)
	if x == nil || y == nil {
		return nil
	}
	n := min(len(x), len(y))
	points := make([][2]float64, n)
	for i := range n {
		points[i] = [2]float64{x[i], y[i]}
	}
	return sy.ScatterChart(map[string][][2]float64{yCol: points}, opts...)
}

// PieChart extracts a label column and a value column to render a pie chart.
// Supports sy.Selectable(); the clicked slice's label arrives in Selection.X.
func PieChart(dt *insyra.DataTable, labelCol, valueCol string, opts ...sy.Option) *sy.ChartSelection {
	labels := dt.GetColByName(labelCol)
	values := extractNumericCol(dt, valueCol)
	if labels == nil || values == nil {
		if labels == nil {
			sy.Warning(fmt.Sprintf("column %q not found", labelCol))
		}
		return nil
	}
	data := make(map[string]float64)
	labelData := labels.Data()
	n := min(len(labelData), len(values))
	for i := range n {
		data[fmt.Sprint(labelData[i])] = values[i]
	}
	return sy.PieChart(data, opts...)
}

// MultiLineChart plots several numeric columns as one line chart over xCol
// labels. Pass nil yCols to plot every numeric column except xCol — the
// equivalent of Streamlit's st.line_chart(df).
func MultiLineChart(dt *insyra.DataTable, xCol string, yCols []string, opts ...sy.Option) *sy.ChartSelection {
	series := multiSeries(dt, xCol, yCols)
	if series == nil {
		return nil
	}
	return sy.LineChart(series, withXLabels(dt, xCol, opts)...)
}

// MultiBarChart is MultiLineChart's bar counterpart (st.bar_chart(df)).
func MultiBarChart(dt *insyra.DataTable, xCol string, yCols []string, opts ...sy.Option) *sy.ChartSelection {
	series := multiSeries(dt, xCol, yCols)
	if series == nil {
		return nil
	}
	return sy.BarChart(series, withXLabels(dt, xCol, opts)...)
}

// MultiAreaChart is MultiLineChart's area counterpart (st.area_chart(df)).
func MultiAreaChart(dt *insyra.DataTable, xCol string, yCols []string, opts ...sy.Option) *sy.ChartSelection {
	series := multiSeries(dt, xCol, yCols)
	if series == nil {
		return nil
	}
	return sy.AreaChart(series, withXLabels(dt, xCol, opts)...)
}

// GroupedBarChart groups the table by keyCol, aggregates valueCol with op
// (insyra.OpSum, OpMean, ...), and renders one bar per group. Combined with
// sy.Selectable() this is the click-to-filter building block: the clicked
// group's key arrives in Selection.X, ready for FilterEquals.
func GroupedBarChart(dt *insyra.DataTable, keyCol, valueCol string, op insyra.AggregateOp, opts ...sy.Option) *sy.ChartSelection {
	agg := aggregate(dt, keyCol, valueCol, op)
	if agg == nil {
		return nil
	}
	return BarChart(agg, keyCol, aggColName(valueCol, op), opts...)
}

// GroupedPieChart is GroupedBarChart's pie counterpart.
func GroupedPieChart(dt *insyra.DataTable, keyCol, valueCol string, op insyra.AggregateOp, opts ...sy.Option) *sy.ChartSelection {
	agg := aggregate(dt, keyCol, valueCol, op)
	if agg == nil {
		return nil
	}
	return PieChart(agg, keyCol, aggColName(valueCol, op), opts...)
}

// aggregate runs GroupBy(keyCol).Aggregate(op on valueCol).
func aggregate(dt *insyra.DataTable, keyCol, valueCol string, op insyra.AggregateOp) *insyra.DataTable {
	if dt == nil {
		sy.Warning("nil DataTable")
		return nil
	}
	return dt.GroupBy(keyCol).Aggregate(insyra.AggregateConfig{
		SourceCol: valueCol, As: aggColName(valueCol, op), Op: op,
	})
}

func aggColName(valueCol string, op insyra.AggregateOp) string {
	return valueCol + "_" + op.String()
}

// FilterEquals returns a new DataTable with only the rows whose col value
// (string-compared) equals want. The source table is left untouched. It is a
// pure data helper (no UI): a nil table returns nil and an unknown column
// returns the input table unchanged.
func FilterEquals(dt *insyra.DataTable, col string, want any) *insyra.DataTable {
	if dt == nil {
		return nil
	}
	headers := dt.Headers()
	colIdx := -1
	for i, h := range headers {
		if h == col {
			colIdx = i
			break
		}
	}
	if colIdx < 0 {
		return dt
	}
	wantStr := fmt.Sprint(want)
	var rows [][]any
	for _, row := range dt.To2DSlice() {
		if colIdx < len(row) && fmt.Sprint(row[colIdx]) == wantStr {
			rows = append(rows, row)
		}
	}
	return tableFromRows(headers, rows)
}

// FilterBySelection filters dt to the rows matching a chart selection's X
// label in col. A nil selection returns dt unchanged, so the call composes
// directly with a selectable chart:
//
//	sel := syi.GroupedBarChart(dt, "region", "revenue", insyra.OpSum,
//	    sy.Selectable(), sy.Key("by_region"))
//	syi.Table(syi.FilterBySelection(dt, "region", sel))
func FilterBySelection(dt *insyra.DataTable, col string, sel *sy.ChartSelection) *insyra.DataTable {
	if sel == nil {
		return dt
	}
	return FilterEquals(dt, col, sel.X)
}

// EditableDataTable renders an editable table and returns the current state
// as a new *insyra.DataTable (the input is untouched), so user edits flow
// straight back into Insyra pipelines.
func EditableDataTable(dt *insyra.DataTable, opts ...sy.Option) *insyra.DataTable {
	if dt == nil {
		sy.Warning("nil DataTable")
		return nil
	}
	headers := dt.Headers()
	return tableFromRows(headers, sy.DataEditor(headers, dt.To2DSlice(), opts...))
}

// tableFromRows rebuilds a DataTable from headers and row-major data.
func tableFromRows(headers []string, rows [][]any) *insyra.DataTable {
	cols := make([][]any, len(headers))
	for _, row := range rows {
		for i := range headers {
			var v any
			if i < len(row) {
				v = row[i]
			}
			cols[i] = append(cols[i], v)
		}
	}
	lists := make([]*insyra.DataList, len(headers))
	for i, h := range headers {
		lists[i] = insyra.NewDataList(cols[i]...).SetName(h)
	}
	return insyra.NewDataTable(lists...)
}

// withXLabels prepends an sy.XLabels option built from a column's values;
// a missing column or empty name simply omits the labels.
func withXLabels(dt *insyra.DataTable, xCol string, opts []sy.Option) []sy.Option {
	if dt == nil || xCol == "" {
		return opts
	}
	dl := dt.GetColByName(xCol)
	if dl == nil {
		return opts
	}
	data := dl.Data()
	labels := make([]string, len(data))
	for i, v := range data {
		labels[i] = fmt.Sprint(v)
	}
	return append([]sy.Option{sy.XLabels(labels)}, opts...)
}

// multiSeries extracts the named numeric columns (or, when yCols is empty,
// every numeric column except xCol) as chart series.
func multiSeries(dt *insyra.DataTable, xCol string, yCols []string) map[string][]float64 {
	if dt == nil {
		sy.Warning("nil DataTable")
		return nil
	}
	if len(yCols) == 0 {
		for _, h := range dt.Headers() {
			if h == xCol {
				continue
			}
			if dl := dt.GetColByName(h); dl != nil && len(numericList(dl)) > 0 {
				yCols = append(yCols, h)
			}
		}
	}
	series := make(map[string][]float64, len(yCols))
	for _, y := range yCols {
		if ys := extractNumericCol(dt, y); len(ys) > 0 {
			series[y] = ys
		}
	}
	if len(series) == 0 {
		sy.Warning("no numeric columns to plot")
		return nil
	}
	return series
}

// Metrics renders summary statistics for a numeric column as Metric widgets.
// It is shorthand for ListMetrics on the named column.
func Metrics(dt *insyra.DataTable, col string) {
	if dt == nil {
		sy.Warning("nil DataTable")
		return
	}
	dl := dt.GetColByName(col)
	if dl == nil {
		sy.Warning(fmt.Sprintf("column %q not found", col))
		return
	}
	ListMetrics(dl)
}

// --- DataList helpers (the single-series counterpart to the DataTable set) ---

// List renders an Insyra DataList as a single-column Syralit DataFrame. The
// column header is the list's name (or "Value" if unnamed).
func List(dl *insyra.DataList, opts ...sy.Option) {
	if dl == nil {
		sy.Warning("nil DataList")
		return
	}
	sy.DataFrame([]string{listName(dl)}, listRows(dl.Data()), opts...)
}

// ListPreview renders the first n values of a DataList as a single-column table.
func ListPreview(dl *insyra.DataList, n int, opts ...sy.Option) {
	if dl == nil {
		sy.Warning("nil DataList")
		return
	}
	data := dl.Data()
	if n > len(data) {
		n = len(data)
	}
	sy.DataFrame([]string{listName(dl)}, listRows(data[:n]), opts...)
}

// EditableList renders a DataList as an editable single-column table and returns
// the current values reflecting user edits.
func EditableList(dl *insyra.DataList, opts ...sy.Option) []any {
	if dl == nil {
		sy.Warning("nil DataList")
		return nil
	}
	edited := sy.DataEditor([]string{listName(dl)}, listRows(dl.Data()), opts...)
	out := make([]any, 0, len(edited))
	for _, row := range edited {
		if len(row) > 0 {
			out = append(out, row[0])
		}
	}
	return out
}

// ListMetrics renders summary statistics for a numeric DataList (count, mean,
// min, max) as a row of Metric widgets.
func ListMetrics(dl *insyra.DataList) {
	if dl == nil {
		sy.Warning("nil DataList")
		return
	}
	cols := sy.Columns(4)
	cols[0](func() { sy.Metric("Count", fmt.Sprintf("%d", dl.Len())) })
	cols[1](func() { sy.Metric("Mean", statStr(dl.Mean(), 2)) })
	cols[2](func() { sy.Metric("Min", numStr(dl.Min())) })
	cols[3](func() { sy.Metric("Max", numStr(dl.Max())) })
}

// ListDescribe renders a pandas-style summary table for a numeric DataList:
// count, mean, std, min, 25%, 50% (median), 75%, max.
func ListDescribe(dl *insyra.DataList, opts ...sy.Option) {
	if dl == nil {
		sy.Warning("nil DataList")
		return
	}
	f := func(v float64) string { return statStr(v, 2) }
	rows := [][]string{
		{"count", fmt.Sprintf("%d", dl.Len())},
		{"mean", f(dl.Mean())},
		{"std", f(dl.Stdev())},
		{"min", f(dl.Min())},
		{"25%", f(dl.Quartile(1))},
		{"50%", f(dl.Median())},
		{"75%", f(dl.Quartile(3))},
		{"max", f(dl.Max())},
	}
	sy.Table([]string{"Statistic", listName(dl)}, rows)
}

// ListLineChart renders a DataList's numeric values as a line chart over their
// index. The series is named after the list.
func ListLineChart(dl *insyra.DataList, opts ...sy.Option) {
	if nums := numericList(dl); nums != nil {
		sy.LineChart(map[string][]float64{listName(dl): nums}, opts...)
	}
}

// ListBarChart renders a DataList's numeric values as a bar chart over index.
func ListBarChart(dl *insyra.DataList, opts ...sy.Option) {
	if nums := numericList(dl); nums != nil {
		sy.BarChart(map[string][]float64{listName(dl): nums}, opts...)
	}
}

// ListAreaChart renders a DataList's numeric values as an area chart over index.
func ListAreaChart(dl *insyra.DataList, opts ...sy.Option) {
	if nums := numericList(dl); nums != nil {
		sy.AreaChart(map[string][]float64{listName(dl): nums}, opts...)
	}
}

// Histogram renders the distribution of a numeric DataList across the given
// number of bins. This is unique to lists — there is no DataTable equivalent.
func Histogram(dl *insyra.DataList, bins int, opts ...sy.Option) {
	if nums := numericList(dl); nums != nil {
		sy.HistogramChart(nums, bins, opts...)
	}
}

// listName returns the DataList's name, falling back to "Value" when unnamed so
// tables and charts always have a usable header/series label.
func listName(dl *insyra.DataList) string {
	if name := dl.GetName(); name != "" {
		return name
	}
	return "Value"
}

// listRows wraps each value in a one-element row for single-column tables.
func listRows(data []any) [][]any {
	rows := make([][]any, len(data))
	for i, v := range data {
		rows[i] = []any{v}
	}
	return rows
}

// numericList extracts the numeric values of a DataList, skipping non-numeric
// entries — the single-list counterpart to extractNumericCol.
func numericList(dl *insyra.DataList) []float64 {
	if dl == nil {
		sy.Warning("nil DataList")
		return nil
	}
	raw := dl.Data()
	nums := make([]float64, 0, len(raw))
	for _, v := range raw {
		if f, ok := toFloat64(v); ok {
			nums = append(nums, f)
		}
	}
	return nums
}

// numStr formats a float without trailing zeros (e.g. 3, 3.5, 3.14). A NaN or
// Inf — which is what Insyra returns for stats on a non-numeric column — renders
// as an em dash rather than a bare "NaN".
func numStr(f float64) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return "—"
	}
	return fmt.Sprintf("%g", f)
}

// statStr formats a statistic to a fixed precision, with the same NaN/Inf
// guard as numStr.
func statStr(f float64, prec int) string {
	if math.IsNaN(f) || math.IsInf(f, 0) {
		return "—"
	}
	return strconv.FormatFloat(f, 'f', prec, 64)
}

func extractNumericCol(dt *insyra.DataTable, col string) []float64 {
	if dt == nil {
		sy.Warning("nil DataTable")
		return nil
	}
	dl := dt.GetColByName(col)
	if dl == nil {
		sy.Warning(fmt.Sprintf("column %q not found", col))
		return nil
	}
	return numericList(dl)
}

func toFloat64(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int64:
		return float64(x), true
	case int32:
		return float64(x), true
	case int16:
		return float64(x), true
	case int8:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint64:
		return float64(x), true
	case uint32:
		return float64(x), true
	}
	return 0, false
}
