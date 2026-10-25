package syinsyra

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	sy "github.com/HazelnutParadise/syralit"

	"github.com/HazelnutParadise/insyra"
	"github.com/xuri/excelize/v2"
)

// UploadTable renders a file uploader and parses the uploaded file into an
// Insyra DataTable. The format is detected from the file extension: .csv,
// .xlsx/.xls, and .json are supported. Returns nil until a valid file is
// uploaded; parse errors are surfaced in the UI.
//
// Parquet is intentionally not handled here — it would drag Apache Arrow into
// every app that uses this adapter. Read parquet directly with
// github.com/HazelnutParadise/insyra/parquet and pass the result to Table.
func UploadTable(label string, opts ...sy.Option) *insyra.DataTable {
	return UploadTableWith(label, insyra.CSVReadOptions{FirstRowToColNames: true}, opts...)
}

// UploadTableWith is UploadTable with explicit CSV read options. Set
// csvOpts.Encoding to skip auto-detection, for example "big5", "gbk",
// "gb18030", "shift_jis" or "utf-8"; the default "" (or "auto") detects it.
// Let end users correct a wrong guess with a selector:
//
//	enc := sy.SelectBox("Encoding", []string{"auto", "utf-8", "big5", "gbk"})
//	dt := syi.UploadTableWith("CSV", insyra.CSVReadOptions{FirstRowToColNames: true, Encoding: enc})
//
// The options only affect .csv files.
func UploadTableWith(label string, csvOpts insyra.CSVReadOptions, opts ...sy.Option) *insyra.DataTable {
	f := sy.FileUploader(label, opts...)
	if f == nil || len(f.Data) == 0 {
		return nil
	}
	dt, err := ParseTableWith(f.Name, f.Data, csvOpts)
	if err != nil {
		sy.Exception(err)
		return nil
	}
	return dt
}

// ParseTable parses raw file bytes into a DataTable by file-name extension.
// Exposed so callers can parse files from sources other than the uploader
// (embedded assets, downloads, …). CSV files are read with the first row as
// column names and their encoding detected automatically (UTF-8, Big5, GBK,
// UTF-16 with BOM, …). Use ParseTableWith to change the CSV options.
func ParseTable(name string, data []byte) (*insyra.DataTable, error) {
	return ParseTableWith(name, data, insyra.CSVReadOptions{FirstRowToColNames: true})
}

// ParseTableWith is ParseTable with explicit CSV read options, for example
// RawStrings to keep IDs such as "0050" as text or Encoding (e.g. "big5") to
// skip encoding detection. An unrecognised Encoding name leaves the bytes
// undecoded, so use common names such as utf-8, big5, gbk, gb18030, shift_jis.
// The options only affect .csv files.
func ParseTableWith(name string, data []byte, csvOpts insyra.CSVReadOptions) (*insyra.DataTable, error) {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".csv":
		return insyra.ReadCSV(bytes.NewReader(data), csvOpts)
	case ".xlsx", ".xls":
		return parseExcel(data)
	case ".json":
		var v any
		if err := json.Unmarshal(data, &v); err != nil {
			return nil, fmt.Errorf("invalid JSON: %w", err)
		}
		return insyra.ReadJSON(v)
	default:
		ext := filepath.Ext(name)
		if ext == "" {
			ext = "(none)"
		}
		return nil, fmt.Errorf("unsupported file type %s — use .csv, .xlsx or .json", ext)
	}
}

func parseExcel(data []byte) (*insyra.DataTable, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to open Excel file: %w", err)
	}
	defer func() { _ = f.Close() }()
	sheet := f.GetSheetName(f.GetActiveSheetIndex())
	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, fmt.Errorf("failed to read sheet %q: %w", sheet, err)
	}
	dt, err := insyra.ReadSlice2D(rows)
	if err != nil {
		return nil, err
	}
	dt.SetRowToColNames(0)
	return dt, nil
}
