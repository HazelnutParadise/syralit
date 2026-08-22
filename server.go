package syralit

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/coder/websocket"
	"github.com/yuin/goldmark"
)

// Config controls the dev server.
type Config struct {
	Title string
	Host  string
	Port  int
	Theme Theme

	// MaxUploadSizeMB caps FileUploader/CameraInput payloads, in megabytes.
	// 0 means the default (10 MB). Configurable via [server] max_upload_size_mb
	// in syralit.toml.
	MaxUploadSizeMB int

	// SSLCertFile / SSLKeyFile serve the app over HTTPS when both are set
	// (PEM files). Configurable via [server] ssl_cert_file / ssl_key_file.
	SSLCertFile string
	SSLKeyFile  string

	// Lang sets the document language of the app shell (the lang attribute on
	// <html>). Defaults to "en". Configurable via the lang key in syralit.toml.
	Lang string

	// Dir sets the writing direction of the app shell (the dir attribute on
	// <html>): "ltr", "rtl" or "auto". Empty omits the attribute, which the
	// browser reads as "ltr". Arabic, Hebrew, Persian and Urdu apps need "rtl"
	// explicitly — the language does not imply the direction. Configurable via
	// the dir key in syralit.toml.
	Dir string

	// HeadHTML is inserted verbatim at the end of <head>, after the stylesheet
	// link and the theme block: a description or Open Graph tag, a favicon
	// link, a preconnect hint. It is neither escaped nor validated, exactly
	// like sy.HTML(), so never build it out of user input. The value applies to
	// every request; there is no per-request variant. Configurable via the
	// head_html key in syralit.toml.
	HeadHTML string

	// UIStrings overrides the framework's built-in UI text (localization).
	// Known keys: "connecting", "loading", "add_new", "file_too_large",
	// "menu", "menu_get_help", "menu_report_bug", "menu_about".
	// Configurable via the [i18n] table in syralit.toml.
	UIStrings map[string]string

	// DocumentFunc, when non-nil, is called once per document request — before
	// the shell is rendered and before any session exists — so the title and
	// <head> can depend on the request (a headline looked up from a query
	// parameter, a canonical URL). Non-empty fields in the returned Document
	// override the corresponding Config values for that response only; empty
	// ones keep them. It is never called during a rerun. The returned HeadHTML
	// goes in verbatim, so anything taken from the request must be passed
	// through html.EscapeString first. Code only; there is no syralit.toml key.
	DocumentFunc func(*http.Request) Document
}

// Document carries the per-request parts of the HTML shell, returned by
// Config.DocumentFunc. A zero field means "use the Config value". Title wins
// over the page-URL title and Config.Title; an invalid Lang or Dir is ignored
// in favour of the Config value rather than logged, since this runs per request.
type Document struct {
	Title    string
	Lang     string
	Dir      string
	HeadHTML string
}

func (c *Config) uploadLimit() int64 {
	if c.MaxUploadSizeMB > 0 {
		return int64(c.MaxUploadSizeMB) << 20
	}
	return 10 << 20
}

func (c *Config) applyDefaults() {
	if c.Host == "" {
		c.Host = "127.0.0.1"
	}
	if c.Port == 0 {
		c.Port = 8600 // not Streamlit's 8501, to avoid clashing when both run locally
	}
	if c.Title == "" {
		c.Title = "Syralit App"
	}
}

// App starts a Syralit app with default config. It blocks until the server stops.
//
// In single-page mode (no AddPage calls), fn is the sole page function.
// In multi-page mode (AddPage called in init()), fn is ignored and pages are
// rendered from the registry. Pass nil when using AddPage.
func App(fn func()) {
	if fn == nil && !hasPages() {
		log.Fatal("syralit: App(nil) called but no pages registered via AddPage")
	}
	if hasPages() {
		fn = nil
	}
	if err := Run(Config{}, fn); err != nil {
		log.Fatalf("syralit: %v", err)
	}
}

// envDevAddr, when set by the dev supervisor, tells this process to run as a hot
// reload child: bind the given internal address instead of the configured port,
// and honor the dev control messages on the WebSocket (state dump/restore).
const envDevAddr = "SYRALIT_DEV_ADDR"

