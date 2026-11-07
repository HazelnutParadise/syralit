package syralit

import (
	"crypto/rand"
	"encoding/hex"
	"sync"
)

// Session is one browser connection's private state: the widget values it has
// sent, transient button presses, and the user-managed session store. Each
// WebSocket connection owns exactly one Session.
type session struct {
	id    string
	appFn func() // the user's App function (nil in multi-page mode)
	// cfg is the resolved config of the server this session belongs to. The
	// page function reads it (sy.GetOption, the upload cap on uploader widgets),
	// and it has to live here rather than in package state so two handlers in
	// one process each see their own values. Zero for sessions without a server
	// (RenderOnce, AppTest).
	cfg Config

	mu            sync.Mutex
	widgets       map[string]any  // widget id -> persisted value (text, checkbox, select...)
	transient     map[string]bool // button id -> pressed during the current cycle
	store         map[string]any  // sy.State / sy.Session() user store
	currentPage   string          // multi-page: active page title (empty = default)
	pendingToasts []map[string]any
	pageConfig    *pageConfig
	needsRerun    bool // set by SwitchPage to re-render after stop
	queryParams   map[string]string
	queryDirty    bool // SetQueryParam called; push the URL update to the client
	reqCtx        RequestContext

	fragmentFns     map[string]func() // fragment key -> registered function
	widgetFragments map[string]string // widget id -> fragment key (for partial reruns)

	clearOnSubmit map[string]bool   // form id -> reset inputs after submit
	formWidgets   map[string]string // widget id -> form id (members of a form)

	// acks maps a browser tab's client id to the highest frame seq handled for
	// it; pushes echo it so the tab can spot trees that predate its typing.
	// ackSeen orders the tabs by last frame, for evicting the stalest one.
	acks    map[string]int64
	ackSeen map[string]uint64
	ackTick uint64

	// handleMu serializes frame handling and pushes for one session. A
	// WebSocket session already handles one frame at a time; SSE handles each
	// POST on its own goroutine, where an ack recorded mid-rerun would
	// otherwise be echoed on a tree that never saw that frame.
	handleMu sync.Mutex

	wake chan struct{} // server-initiated rerun signal (background Tasks)
	sink uiSink        // SSE transport: where pushes go for POST-driven updates
}

type pageConfig struct {
	title        string
	icon         string
	layout       string // "centered" or "wide"
	sidebarState string // "expanded" (default) or "collapsed"
	logo         string // sidebar logo URL
	primaryColor string
	bgColor      string
	textColor    string
	menuHelpURL  string // "Get help" link in the app menu
	menuBugURL   string // "Report a bug" link in the app menu
	menuAbout    string // markdown for the About dialog
}

func newSession(appFn func()) *session {
	return &session{
		id:        newID(),
		appFn:     appFn,
		widgets:   map[string]any{},
		transient: map[string]bool{},
		store:     map[string]any{},
		wake:      make(chan struct{}, 1),
	}
}

// requestRerun asks the connection's event loop to re-render and push. It is
// safe to call from any goroutine (e.g. a background Task) and coalesces:
// multiple calls before the loop wakes collapse into one re-render.
func (s *session) requestRerun() {
	if s.wake == nil {
		return
	}
	select {
	case s.wake <- struct{}{}:
	default:
	}
}

func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (s *session) widgetValue(id string) (any, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.widgets[id]
	return v, ok
}

func (s *session) setWidget(id string, v any) {
	s.mu.Lock()
	s.widgets[id] = v
	s.mu.Unlock()
}

// takeWidget returns and removes a widget value — one-shot semantics for
// button-like widgets that carry a payload (e.g. MenuButton).
func (s *session) takeWidget(id string) any {
	s.mu.Lock()
	defer s.mu.Unlock()
	v, ok := s.widgets[id]
	if ok {
		delete(s.widgets, id)
	}
	return v
}

func (s *session) pressButton(id string) {
	s.mu.Lock()
	s.transient[id] = true
	s.mu.Unlock()
}

func (s *session) buttonPressed(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.transient[id]
}

func (s *session) resetTransient() {
	s.mu.Lock()
	clear(s.transient)
	s.mu.Unlock()
}

// maxAckClients caps how many tabs' acks one session remembers. A normal
// session serves one tab; the dev supervisor shares one child session among
// every tab and reload, which would otherwise grow the map without bound.
const maxAckClients = 16

