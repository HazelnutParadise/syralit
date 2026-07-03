package insyradsl

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/HazelnutParadise/insyra"
	sy "github.com/HazelnutParadise/syralit"
)

func TestRunDSLComputeGroupby(t *testing.T) {
	script := `
newdl North South North West as region
newdl 10 20 30 40 as revenue
newdt region revenue as sales
setcolnames sales region revenue
groupby sales by region agg revenue:sum:total as report
`
	res := RunDSL(script)
	if res.Err != nil {
		t.Fatalf("RunDSL error: %v", res.Err)
	}
	dt, ok := res.Vars["report"].(*insyra.DataTable)
	if !ok {
		t.Fatalf("report is %T, want *insyra.DataTable", res.Vars["report"])
	}
	if dt.NumRows() != 3 {
		t.Fatalf("want 3 groups, got %d", dt.NumRows())
	}
}

func TestRunDSLSafeModeDenies(t *testing.T) {
	denied := []string{
		"load data.csv as t",
		"save t out.csv",
		"db connect main sqlite:x.db",
		"fetch stock AAPL",
		"run script.isr",
		"env create scratch",
		"config get x",
		"plot t bar",
	}
	for _, cmd := range denied {
		res := RunDSL(cmd)
		if res.Err == nil || !strings.Contains(res.Err.Error(), "not allowed") {
			t.Errorf("command %q: want 'not allowed' error, got %v", cmd, res.Err)
		}
	}
}

func TestRunDSLUnrestrictedTogglesWhitelist(t *testing.T) {
	safe := RunDSL("load definitely-missing.csv as t")
	if safe.Err == nil || !strings.Contains(safe.Err.Error(), "not allowed") {
		t.Fatalf("safe mode should block load, got %v", safe.Err)
	}
	unres := RunDSL("load definitely-missing.csv as t", Unrestricted())
	if unres.Err == nil {
		t.Fatalf("loading a missing file should still error")
	}
	if strings.Contains(unres.Err.Error(), "not allowed") {
		t.Fatalf("unrestricted mode should pass the whitelist, got %v", unres.Err)
	}
}

func TestRunDSLInject(t *testing.T) {
	dl := insyra.NewDataList(3, 1, 2)
	dl.SetName("x")
	res := RunDSL("rank x as rx", WithVars(map[string]any{"x": dl}))
	if res.Err != nil {
		t.Fatalf("RunDSL error: %v", res.Err)
	}
	if _, ok := res.Vars["rx"].(*insyra.DataList); !ok {
		t.Fatalf("rx is %T, want *insyra.DataList", res.Vars["rx"])
	}
}

func TestRunDSLMaxLines(t *testing.T) {
	res := RunDSL("newdl 1 as a\nnewdl 2 as b\nnewdl 3 as c", MaxLines(2))
	if res.Err == nil || !strings.Contains(res.Err.Error(), "exceeds limit") {
		t.Fatalf("want line-limit error, got %v", res.Err)
	}
}

func TestRunDSLEnvRootPersists(t *testing.T) {
	root := t.TempDir()

	res1 := RunDSL("newdl 1 2 3 4 5 as x", EnvRoot(root))
	if res1.Err != nil {
		t.Fatalf("run 1: %v", res1.Err)
	}
	statePath := filepath.Join(root, "envs", "default", "state.json")
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("EnvRoot should persist state at %s: %v", statePath, err)
	}

	// A second run against the same root restores x from the first.
	res2 := RunDSL("mean x", EnvRoot(root))
	if res2.Err != nil {
		t.Fatalf("run 2: %v", res2.Err)
	}
	if _, ok := res2.Vars["x"].(*insyra.DataList); !ok {
		t.Fatalf("x not restored across runs: %T", res2.Vars["x"])
	}
	if !strings.Contains(res2.Output, "3") { // mean(1..5) = 3
		t.Fatalf("mean output = %q, want it to contain 3", res2.Output)
	}
}

func TestChartSeriesByExcelLetter(t *testing.T) {
	res := RunDSL("newdl 10 20 30 as a\nnewdl 1 2 3 as b\nnewdt a b as t")
	if res.Err != nil {
		t.Fatal(res.Err)
	}
	xLabels, series, err := chartSeries(res.Vars["t"], "A", []string{"B"})
	if err != nil {
		t.Fatal(err)
	}
	if len(xLabels) != 3 || xLabels[0] != "10" {
		t.Fatalf("xLabels = %v", xLabels)
	}
	got := series["B"]
	if len(got) != 3 || got[0] != 1 || got[2] != 3 {
		t.Fatalf("series B = %v", got)
	}
}

