// Package syinsyra provides first-class Insyra integration for Syralit.
// Import with alias syi:
//
//	import syi "github.com/HazelnutParadise/syralit/integrations/insyra"
package syinsyra

import (
	"fmt"

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

// LineChart extracts two columns from a DataTable and renders a line chart.
// xCol provides x-axis labels, yCol provides numeric y values.
func LineChart(dt *insyra.DataTable, xCol, yCol string) {
	ys := extractNumericCol(dt, yCol)
	if ys == nil {
		return
	}
	sy.LineChart(map[string][]float64{yCol: ys})
}

// BarChart extracts a numeric column and renders a bar chart.
func BarChart(dt *insyra.DataTable, xCol, yCol string) {
	ys := extractNumericCol(dt, yCol)
	if ys == nil {
		return
	}
	sy.BarChart(map[string][]float64{yCol: ys})
}

// AreaChart extracts a numeric column and renders an area chart.
func AreaChart(dt *insyra.DataTable, xCol, yCol string) {
	ys := extractNumericCol(dt, yCol)
	if ys == nil {
		return
	}
	sy.AreaChart(map[string][]float64{yCol: ys})
}

// ScatterChart extracts x and y columns and renders a scatter chart.
func ScatterChart(dt *insyra.DataTable, xCol, yCol string) {
	x := extractNumericCol(dt, xCol)
	y := extractNumericCol(dt, yCol)
	if x == nil || y == nil {
		return
	}
	n := min(len(x), len(y))
	points := make([][2]float64, n)
	for i := range n {
		points[i] = [2]float64{x[i], y[i]}
	}
	sy.ScatterChart(map[string][][2]float64{yCol: points})
}

// PieChart extracts a label column and a value column to render a pie chart.
func PieChart(dt *insyra.DataTable, labelCol, valueCol string) {
	labels := dt.GetColByName(labelCol)
	values := extractNumericCol(dt, valueCol)
	if labels == nil || values == nil {
		if labels == nil {
			sy.Warning(fmt.Sprintf("column %q not found", labelCol))
		}
		return
	}
	data := make(map[string]float64)
	labelData := labels.Data()
	n := min(len(labelData), len(values))
	for i := range n {
		data[fmt.Sprint(labelData[i])] = values[i]
	}
	sy.PieChart(data)
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
	cols[1](func() { sy.Metric("Mean", fmt.Sprintf("%.2f", dl.Mean())) })
	cols[2](func() { sy.Metric("Min", numStr(dl.Min())) })
	cols[3](func() { sy.Metric("Max", numStr(dl.Max())) })
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

// numStr formats a float without trailing zeros (e.g. 3, 3.5, 3.14).
func numStr(f float64) string {
	return fmt.Sprintf("%g", f)
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
