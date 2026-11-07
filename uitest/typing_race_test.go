package uitest

import (
	"context"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/chromedp/chromedp"

	sy "github.com/HazelnutParadise/syralit"
)

const (
	typingRaceDelay = 150 * time.Millisecond

	raceSlowInput  = `input.sy-input[data-id="slow"]`
	raceSlowArea   = `textarea.sy-input[data-id="slowarea"]`
	raceFragInput  = `input.sy-input[data-id="fragslow"]`
	raceResetInput = `input.sy-input[data-id="resettable"]`
)

func raceApp() {
	v := sy.TextInput("Slow", sy.Key("slow"))
	if v != "" {
		time.Sleep(typingRaceDelay)
	}
	sy.Text("echo:" + v)

	area := sy.TextArea("Slow area", sy.Key("slowarea"))
	if area != "" {
		time.Sleep(typingRaceDelay)
	}
	sy.Text("areaecho:" + area)

	sy.Fragment("frag", func() {
		fv := sy.TextInput("Slow fragment", sy.Key("fragslow"))
		if fv != "" {
			time.Sleep(typingRaceDelay)
		}
		sy.Text("fragecho:" + fv)
	})

	resettable := sy.TextInput("Resettable", sy.Key("resettable"))
	if resettable == "reset" {
		sy.ResetWidget("resettable")
		sy.Rerun()
	}
}

func startRaceServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(sy.Handler(sy.Config{Title: "Race"}, raceApp))
	t.Cleanup(srv.Close)
	return srv
}

func openRaceApp(t *testing.T, ctx context.Context, url string) {
	t.Helper()
	if err := chromedp.Run(ctx,
		chromedp.Navigate(url),
		chromedp.WaitVisible(raceSlowInput, chromedp.ByQuery),
	); err != nil {
		t.Fatalf("open race app: %v", err)
	}
}

// raceKeyGap is shorter than typingRaceDelay, so replies to earlier
// keystrokes land while later ones are still being typed (issue #13).
const raceKeyGap = 60 * time.Millisecond

func typeRace(t *testing.T, ctx context.Context, selector, text string) {
	t.Helper()
	if err := chromedp.Run(ctx, chromedp.Click(selector, chromedp.ByQuery)); err != nil {
		t.Fatalf("focus %s: %v", selector, err)
	}
	sendRace(t, ctx, selector, text)
}

// sendRace types text one key at a time into the focused element, pausing
// raceKeyGap between keys like a fast typist.
func sendRace(t *testing.T, ctx context.Context, selector, text string) {
	t.Helper()
	for _, r := range text {
		if err := chromedp.Run(ctx,
			chromedp.KeyEvent(string(r)),
			chromedp.Sleep(raceKeyGap),
		); err != nil {
			t.Fatalf("send %q to %s: %v", text, selector, err)
		}
	}
}

func waitRaceText(t *testing.T, ctx context.Context, want string) {
	t.Helper()
	var ok bool
	expr := fmt.Sprintf(`document.getElementById("syralit-app").textContent.indexOf(%q) >= 0`, want)
	if err := chromedp.Run(ctx, chromedp.Poll(expr, &ok, chromedp.WithPollingTimeout(10*time.Second))); err != nil {
		t.Fatalf("waiting for page text %q: %v", want, err)
	}
	if !ok {
		t.Fatalf("waiting for page text %q: never matched", want)
	}
}

func raceInputValue(t *testing.T, ctx context.Context, selector string) string {
	t.Helper()
	var got string
	expr := fmt.Sprintf(`document.querySelector(%q).value`, selector)
	if err := chromedp.Run(ctx, chromedp.Evaluate(expr, &got)); err != nil {
		t.Fatalf("read value of %s: %v", selector, err)
	}
	return got
}

func waitRaceInputValue(t *testing.T, ctx context.Context, selector, want string) {
	t.Helper()
	var ok bool
	expr := fmt.Sprintf(`document.querySelector(%q).value === %q`, selector, want)
	if err := chromedp.Run(ctx, chromedp.Poll(expr, &ok, chromedp.WithPollingTimeout(10*time.Second))); err != nil {
		t.Fatalf("waiting for %s value %q: %v", selector, want, err)
	}
	if !ok {
		t.Fatalf("waiting for %s value %q: never matched", selector, want)
	}
}