// maxClientIDLen bounds a client id; the runtime sends about 20 characters.
const maxClientIDLen = 64

// recordAck notes that frame seq from client has been handled. Frames without
// a usable client id (older runtimes, supervisor-internal frames) are ignored.
// When the map is full, the tab that sent a frame least recently is dropped.
func (s *session) recordAck(client string, seq int64) {
	if client == "" || len(client) > maxClientIDLen || seq <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.acks == nil {
		s.acks = map[string]int64{}
		s.ackSeen = map[string]uint64{}
	}
	if _, ok := s.acks[client]; !ok && len(s.acks) >= maxAckClients {
		stalest, oldest := "", uint64(0)
		for k, t := range s.ackSeen {
			if stalest == "" || t < oldest {
				stalest, oldest = k, t
			}
		}
		delete(s.acks, stalest)
		delete(s.ackSeen, stalest)
	}
	s.ackTick++
	s.ackSeen[client] = s.ackTick
	if seq > s.acks[client] {
		s.acks[client] = seq
	}
}

// applyChange records an incoming widget_change from the browser. Buttons are
// recorded as a one-shot press; everything else overwrites the stored value.
func (s *session) applyChange(widgetID string, value any, isButton bool) {
	if isButton {
		s.pressButton(widgetID)
		return
	}
	s.setWidget(widgetID, value)
}

func (s *session) activePage() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.currentPage == "" {
		return defaultPageTitle()
	}
	return s.currentPage
}

func (s *session) setCurrentPage(page string) {
	s.mu.Lock()
	if s.currentPage != page {
		s.currentPage = page
		s.needsRerun = true
	}
	s.mu.Unlock()
}

// dumpState snapshots widget values, session store, and active page for hot
// reload (transient button presses are intentionally not carried).
func (s *session) dumpState() sessionState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return sessionState{
		Widgets:     cloneAnyMap(s.widgets),
		Store:       cloneAnyMap(s.store),
		CurrentPage: s.currentPage,
	}
}

// restoreState seeds a fresh child with state from before the reload.
func (s *session) restoreState(st *sessionState) {
	if st == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, v := range st.Widgets {
		s.widgets[k] = v
	}
	for k, v := range st.Store {
		s.store[k] = v
	}
	if st.CurrentPage != "" {
		s.currentPage = st.CurrentPage
	}
}

func (s *session) registerFragment(key string, fn func()) {
	s.mu.Lock()
	if s.fragmentFns == nil {
		s.fragmentFns = map[string]func(){}
	}
	s.fragmentFns[key] = fn
	s.mu.Unlock()
}

func (s *session) registerWidgetFragment(widgetID, fragmentKey string) {
	s.mu.Lock()
	if s.widgetFragments == nil {
		s.widgetFragments = map[string]string{}
	}
	s.widgetFragments[widgetID] = fragmentKey
	s.mu.Unlock()
}

func (s *session) fragmentForWidget(widgetID string) (string, func(), bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, ok := s.widgetFragments[widgetID]
	if !ok {
		return "", nil, false
	}
	fn, ok := s.fragmentFns[key]
	return key, fn, ok
}

// fragmentByKey looks up a registered fragment function by its key, used by the
// auto-refresh (RunEvery) path.
func (s *session) fragmentByKey(key string) (func(), bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	fn, ok := s.fragmentFns[key]
	return fn, ok
}

func (s *session) registerClearOnSubmit(formID string) {
	s.mu.Lock()
	if s.clearOnSubmit == nil {
		s.clearOnSubmit = map[string]bool{}
	}
	s.clearOnSubmit[formID] = true
	s.mu.Unlock()
}

func (s *session) registerFormWidget(widgetID, formID string) {
	s.mu.Lock()
	if s.formWidgets == nil {
		s.formWidgets = map[string]string{}
	}
	s.formWidgets[widgetID] = formID
	s.mu.Unlock()
}

// formOf returns the form id a widget belongs to (the submit button included).
func (s *session) formOf(widgetID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.formWidgets[widgetID]
}

func (s *session) isClearOnSubmit(formID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.clearOnSubmit[formID]
}

// clearFormWidgets drops the stored values of every widget in a form, so the
// next render shows them empty (paired with clearing the just-sent tree).
func (s *session) clearFormWidgets(formID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for wid, fid := range s.formWidgets {
		if fid == formID {
			delete(s.widgets, wid)
		}
	}
}

func cloneStrMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func cloneAnyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
