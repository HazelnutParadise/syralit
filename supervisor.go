package syralit

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/fsnotify/fsnotify"
)

// DevOptions configures the hot reload supervisor.
type DevOptions struct {
	Dir       string // project directory to build & watch (default ".")
	Target    string // build target package (default Dir)
	Host      string // outward bind host (default 127.0.0.1)
	Port      int    // outward port (default 8600)
	Title     string // page title
	AssetsDir string // optional: serve front-end assets from disk and hot-reload them
	PublicDir string // optional: serve user static files (public/) from disk at site root
	Theme     Theme
}

func (o *DevOptions) applyDefaults() {
	if o.Dir == "" {
		o.Dir = "."
	}
	if o.Target == "" {
		o.Target = "." // built with cmd.Dir = o.Dir, so "." is the package in Dir
	}
	if o.Host == "" {
		o.Host = "127.0.0.1"
	}
	if o.Port == 0 {
		o.Port = 8600
	}
	if o.Title == "" {
		o.Title = "Syralit App (dev)"
	}
}

// RunDev starts the hot reload supervisor. The supervisor owns the outward port
// and the browser connections for its whole lifetime ("the app never stops"),
// while the user's program runs as a child process that is rebuilt and restarted
// on file changes. Session state is carried across restarts (SPEC §10.3).
func RunDev(opts DevOptions) error {
	s, mux, err := startSupervisor(opts)
	if err != nil {
		return err
	}
	addr := fmt.Sprintf("%s:%d", s.opts.Host, s.opts.Port)
	log.Printf("syralit[dev]: %q running on http://%s (child on %s)", s.opts.Title, addr, s.childAddr)
	return http.ListenAndServe(addr, mux)
}

// startSupervisor builds the supervisor, performs the initial build/child start,
// and begins watching. It returns the handler so the caller can serve it (RunDev)
// or drive it in-process (tests). Call s.shutdown() to clean up.
func startSupervisor(opts DevOptions) (*supervisor, http.Handler, error) {
	if opts.Dir == "" {
		opts.Dir = "."
	}
	loadFileConfig(opts.Dir).applyToDev(&opts) // syralit.toml fills unset values
	opts.applyDefaults()

	childAddr, err := freePort()
	if err != nil {
		return nil, nil, fmt.Errorf("allocate child port: %w", err)
	}
	bin := filepath.Join(os.TempDir(), fmt.Sprintf("syralit-dev-%d", os.Getpid()))
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}

	s := &supervisor{
		opts:      opts,
		binPath:   bin,
		childAddr: childAddr,
		ctx:       context.Background(),
		browsers:  map[*browserConn]struct{}{},
		stateCh:   make(chan json.RawMessage, 1),
		done:      make(chan struct{}),
	}

	// A broken initial build still starts the supervisor so the browser shows
	// the error overlay instead of failing to connect.
	if out, err := s.build(); err != nil {
		log.Printf("syralit[dev]: initial build failed:\n%s", out)
		s.setBuildErr(out)
	} else if err := s.spawnChild(); err != nil {
		log.Printf("syralit[dev]: start child: %v", err)
		s.setBuildErr(err.Error())
	}

	go s.watch()

	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)
	mux.Handle("GET /_syralit/assets/", s.assetsHandler())
	mux.HandleFunc("GET /_syralit/ws", s.handleBrowserWS)
	return s, mux, nil
}

// shutdown stops the watcher and the child process. The supervisor is not
// reusable afterward.
func (s *supervisor) shutdown() {
	select {
	case <-s.done:
	default:
		close(s.done)
	}
	s.stopChild()
	_ = os.Remove(s.binPath)
}

type supervisor struct {
	opts      DevOptions
	binPath   string
	childAddr string
	ctx       context.Context

	mu           sync.Mutex
	childCmd     *exec.Cmd
	childConn    *websocket.Conn
	childWriteMu sync.Mutex
	lastPatch    []byte // last ui_patch from child, replayed to new browsers
	buildErr     string

	browsersMu sync.Mutex
	browsers   map[*browserConn]struct{}

	stateCh  chan json.RawMessage
	reloadMu sync.Mutex // serializes reload()
	done     chan struct{}
}

// --- build & child process ---

