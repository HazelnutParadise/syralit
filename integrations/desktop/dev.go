package sydesktop

// Hot-reload support: under `syralit dev` the supervisor owns the outward port
// and kills/restarts the app as a child process on every rebuild, so a window
// opened by the child would die with it. Instead the child serves the
// supervisor like any dev child (sy.Run) and hands the native window to a
// detached "window host" process — a copy of this same binary re-entering
// Run in window-only mode — pointed at the supervisor's outward URL. Rebuilds
// then behave exactly like they do in a browser: the window's connection to
// the supervisor never drops, state is preserved, build errors overlay.

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// Set by the dev supervisor for its child (core's envDevAddr/URL/Session).
	envDevAddr    = "SYRALIT_DEV_ADDR"
	envDevURL     = "SYRALIT_URL"
	envDevSession = "SYRALIT_DEV_SESSION"
	// Internal to this package: marks the window-host process; value is the URL.
	envWindowHost = "SYRALIT_DESKTOP_WINDOW"
)

// spawnDevWindow starts the detached window host, at most once per dev
// session. A lockfile keyed by the session id guards against respawning on
// every rebuild — and deliberately survives the window host's exit, so a
// window the user closed stays closed for the rest of the session (the app
// remains reachable in a browser via $SYRALIT_URL).
func spawnDevWindow() {
	url := os.Getenv(envDevURL)
	if url == "" {
		log.Printf("sydesktop: %s not set; no native window (syralit CLI too old — rebuild it from the current module)", envDevURL)
		return
	}
	key := sanitizeKey(os.Getenv(envDevSession))
	if key == "" {
		key = sanitizeKey(os.Getenv(envDevAddr))
	}
	cleanStale()

	lock := filepath.Join(os.TempDir(), "syralit-desktop-"+key+".lock")
	lf, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return // window already spawned (or closed by the user) this session
	}
	fmt.Fprintf(lf, "%d\n", os.Getpid())
	_ = lf.Close()

	fail := func(err error) {
		_ = os.Remove(lock) // let the next rebuild retry
		log.Printf("sydesktop: dev window: %v", err)
	}

	// The supervisor rebuilds onto the child's own executable path, and
	// Windows locks a running exe — a window host running from that path
	// would fail every rebuild. Run a copy instead.
	self, err := os.Executable()
	if err != nil {
		fail(err)
		return
	}
	host := filepath.Join(os.TempDir(), "syralit-desktop-"+key+exeSuffix())
	if err := copyFile(self, host); err != nil {
		fail(err)
		return
	}
	cmd := exec.Command(host)
	cmd.Env = append(os.Environ(), envWindowHost+"="+url)
	if err := cmd.Start(); err != nil {
		fail(err)
		return
	}
	_ = cmd.Process.Release() // detach: must survive this child's kill on rebuild
}

// runWindowHost is Run's window-only mode: no Syralit server, just the native
// window on the dev supervisor's URL. It waits for the server to come up, and
// quits itself when the server stays gone (supervisor stopped) so dev sessions
// don't leave dead windows behind.
func runWindowHost(url string, o *opts) error {
	if !waitReachable(url, 15*time.Second) {
		return fmt.Errorf("dev server not reachable at %s", url)
	}
	if o.title == "" {
		o.title = o.cfg.Title
	}
	if o.title == "" {
		o.title = "Syralit App (dev)"
	}
	app := newWindowApp(o, url)
	go func() {
		fails := 0
		for {
			time.Sleep(2 * time.Second)
			if reachable(url) {
				fails = 0
				continue
			}
			if fails++; fails >= 3 {
				app.Quit()
				return
			}
		}
	}()
	return app.Run()
}

func reachable(url string) bool {
	client := http.Client{Timeout: 1500 * time.Millisecond}
	resp, err := client.Get(url)
	if err != nil {
		return false
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
	return true // any HTTP response means the supervisor is alive
}

func waitReachable(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for {
		if reachable(url) {
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(200 * time.Millisecond)
	}
}

// cleanStale removes window-host copies and lockfiles from past dev sessions
// (>24h old) so temp doesn't accumulate one binary per session forever.
func cleanStale() {
	matches, _ := filepath.Glob(filepath.Join(os.TempDir(), "syralit-desktop-*"))
	for _, m := range matches {
		if fi, err := os.Stat(m); err == nil && time.Since(fi.ModTime()) > 24*time.Hour {
			_ = os.Remove(m) // best-effort; a still-running exe just stays
		}
	}
}

func sanitizeKey(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '-', r == '.':
			return r
		default:
			return '_'
		}
	}, s)
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	return out.Close()
}
