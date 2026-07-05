// Package uitest drives a real (headless) Chrome against an in-process
// Syralit app and asserts on actual rendered behavior — the layer AppTest
// (headless tree assertions) cannot reach: CSS visibility, browser events,
// WebSocket round-trips, uploads, canvas clicks.
package uitest

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/chromedp/chromedp"

	sy "github.com/HazelnutParadise/syralit"
)

// testApp is the single app exercised by every test.
func testApp() {
	sy.SetPageConfig(sy.PageTitle("UITest"), sy.InitialSidebarState("collapsed"))

	sy.Sidebar(func() {
		sy.Header("Side")
	})

	sy.Title("UI Test App")

	name := sy.TextInput("Name", sy.Key("name"))
	if sy.Button("Greet", sy.Key("greet")) {
		sy.Success("Hello, " + name)
	}

	if sy.Button("Toast8", sy.Key("toast8")) {
		sy.Toast("long toast", "success", "", "8s")
	}

	if sel := sy.BarChart(map[string][]float64{"S": {5, 9, 3}},
		sy.XLabels([]string{"a", "b", "c"}), sy.Selectable(), sy.Key("chart")); sel != nil {
		sy.Textf("sel:%s/%s/%.0f", sel.Series, sel.X, sel.Value)
	}

	files := sy.FileUploaderMultiple("Files", sy.Key("files"))
	for _, f := range files {
		sy.Textf("file:%s:%d", f.Name, f.Size)
	}
}

// browser spawns a headless Chrome tab; skips the suite when Chrome is absent.
func browser(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", "new"),
	)
	alloc, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancelCtx := chromedp.NewContext(alloc)
	ctx, cancelTO := context.WithTimeout(ctx, 60*time.Second)
	// Probe: fail fast (and clearly) when no Chrome is installed.
	if err := chromedp.Run(ctx); err != nil {
		cancelTO()
		cancelCtx()
		cancelAlloc()
		t.Skipf("chrome not available: %v", err)
	}
	cancel := func() { cancelTO(); cancelCtx(); cancelAlloc() }
	return ctx, cancel
}

func startApp(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(sy.Handler(sy.Config{Title: "UITest"}, testApp))
	t.Cleanup(srv.Close)
	return srv
}

// waitApp waits until the WebSocket delivered the first UI tree.
func waitApp(url string) chromedp.Tasks {
	return chromedp.Tasks{
		chromedp.Navigate(url),
		chromedp.WaitVisible(".sy-title", chromedp.ByQuery),
	}
}

