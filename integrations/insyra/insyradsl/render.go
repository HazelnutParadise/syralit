package insyradsl

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/HazelnutParadise/insyra"
)

// renderSpec describes how to turn a DSL result into one rendered element. It is
// shared by the Go widget (emits Syralit widgets) and the Artifact component
// (builds a Node), so both stay behaviourally identical.
type renderSpec struct {
	kind        string   // dataframe|table|line_chart|bar_chart|area_chart|pie_chart|metric|text
	output      string   // variable to render; empty = auto-resolve
	x           string   // chart x-axis column (labels)
	y           []string // chart y-axis columns (series); empty = all numeric except x
	label       string   // pie label column
	value       string   // pie value column / metric value column
	title       string   // chart title
	metricLabel string   // metric label
	height      int
}

// resolveOutputVar picks which variable to render. An explicit name wins;
// otherwise it prefers $result (where unnamed transforms land) and finally the
// sole DataTable/DataList in the result.
func resolveOutputVar(res DSLResult, name string) (any, string, bool) {
	if name != "" {
		v, ok := res.Vars[name]
		return v, name, ok
	}
	if v, ok := res.Vars["$result"]; ok {
		return v, "$result", true
	}
	var found any
	var foundName string
	count := 0
	for k, v := range res.Vars {
		switch v.(type) {
		case *insyra.DataTable, *insyra.DataList:
			found, foundName = v, k
			count++
		}
	}
	if count == 1 {
		return found, foundName, true
	}
	return nil, "", false
}

// tableData returns the headers and row matrix of a DataTable/DataList for
// table-like rendering. A DataList becomes a single named column. Unnamed
// DataTable columns (e.g. from newdt) get Excel-style labels (A, B, ...) so the
// table is never headerless.
func tableData(v any) (headers []string, rows [][]any, ok bool) {
	switch t := v.(type) {
	case *insyra.DataTable:
		headers := displayHeaders(t.Headers())
		rows := t.To2DSlice()
		// Surface row names (e.g. describe's stat labels) as a leading column so
		// the values are not shown unlabeled.
		labels := make([]string, len(rows))
		labeled := false
		for i := range rows {
			if name, named := t.GetRowNameByIndex(i); named && name != "" {
				labels[i] = name
				labeled = true
			}
		}
		if labeled {
			headers = append([]string{""}, headers...)
			for i := range rows {
				rows[i] = append([]any{labels[i]}, rows[i]...)
			}
		}
		return headers, rows, true
	case *insyra.DataList:
		data := t.Data()
		rows = make([][]any, len(data))
		for i, cell := range data {
			rows[i] = []any{cell}
		}
		return []string{dataListName(t)}, rows, true
	default:
		return nil, nil, false
	}
}

// chartSeries extracts x-axis labels and numeric series from a DataTable for
// line/bar/area charts. Columns are addressed by name, Excel-style letter
// (A, B, ...), or 0-based index. yCols empty means "every numeric column except
// x".
func chartSeries(v any, xCol string, yCols []string) (xLabels []string, series map[string][]float64, err error) {
	dt, ok := v.(*insyra.DataTable)
	if !ok {
		if dl, isList := v.(*insyra.DataList); isList {
			return nil, map[string][]float64{dataListName(dl): numericValues(dl.Data())}, nil
		}
		return nil, nil, fmt.Errorf("chart requires a DataTable or DataList output")
	}
	headers := dt.Headers()

	xIdx := -1
	if xCol != "" {
		col, idx, found := resolveColumn(dt, headers, xCol)
		if !found {
			return nil, nil, fmt.Errorf("x column %q not found", xCol)
		}
		xIdx = idx
		for _, cell := range col.Data() {
			xLabels = append(xLabels, fmt.Sprint(cell))
		}
	}

	type ycol struct {
		name string
		data []any
	}
	var chosen []ycol
	if len(yCols) == 0 {
		for i := range headers {
			if i == xIdx {
				continue
			}
			col := dt.GetColByNumber(i)
			if col != nil && hasNumeric(col.Data()) {
				chosen = append(chosen, ycol{name: headerLabel(headers, i), data: col.Data()})
			}
		}
	} else {
		for _, ref := range yCols {
			col, idx, found := resolveColumn(dt, headers, ref)
			if !found {
				return nil, nil, fmt.Errorf("y column %q not found", ref)
			}
			chosen = append(chosen, ycol{name: headerLabel(headers, idx), data: col.Data()})
		}
	}
	if len(chosen) == 0 {
		return nil, nil, fmt.Errorf("no numeric y columns to chart")
	}

	series = make(map[string][]float64, len(chosen))
	for _, c := range chosen {
		series[c.name] = numericValues(c.data)
	}
	return xLabels, series, nil
}

