package syinsyra

import (
	"bytes"
	"testing"

	"github.com/HazelnutParadise/insyra"
	"github.com/xuri/excelize/v2"
	"golang.org/x/text/encoding/traditionalchinese"
	"golang.org/x/text/encoding/unicode"
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

const csvZh = "股票,收盤\n台積電,1000\n鴻海,200\n聯發科,1500\n中華電,130\n"

func checkZhTable(t *testing.T, dt *insyra.DataTable) {
	t.Helper()
	if h := dt.Headers(); len(h) != 2 || h[0] != "股票" || h[1] != "收盤" {
		t.Fatalf("unexpected headers: %v", h)
	}
	rows := dt.To2DSlice()
	if len(rows) != 4 || rows[0][0] != "台積電" {
		t.Fatalf("unexpected rows: %v", rows)
	}
}

func TestParseCSVBig5(t *testing.T) {
	data, err := traditionalchinese.Big5.NewEncoder().Bytes([]byte(csvZh))
	if err != nil {
		t.Fatal(err)
	}
	dt, err := ParseTable("broker.csv", data)
	if err != nil {
		t.Fatal(err)
	}
	checkZhTable(t, dt)
}

func TestParseCSVUTF16BOM(t *testing.T) {
	data, err := unicode.UTF16(unicode.LittleEndian, unicode.UseBOM).NewEncoder().Bytes([]byte(csvZh))
	if err != nil {
		t.Fatal(err)
	}
	dt, err := ParseTable("excel.csv", data)
	if err != nil {
		t.Fatal(err)
	}
	checkZhTable(t, dt)
}

func TestParseCSVUTF8BOM(t *testing.T) {
	dt, err := ParseTable("excel.csv", append([]byte("\xef\xbb\xbf"), csvZh...))
	if err != nil {
		t.Fatal(err)
	}
	checkZhTable(t, dt)
}

func TestParseTableWithRawStrings(t *testing.T) {
	csv := []byte("code,name\n0050,元大台灣50\n0056,元大高股息\n")
	dt, err := ParseTableWith("etf.csv", csv, insyra.CSVReadOptions{FirstRowToColNames: true, RawStrings: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := dt.To2DSlice()[0][0]; got != "0050" {
		t.Fatalf("code = %#v, want \"0050\"", got)
	}
}

func TestParseTableWithExplicitEncoding(t *testing.T) {
	// A single short column of Big5 text is exactly what detection can guess wrong.
	data, err := traditionalchinese.Big5.NewEncoder().Bytes([]byte(csvZh))
	if err != nil {
		t.Fatal(err)
	}
	dt, err := ParseTableWith("broker.csv", data, insyra.CSVReadOptions{FirstRowToColNames: true, Encoding: "big5"})
	if err != nil {
		t.Fatal(err)
	}
	checkZhTable(t, dt)
	bad, err := ParseTableWith("broker.csv", data, insyra.CSVReadOptions{FirstRowToColNames: true, Encoding: "utf-8"})
	if err == nil && bad.Headers()[0] == "股票" {
		t.Fatal("forcing utf-8 on Big5 bytes must not decode correctly")
	}
}
