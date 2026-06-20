package syralit

import (
	"database/sql"
	"os"
	"reflect"
	"sync"
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

// QueryParams returns a copy of the URL query parameters from the current
// session's WebSocket connection. Useful for deep linking and sharing app
// state via URL.
//
//	params := sy.QueryParams()
//	filter := params["filter"]
func QueryParams() map[string]string {
	rc := current()
	rc.sess.mu.Lock()
	defer rc.sess.mu.Unlock()
	cp := make(map[string]string, len(rc.sess.queryParams))
	for k, v := range rc.sess.queryParams {
		cp[k] = v
	}
	return cp
}

// QueryParam returns a single URL query parameter value.
func QueryParam(key string) string {
	rc := current()
	rc.sess.mu.Lock()
	defer rc.sess.mu.Unlock()
	return rc.sess.queryParams[key]
}

// --- Connection --------------------------------------------------------

var (
	connMu    sync.Mutex
	connCache = map[string]*sql.DB{}
)

// Connection returns a cached *sql.DB for the given name. The DSN is read
// from Secrets(name + "_DSN") or the [connections.<name>] section of
// syralit.toml. The driver must be registered by importing the appropriate
// database/sql driver package (e.g., _ "github.com/mattn/go-sqlite3").
//
//	db := sy.Connection("mydb")
//	rows, _ := db.Query("SELECT * FROM users")
func Connection(name string) *sql.DB {
	connMu.Lock()
	defer connMu.Unlock()

	if db, ok := connCache[name]; ok {
		return db
	}

	dsn := Secrets(name + "_DSN")
	if dsn == "" {
		dsn = Secrets(name)
	}

	driver := Secrets(name + "_DRIVER")
	if driver == "" {
		driver = "sqlite3"
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		panic("syralit: connection " + name + ": " + err.Error())
	}
	connCache[name] = db
	return db
}

// SQLQuery executes a SQL query and returns the results as headers and rows,
// ready to pass to DataFrame or DataEditor.
func SQLQuery(db *sql.DB, query string, args ...any) ([]string, [][]any) {
	rows, err := db.Query(query, args...)
	if err != nil {
		Error("SQL error: " + err.Error())
		return nil, nil
	}
	defer rows.Close()

	cols, _ := rows.Columns()
	var result [][]any
	for rows.Next() {
		values := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range values {
			ptrs[i] = &values[i]
		}
		if err := rows.Scan(ptrs...); err != nil {
			continue
		}
		row := make([]any, len(cols))
		for i, v := range values {
			if b, ok := v.([]byte); ok {
				row[i] = string(b)
			} else {
				row[i] = v
			}
		}
		result = append(result, row)
	}
	return cols, result
}

// --- Auth ---------------------------------------------------------------

// User returns the current session's authenticated user info. Returns nil
// if no user is logged in.
func User() map[string]string {
	sess := current().sess
	sess.mu.Lock()
	defer sess.mu.Unlock()
	if u, ok := sess.store["__syralit_user"].(map[string]string); ok {
		return u
	}
	return nil
}

// Login sets the current session's authenticated user. Call this after
// validating credentials.
func Login(user map[string]string) {
	sess := current().sess
	sess.mu.Lock()
	sess.store["__syralit_user"] = user
	sess.mu.Unlock()
}

// Logout clears the current session's authenticated user.
func Logout() {
	sess := current().sess
	sess.mu.Lock()
	delete(sess.store, "__syralit_user")
	sess.mu.Unlock()
}

// LoginGate renders a login form and blocks the rest of the app until the
// user authenticates. The check function receives (username, password) and
// returns true if the credentials are valid. Returns the logged-in username.
//
//	user := sy.LoginGate(func(u, p string) bool {
//	    return u == "admin" && p == sy.Secrets("ADMIN_PASS")
//	})
//	sy.Title("Welcome, " + user)
func LoginGate(check func(username, password string) bool) string {
	u := User()
	if u != nil {
		return u["username"]
	}

	Title("Login")
	username := TextInput("Username", Key("__login_user"), Placeholder("Username"))
	password := PasswordInput("Password", Key("__login_pass"))

	if Button("Login", Key("__login_btn")) {
		if check(username, password) {
			Login(map[string]string{"username": username})
			Rerun()
		} else {
			Error("Invalid credentials")
		}
	}
	Stop()
	return ""
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
