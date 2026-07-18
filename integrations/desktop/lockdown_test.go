package sydesktop

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLockdown(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	token := newLockdownToken()
	h := lockdown(inner, token)

	get := func(path string, cookie string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: lockdownCookie, Value: cookie})
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		return rec
	}

	// No credentials: rejected.
	if rec := get("/", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("bare request: %d, want 403", rec.Code)
	}
	// Wrong token: rejected.
	if rec := get("/?"+lockdownParam+"=nope", ""); rec.Code != http.StatusForbidden {
		t.Fatalf("wrong token: %d, want 403", rec.Code)
	}
	if rec := get("/", "nope"); rec.Code != http.StatusForbidden {
		t.Fatalf("wrong cookie: %d, want 403", rec.Code)
	}
	// Valid token param: served, and the auth cookie is established.
	rec := get("/?"+lockdownParam+"="+token, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("token param: %d, want 200", rec.Code)
	}
	found := false
	for _, c := range rec.Result().Cookies() {
		if c.Name == lockdownCookie && c.Value == token && c.HttpOnly {
			found = true
		}
	}
	if !found {
		t.Fatal("valid token request did not set the auth cookie")
	}
	// Valid cookie alone: served (later requests from the window).
	if rec := get("/_syralit/ws", token); rec.Code != http.StatusOK {
		t.Fatalf("cookie auth: %d, want 200", rec.Code)
	}
	// /api/ endpoints pass through — they carry their own authentication.
	if rec := get("/api/agent/artifacts", ""); rec.Code != http.StatusOK {
		t.Fatalf("/api/ passthrough: %d, want 200", rec.Code)
	}
}
