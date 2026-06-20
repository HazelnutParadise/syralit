package syralit

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/coder/websocket"
)

// Config controls the dev server. Fields beyond these (Theme, upload limits) are
// reserved for later stages; see SPEC §6.1 / §15.
type Config struct {
	Title string
	Host  string
	Port  int
	Theme Theme
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

// Run starts a Syralit app with explicit config. Values left unset are filled
// from syralit.toml in the working directory if present, then by defaults.
func Run(cfg Config, fn func()) error {
	loadFileConfig(".").applyToConfig(&cfg)
	cfg.applyDefaults()
	s := &server{cfg: cfg, appFn: fn}
	if addr := os.Getenv(envDevAddr); addr != "" {
		log.Printf("syralit[dev-child]: listening on %s", addr)
		return http.ListenAndServe(addr, s.handler())
	}
	return s.listenAndServe()
}

type server struct {
	cfg   Config
	appFn func()
}

func (s *server) handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /", s.handleIndex)

	sub, _ := fs.Sub(assetsFS, "assets")
	mux.Handle("GET /_syralit/assets/", http.StripPrefix("/_syralit/assets/", http.FileServer(http.FS(sub))))

	mux.HandleFunc("GET /_syralit/ws", s.handleWS)
	return mux
}

func (s *server) listenAndServe() error {
	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	log.Printf("syralit: %q running on http://%s", s.cfg.Title, addr)
	return http.ListenAndServe(addr, s.handler())
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(renderIndex(s.cfg.Title, s.cfg.Theme)))
}

// inbound message from the browser (SPEC §13). The __dev_* fields are only used
// in dev child mode, where the supervisor (not a real browser) drives the socket.
type clientMsg struct {
	Type     string         `json:"type"`
	WidgetID string         `json:"widget_id"`
	Value    any            `json:"value"`
	IsButton bool           `json:"is_button"`
	Page     string         `json:"page,omitempty"`    // for page_change
	Changes  []widgetChange `json:"changes,omitempty"` // for form_submit
	State    *sessionState  `json:"state,omitempty"`   // for __dev_restore
}

type widgetChange struct {
	WidgetID string `json:"widget_id"`
	Value    any    `json:"value"`
}

func (s *server) handleWS(w http.ResponseWriter, r *http.Request) {
	// InsecureSkipVerify is acceptable here: MVP binds to localhost only (SPEC §15).
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	c.SetReadLimit(32 << 20) // dev restore payloads can be large
	defer c.CloseNow()

	ctx := context.Background()
	sess := newSession(s.appFn)

	qp := make(map[string]string)
	for k, v := range r.URL.Query() {
		if len(v) > 0 {
			qp[k] = v[0]
		}
	}
	sess.mu.Lock()
	sess.queryParams = qp
	sess.mu.Unlock()

	// Initial render.
	if err := pushUI(ctx, c, sess); err != nil {
		return
	}

	for {
		_, data, err := c.Read(ctx)
		if err != nil {
			return // connection closed
		}
		var msg clientMsg
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		switch msg.Type {
		case "widget_change":
			sess.applyChange(msg.WidgetID, msg.Value, msg.IsButton)
			if fragKey, fragFn, ok := sess.fragmentForWidget(msg.WidgetID); ok && !msg.IsButton {
				if err := pushFragmentUI(ctx, c, sess, fragKey, fragFn); err != nil {
					return
				}
			} else {
				if err := pushUI(ctx, c, sess); err != nil {
					return
				}
			}
		case "page_change":
			sess.setCurrentPage(msg.Page)
			if err := pushUI(ctx, c, sess); err != nil {
				return
			}
		case "form_submit":
			for _, ch := range msg.Changes {
				sess.applyChange(ch.WidgetID, ch.Value, false)
			}
			sess.pressButton(msg.WidgetID)
			if err := pushUI(ctx, c, sess); err != nil {
				return
			}
		case devMsgDump: // supervisor asks for state before restarting this child
			st := sess.dumpState()
			out, _ := json.Marshal(map[string]any{"type": devMsgState, "state": st})
			if err := c.Write(ctx, websocket.MessageText, out); err != nil {
				return
			}
		case devMsgRestore: // supervisor hands back state into the fresh child
			sess.restoreState(msg.State)
			if err := pushUI(ctx, c, sess); err != nil {
				return
			}
		}
	}
}

// pushUI runs a rerun and sends the resulting UI tree to the browser.
// Sidebar user content (nodes of type "sidebar_content") is separated from the
// main tree and sent in a dedicated "sidebar" field.
func pushUI(ctx context.Context, c *websocket.Conn, sess *session) error {
	var root *Node
	for range 5 {
		sess.mu.Lock()
		sess.needsRerun = false
		sess.mu.Unlock()
		root = runRerun(sess)
		sess.mu.Lock()
		stable := !sess.needsRerun
		sess.mu.Unlock()
		if stable {
			break
		}
	}

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
		if len(pc) > 0 {
			msg["page_config"] = pc
		}
	}
	sess.mu.Unlock()

	out, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.Write(ctx, websocket.MessageText, out)
}

// pushFragmentUI runs only a fragment's function and sends a partial update.
func pushFragmentUI(ctx context.Context, c *websocket.Conn, sess *session, key string, fn func()) error {
	root := runFragment(sess, key, fn)

	msg := map[string]any{
		"type":         "fragment_patch",
		"fragment_key": key,
		"nodes":        root.Children,
	}

	sess.mu.Lock()
	if len(sess.pendingToasts) > 0 {
		msg["toasts"] = sess.pendingToasts
		sess.pendingToasts = nil
	}
	sess.mu.Unlock()

	out, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return c.Write(ctx, websocket.MessageText, out)
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