// pieData maps a label column to a value column for a pie chart. Columns are
// addressed by name, Excel-style letter, or index.
func pieData(v any, labelCol, valueCol string) (map[string]float64, error) {
	dt, ok := v.(*insyra.DataTable)
	if !ok {
		return nil, fmt.Errorf("pie_chart requires a DataTable output")
	}
	if labelCol == "" || valueCol == "" {
		return nil, fmt.Errorf("pie_chart requires both label and value columns")
	}
	headers := dt.Headers()
	labels, _, lok := resolveColumn(dt, headers, labelCol)
	if !lok {
		return nil, fmt.Errorf("label column %q not found", labelCol)
	}
	values, _, vok := resolveColumn(dt, headers, valueCol)
	if !vok {
		return nil, fmt.Errorf("value column %q not found", valueCol)
	}
	labelData := labels.Data()
	valueData := numericValues(values.Data())
	n := len(labelData)
	if len(valueData) < n {
		n = len(valueData)
	}
	out := make(map[string]float64, n)
	for i := 0; i < n; i++ {
		out[fmt.Sprint(labelData[i])] = valueData[i]
	}
	return out, nil
}

// resolveColumn finds a column by name, Excel-style letter (A, B, ...), or
// 0-based index. It matches header names manually (rather than GetColByName) to
// avoid Insyra's not-found warning log for the letter/index fallbacks.
func resolveColumn(dt *insyra.DataTable, headers []string, ref string) (*insyra.DataList, int, bool) {
	for i, h := range headers {
		if h != "" && h == ref {
			return dt.GetColByNumber(i), i, true
		}
	}
	if idx, ok := excelIndex(ref); ok && idx >= 0 && idx < len(headers) {
		return dt.GetColByNumber(idx), idx, true
	}
	if n, err := strconv.Atoi(ref); err == nil && n >= 0 && n < len(headers) {
		return dt.GetColByNumber(n), n, true
	}
	return nil, -1, false
}

// displayHeaders replaces empty column names with Excel-style labels.
func displayHeaders(headers []string) []string {
	out := make([]string, len(headers))
	for i, h := range headers {
		out[i] = headerLabel(headers, i)
		_ = h
	}
	return out
}

// headerLabel returns the column's name, or its Excel-style label when unnamed.
func headerLabel(headers []string, i int) string {
	if i >= 0 && i < len(headers) && headers[i] != "" {
		return headers[i]
	}
	return excelLabel(i)
}

// excelIndex parses an Excel-style column letter (A=0, B=1, ..., AA=26) to a
// 0-based index. It returns false for anything that isn't all letters.
func excelIndex(ref string) (int, bool) {
	ref = strings.ToUpper(strings.TrimSpace(ref))
	if ref == "" {
		return 0, false
	}
	idx := 0
	for _, r := range ref {
		if r < 'A' || r > 'Z' {
			return 0, false
		}
		idx = idx*26 + int(r-'A'+1)
	}
	return idx - 1, true
}

// excelLabel is the inverse of excelIndex: 0 -> "A", 25 -> "Z", 26 -> "AA".
func excelLabel(i int) string {
	if i < 0 {
		return ""
	}
	i++
	var b []byte
	for i > 0 {
		i--
		b = append([]byte{byte('A' + i%26)}, b...)
		i /= 26
	}
	return string(b)
}

// scalarValue renders a single value for a metric: a scalar as-is, or the first
// element of a one-value DataList.
func scalarValue(v any) (string, bool) {
	switch t := v.(type) {
	case *insyra.DataList:
		data := t.Data()
		if len(data) == 0 {
			return "", false
		}
		return fmt.Sprint(data[0]), true
	case *insyra.DataTable:
		return "", false
	case nil:
		return "", false
	default:
		return fmt.Sprint(t), true
	}
}

// dataListName returns the DataList's name, or "Value" when unnamed, so a
// single-column table always has a header.
func dataListName(dl *insyra.DataList) string {
	if name := dl.GetName(); name != "" {
		return name
	}
	return "Value"
}

// numericValues extracts the numeric entries of a slice, skipping non-numerics.
func numericValues(data []any) []float64 {
	out := make([]float64, 0, len(data))
	for _, v := range data {
		if f, ok := toFloat64(v); ok {
			out = append(out, f)
		}
	}
	return out
}

func hasNumeric(data []any) bool {
	for _, v := range data {
		if _, ok := toFloat64(v); ok {
			return true
		}
	}
	return false
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
	case string:
		if f, err := strconv.ParseFloat(strings.TrimSpace(x), 64); err == nil && !math.IsNaN(f) {
			return f, true
		}
	}
	return 0, false
}

// stringRows flattens an [][]any into [][]string for the plain "table" node,
// which expects string cells.
func stringRows(rows [][]any) [][]string {
	out := make([][]string, len(rows))
	for i, row := range rows {
		cells := make([]string, len(row))
		for j, cell := range row {
			cells[j] = fmt.Sprint(cell)
		}
		out[i] = cells
	}
	return out
}
