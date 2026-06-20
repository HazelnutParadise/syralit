package syralit

import (
	"os"
	"reflect"
)

// coerceNumeric converts a JSON-decoded number (float64) back to the declared
// type T when both are numeric kinds. Used to repair typed State after a hot
// reload round trip. Non-numeric or interface T returns false.
func coerceNumeric[T any](raw any) (T, bool) {
	var zero T
	if raw == nil {
		return zero, false
	}
	tt := reflect.TypeOf(zero)
	if tt == nil { // T is an interface type
		return zero, false
	}
	rv := reflect.ValueOf(raw)
	if isNumericKind(rv.Kind()) && isNumericKind(tt.Kind()) && rv.Type().ConvertibleTo(tt) {
		return rv.Convert(tt).Interface().(T), true
	}
	return zero, false
}

func isNumericKind(k reflect.Kind) bool {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	return false
}

// State returns a typed handle to a value that persists across reruns within the
// current session. On first use the default is stored; later reruns see whatever
// was last Set.
//
//	count := sy.State("count", 0)
//	if sy.Button("Add") { count.Set(count.Get() + 1) }
//	sy.Textf("Count: %d", count.Get())
//
// Note (deviation from SPEC §7): SPEC sketched both a `type State[T]` interface
// and a `sy.State(...)` constructor, which collide in Go (a package can't have a
// type and a func with the same name). We keep the constructor and return the
// concrete *StateValue[T].
func State[T any](key string, def T) *StateValue[T] {
	sess := current().sess
	sess.mu.Lock()
	if _, ok := sess.store[key]; !ok {
		sess.store[key] = def
	}
	sess.mu.Unlock()
	return &StateValue[T]{key: key, sess: sess}
}

type StateValue[T any] struct {
	key  string
	sess *session
}

func (s *StateValue[T]) Get() T {
	s.sess.mu.Lock()
	defer s.sess.mu.Unlock()
	raw := s.sess.store[s.key]
	if v, ok := raw.(T); ok {
		return v
	}
	// After a hot reload, state comes back through a JSON round trip, so an int
	// arrives as float64. Coerce numeric kinds back to the declared type T.
	if v, ok := coerceNumeric[T](raw); ok {
		return v
	}
	var zero T
	return zero
}

func (s *StateValue[T]) Set(v T) {
	s.sess.mu.Lock()
	s.sess.store[s.key] = v
	s.sess.mu.Unlock()
}

func (s *StateValue[T]) Clear() {
	s.sess.mu.Lock()
	delete(s.sess.store, s.key)
	s.sess.mu.Unlock()
}

// SessionStore is the non-generic session store API (SPEC §7).
type SessionStore struct{ sess *session }

// Session returns the current rerun's session store for untyped access.
func Session() *SessionStore { return &SessionStore{sess: current().sess} }

func (s *SessionStore) Set(key string, value any) {
	s.sess.mu.Lock()
	s.sess.store[key] = value
	s.sess.mu.Unlock()
}

func (s *SessionStore) Get(key string) (any, bool) {
	s.sess.mu.Lock()
	defer s.sess.mu.Unlock()
	v, ok := s.sess.store[key]
	return v, ok
}

// Secrets returns a secret value from the [secrets] section of syralit.toml,
// falling back to an environment variable of the same name. This lets apps
// keep API keys out of source code.
//
//	apiKey := sy.Secrets("OPENAI_API_KEY")
func Secrets(key string) string {
	if loadedSecrets != nil {
		if v, ok := loadedSecrets[key]; ok {
			return v
		}
	}
	return os.Getenv(key)
}
