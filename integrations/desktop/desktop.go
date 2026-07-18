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
}

// Config passes an explicit Syralit config through to the app (theme, upload
// limit, UI strings, ...). Host/Port are ignored — the desktop shell always
// binds a random loopback port. Values left unset resolve from syralit.toml
// and defaults, exactly like sy.Run.
func Config(cfg sy.Config) Option { return func(o *opts) { o.cfg = cfg } }

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
func Run(fn func(), options ...Option) error {
	o := &opts{width: 1024, height: 768}
	for _, apply := range options {
		apply(o)
	}

	// sy.Handler resolves config (code > syralit.toml > defaults) as a side
	// effect, so the resolved app title is available right after.
	handler := sy.Handler(o.cfg, fn)
	if o.title == "" {
		if t, _ := sy.GetOption("title").(string); t != "" {
			o.title = t
		}
	}

	// Loopback only: a desktop app must not expose itself to the network.
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	srv := &http.Server{Handler: handler}
	serveErr := make(chan error, 1)
	go func() {
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
		}
	}()

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
		URL:       "http://" + ln.Addr().String(),
	})

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
