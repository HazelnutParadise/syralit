package syinsyra

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"time"

	sy "github.com/HazelnutParadise/syralit"

	"github.com/HazelnutParadise/insyra"
)

// DownloadCSV renders a download button that exports the DataTable as a CSV
// file — the natural exit point after filtering or editing a table.
func DownloadCSV(label string, dt *insyra.DataTable, filename string, opts ...sy.Option) {
	if dt == nil {
		sy.Warning("nil DataTable")
		return
	}
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write(dt.Headers())
	for _, row := range dt.To2DSlice() {
		rec := make([]string, len(row))
		for i, v := range row {
			if v != nil {
				rec[i] = fmt.Sprint(v)
			}
		}
		_ = w.Write(rec)
	}
	w.Flush()
	sy.DownloadButton(label, buf.Bytes(), filename, append([]sy.Option{sy.MimeType("text/csv")}, opts...)...)
}

// RollingMeanChart plots a numeric column together with its rolling mean over
// the given window — raw values and the smoothed series in one line chart.
// Supports sy.Selectable() like the other chart helpers.
func RollingMeanChart(dt *insyra.DataTable, xCol, yCol string, window int, opts ...sy.Option) *sy.ChartSelection {
	dl := columnList(dt, yCol)
	if dl == nil {
		return nil
	}
	raw := numericList(dl)
	rolled := numericList(dl.Rolling(insyra.RollingOptions{Window: window}).Mean())
	series := map[string][]float64{
		yCol: raw,
		fmt.Sprintf("%s (rolling %d)", yCol, window): rolled,
	}
	return sy.LineChart(series, withXLabels(dt, xCol, opts)...)
}

// CumSumChart plots a numeric column's cumulative sum as a line chart.
func CumSumChart(dt *insyra.DataTable, xCol, yCol string, opts ...sy.Option) *sy.ChartSelection {
	if dt == nil {
		sy.Warning("nil DataTable")
		return nil
	}
	cum := numericList(dt.CumSumCol(yCol))
	if len(cum) == 0 {
		sy.Warning(fmt.Sprintf("column %q has no numeric values", yCol))
		return nil
	}
	series := map[string][]float64{yCol + " (cumsum)": cum}
	return sy.LineChart(series, withXLabels(dt, xCol, opts)...)
}

// PctChangeChart plots a numeric column's percentage change over the given
// number of periods as a bar chart.
func PctChangeChart(dt *insyra.DataTable, xCol, yCol string, periods int, opts ...sy.Option) *sy.ChartSelection {
	if dt == nil {
		sy.Warning("nil DataTable")
		return nil
	}
	pct := numericList(dt.PctChangeCol(yCol, periods))
	if len(pct) == 0 {
		sy.Warning(fmt.Sprintf("column %q has no numeric values", yCol))
		return nil
	}
	series := map[string][]float64{fmt.Sprintf("%s (%%chg %d)", yCol, periods): pct}
	return sy.BarChart(series, withXLabels(dt, xCol, opts)...)
}

// AddFormulaColumn renders a formula editor (CCL expression + column name)
// and returns a NEW DataTable with the computed column appended — the source
// table is untouched, and it comes back unchanged while the formula is empty
// or invalid. Columns are referenced by letter (A, B, ...) or by name in
// brackets: ["Revenue"] - ["Cost"].
//
// The evaluation runs in a goroutine guarded by a timeout: CCL can loop on
// certain malformed inputs, and a UI must never hang on user input. On
// timeout the goroutine is abandoned (it may leak until it finishes) and the
// user sees an error instead.
func AddFormulaColumn(dt *insyra.DataTable, key string) *insyra.DataTable {
	if dt == nil {
		sy.Warning("nil DataTable")
		return nil
	}
	formula := sy.TextInput("ƒx CCL formula (new column)",
		sy.Key(key+"_formula"), sy.Mono(),
		sy.Placeholder(`["Revenue"] - ["Cost"]  ·  IF(["Revenue"] > 1000, 'high', 'low')`))
	name := sy.TextInput("Column name",
		sy.Key(key+"_name"), sy.Placeholder("computed"))
	if formula == "" {
		return dt
	}
	if name == "" {
		name = "computed"
	}
	out, err := evalCCL(dt, name, formula, 3*time.Second)
	if err != nil {
		sy.Error("CCL: " + err.Error())
		return dt
	}
	return out
}

// evalCCL runs AddColUsingCCL on a copy of dt with a hard timeout.
func evalCCL(dt *insyra.DataTable, name, formula string, timeout time.Duration) (*insyra.DataTable, error) {
	// Work on a copy so a failed/partial evaluation never mutates the source.
	cp := tableFromRows(dt.Headers(), dt.To2DSlice())
	done := make(chan error, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				done <- fmt.Errorf("formula panicked: %v", r)
			}
		}()
		cp.AddColUsingCCL(name, formula)
		done <- nil
	}()
	select {
	case err := <-done:
		if err != nil {
			return nil, err
		}
		return cp, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("formula evaluation timed out after %s", timeout)
	}
}

// columnList fetches a column with UI warnings on failure.
func columnList(dt *insyra.DataTable, col string) *insyra.DataList {
	if dt == nil {
		sy.Warning("nil DataTable")
		return nil
	}
	dl := dt.GetColByName(col)
	if dl == nil {
		sy.Warning(fmt.Sprintf("column %q not found", col))
	}
	return dl
}
