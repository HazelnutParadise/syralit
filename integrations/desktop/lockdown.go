package sydesktop

// Browser lockdown: a desktop app's loopback server should serve its own
// window, not every browser on the machine. The window's URL carries a
// per-launch random token; the middleware sets a cookie on the first tokened
// request and rejects everything else. Paths under /api/ pass through — those
// are programmatic endpoints (agent artifacts, ...) that carry their own
// authentication, mirroring what the dev supervisor proxies.

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"net/http"
	"strings"
)

const (
	lockdownCookie = "_syralit_desktop"
	lockdownParam  = "syd_token"
)

func newLockdownToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("sydesktop: crypto/rand unavailable: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// lockdown wraps next so only requests bearing the token (query param or the
// cookie it establishes) get through, except /api/ endpoints which keep their
// own auth. The token stays in the window's URL — invisible in a frameless
// webview — so a webview that dropped its cookies re-authenticates on reload.
func lockdown(next http.Handler, token string) http.Handler {
	tok := []byte(token)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}
		if c, err := r.Cookie(lockdownCookie); err == nil &&
			subtle.ConstantTimeCompare([]byte(c.Value), tok) == 1 {
			next.ServeHTTP(w, r)
			return
		}
		if q := r.URL.Query().Get(lockdownParam); q != "" &&
			subtle.ConstantTimeCompare([]byte(q), tok) == 1 {
			http.SetCookie(w, &http.Cookie{
				Name: lockdownCookie, Value: token, Path: "/",
				HttpOnly: true, SameSite: http.SameSiteLaxMode,
			})
			next.ServeHTTP(w, r)
			return
		}
		http.Error(w, "This Syralit app only serves its own desktop window (start with sydesktop.AllowBrowser() to open it up).", http.StatusForbidden)
	})
}