func TestRenderAndWidgetRoundTrip(t *testing.T) {
	srv := startApp(t)
	ctx, cancel := browser(t)
	defer cancel()

	var greeting string
	err := chromedp.Run(ctx,
		waitApp(srv.URL),
		chromedp.SendKeys(`input.sy-input[data-id="name"]`, "Ada", chromedp.ByQuery),
		// blur to flush the input's change event before clicking
		chromedp.Evaluate(`document.querySelector('input.sy-input[data-id="name"]').blur()`, nil),
		chromedp.Sleep(300*time.Millisecond),
		chromedp.Click(`//button[contains(., "Greet")]`),
		chromedp.WaitVisible(".sy-status-success", chromedp.ByQuery),
		chromedp.Text(".sy-status-success", &greeting, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(greeting, "Hello, Ada") {
		t.Fatalf("greeting = %q", greeting)
	}
}

func TestSidebarCollapsedAndToggle(t *testing.T) {
	srv := startApp(t)
	ctx, cancel := browser(t)
	defer cancel()

	var collapsed, visibleAfter bool
	err := chromedp.Run(ctx,
		waitApp(srv.URL),
		chromedp.Evaluate(`document.getElementById("syralit-root").classList.contains("sidebar-collapsed")`, &collapsed),
		chromedp.Click("#syralit-sidebar-toggle", chromedp.ByID),
		chromedp.Sleep(200*time.Millisecond),
		chromedp.Evaluate(`getComputedStyle(document.getElementById("syralit-sidebar")).display !== "none"`, &visibleAfter),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !collapsed {
		t.Fatal("sidebar should start collapsed (initial_sidebar_state)")
	}
	if !visibleAfter {
		t.Fatal("sidebar should open after clicking the toggle")
	}
}

func TestToastVisibleWithDuration(t *testing.T) {
	srv := startApp(t)
	ctx, cancel := browser(t)
	defer cancel()

	var opacity string
	var aliveAt5s bool
	err := chromedp.Run(ctx,
		waitApp(srv.URL),
		chromedp.Click(`//button[contains(., "Toast8")]`),
		chromedp.WaitVisible(".sy-toast", chromedp.ByQuery),
		chromedp.Sleep(1*time.Second),
		chromedp.Evaluate(`getComputedStyle(document.querySelector(".sy-toast")).opacity`, &opacity),
		chromedp.Sleep(4*time.Second), // past the 3s default, within the 8s custom
		chromedp.Evaluate(`!!document.querySelector(".sy-toast")`, &aliveAt5s),
	)
	if err != nil {
		t.Fatal(err)
	}
	if opacity != "1" {
		t.Fatalf("toast opacity = %s, want 1 (actually visible)", opacity)
	}
	if !aliveAt5s {
		t.Fatal("8s toast disappeared before 5s")
	}
}

func TestChartClickSelection(t *testing.T) {
	srv := startApp(t)
	ctx, cancel := browser(t)
	defer cancel()

	// Resolve the middle bar's viewport coordinates from Chart.js metadata,
	// then click with a real (trusted) mouse event.
	var coords []float64
	var selText string
	err := chromedp.Run(ctx,
		waitApp(srv.URL),
		chromedp.WaitVisible(".sy-chart-wrap canvas", chromedp.ByQuery),
		chromedp.ScrollIntoView(".sy-chart-wrap", chromedp.ByQuery),
		chromedp.Sleep(500*time.Millisecond), // let the scroll and chart animation settle
		chromedp.Poll(`(function(){
			var c = document.querySelector(".sy-chart-wrap canvas");
			if (!window.Chart || !Chart.getChart(c)) return null;
			var chart = Chart.getChart(c);
			var el = chart.getDatasetMeta(0).data[1];
			if (!el) return null;
			var r = c.getBoundingClientRect();
			return [r.left + el.x, r.top + el.y + 5];
		})()`, &coords, chromedp.WithPollingTimeout(15*time.Second)),
		chromedp.ActionFunc(func(ctx context.Context) error {
			if len(coords) != 2 {
				return fmt.Errorf("no bar coordinates: %v", coords)
			}
			return chromedp.MouseClickXY(coords[0], coords[1]).Do(ctx)
		}),
		chromedp.WaitVisible(`//p[contains(., "sel:")]`),
		chromedp.Text(`//p[contains(., "sel:")]`, &selText),
	)
	if err != nil {
		t.Fatal(err)
	}
	if selText != "sel:S/b/9" {
		t.Fatalf("selection = %q, want sel:S/b/9", selText)
	}
}

func TestMultiFileUpload(t *testing.T) {
	srv := startApp(t)
	ctx, cancel := browser(t)
	defer cancel()

	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.txt")
	f2 := filepath.Join(dir, "b.csv")
	if err := os.WriteFile(f1, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(f2, []byte("1,2,3"), 0o644); err != nil {
		t.Fatal(err)
	}

	var texts string
	err := chromedp.Run(ctx,
		waitApp(srv.URL),
		chromedp.SetUploadFiles(`input[type="file"]`, []string{f1, f2}, chromedp.ByQuery),
		chromedp.WaitVisible(`//p[contains(., "file:a.txt")]`),
		chromedp.Text("#syralit-app", &texts, chromedp.ByID),
	)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(texts, "file:a.txt:5") || !strings.Contains(texts, "file:b.csv:5") {
		t.Fatalf("uploaded files not echoed: %q", texts)
	}
}

func TestSubPathMount(t *testing.T) {
	mux := http.NewServeMux()
	mux.Handle("/dash/", http.StripPrefix("/dash", sy.Handler(sy.Config{Title: "Sub"}, func() {
		sy.Title("Mounted Under Dash")
	})))
	srv := httptest.NewServer(mux)
	defer srv.Close()

	ctx, cancel := browser(t)
	defer cancel()

	var title string
	err := chromedp.Run(ctx,
		chromedp.Navigate(srv.URL+"/dash/"),
		chromedp.WaitVisible(".sy-title", chromedp.ByQuery),
		chromedp.Text(".sy-title", &title, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatal(err)
	}
	if title != "Mounted Under Dash" {
		t.Fatalf("title = %q — WebSocket under sub-path mount failed", title)
	}
}
