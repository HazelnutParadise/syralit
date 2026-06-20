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
func Metrics(dt *insyra.DataTable, col string) {
	dl := dt.GetColByName(col)
	if dl == nil {
		sy.Warning(fmt.Sprintf("column %q not found", col))
		return
	}
	cols := sy.Columns(4)
	cols[0](func() { sy.Metric("Count", fmt.Sprintf("%d", dl.Len())) })
	cols[1](func() { sy.Metric("Mean", fmt.Sprintf("%.2f", dl.Mean())) })
	cols[2](func() { sy.Metric("Min", fmt.Sprint(dl.Min())) })
	cols[3](func() { sy.Metric("Max", fmt.Sprint(dl.Max())) })
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
	raw := dl.Data()
	nums := make([]float64, 0, len(raw))
	for _, v := range raw {
		if f, ok := toFloat64(v); ok {
			nums = append(nums, f)
		}
	}
	return nums
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