// envDevURL and envDevSession are also set by the dev supervisor for its child:
// the outward URL browsers use (the documented $SYRALIT_URL convention), and an
// identifier that is unique per supervisor run but stable across child rebuilds.
// integrations/desktop uses them to keep one native window alive per dev session.
const (
	envDevURL     = "SYRALIT_URL"
	envDevSession = "SYRALIT_DEV_SESSION"
)

// Run starts a Syralit app with explicit config. Values left unset are filled
// from syralit.toml in the working directory if present, then by defaults.
func Run(cfg Config, fn func()) error {
	loadFileConfig(".").applyToConfig(&cfg)
	cfg.applyDefaults()
	s := &server{cfg: cfg, appFn: fn, shell: resolveShell(cfg.Lang, cfg.Dir, cfg.HeadHTML, cfg.UIStrings)}
	if addr := os.Getenv(envDevAddr); addr != "" {
		log.Printf("syralit[dev-child]: listening on %s", addr)
		return http.ListenAndServe(addr, s.handler())
	}
	return s.listenAndServe()
}

// Handler returns the app as an http.Handler, so a Syralit app can be mounted
// inside an existing Go HTTP server instead of owning the process:
//
//	mux.Handle("/dashboard/", http.StripPrefix("/dashboard", sy.Handler(sy.Config{}, myApp)))
//
// Config resolution matches Run (syralit.toml, then defaults); Host/Port are
// ignored since the caller owns the listener.
func Handler(cfg Config, fn func()) http.Handler {
	loadFileConfig(".").applyToConfig(&cfg)
	cfg.applyDefaults()
	s := &server{cfg: cfg, appFn: fn, shell: resolveShell(cfg.Lang, cfg.Dir, cfg.HeadHTML, cfg.UIStrings)}
	return s.handler()
}

// GetOption returns a resolved configuration value by key: "title",
// "server.host", "server.port", "server.max_upload_size_mb", "theme.mode",
// "theme.accent", "theme.radius". Unknown keys return nil. Values reflect the
// effective config of the server serving the current session (code >
// syralit.toml > defaults), so call it from inside the page function. Outside
// a rerun there is no server to ask and the zero Config's values come back.
func GetOption(key string) any {
	var resolvedConfig Config
	if cur != nil {
		resolvedConfig = cur.sess.cfg
	}
	switch key {
	case "title":
		return resolvedConfig.Title
	case "server.host":
		return resolvedConfig.Host
	case "server.port":
		return resolvedConfig.Port
	case "server.max_upload_size_mb":
		return int(resolvedConfig.uploadLimit() >> 20)
	case "theme.mode":
		return resolvedConfig.Theme.Mode
	case "theme.accent":
		return resolvedConfig.Theme.Accent
	case "theme.radius":
		return resolvedConfig.Theme.Radius
	}
	return nil
}

type server struct {
	cfg   Config
	appFn func()
	shell shellConfig // resolved from cfg once; read by every index render
}

// newSession starts a session that belongs to this server, so the page
// function sees this server's config rather than some other handler's.
func (s *server) newSession() *session {
	sess := newSession(s.appFn)
	sess.cfg = s.cfg
	return sess
}

func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)

	sub, _ := fs.Sub(assetsFS, "assets")
	frameworkAssets := http.StripPrefix("/_syralit/assets/", http.FileServer(http.FS(sub)))
	mux.Handle("GET /_syralit/assets/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A user override (sy.StaticAssets) shadows the built-in asset of the
		// same name; otherwise fall through to the embedded framework copy.
		name := strings.TrimPrefix(r.URL.Path, "/_syralit/assets/")
		if serveOverlayAsset(w, r, name) {
			return
		}
		frameworkAssets.ServeHTTP(w, r)
	}))

	mux.HandleFunc("GET /_syralit/ws", s.handleWS)
	// SSE fallback transport for environments where WebSocket can't connect.
	mux.HandleFunc("GET /_syralit/sse", s.handleSSE)
	mux.HandleFunc("POST /_syralit/msg", s.handleMsg)
	registerArtifactEndpoints(mux)
	return mux
}