func TestExcelIndexRoundTrip(t *testing.T) {
	cases := map[string]int{"A": 0, "B": 1, "Z": 25, "AA": 26, "AB": 27}
	for letter, want := range cases {
		if got, ok := excelIndex(letter); !ok || got != want {
			t.Errorf("excelIndex(%q) = %d,%v; want %d", letter, got, ok, want)
		}
		if got := excelLabel(want); got != letter {
			t.Errorf("excelLabel(%d) = %q; want %q", want, got, letter)
		}
	}
	if _, ok := excelIndex("A1"); ok {
		t.Errorf("excelIndex(\"A1\") should fail")
	}
}

func TestArtifactInsyraBarChart(t *testing.T) {
	spec := sy.ArtifactSpec{
		Version: "v1",
		Nodes: []sy.ArtifactNode{{
			ID:        "chart",
			Component: "insyra",
			Props: map[string]any{
				"script": "newdl 1 2 3 as idx\nnewdl 12 18 9 as deals\nnewdt idx deals as t",
				"render": "bar_chart",
				"output": "t",
				"x":      "A",
				"y":      "B",
			},
			Layout: sy.ArtifactLayoutItem{ColumnSpan: 2},
		}},
	}
	store := sy.NewArtifactStore("test-bar", spec)
	if err := store.Set(spec); err != nil {
		t.Fatalf("Set: %v", err)
	}
	root := sy.RenderOnce(func() { sy.ArtifactCanvas(store) })
	bars := root.Find("bar_chart")
	if len(bars) != 1 {
		t.Fatalf("want 1 bar_chart, got %d (types: %v)", len(bars), nodeTypes(root))
	}
	if _, ok := bars[0].Props["series"]; !ok {
		t.Fatalf("bar_chart missing series prop: %v", bars[0].Props)
	}
	// Props go through a JSON clone in the store, so numbers arrive as float64.
	if layout, ok := bars[0].Props["artifact_layout"].(map[string]any); !ok || fmt.Sprint(layout["column_span"]) != "2" {
		t.Fatalf("expected column_span 2, got %v", bars[0].Props["artifact_layout"])
	}
}

func TestArtifactInsyraTableSynthesizesHeaders(t *testing.T) {
	spec := sy.ArtifactSpec{
		Version: "v1",
		Nodes: []sy.ArtifactNode{{
			ID:        "grid",
			Component: "insyra",
			Props: map[string]any{
				"script": "newdl 1 2 as a\nnewdl 3 4 as b\nnewdt a b as t",
				"render": "table",
				"output": "t",
			},
		}},
	}
	store := sy.NewArtifactStore("test-table", spec)
	if err := store.Set(spec); err != nil {
		t.Fatalf("Set: %v", err)
	}
	root := sy.RenderOnce(func() { sy.ArtifactCanvas(store) })
	tables := root.Find("table")
	if len(tables) != 1 {
		t.Fatalf("want 1 table, got %d (types: %v)", len(tables), nodeTypes(root))
	}
	// The store JSON-clones nodes, so []string arrives as []any.
	headers := toStringSlice(tables[0].Props["headers"])
	if len(headers) != 2 || headers[0] != "A" || headers[1] != "B" {
		t.Fatalf("synthesized headers = %v, want [A B]", headers)
	}
}

func TestArtifactInsyraTextTranscript(t *testing.T) {
	spec := sy.ArtifactSpec{
		Version: "v1",
		Nodes: []sy.ArtifactNode{{
			ID:        "stat",
			Component: "insyra",
			Props: map[string]any{
				"script": "newdl 10 20 30 as x\nmean x",
				"render": "text",
			},
		}},
	}
	store := sy.NewArtifactStore("test-text", spec)
	if err := store.Set(spec); err != nil {
		t.Fatalf("Set: %v", err)
	}
	root := sy.RenderOnce(func() { sy.ArtifactCanvas(store) })
	texts := root.Find("text")
	if len(texts) != 1 {
		t.Fatalf("want 1 text node, got %d (types: %v)", len(texts), nodeTypes(root))
	}
	if s, _ := texts[0].Props["text"].(string); !strings.Contains(s, "20") {
		t.Fatalf("text transcript missing mean 20: %q", s)
	}
}

