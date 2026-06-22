package syinsyra

import (
	"bytes"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParseCSV(t *testing.T) {
	dt, err := ParseTable("data.csv", []byte("Name,Sales\nAlice,10\nBob,20\nCarol,30"))
	if err != nil {
		t.Fatal(err)
	}
	if h := dt.Headers(); len(h) != 2 || h[0] != "Name" || h[1] != "Sales" {
		t.Fatalf("unexpected headers: %v", h)
	}
	if rows := dt.To2DSlice(); len(rows) != 3 {
		t.Fatalf("expected 3 data rows, got %d", len(rows))
	}
}

func TestParseExcel(t *testing.T) {
	f := excelize.NewFile()
	sheet := f.GetSheetName(0)
	_ = f.SetSheetRow(sheet, "A1", &[]any{"Name", "Sales"})
	_ = f.SetSheetRow(sheet, "A2", &[]any{"Alice", 10})
	_ = f.SetSheetRow(sheet, "A3", &[]any{"Bob", 20})
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}

	dt, err := ParseTable("report.xlsx", buf.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	if h := dt.Headers(); len(h) != 2 || h[0] != "Name" || h[1] != "Sales" {
		t.Fatalf("unexpected headers: %v", h)
	}
	if rows := dt.To2DSlice(); len(rows) != 2 {
		t.Fatalf("expected 2 data rows, got %d", len(rows))
	}
}

func TestParseUnsupported(t *testing.T) {
	if _, err := ParseTable("notes.txt", []byte("hello")); err == nil {
		t.Fatal("expected an error for unsupported extension")
	}
}
