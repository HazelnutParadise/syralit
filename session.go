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

	mu            sync.Mutex
	widgets       map[string]any  // widget id -> persisted value (text, checkbox, select...)
	transient     map[string]bool // button id -> pressed during the current cycle
	store         map[string]any  // sy.State / sy.Session() user store
	currentPage   string          // multi-page: active page title (empty = default)
	pendingToasts []map[string]any
	pageConfig    *pageConfig
	needsRerun    bool // set by SwitchPage to re-render after stop
	queryParams   map[string]string
}

type pageConfig struct {
	title        string
	icon         string
	layout       string // "centered" or "wide"
	logo         string // sidebar logo URL
	primaryColor string
	bgColor      string
	textColor    string
}

func newSession(appFn func()) *session {
	return &session{
		id:        newID(),
		appFn:     appFn,
		widgets:   map[string]any{},
		transient: map[string]bool{},
		store:     map[string]any{},
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

func cloneAnyMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
