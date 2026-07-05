package syoidc

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func testConfig() Config {
	return Config{
		Issuer: "https://issuer.example", ClientID: "cid", ClientSecret: "sec",
		RedirectURL:  "http://localhost/auth/callback",
		CookieSecret: []byte("0123456789abcdef0123456789abcdef"),
	}
}

func newTestMiddleware(t *testing.T) *middleware {
	t.Helper()
	app := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("app"))
	})
	h, err := Protect(app, testConfig())
	if err != nil {
		t.Fatal(err)
	}
	return h.(*middleware)
}

func TestCookieSignRoundTrip(t *testing.T) {
	m := newTestMiddleware(t)
	id := identity{Sub: "u1", Email: "a@b.c", Name: "Ada", Exp: time.Now().Add(time.Hour).Unix()}
	v := m.signIdentity(id)
	got, err := m.verifyCookieValue(v)
	if err != nil {
		t.Fatal(err)
	}
	if got.Sub != "u1" || got.Email != "a@b.c" || got.Name != "Ada" {
		t.Fatalf("roundtrip = %+v", got)
	}
}

func TestCookieTamperRejected(t *testing.T) {
	m := newTestMiddleware(t)
	id := identity{Sub: "u1", Exp: time.Now().Add(time.Hour).Unix()}
	v := m.signIdentity(id)
	// Flip a payload byte — signature must fail.
	tampered := "x" + v[1:]
	if _, err := m.verifyCookieValue(tampered); err == nil {
		t.Fatal("tampered cookie accepted")
	}
	if _, err := m.verifyCookieValue(""); err == nil {
		t.Fatal("empty cookie accepted")
	}
	if _, err := m.verifyCookieValue("no-dot"); err == nil {
		t.Fatal("malformed cookie accepted")
	}
}

func TestCookieExpiryRejected(t *testing.T) {
	m := newTestMiddleware(t)
	id := identity{Sub: "u1", Exp: time.Now().Add(-time.Minute).Unix()}
	if _, err := m.verifyCookieValue(m.signIdentity(id)); err == nil {
		t.Fatal("expired cookie accepted")
	}
}

func TestMiddlewareGate(t *testing.T) {
	m := newTestMiddleware(t)

	// Unauthenticated app request → redirect to login.
	rec := httptest.NewRecorder()
	m.ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/auth/login" {
		t.Fatalf("expected redirect to login, got %d %s", rec.Code, rec.Header().Get("Location"))
	}

	// Framework assets are public (the login page itself needs them).
	rec = httptest.NewRecorder()
	m.ServeHTTP(rec, httptest.NewRequest("GET", "/_syralit/assets/runtime.css", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("assets should be public, got %d", rec.Code)
	}

	// Valid identity cookie → app served.
	id := identity{Sub: "u1", Exp: time.Now().Add(time.Hour).Unix()}
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: "sy_oidc", Value: m.signIdentity(id)})
	rec = httptest.NewRecorder()
	m.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || rec.Body.String() != "app" {
		t.Fatalf("authenticated request not served: %d %q", rec.Code, rec.Body.String())
	}

	// Logout clears the cookie and redirects home.
	rec = httptest.NewRecorder()
	m.ServeHTTP(rec, httptest.NewRequest("GET", "/auth/logout", nil))
	if rec.Code != http.StatusFound || rec.Header().Get("Location") != "/" {
		t.Fatalf("logout redirect wrong: %d", rec.Code)
	}
	if !strings.Contains(rec.Header().Get("Set-Cookie"), "Max-Age=0") &&
		!strings.Contains(rec.Header().Get("Set-Cookie"), "Expires") {
		t.Fatalf("logout did not clear cookie: %s", rec.Header().Get("Set-Cookie"))
	}
}

func TestConfigValidation(t *testing.T) {
	if _, err := Protect(nil, Config{}); err == nil {
		t.Fatal("empty config accepted")
	}
	c := testConfig()
	c.CookieSecret = []byte("short")
	if _, err := Protect(nil, c); err == nil {
		t.Fatal("short cookie secret accepted")
	}
}
