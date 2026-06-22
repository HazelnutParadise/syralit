package syinsyra

import (
	"testing"

	sy "github.com/HazelnutParadise/syralit"

	"github.com/HazelnutParadise/insyra"
)

func abTable() *insyra.DataTable {
	return insyra.NewDataTable(
		insyra.NewDataList(1.0, 2, 3).SetName("A"),
		insyra.NewDataList(10.0, 20, 30).SetName("B"),
	)
}

func TestMatchOp(t *testing.T) {
	cases := []struct {
		cell any
		op   string
		want string
		out  bool
	}{
		{5.0, ">", "3", true},
		{5.0, "<", "3", false},
		{3.0, "==", "3", true},
		{3.0, "!=", "3", false},
		{"hello", "contains", "ell", true},
		{"hello", "==", "hello", true},
		{"apple", ">", "banana", false},
	}
	for _, c := range cases {
		if got := matchOp(c.cell, c.op, c.want); got != c.out {
			t.Errorf("matchOp(%v, %q, %q) = %v, want %v", c.cell, c.op, c.want, got, c.out)
		}
	}
}

func TestRunCCL(t *testing.T) {
	out, ok := runCCL(abTable().Clone(), "Sum", "A + B")
	if !ok {
		t.Fatal("expected runCCL to succeed on a valid formula")
	}
	if h := out.Headers(); len(h) != 3 || h[2] != "Sum" {
		t.Fatalf("expected 3 columns ending in Sum, got %v", h)
	}
	got := out.GetColByName("Sum").Data()
	want := []float64{11, 22, 33}
	for i, w := range want {
		if f, ok := toFloat64(got[i]); !ok || f != w {
			t.Fatalf("Sum[%d] = %v, want %v", i, got[i], w)
		}
	}
}

func TestFilterBuilderRendersPassthrough(t *testing.T) {
	// With no value entered (RenderOnce has no interaction), FilterBuilder
	// passes the table through and renders it as a dataframe.
	var ret *insyra.DataTable
	tree := sy.RenderOnce(func() { ret = FilterBuilder(abTable()) })
	if len(tree.Find("dataframe")) != 1 {
		t.Fatalf("expected a dataframe, got %d", len(tree.Find("dataframe")))
	}
	if ret == nil || len(ret.To2DSlice()) != 3 {
		t.Fatal("expected pass-through of all 3 rows")
	}
}

func TestCCLBuilderRenders(t *testing.T) {
	tree := sy.RenderOnce(func() { CCLBuilder(abTable()) })
	if len(tree.Find("dataframe")) != 1 {
		t.Fatalf("expected a dataframe, got %d", len(tree.Find("dataframe")))
	}
}