func TestTypingRaceFastTypingKeepsFullValue(t *testing.T) {
	srv := startRaceServer(t)
	ctx, cancel := browser(t)
	defer cancel()

	openRaceApp(t, ctx, srv.URL)
	typeRace(t, ctx, raceSlowInput, "abcdefgh")
	waitRaceText(t, ctx, "echo:abcdefgh")

	if got := raceInputValue(t, ctx, raceSlowInput); got != "abcdefgh" {
		t.Fatalf("input value = %q, want %q", got, "abcdefgh")
	}
}

func TestTypingRaceAppendAfterFirstBurst(t *testing.T) {
	srv := startRaceServer(t)
	ctx, cancel := browser(t)
	defer cancel()

	openRaceApp(t, ctx, srv.URL)
	typeRace(t, ctx, raceSlowInput, "abcdefgh")
	waitRaceText(t, ctx, "echo:abcdefgh")

	sendRace(t, ctx, raceSlowInput, "XY")
	waitRaceText(t, ctx, "echo:abcdefghXY")

	if got := raceInputValue(t, ctx, raceSlowInput); got != "abcdefghXY" {
		t.Fatalf("input value = %q, want %q", got, "abcdefghXY")
	}
}

func TestTypingRaceMidInsertKeepsCaret(t *testing.T) {
	srv := startRaceServer(t)
	ctx, cancel := browser(t)
	defer cancel()

	openRaceApp(t, ctx, srv.URL)
	typeRace(t, ctx, raceSlowInput, "abcdef")
	waitRaceText(t, ctx, "echo:abcdef")

	if err := chromedp.Run(ctx, chromedp.Evaluate(
		`(function(){var el=document.querySelector('input.sy-input[data-id="slow"]');el.focus();el.setSelectionRange(3,3);})()`,
		nil,
	)); err != nil {
		t.Fatalf("move caret: %v", err)
	}

	sendRace(t, ctx, raceSlowInput, "123")
	waitRaceText(t, ctx, "echo:abc123def")

	if got := raceInputValue(t, ctx, raceSlowInput); got != "abc123def" {
		t.Fatalf("input value = %q, want %q", got, "abc123def")
	}
}

func TestRaceTextAreaFastTypingKeepsFullValue(t *testing.T) {
	srv := startRaceServer(t)
	ctx, cancel := browser(t)
	defer cancel()

	openRaceApp(t, ctx, srv.URL)
	typeRace(t, ctx, raceSlowArea, "abcdefgh")
	waitRaceText(t, ctx, "areaecho:abcdefgh")

	if got := raceInputValue(t, ctx, raceSlowArea); got != "abcdefgh" {
		t.Fatalf("textarea value = %q, want %q", got, "abcdefgh")
	}
}

func TestRaceFragmentFastTypingKeepsValueAndFocus(t *testing.T) {
	srv := startRaceServer(t)
	ctx, cancel := browser(t)
	defer cancel()

	openRaceApp(t, ctx, srv.URL)
	typeRace(t, ctx, raceFragInput, "abcdefgh")
	waitRaceText(t, ctx, "fragecho:abcdefgh")

	if got := raceInputValue(t, ctx, raceFragInput); got != "abcdefgh" {
		t.Fatalf("fragment input value = %q, want %q", got, "abcdefgh")
	}

	var active string
	if err := chromedp.Run(ctx, chromedp.Evaluate(
		`(document.activeElement && document.activeElement.dataset && document.activeElement.dataset.id) || ""`,
		&active,
	)); err != nil {
		t.Fatalf("read active element: %v", err)
	}
	if active != "fragslow" {
		t.Fatalf("active element data-id = %q, want %q", active, "fragslow")
	}
}

func TestRaceResetWidgetClearsInput(t *testing.T) {
	srv := startRaceServer(t)
	ctx, cancel := browser(t)
	defer cancel()

	openRaceApp(t, ctx, srv.URL)
	typeRace(t, ctx, raceResetInput, "reset")
	waitRaceInputValue(t, ctx, raceResetInput, "")

	if got := raceInputValue(t, ctx, raceResetInput); got != "" {
		t.Fatalf("input value = %q, want empty after ResetWidget", got)
	}
}