func (s *server) listenAndServe() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	if s.cfg.SSLCertFile != "" && s.cfg.SSLKeyFile != "" {
		log.Printf("syralit: %q running on https://%s", s.cfg.Title, addr)
		return http.ListenAndServeTLS(addr, s.cfg.SSLCertFile, s.cfg.SSLKeyFile, s.handler())
	}
	log.Printf("syralit: %q running on http://%s", s.cfg.Title, addr)
	return http.ListenAndServe(addr, s.handler())
}

// requestBasePath recovers the mount prefix when the app runs behind
// http.StripPrefix (sy.Handler under a sub-path): RequestURI keeps the
// original path while URL.Path has been stripped.
func requestBasePath(r *http.Request) string {
	orig := r.RequestURI
	if i := strings.IndexByte(orig, '?'); i >= 0 {
		orig = orig[:i]
	}
	if orig != r.URL.Path && strings.HasSuffix(orig, r.URL.Path) {
		return strings.TrimSuffix(orig, r.URL.Path)
	}
	return ""
}

// pageQueryKey marks the page a browser is opening. The client puts it on the
// WebSocket/SSE URL — not in the address bar — because those transports carry
// only the query string, so without it the first render would always be the
// default page even when the visitor asked for /reports.
const pageQueryKey = "__sy_page"

// takeInitialPage pulls the marker out of a transport's query parameters and
// returns the page title it names. The key is removed so it never shows up in
// sy.QueryParams(), which is the app's own namespace.
func takeInitialPage(qp map[string]string) string {
	slug := qp[pageQueryKey]
	delete(qp, pageQueryKey)
	if p, ok := pageBySlug(slug); ok {
		return p.title
	}
	return ""
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	title := s.cfg.Title
	if r.URL.Path != "/" {
		// Non-root paths fall through to user static files (sy.Static / an
		// embedded public/ dir), then to the page URLs, before 404. Files win:
		// a real robots.txt must stay reachable whatever the pages are called.
		if serveRootStatic(w, r) {
			return
		}
		p, ok := pageBySlug(strings.Trim(r.URL.Path, "/"))
		if !ok {
			http.NotFound(w, r)
			return
		}
		// The shell carries the page's own title, so a crawler or View Source
		// sees the page that was asked for rather than the app-wide name.
		title = p.title
	}
	shell := s.shell
	if s.cfg.DocumentFunc != nil {
		doc := s.cfg.DocumentFunc(r)
		if doc.Title != "" {
			title = doc.Title
		}
		shell = shell.override(doc)
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(renderIndex(title, s.cfg.Theme, shell, requestBasePath(r))))
}

// inbound message from the browser (SPEC §13). The __dev_* fields are only used
// in dev child mode, where the supervisor (not a real browser) drives the socket.
type clientMsg struct {
	Type        string         `json:"type"`
	WidgetID    string         `json:"widget_id"`
	Value       any            `json:"value"`
	IsButton    bool           `json:"is_button"`
	Page        string         `json:"page,omitempty"`         // for page_change
	Changes     []widgetChange `json:"changes,omitempty"`      // for form_submit
	State       *sessionState  `json:"state,omitempty"`        // for __dev_restore
	FragmentKey string         `json:"fragment_key,omitempty"` // for fragment_rerun
	SessionID   string         `json:"session_id,omitempty"`   // SSE transport (POST /_syralit/msg)
}

type widgetChange struct {
	WidgetID string `json:"widget_id"`
	Value    any    `json:"value"`
}

