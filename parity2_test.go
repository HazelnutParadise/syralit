package syralit

import (
	"strings"
	"testing"
)

func TestChartRangeSelection(t *testing.T) {
	at := NewAppTest(func() {
		sel := LineChart(map[string][]float64{"A": {1, 2, 3, 4, 5}},
			XLabels([]string{"a", "b", "c", "d", "e"}), RangeSelectable(), Key("rc"))
		if sel != nil && sel.Range {
			Textf("range:%d-%d(%s..%s)", sel.Index, sel.EndIndex, sel.X, sel.EndX)
		}
	})
	at.Run()
	n := at.FindAll("line_chart")
	if len(n) != 1 || n[0].Props["range_selectable"] != true {
		t.Fatalf("chart not range_selectable: %+v", n)
	}
	at.SetValue("rc", map[string]any{
		"range": true, "index": float64(1), "x": "b",
		"end_index": float64(3), "end_x": "d",
	})
	at.Run()
	if got := at.Texts("text"); len(got) != 1 || got[0] != "range:1-3(b..d)" {
		t.Fatalf("range selection = %v", got)
	}
}

func TestDataFrameColumnSelection(t *testing.T) {
	at := NewAppTest(func() {
		cols := DataFrame([]string{"A", "B", "C"}, [][]any{{1, 2, 3}},
			Selectable(), SelectionMode("multi-column"), Key("df"))
		if len(cols) > 0 {
			Textf("cols:%v", cols)
		}
	})
	at.Run()
	dfs := at.FindAll("dataframe")
	if len(dfs) != 1 || dfs[0].Props["selection_mode"] != "multi-column" {
		t.Fatalf("selection_mode missing: %+v", dfs)
	}
	at.SetValue("df", []any{float64(0), float64(2)})
	at.Run()
	if got := at.Texts("text"); len(got) != 1 || got[0] != "cols:[0 2]" {
		t.Fatalf("column selection = %v", got)
	}
}

func TestI18nInjection(t *testing.T) {
	shell := resolveShell("", "", "", map[string]string{"connecting": "連線中…", "loading": "載入中…"})
	html := renderIndex("X", Theme{}, shell)
	if !strings.Contains(html, `window.__SY_I18N={"connecting":"連線中…","loading":"載入中…"}`) ||
		!strings.Contains(html, "<p>連線中…</p>") {
		t.Fatalf("i18n script missing:\n%s", html)
	}

	// A value trying to break out of the script element must stay escaped.
	shell = resolveShell("", "", "", map[string]string{"loading": "</script><script>alert(1)</script>"})
	html = renderIndex("X", Theme{}, shell)
	if strings.Contains(html, "</script><script>alert") {
		t.Fatalf("i18n value broke out of script context:\n%s", html)
	}

	// No overrides: no script, English default.
	html = renderIndex("X", Theme{}, shellConfig{})
	if strings.Contains(html, "__SY_I18N") || !strings.Contains(html, "<p>Connecting…</p>") {
		t.Fatalf("unexpected i18n output without overrides:\n%s", html)
	}
}

func TestI18nConfig(t *testing.T) {
	cfg := Config{}
	fc := &fileConfig{I18n: map[string]string{"loading": "載入中"}}
	fc.applyToConfig(&cfg)
	if cfg.UIStrings["loading"] != "載入中" {
		t.Fatalf("i18n config not applied: %+v", cfg.UIStrings)
	}
}

func TestUserResolver(t *testing.T) {
	defer SetUserResolver(nil)
	SetUserResolver(func(rc RequestContext) map[string]string {
		if rc.Cookies["auth"] == "good" {
			return map[string]string{"username": "ada", "email": "ada@example.com"}
		}
		return nil
	})

	sess := newSession(func() {
		if u := User(); u != nil {
			Text("user:" + u["username"])
		} else {
			Text("anon")
		}
	})
	sess.reqCtx = RequestContext{Cookies: map[string]string{"auth": "good"}}
	resolveSessionUser(sess)
	at := &AppTest{sess: sess}
	at.Run()
	if got := at.Texts("text"); len(got) != 1 || got[0] != "user:ada" {
		t.Fatalf("resolved user = %v", got)
	}

	// Bad cookie → anonymous.
	sess2 := newSession(func() {
		if User() == nil {
			Text("anon")
		}
	})
	sess2.reqCtx = RequestContext{Cookies: map[string]string{"auth": "bad"}}
	resolveSessionUser(sess2)
	at2 := &AppTest{sess: sess2}
	at2.Run()
	if got := at2.Texts("text"); len(got) != 1 || got[0] != "anon" {
		t.Fatalf("anonymous path = %v", got)
	}
}