func TestArtifactInsyraRejectsMissingScript(t *testing.T) {
	spec := sy.ArtifactSpec{
		Version: "v1",
		Nodes: []sy.ArtifactNode{{
			ID:        "bad",
			Component: "insyra",
			Props:     map[string]any{"render": "table"},
		}},
	}
	store := sy.NewArtifactStore("test-missing", spec)
	if err := store.Set(spec); err == nil || !strings.Contains(err.Error(), "script") {
		t.Fatalf("want script-required error, got %v", err)
	}
}

func TestArtifactInsyraRejectsUnsafeScript(t *testing.T) {
	spec := sy.ArtifactSpec{
		Version: "v1",
		Nodes: []sy.ArtifactNode{{
			ID:        "danger",
			Component: "insyra",
			Props: map[string]any{
				"script": "load /etc/passwd as secrets",
				"render": "table",
				"output": "secrets",
			},
		}},
	}
	store := sy.NewArtifactStore("test-unsafe", spec)
	if err := store.Set(spec); err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("want safe-mode rejection, got %v", err)
	}
}

func TestWidgetDSLRendersDataframe(t *testing.T) {
	root := sy.RenderOnce(func() {
		DSL("newdl 10 20 30 as a\nnewdl 1 2 3 as b\nnewdt a b as t",
			Render("dataframe"), Output("t"))
	})
	if got := len(root.Find("dataframe")); got != 1 {
		t.Fatalf("want 1 dataframe, got %d (types: %v)", got, nodeTypes(root))
	}
}

func TestWidgetDSLDefaultTranscript(t *testing.T) {
	root := sy.RenderOnce(func() {
		DSL("newdl 1 2 3 as x\nmean x")
	})
	if got := len(root.Find("code")); got != 1 {
		t.Fatalf("want 1 code transcript, got %d (types: %v)", got, nodeTypes(root))
	}
}

func toStringSlice(v any) []string {
	switch s := v.(type) {
	case []string:
		return s
	case []any:
		out := make([]string, len(s))
		for i, e := range s {
			out[i] = fmt.Sprint(e)
		}
		return out
	}
	return nil
}

func TestSafeCommandCatalogExcludesUnsafe(t *testing.T) {
	cat := SafeCommandCatalog()
	if len(cat) == 0 {
		t.Fatal("empty catalog")
	}
	seen := map[string]string{}
	for _, c := range cat {
		seen[c.Name] = c.Usage
	}
	if seen["groupby"] == "" {
		t.Fatalf("catalog should include groupby with usage; got %d entries", len(cat))
	}
	for _, unsafe := range []string{"load", "save", "db", "fetch", "run", "env", "plot"} {
		if _, bad := seen[unsafe]; bad {
			t.Fatalf("catalog exposed unsafe command %q", unsafe)
		}
	}
}

func TestDiscoveryReportsInsyraCapabilities(t *testing.T) {
	store := sy.NewArtifactStore("caps", sy.ArtifactSpec{
		Version: "v1",
		Nodes:   []sy.ArtifactNode{{ID: "n", Component: "text", Props: map[string]any{"text": "hi"}}},
	})
	handler := sy.ArtifactAPIHandler(sy.StaticAgentKey("a", "tok"), store)

	req := httptest.NewRequest(http.MethodGet, "/api/agent/artifacts", nil)
	req.Header.Set("Authorization", "Bearer tok")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Components struct {
			Custom       []string `json:"custom"`
			Capabilities struct {
				Insyra struct {
					InsyraVersion string       `json:"insyra_version"`
					Commands      []CommandDoc `json:"commands"`
				} `json:"insyra"`
			} `json:"capabilities"`
		} `json:"components"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v (body=%s)", err, rec.Body.String())
	}

	found := false
	for _, c := range resp.Components.Custom {
		if c == "insyra" {
			found = true
		}
	}
	if !found {
		t.Fatalf("components.custom should include insyra: %v", resp.Components.Custom)
	}
	if resp.Components.Capabilities.Insyra.InsyraVersion == "" {
		t.Fatalf("capabilities.insyra.insyra_version is empty")
	}
	hasGroupby := false
	for _, cmd := range resp.Components.Capabilities.Insyra.Commands {
		if cmd.Name == "groupby" && cmd.Usage != "" {
			hasGroupby = true
		}
		if cmd.Name == "load" || cmd.Name == "fetch" {
			t.Fatalf("capabilities exposed unsafe command %q", cmd.Name)
		}
	}
	if !hasGroupby {
		t.Fatalf("capabilities catalog missing groupby")
	}
}

func nodeTypes(root *sy.Node) []string {
	var out []string
	var walk func(*sy.Node)
	walk = func(n *sy.Node) {
		if n == nil {
			return
		}
		out = append(out, n.Type)
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	return out
}
