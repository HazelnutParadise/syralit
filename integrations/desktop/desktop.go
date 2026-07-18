// Package sydesktop ships a Syralit app as a native desktop application.
// Import with alias sydesktop:
//
//	import sydesktop "github.com/HazelnutParadise/syralit/integrations/desktop"
//
// It runs the app on a loopback-only listener (127.0.0.1, random port) and
// opens a native webview window (Wails v3) pointing at it, so the exact same
// app function works as a webapp with sy.App and as a desktop app with
// sydesktop.App — only the entry point differs:
//
//	func main() {
//	    sydesktop.App(func() {
//	        sy.Title("My tool")
//	        // ... any Syralit app
//	    }, sydesktop.WindowSize(1200, 800))
//	}
//
// Platform requirements are Wails v3's: Windows needs the WebView2 runtime
// (preinstalled on Windows 10/11), macOS needs the Xcode command-line tools
// at build time, Linux needs webkit2gtk.
package sydesktop

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	sy "github.com/HazelnutParadise/syralit"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// Option configures the desktop window and app.
type Option func(*opts)

type opts struct {
	cfg                 sy.Config
	title               string
	width, height       int
	minWidth, minHeight int
	frameless           bool
	icon                []byte
	allowBrowser        bool
}

// Config passes an explicit Syralit config through to the app (theme, upload
// limit, UI strings, ...). Values left unset resolve from syralit.toml and
// defaults, exactly like sy.Run. Host is ignored — a desktop app always binds
// loopback only. An explicitly set Port pins the port (useful so agents can
// reach the /api/ endpoints on a stable address across launches); left zero,
// a random free port is used.
func Config(cfg sy.Config) Option { return func(o *opts) { o.cfg = cfg } }

// AllowBrowser lets other local browsers open the app too. By default the
// loopback server only answers its own window: requests without the window's
// per-launch token are rejected (except /api/ endpoints, which carry their
// own authentication).
func AllowBrowser() Option { return func(o *opts) { o.allowBrowser = true } }

// WindowTitle sets the native window title. Defaults to the resolved Syralit
// app title (code > syralit.toml > "Syralit App").
func WindowTitle(t string) Option { return func(o *opts) { o.title = t } }

// WindowSize sets the initial window size in pixels. Defaults to 1024×768.
func WindowSize(w, h int) Option { return func(o *opts) { o.width, o.height = w, h } }

// MinSize sets the minimum window size in pixels.
func MinSize(w, h int) Option { return func(o *opts) { o.minWidth, o.minHeight = w, h } }

// Frameless removes the native window frame.
func Frameless() Option { return func(o *opts) { o.frameless = true } }

// Icon sets the application icon from raw image bytes (PNG recommended).
func Icon(data []byte) Option { return func(o *opts) { o.icon = data } }

// App starts fn as a desktop app and blocks until the window closes. It is
// the desktop counterpart of sy.App: fatal on error, nil fn allowed in
// multi-page mode (pages registered via sy.AddPage render instead).
//
// Must be called from the main goroutine — macOS requires the native event
// loop to run on the main thread.
func App(fn func(), options ...Option) {
	if err := Run(fn, options...); err != nil {
		log.Fatalf("sydesktop: %v", err)
	}
}

// Run is App with the error returned instead of being fatal.
//
// Under `syralit dev` the process is a hot-reload child: it serves the
// supervisor like sy.Run does, and the native window is handed to a detached
// helper process that outlives rebuilds — so the window gets the same
// hot-reload experience as a browser tab (state preserved, error overlays).
// Closing that window leaves the dev session running; closing it is for the
// session (the app stays reachable in a browser at $SYRALIT_URL).
func Run(fn func(), options ...Option) error {
	o := &opts{width: 1024, height: 768}
	for _, apply := range options {
		apply(o)
	}

	// Window-host mode (internal): render the dev window only; the supervisor
	// owns the server. Entered by the helper process spawnDevWindow starts.
	if url := os.Getenv(envWindowHost); url != "" {
		return runWindowHost(url, o)
	}

	// Dev-child mode under `syralit dev`: spawn the window helper (once per
	// dev session), then serve the supervisor's internal address via sy.Run.
	if os.Getenv(envDevAddr) != "" {
		spawnDevWindow()
		return sy.Run(o.cfg, fn)
	}

	// sy.Handler resolves config (code > syralit.toml > defaults) as a side
	// effect, so the resolved app title is available right after.
	handler := sy.Handler(o.cfg, fn)
	if o.title == "" {
		if t, _ := sy.GetOption("title").(string); t != "" {
			o.title = t
		}
	}

	// Loopback only: a desktop app must not expose itself to the network. An
	// explicit Config Port pins the port; otherwise the OS picks a free one.
	listenAddr := "127.0.0.1:0"
	if o.cfg.Port > 0 {
		listenAddr = fmt.Sprintf("127.0.0.1:%d", o.cfg.Port)
	}
	ln, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	url := "http://" + ln.Addr().String()
	windowURL := url
	if !o.allowBrowser {
		token := newLockdownToken()
		handler = lockdown(handler, token)
		windowURL = url + "/?" + lockdownParam + "=" + token
	}
	// Publish the app's URL the way the dev supervisor does, so agent
	// subprocesses this app spawns can find the /api/ endpoints.
	_ = os.Setenv(envDevURL, url)
	log.Printf("sydesktop: %q running on %s (loopback only)", o.title, url)

	srv := &http.Server{Handler: handler}
	serveErr := make(chan error, 1)
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

	app := newWindowApp(o, windowURL)

	runErr := app.Run()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)

	select {
	case err := <-serveErr:
		return err
	default:
	}
	return runErr
}

// newWindowApp builds the Wails application with a single webview window
// pointed at url, from the shared window options.
func newWindowApp(o *opts, url string) *application.App {
	app := application.New(application.Options{
		Name: o.title,
		Icon: o.icon,
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	})
	app.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:     o.title,
		Width:     o.width,
		Height:    o.height,
		MinWidth:  o.minWidth,
		MinHeight: o.minHeight,
		Frameless: o.frameless,
		URL:       url,
	})
	return app
}