func (s *supervisor) build() (string, error) {
	cmd := exec.Command("go", "build", "-o", s.binPath, s.opts.Target)
	cmd.Dir = s.opts.Dir
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (s *supervisor) spawnChild() error {
	cmd := exec.Command(s.binPath)
	cmd.Env = append(os.Environ(), envDevAddr+"="+s.childAddr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}

	conn, err := dialChildWithRetry(s.ctx, s.childAddr, 5*time.Second)
	if err != nil {
		_ = cmd.Process.Kill()
		return err
	}
	conn.SetReadLimit(32 << 20)

	s.mu.Lock()
	s.childCmd = cmd
	s.childConn = conn
	s.mu.Unlock()

	go s.childPump(conn)
	return nil
}

func (s *supervisor) stopChild() {
	s.mu.Lock()
	conn, cmd := s.childConn, s.childCmd
	s.childConn, s.childCmd = nil, nil
	s.mu.Unlock()

	if conn != nil {
		_ = conn.Close(websocket.StatusNormalClosure, "reload")
	}
	if cmd != nil && cmd.Process != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}
}

// childPump reads frames from the child and either captures dumped state or
// broadcasts the frame (ui_patch) to all browsers.
func (s *supervisor) childPump(conn *websocket.Conn) {
	for {
		_, data, err := conn.Read(s.ctx)
		if err != nil {
			return
		}
		var probe struct {
			Type  string          `json:"type"`
			State json.RawMessage `json:"state"`
		}
		_ = json.Unmarshal(data, &probe)

		if probe.Type == devMsgState {
			select {
			case s.stateCh <- probe.State:
			default:
			}
			continue
		}
		if probe.Type == "ui_patch" {
			s.mu.Lock()
			s.lastPatch = data
			s.mu.Unlock()
		}
		s.broadcast(data)
	}
}

func (s *supervisor) sendToChild(b []byte) {
	s.mu.Lock()
	conn := s.childConn
	s.mu.Unlock()
	if conn == nil {
		return
	}
	s.childWriteMu.Lock()
	defer s.childWriteMu.Unlock()
	_ = conn.Write(s.ctx, websocket.MessageText, b)
}

// --- reload ---

func (s *supervisor) reload() {
	s.reloadMu.Lock()
	defer s.reloadMu.Unlock()

	s.broadcast(devStatus("building"))

	out, err := s.build()
	if err != nil {
		log.Printf("syralit[dev]: build failed:\n%s", out)
		s.setBuildErr(out)
		s.broadcast(devBuildError(out))
		return // keep the old child serving
	}
	s.clearBuildErr()

	// Hand session state from the old child to the new one.
	var saved json.RawMessage
	s.mu.Lock()
	hasChild := s.childConn != nil
	s.mu.Unlock()
	if hasChild {
		s.sendToChild(devSimple(devMsgDump))
		select {
		case saved = <-s.stateCh:
		case <-time.After(2 * time.Second):
			log.Printf("syralit[dev]: state dump timed out")
		}
	}

	s.stopChild()
	if err := s.spawnChild(); err != nil {
		log.Printf("syralit[dev]: restart failed: %v", err)
		s.broadcast(devBuildError("failed to start app: " + err.Error()))
		return
	}

	if len(saved) > 0 {
		restore := append([]byte(`{"type":"`+devMsgRestore+`","state":`), saved...)
		restore = append(restore, '}')
		s.sendToChild(restore)
	}
	s.broadcast(devStatus("ready"))
}

// --- browser side ---

type browserConn struct {
	c   *websocket.Conn
	wmu sync.Mutex
}

func (b *browserConn) write(ctx context.Context, data []byte) {
	b.wmu.Lock()
	defer b.wmu.Unlock()
	_ = b.c.Write(ctx, websocket.MessageText, data)
}

func (s *supervisor) handleBrowserWS(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	c.SetReadLimit(32 << 20)
	bc := &browserConn{c: c}

	s.browsersMu.Lock()
	s.browsers[bc] = struct{}{}
	s.browsersMu.Unlock()
	defer func() {
		s.browsersMu.Lock()
		delete(s.browsers, bc)
		s.browsersMu.Unlock()
		c.CloseNow()
	}()

	// Bring the freshly connected browser up to date.
	s.mu.Lock()
	patch, berr := s.lastPatch, s.buildErr
	s.mu.Unlock()
	if patch != nil {
		bc.write(s.ctx, patch)
	}
	if berr != "" {
		bc.write(s.ctx, devBuildError(berr))
	}

	// Forward browser frames (widget_change) to the child.
	for {
		_, data, err := c.Read(s.ctx)
		if err != nil {
			return
		}
		s.sendToChild(data)
	}
}

