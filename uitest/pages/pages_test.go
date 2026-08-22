// Package pages drives a real Chrome against a multi-page Syralit app. It lives
// in its own package because sy.AddPage writes to a process-wide registry —
// registering pages inside the main uitest package would put a page sidebar on
// every other test's app.
package pages

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chromedp/chromedp"

	sy "github.com/HazelnutParadise/syralit"
)

func init() {
	sy.AddPage("Home", func() { sy.Title("Home Page") }, sy.PageOrder(1))
	sy.AddPage("Data Explorer", func() { sy.Title("Explorer Page") }, sy.PageOrder(2))
	sy.AddPage("報表", func() { sy.Title("Report Page") }, sy.PageOrder(3), sy.PageSlug("reports"))
}

func browser(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	opts := append(chromedp.DefaultExecAllocatorOptions[:], chromedp.Flag("headless", "new"))
	alloc, cancelAlloc := chromedp.NewExecAllocator(context.Background(), opts...)
	ctx, cancelCtx := chromedp.NewContext(alloc)
	ctx, cancelTO := context.WithTimeout(ctx, 60*time.Second)
	if err := chromedp.Run(ctx); err != nil {
		cancelTO()
		cancelCtx()
		cancelAlloc()
		t.Skipf("chrome not available: %v", err)
	}
	return ctx, func() { cancelTO(); cancelCtx(); cancelAlloc() }
}

func startApp(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(sy.Handler(sy.Config{Title: "Pages"}, nil))
	t.Cleanup(srv.Close)
	return srv
}

// TestPageNavigationUpdatesURL is the whole point of page URLs: clicking a page
// changes the address bar, the back button returns to the previous page, and
// opening a page URL cold renders that page rather than the first one.
func TestPageNavigationUpdatesURL(t *testing.T) {
	srv := startApp(t)
	ctx, cancel := browser(t)
	defer cancel()

	var url, title, docTitle string
	err := chromedp.Run(ctx,
		// Below 768px the sidebar slides off-canvas, so a real click on a page
		// link would land on the backdrop instead.
		chromedp.EmulateViewport(1280, 800),
		chromedp.Navigate(srv.URL+"/"),
		chromedp.WaitVisible(".sy-title", chromedp.ByQuery),
		chromedp.Text(".sy-title", &title, chromedp.ByQuery),
	)
	if err != nil {
		t.Fatal(err)
	}
	if title != "Home Page" {
		t.Fatalf("root should render the first page, got %q", title)
	}

	// Click the second page link.
	err = chromedp.Run(ctx,
		chromedp.Click(`.sy-sidebar-pages a[href$="/data_explorer"]`, chromedp.ByQuery),
		chromedp.WaitVisible(`.sy-title`, chromedp.ByQuery),
		chromedp.Sleep(300*time.Millisecond),
		chromedp.Text(".sy-title", &title, chromedp.ByQuery),
		chromedp.Location(&url),
	)
	if err != nil {
		t.Fatal(err)
	}
	if title != "Explorer Page" {
		t.Fatalf("after click, page = %q", title)
	}
	if url != srv.URL+"/data_explorer" {
		t.Fatalf("URL after click = %q", url)
	}

	// Back button returns to the first page without a reload.
	err = chromedp.Run(ctx,
		chromedp.NavigateBack(),
		chromedp.Sleep(500*time.Millisecond),
		chromedp.Text(".sy-title", &title, chromedp.ByQuery),
		chromedp.Location(&url),
	)
	if err != nil {
		t.Fatal(err)
	}
	if title != "Home Page" {
		t.Fatalf("after back, page = %q", title)
	}
	if url != srv.URL+"/" {
		t.Fatalf("URL after back = %q", url)
	}

	// A cold load of an explicit slug renders that page, and the document title
	// comes from the server rather than from the client after connecting.
	err = chromedp.Run(ctx,
		chromedp.Navigate(srv.URL+"/reports"),
		chromedp.WaitVisible(".sy-title", chromedp.ByQuery),
		chromedp.Text(".sy-title", &title, chromedp.ByQuery),
		chromedp.Title(&docTitle),
	)
	if err != nil {
		t.Fatal(err)
	}
	if title != "Report Page" {
		t.Fatalf("cold load of /reports rendered %q", title)
	}
	if docTitle != "報表" {
		t.Fatalf("document title = %q, want 報表", docTitle)
	}
}
