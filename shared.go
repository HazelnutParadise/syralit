package syralit

import "sync"

// App-wide shared state, distinct from per-session State. Values live for the
// life of the process and are visible to every connected session, enabling
// real-time collaboration — something Streamlit's per-session model can't do.
var (
	sharedMu    sync.RWMutex
	sharedStore = map[string]any{}

	sessionsMu     sync.Mutex
	activeSessions = map[*session]struct{}{}
)

func registerSession(s *session) {
	sessionsMu.Lock()
	activeSessions[s] = struct{}{}
	sessionsMu.Unlock()
}

func deregisterSession(s *session) {
	sessionsMu.Lock()
	delete(activeSessions, s)
	sessionsMu.Unlock()
}

// broadcastRerun wakes every active session so they re-render and pick up new
// shared state. Safe to call from any goroutine.
func broadcastRerun() {
	sessionsMu.Lock()
	sessions := make([]*session, 0, len(activeSessions))
	for s := range activeSessions {
		sessions = append(sessions, s)
	}
	sessionsMu.Unlock()
	for _, s := range sessions {
		s.requestRerun()
	}
}

// SharedVar is a typed handle to one app-wide shared value.
type SharedVar[T any] struct {
	key string
}

// Shared returns a handle to an app-wide value shared across all sessions,
// seeding it with def on first use. Unlike State (per-session), a Set is visible
// to every connected client and pushes a live update to all of them.
//
//	online := sy.Shared("online", 0)
//	online.Set(online.Get() + 1) // every other browser sees the new count
func Shared[T any](key string, def T) *SharedVar[T] {
	sharedMu.Lock()
	if _, ok := sharedStore[key]; !ok {
		sharedStore[key] = def
	}
	sharedMu.Unlock()
	return &SharedVar[T]{key: key}
}

// Get returns the current shared value (the zero value of T if the stored value
// has a different type).
func (v *SharedVar[T]) Get() T {
	sharedMu.RLock()
	defer sharedMu.RUnlock()
	t, _ := sharedStore[v.key].(T)
	return t
}

// Set updates the shared value and pushes a live re-render to every session.
func (v *SharedVar[T]) Set(val T) {
	sharedMu.Lock()
	sharedStore[v.key] = val
	sharedMu.Unlock()
	broadcastRerun()
}

// Update applies fn to the current value atomically and broadcasts the result —
// use it for read-modify-write (e.g. counters) without a race between sessions.
func (v *SharedVar[T]) Update(fn func(T) T) {
	sharedMu.Lock()
	cur, _ := sharedStore[v.key].(T)
	next := fn(cur)
	sharedStore[v.key] = next
	sharedMu.Unlock()
	broadcastRerun()
}