func (s *server) handleWS(w http.ResponseWriter, r *http.Request) {
	// coder/websocket authorizes the request host and rejects cross-origin
	// browser handshakes by default.
	c, err := websocket.Accept(w, r, nil)
	if err != nil {
		return
	}
	limit := int64(32 << 20) // dev restore payloads can be large
	if l := s.cfg.uploadLimit() + (4 << 20); l > limit {
		limit = l // base64 upload + envelope headroom
	}
	c.SetReadLimit(limit)
	defer c.CloseNow()

	ctx := context.Background()
	sess := s.newSession()
	registerSession(sess)
	defer deregisterSession(sess)

	qp := make(map[string]string)
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			qp[k] = v[0]
		}
	}
	sess.mu.Lock()
	sess.currentPage = takeInitialPage(qp)
	sess.queryParams = qp
	sess.reqCtx = captureRequest(r)
	sess.mu.Unlock()
	resolveSessionUser(sess)

	sink := wsSink{c: c, ctx: ctx}

	// Initial render.
	if err := pushUI(sink, sess); err != nil {
		return
	}

	// Decode client frames on a reader goroutine so the event loop can also
	// react to server-initiated reruns (background Tasks signalling sess.wake).
	msgCh := make(chan clientMsg)
	go func() {
		for {
			_, data, err := c.Read(ctx)
			if err != nil {
				close(msgCh)
				return
			}
			var msg clientMsg
			if err := json.Unmarshal(data, &msg); err != nil {
				continue
			}
			select {
			case msgCh <- msg:
			case <-ctx.Done():
				return
			}
		}
	}()

	for {
		select {
		case msg, ok := <-msgCh:
			if !ok {
				return // connection closed
			}
			if !s.handleClientMsg(sink, sess, msg) {
				return
			}
		case <-sess.wake: // a background Task (or other server event) finished
			if err := pushUI(sink, sess); err != nil {
				return
			}
		}
	}
}

// handleClientMsg processes one inbound frame, pushing any resulting UI through
// sink. It returns false if the connection should be torn down (write error).
func (s *server) handleClientMsg(sink uiSink, sess *session, msg clientMsg) bool {
	switch msg.Type {
	case "widget_change":
		sess.applyChange(msg.WidgetID, msg.Value, msg.IsButton)
		if fragKey, fragFn, ok := sess.fragmentForWidget(msg.WidgetID); ok && !msg.IsButton {
			return pushFragmentUI(sink, sess, fragKey, fragFn) == nil
		}
		return pushUI(sink, sess) == nil
	case "fragment_rerun": // auto-refresh (RunEvery) tick from the client
		if fn, ok := sess.fragmentByKey(msg.FragmentKey); ok {
			return pushFragmentUI(sink, sess, msg.FragmentKey, fn) == nil
		}
		return true
	case "page_change":
		page := msg.Page
		if pageByTitle(page) == nil {
			// The dev supervisor forwards the slug from the browser URL; it has
			// no page registry of its own to resolve it with.
			if p, ok := pageBySlug(page); ok {
				page = p.title
			}
		}
		sess.setCurrentPage(page)
		return pushUI(sink, sess) == nil
	case "form_submit":
		for _, ch := range msg.Changes {
			sess.applyChange(ch.WidgetID, ch.Value, false)
		}
		sess.pressButton(msg.WidgetID)
		formID := sess.formOf(msg.WidgetID)
		if formID != "" && sess.isClearOnSubmit(formID) {
			// Render once with the submitted values (so the handler sees them),
			// but blank the form's inputs in the sent tree; then drop the stored
			// values so later renders stay cleared.
			ok := pushUI(sink, sess, func(root *Node) {
				if form := findNodeByID(root, formID); form != nil {
					clearFormInputs(form)
				}
			}) == nil
			sess.clearFormWidgets(formID)
			return ok
		}
		return pushUI(sink, sess) == nil
	case devMsgDump: // supervisor asks for state before restarting this child
		st := sess.dumpState()
		out, _ := json.Marshal(map[string]any{"type": devMsgState, "state": st})
		return sink.send(out) == nil
	case devMsgRestore: // supervisor hands back state into the fresh child
		sess.restoreState(msg.State)
		return pushUI(sink, sess) == nil
	}
	return true
}

