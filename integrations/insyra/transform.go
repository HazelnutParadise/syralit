package syinsyra

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	sy "github.com/HazelnutParadise/syralit"

	"github.com/HazelnutParadise/insyra"
)

// cclTimeout bounds a single CCL evaluation. Insyra's CCL engine can run away
// on certain malformed input (an infinite loop a deferred recover cannot
// catch), so every evaluation is run in a goroutine and abandoned if it
// overruns. A well-formed formula completes in well under a millisecond.
const cclTimeout = 3 * time.Second

// FilterBuilder renders an interactive row filter over a DataTable (column +
// operator + value) and returns the filtered DataTable, also rendering it. The
// source table is never mutated (it operates on a Clone). Use the returned
// table for downstream steps (charts, stats, …).
func FilterBuilder(dt *insyra.DataTable, opts ...sy.Option) *insyra.DataTable {
	if dt == nil {
		sy.Warning("nil DataTable")
		return nil
	}
	headers := dt.Headers()
	if len(headers) == 0 {
		Table(dt, opts...)
		return dt
	}

	var col, op, val string
	c := sy.Columns(3)
	c[0](func() { col = sy.SelectBox("Column", headers, sy.Key("syi_flt_col")) })
	c[1](func() {
		op = sy.SelectBox("Operator", []string{">", ">=", "<", "<=", "==", "!=", "contains"}, sy.Key("syi_flt_op"))
	})
	c[2](func() { val = sy.TextInput("Value", sy.Key("syi_flt_val")) })

	result := dt
	if col != "" && strings.TrimSpace(val) != "" {
		want := strings.TrimSpace(val)
		if target := dt.GetColByName(col); target != nil {
			data := target.Data()
			keep := make([]bool, len(data))
			for i, v := range data {
				keep[i] = matchOp(v, op, want)
			}
			// Filter is cell-level: returning the row's verdict for every cell
			// keeps a matching row whole and drops a non-matching row entirely.
			result = dt.Clone().Filter(func(rowIdx int, _ string, _ any) bool {
				return rowIdx >= 0 && rowIdx < len(keep) && keep[rowIdx]
			})
		}
	}

	sy.Caption(fmt.Sprintf("%d row(s)", len(result.To2DSlice())))
	Table(result, opts...)
	return result
}

// CCLBuilder renders an interactive CCL (column-computation language) box that
// adds a computed column to a DataTable and returns the result, also rendering
// it. The source table is never mutated. The formula is applied only when the
// user presses Apply, and each evaluation is time-bounded — a formula that
// fails or runs away is reported in the UI and not applied, so it can never
// hang or crash the app.
//
// Example formula (referencing columns by their index letter): A * B + 1
func CCLBuilder(dt *insyra.DataTable, opts ...sy.Option) *insyra.DataTable {
	if dt == nil {
		sy.Warning("nil DataTable")
		return nil
	}

	applied := sy.State("syi_ccl_applied", "")
	appliedName := sy.State("syi_ccl_applied_name", "computed")

	var name, formula string
	c := sy.Columns(2)
	c[0](func() { name = sy.TextInput("New column", sy.Key("syi_ccl_name"), sy.DefaultValue("computed")) })
	c[1](func() { formula = sy.TextInput("CCL formula (e.g. A * B)", sy.Key("syi_ccl_formula")) })

	if sy.Button("Apply", sy.Key("syi_ccl_apply")) {
		f := strings.TrimSpace(formula)
		n := strings.TrimSpace(name)
		if n == "" {
			n = "computed"
		}
		if f == "" {
			applied.Clear()
		} else if _, ok := runCCL(dt.Clone(), n, f); ok {
			applied.Set(f)
			appliedName.Set(n)
		} else {
			sy.Warning("CCL formula failed or timed out — not applied")
			applied.Clear()
		}
	}

	result := dt
	if f := applied.Get(); f != "" {
		if out, ok := runCCL(dt.Clone(), appliedName.Get(), f); ok {
			result = out
		}
	}

	Table(result, opts...)
	return result
}

// runCCL applies a CCL formula to dt in a goroutine, returning (result, true)
// on success or (nil, false) if it panics or exceeds cclTimeout. dt should be a
// Clone — it is mutated in place by AddColUsingCCL.
func runCCL(dt *insyra.DataTable, colName, formula string) (*insyra.DataTable, bool) {
	ch := make(chan *insyra.DataTable, 1)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				ch <- nil
			}
		}()
		ch <- dt.AddColUsingCCL(colName, formula)
	}()
	select {
	case out := <-ch:
		return out, out != nil
	case <-time.After(cclTimeout):
		return nil, false
	}
}

// matchOp evaluates a single cell against an operator and target value. It
// compares numerically when both sides parse as numbers, otherwise as strings.
func matchOp(cell any, op, want string) bool {
	if cf, ok := toFloat64(cell); ok {
		if wf, err := strconv.ParseFloat(want, 64); err == nil {
			switch op {
			case ">":
				return cf > wf
			case ">=":
				return cf >= wf
			case "<":
				return cf < wf
			case "<=":
				return cf <= wf
			case "==":
				return cf == wf
			case "!=":
				return cf != wf
			case "contains":
				return strings.Contains(fmt.Sprint(cell), want)
			}
		}
	}
	s := fmt.Sprint(cell)
	switch op {
	case "==":
		return s == want
	case "!=":
		return s != want
	case "contains":
		return strings.Contains(s, want)
	case ">":
		return s > want
	case ">=":
		return s >= want
	case "<":
		return s < want
	case "<=":
		return s <= want
	}
	return true
}
