package syinsyra

import (
	"fmt"
	"strings"

	sy "github.com/HazelnutParadise/syralit"

	"github.com/HazelnutParadise/insyra"
	"github.com/HazelnutParadise/insyra/stats"
)

// Describe renders a full summary table (count, mean, std, min, quartiles, max,
// …) for every column of a DataTable, using Insyra's DataTable.Describe.
func Describe(dt *insyra.DataTable, opts ...sy.Option) {
	if dt == nil {
		sy.Warning("nil DataTable")
		return
	}
	renderTableWithRowNames(dt.Describe(), "Statistic", opts...)
}

// CorrelationMatrix renders the pairwise correlation matrix of a DataTable's
// numeric columns. method is "pearson" (default), "spearman", or "kendall".
func CorrelationMatrix(dt *insyra.DataTable, method string, opts ...sy.Option) {
	if dt == nil {
		sy.Warning("nil DataTable")
		return
	}
	corr, _, err := stats.CorrelationMatrix(dt, corrMethod(method))
	if err != nil {
		sy.Exception(err)
		return
	}
	renderTableWithRowNames(corr, "", opts...)
}

// Correlation renders the correlation coefficient and p-value between two
// numeric columns as Metric widgets.
func Correlation(dt *insyra.DataTable, colX, colY, method string) {
	if dt == nil {
		sy.Warning("nil DataTable")
		return
	}
	x, y := dt.GetColByName(colX), dt.GetColByName(colY)
	if x == nil || y == nil {
		sy.Warning("column not found")
		return
	}
	res, err := stats.Correlation(x, y, corrMethod(method))
	if err != nil {
		sy.Exception(err)
		return
	}
	cols := sy.Columns(2)
	cols[0](func() { sy.Metric(fmt.Sprintf("r (%s)", methodName(method)), statStr(res.Statistic, 3)) })
	cols[1](func() { sy.Metric("p-value", statStr(res.PValue, 4)) })
}

// LinearRegression fits yCol against one or more xCols and renders the fit
// (R², adjusted R², coefficients with std error / t / p). For a single
// predictor it also shows the equation and a scatter of the data.
func LinearRegression(dt *insyra.DataTable, yCol string, xCols ...string) {
	if dt == nil || len(xCols) == 0 {
		sy.Warning("LinearRegression needs a DataTable and at least one xCol")
		return
	}
	y := dt.GetColByName(yCol)
	if y == nil {
		sy.Warning(fmt.Sprintf("column %q not found", yCol))
		return
	}
	xs := make([]insyra.IDataList, 0, len(xCols))
	for _, c := range xCols {
		dl := dt.GetColByName(c)
		if dl == nil {
			sy.Warning(fmt.Sprintf("column %q not found", c))
			return
		}
		xs = append(xs, dl)
	}
	res, err := stats.LinearRegression(y, xs...)
	if err != nil {
		sy.Exception(err)
		return
	}

	m := sy.Columns(3)
	m[0](func() { sy.Metric("R²", statStr(res.RSquared, 3)) })
	m[1](func() { sy.Metric("Adj. R²", statStr(res.AdjustedRSquared, 3)) })
	m[2](func() { sy.Metric("p-value", statStr(res.PValue, 4)) })

	// Coefficient table.
	rows := [][]string{{"(Intercept)", statStr(res.Intercept, 4), statStr(res.StandardErrorIntercept, 4), statStr(res.TValueIntercept, 3), statStr(res.PValueIntercept, 4)}}
	if len(res.Coefficients) == len(xCols) {
		for i, c := range xCols {
			rows = append(rows, []string{c, statStr(res.Coefficients[i], 4), coefAt(res.StandardErrors, i), coefAt(res.TValues, i), coefAt(res.PValues, i)})
		}
	} else {
		// Simple regression exposes a single Slope.
		rows = append(rows, []string{strings.Join(xCols, ", "), statStr(res.Slope, 4), statStr(res.StandardError, 4), statStr(res.TValue, 3), statStr(res.PValue, 4)})
	}
	sy.Table([]string{"Term", "Coef", "Std.Err", "t", "p"}, rows)

	if len(xCols) == 1 {
		sy.Caption(fmt.Sprintf("%s = %.4g + %.4g·%s", yCol, res.Intercept, res.Slope, xCols[0]))
		x := dt.GetColByName(xCols[0])
		ScatterFromLists(xCols[0], yCol, x, y)
	}
}

// TTest renders an independent two-sample t-test between two numeric columns.
func TTest(dt *insyra.DataTable, col1, col2 string, equalVariance bool) {
	if dt == nil {
		sy.Warning("nil DataTable")
		return
	}
	a, b := dt.GetColByName(col1), dt.GetColByName(col2)
	if a == nil || b == nil {
		sy.Warning("column not found")
		return
	}
	res, err := stats.TwoSampleTTest(a, b, equalVariance)
	if err != nil {
		sy.Exception(err)
		return
	}
	cols := sy.Columns(3)
	cols[0](func() { sy.Metric("t", statStr(res.Statistic, 3)) })
	cols[1](func() { sy.Metric("p-value", statStr(res.PValue, 4)) })
	cols[2](func() {
		diff := ""
		if res.Mean != nil && res.Mean2 != nil {
			diff = statStr(*res.Mean-*res.Mean2, 3)
		}
		sy.Metric("Mean diff", diff)
	})
}

// --- helpers ---

// ScatterFromLists renders a scatter chart from two equal-length DataLists.
func ScatterFromLists(xName, yName string, x, y *insyra.DataList) {
	xn, yn := numericList(x), numericList(y)
	n := min(len(xn), len(yn))
	if n == 0 {
		return
	}
	points := make([][2]float64, n)
	for i := range n {
		points[i] = [2]float64{xn[i], yn[i]}
	}
	sy.ScatterChart(map[string][][2]float64{yName + " vs " + xName: points})
}

func corrMethod(s string) stats.CorrelationMethod {
	switch strings.ToLower(s) {
	case "spearman":
		return stats.SpearmanCorrelation
	case "kendall":
		return stats.KendallCorrelation
	default:
		return stats.PearsonCorrelation
	}
}

func methodName(s string) string {
	switch strings.ToLower(s) {
	case "spearman":
		return "Spearman"
	case "kendall":
		return "Kendall"
	default:
		return "Pearson"
	}
}

func coefAt(s []float64, i int) string {
	if i < len(s) {
		return statStr(s[i], 4)
	}
	return "—"
}

// renderTableWithRowNames renders an Insyra DataTable as a Syralit DataFrame,
// prepending each row's name as a leading column (for describe / matrices).
func renderTableWithRowNames(dt *insyra.DataTable, firstCol string, opts ...sy.Option) {
	if dt == nil {
		sy.Warning("nil result")
		return
	}
	headers := append([]string{firstCol}, dt.Headers()...)
	names := dt.RowNames()
	data := dt.To2DSlice()
	rows := make([][]any, len(data))
	for i, r := range data {
		name := ""
		if i < len(names) {
			name = names[i]
		}
		rows[i] = append([]any{name}, r...)
	}
	sy.DataFrame(headers, rows, opts...)
}