// pushUI runs a rerun and sends the resulting UI tree to the browser.
// Sidebar user content (nodes of type "sidebar_content") is separated from the
// main tree and sent in a dedicated "sidebar" field.
func pushUI(sink uiSink, sess *session, transforms ...func(*Node)) error {
	streamer := func(rc *renderContext) {
		rc.streamer = func(id, chunk string) {
			out, _ := json.Marshal(map[string]any{
				"type": "stream_append", "id": id, "chunk": chunk,
			})
			_ = sink.send(out)
		}
	}

	var root *Node
	for range 5 {
		sess.mu.Lock()
		sess.needsRerun = false
		sess.mu.Unlock()
		root = runRerun(sess, streamer)
		sess.mu.Lock()
		stable := !sess.needsRerun
		sess.mu.Unlock()
		if stable {
			break
		}
	}

	for _, tf := range transforms {
		tf(root)
	}
	updateArtifactPlacements(sess, root)

	var mainNodes, sidebarNodes []*Node
	for _, n := range root.Children {
		if n.Type == "sidebar_content" {
			sidebarNodes = append(sidebarNodes, n.Children...)
		} else {
			mainNodes = append(mainNodes, n)
		}
	}

	msg := map[string]any{
		"type":  "ui_patch",
		"nodes": mainNodes,
	}
	if len(sidebarNodes) > 0 {
		msg["sidebar"] = sidebarNodes
	}
	if hasPages() || len(sidebarNodes) > 0 {
		msg["pages"] = pageInfos()
		msg["active_page"] = sess.activePage()
	}

	sess.mu.Lock()
	if sess.queryDirty {
		msg["set_query"] = cloneStrMap(sess.queryParams)
		sess.queryDirty = false
	}
	if len(sess.pendingToasts) > 0 {
		msg["toasts"] = sess.pendingToasts
		sess.pendingToasts = nil
	}
	if sess.pageConfig != nil {
		pc := map[string]any{}
		if sess.pageConfig.title != "" {
			pc["title"] = sess.pageConfig.title
		}
		if sess.pageConfig.icon != "" {
			pc["icon"] = sess.pageConfig.icon
		}
		if sess.pageConfig.layout != "" {
			pc["layout"] = sess.pageConfig.layout
		}
		if sess.pageConfig.logo != "" {
			pc["logo"] = sess.pageConfig.logo
		}
		if sess.pageConfig.primaryColor != "" {
			pc["primary_color"] = sess.pageConfig.primaryColor
		}
		if sess.pageConfig.bgColor != "" {
			pc["bg_color"] = sess.pageConfig.bgColor
		}
		if sess.pageConfig.textColor != "" {
			pc["text_color"] = sess.pageConfig.textColor
		}
		if sess.pageConfig.sidebarState != "" {
			pc["sidebar_state"] = sess.pageConfig.sidebarState
		}
		if sess.pageConfig.menuHelpURL != "" {
			pc["menu_help_url"] = sess.pageConfig.menuHelpURL
		}
		if sess.pageConfig.menuBugURL != "" {
			pc["menu_bug_url"] = sess.pageConfig.menuBugURL
		}
		if sess.pageConfig.menuAbout != "" {
			var buf bytes.Buffer
			if err := goldmark.Convert([]byte(sess.pageConfig.menuAbout), &buf); err == nil {
				pc["menu_about"] = buf.String()
			}
		}
		if len(pc) > 0 {
			msg["page_config"] = pc
		}
	}
	sess.mu.Unlock()

	out, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return sink.send(out)
}

// pushFragmentUI runs only a fragment's function and sends a partial update.
func pushFragmentUI(sink uiSink, sess *session, key string, fn func()) error {
	root := runFragment(sess, key, fn)

	msg := map[string]any{
		"type":         "fragment_patch",
		"fragment_key": key,
		"nodes":        root.Children,
	}

	sess.mu.Lock()
	if sess.queryDirty {
		msg["set_query"] = cloneStrMap(sess.queryParams)
		sess.queryDirty = false
	}
	if len(sess.pendingToasts) > 0 {
		msg["toasts"] = sess.pendingToasts
		sess.pendingToasts = nil
	}
	sess.mu.Unlock()

	out, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return sink.send(out)
}

func htmlEscape(s string) string {
	repl := map[rune]string{'&': "&amp;", '<': "&lt;", '>': "&gt;", '"': "&quot;"}
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if e, ok := repl[r]; ok {
			out = append(out, []rune(e)...)
		} else {
			out = append(out, r)
		}
	}
	return string(out)
}