func (s *supervisor) broadcast(data []byte) {
	s.browsersMu.Lock()
	conns := make([]*browserConn, 0, len(s.browsers))
	for bc := range s.browsers {
		conns = append(conns, bc)
	}
	s.browsersMu.Unlock()
	for _, bc := range conns {
		bc.write(s.ctx, data)
	}
}

// --- http (served by supervisor, independent of child) ---

func (s *supervisor) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		// In dev the supervisor owns the outward port, so it must serve the
		// project's public/ files itself (the child isn't proxied for HTTP).
		if s.opts.PublicDir != "" && s.servePublic(w, r) {
			return
		}
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(renderIndex(s.opts.Title, s.opts.Theme)))
}

// servePublic serves a file from the project's public/ dir for the request path,
// returning true if a file was written. Paths are confined to the dir.
func (s *supervisor) servePublic(w http.ResponseWriter, r *http.Request) bool {
	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" || strings.Contains(name, "..") {
		return false
	}
	full := filepath.Join(s.opts.PublicDir, filepath.FromSlash(name))
	fi, err := os.Stat(full)
	if err != nil || fi.IsDir() {
		return false
	}
	http.ServeFile(w, r, full)
	return true
}

func (s *supervisor) assetsHandler() http.Handler {
	if s.opts.AssetsDir != "" {
		return http.StripPrefix("/_syralit/assets/", http.FileServer(http.Dir(s.opts.AssetsDir)))
	}
	sub, _ := fs.Sub(assetsFS, "assets")
	return http.StripPrefix("/_syralit/assets/", http.FileServer(http.FS(sub)))
}

// --- file watching ---

func (s *supervisor) watch() {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("syralit[dev]: watcher: %v", err)
		return
	}
	defer w.Close()

	addTree(w, s.opts.Dir)
	if s.opts.AssetsDir != "" {
		addTree(w, s.opts.AssetsDir)
	}

	var debounce *time.Timer
	triggerReload := func() {
		if debounce != nil {
			debounce.Stop()
		}
		debounce = time.AfterFunc(250*time.Millisecond, s.reload)
	}

	for {
		select {
		case <-s.done:
			return
		case ev, ok := <-w.Events:
			if !ok {
				return
			}
			name := ev.Name
			switch {
			case strings.HasSuffix(name, ".go"):
				triggerReload()
			case s.opts.AssetsDir != "" && strings.HasSuffix(name, ".css"):
				s.broadcast(devAssetReload("css"))
			case s.opts.AssetsDir != "" && strings.HasSuffix(name, ".js"):
				s.broadcast(devAssetReload("js"))
			}
			// pick up newly created subdirectories
			if ev.Op&fsnotify.Create != 0 {
				if fi, err := os.Stat(name); err == nil && fi.IsDir() {
					_ = w.Add(name)
				}
			}
		case err, ok := <-w.Errors:
			if !ok {
				return
			}
			log.Printf("syralit[dev]: watch error: %v", err)
		}
	}
}

func addTree(w *fsnotify.Watcher, root string) {
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		base := d.Name()
		if base == ".git" || base == "node_modules" || base == "vendor" {
			return filepath.SkipDir
		}
		_ = w.Add(path)
		return nil
	})
}

// --- helpers ---

func (s *supervisor) setBuildErr(msg string) {
	s.mu.Lock()
	s.buildErr = msg
	s.mu.Unlock()
}

func (s *supervisor) clearBuildErr() {
	s.mu.Lock()
	s.buildErr = ""
	s.mu.Unlock()
}

func freePort() (string, error) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	defer l.Close()
	return l.Addr().String(), nil
}

func dialChildWithRetry(ctx context.Context, addr string, timeout time.Duration) (*websocket.Conn, error) {
	url := "ws://" + addr + "/_syralit/ws"
	deadline := time.Now().Add(timeout)
	for {
		dctx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		conn, _, err := websocket.Dial(dctx, url, nil)
		cancel()
		if err == nil {
			return conn, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("child did not come up at %s: %w", addr, err)
		}
		time.Sleep(80 * time.Millisecond)
	}
}

func devStatus(state string) []byte {
	b, _ := json.Marshal(map[string]any{"type": devMsgStatus, "state": state})
	return b
}

func devBuildError(msg string) []byte {
	b, _ := json.Marshal(map[string]any{"type": devMsgBuildError, "error": msg})
	return b
}

func devAssetReload(asset string) []byte {
	b, _ := json.Marshal(map[string]any{"type": devMsgAssetReload, "asset": asset})
	return b
}

func devSimple(typ string) []byte {
	b, _ := json.Marshal(map[string]any{"type": typ})
	return b
}
