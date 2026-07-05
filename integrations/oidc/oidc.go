// Package syoidc adds OpenID Connect login to a Syralit app — the Go
// counterpart of Streamlit's st.login. Wrap the app handler with Protect and
// every visitor is sent through the provider's login before reaching the app;
// sy.User() then returns the verified claims.
//
//	handler, err := syoidc.Protect(sy.Handler(sy.Config{}, app), syoidc.Config{
//	    Issuer:       "https://accounts.google.com",
//	    ClientID:     os.Getenv("OIDC_CLIENT_ID"),
//	    ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
//	    RedirectURL:  "http://localhost:8600/auth/callback",
//	    CookieSecret: []byte(os.Getenv("COOKIE_SECRET")),
//	})
//	http.ListenAndServe(":8600", handler)
package syoidc

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	sy "github.com/HazelnutParadise/syralit"
)

// Config configures the OIDC middleware.
type Config struct {
	Issuer       string // e.g. "https://accounts.google.com"
	ClientID     string
	ClientSecret string
	RedirectURL  string // must match the provider's registered redirect URI

	// CookieSecret signs the identity cookie (HMAC-SHA256). Required;
	// use 32+ random bytes and keep it stable across restarts.
	CookieSecret []byte

	Scopes []string // defaults to [openid profile email]

	CookieName   string        // identity cookie name (default "sy_oidc")
	CookieMaxAge time.Duration // identity cookie lifetime (default 24h)

	LoginPath    string // default "/auth/login"
	CallbackPath string // default "/auth/callback"
	LogoutPath   string // default "/auth/logout"

	// Public lists path prefixes reachable without login (e.g. "/health").
	// Framework assets are always public so the login redirect can render.
	Public []string
}

func (c *Config) applyDefaults() error {
	if c.Issuer == "" || c.ClientID == "" || c.RedirectURL == "" {
		return errors.New("syoidc: Issuer, ClientID and RedirectURL are required")
	}
	if len(c.CookieSecret) < 16 {
		return errors.New("syoidc: CookieSecret must be at least 16 bytes")
	}
	if len(c.Scopes) == 0 {
		c.Scopes = []string{oidc.ScopeOpenID, "profile", "email"}
	}
	if c.CookieName == "" {
		c.CookieName = "sy_oidc"
	}
	if c.CookieMaxAge == 0 {
		c.CookieMaxAge = 24 * time.Hour
	}
	if c.LoginPath == "" {
		c.LoginPath = "/auth/login"
	}
	if c.CallbackPath == "" {
		c.CallbackPath = "/auth/callback"
	}
	if c.LogoutPath == "" {
		c.LogoutPath = "/auth/logout"
	}
	return nil
}

// identity is what the signed cookie carries.
type identity struct {
	Sub     string `json:"sub"`
	Email   string `json:"email,omitempty"`
	Name    string `json:"name,omitempty"`
	Picture string `json:"picture,omitempty"`
	Exp     int64  `json:"exp"`
}

// Protect wraps app with the OIDC login flow and registers a user resolver so
// sy.User() returns the verified claims ("sub", "email", "name", "picture").
// Provider discovery runs on the first request needing it, so construction
// never blocks on the network.
func Protect(app http.Handler, cfg Config) (http.Handler, error) {
	if err := cfg.applyDefaults(); err != nil {
		return nil, err
	}
	m := &middleware{app: app, cfg: cfg}

	// Make sy.User() work inside the app: sessions resolve their user from
	// the identity cookie captured at connection time.
	sy.SetUserResolver(func(rc sy.RequestContext) map[string]string {
		id, err := m.verifyCookieValue(rc.Cookies[cfg.CookieName])
		if err != nil {
			return nil
		}
		return map[string]string{
			"sub": id.Sub, "email": id.Email, "name": id.Name,
			"picture": id.Picture, "username": id.Email,
		}
	})
	return m, nil
}

type middleware struct {
	app http.Handler
	cfg Config

	provider *oidc.Provider
	oauth    *oauth2.Config
	verifier *oidc.IDTokenVerifier
}

// ensureProvider performs (lazy, once-successful) OIDC discovery.
func (m *middleware) ensureProvider(ctx context.Context) error {
	if m.provider != nil {
		return nil
	}
	p, err := oidc.NewProvider(ctx, m.cfg.Issuer)
	if err != nil {
		return fmt.Errorf("syoidc: provider discovery: %w", err)
	}
	m.provider = p
	m.verifier = p.Verifier(&oidc.Config{ClientID: m.cfg.ClientID})
	m.oauth = &oauth2.Config{
		ClientID:     m.cfg.ClientID,
		ClientSecret: m.cfg.ClientSecret,
		RedirectURL:  m.cfg.RedirectURL,
		Endpoint:     p.Endpoint(),
		Scopes:       m.cfg.Scopes,
	}
	return nil
}

func (m *middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case m.cfg.LoginPath:
		m.handleLogin(w, r)
		return
	case m.cfg.CallbackPath:
		m.handleCallback(w, r)
		return
	case m.cfg.LogoutPath:
		m.clearCookie(w, m.cfg.CookieName)
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	if m.isPublic(r.URL.Path) {
		m.app.ServeHTTP(w, r)
		return
	}
	if c, err := r.Cookie(m.cfg.CookieName); err == nil {
		if _, err := m.verifyCookieValue(c.Value); err == nil {
			m.app.ServeHTTP(w, r)
			return
		}
	}
	http.Redirect(w, r, m.cfg.LoginPath, http.StatusFound)
}

// isPublic: framework assets must load so the redirect target can render, and
// user-configured prefixes (health checks, static files) skip auth too.
func (m *middleware) isPublic(path string) bool {
	if strings.HasPrefix(path, "/_syralit/assets/") {
		return true
	}
	for _, p := range m.cfg.Public {
		if p != "" && strings.HasPrefix(path, p) {
			return true
		}
	}
	return false
}

func (m *middleware) handleLogin(w http.ResponseWriter, r *http.Request) {
	if err := m.ensureProvider(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	state := randomHex(16)
	nonce := randomHex(16)
	http.SetCookie(w, &http.Cookie{
		Name: m.cfg.CookieName + "_state", Value: state + "." + nonce,
		Path: "/", MaxAge: 600, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
	http.Redirect(w, r, m.oauth.AuthCodeURL(state, oidc.Nonce(nonce)), http.StatusFound)
}

func (m *middleware) handleCallback(w http.ResponseWriter, r *http.Request) {
	if err := m.ensureProvider(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	stateCookie, err := r.Cookie(m.cfg.CookieName + "_state")
	if err != nil {
		http.Error(w, "syoidc: missing state cookie", http.StatusBadRequest)
		return
	}
	m.clearCookie(w, m.cfg.CookieName+"_state")
	parts := strings.SplitN(stateCookie.Value, ".", 2)
	if len(parts) != 2 || r.URL.Query().Get("state") != parts[0] {
		http.Error(w, "syoidc: state mismatch", http.StatusBadRequest)
		return
	}

	token, err := m.oauth.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		http.Error(w, "syoidc: code exchange failed", http.StatusBadGateway)
		return
	}
	rawID, _ := token.Extra("id_token").(string)
	if rawID == "" {
		http.Error(w, "syoidc: no id_token in response", http.StatusBadGateway)
		return
	}
	idToken, err := m.verifier.Verify(r.Context(), rawID)
	if err != nil {
		http.Error(w, "syoidc: id_token verification failed", http.StatusUnauthorized)
		return
	}
	if idToken.Nonce != parts[1] {
		http.Error(w, "syoidc: nonce mismatch", http.StatusUnauthorized)
		return
	}

	var claims struct {
		Email   string `json:"email"`
		Name    string `json:"name"`
		Picture string `json:"picture"`
	}
	_ = idToken.Claims(&claims)
	id := identity{
		Sub: idToken.Subject, Email: claims.Email, Name: claims.Name,
		Picture: claims.Picture, Exp: time.Now().Add(m.cfg.CookieMaxAge).Unix(),
	}
	http.SetCookie(w, &http.Cookie{
		Name: m.cfg.CookieName, Value: m.signIdentity(id), Path: "/",
		MaxAge: int(m.cfg.CookieMaxAge.Seconds()), HttpOnly: true,
		SameSite: http.SameSiteLaxMode, Secure: r.TLS != nil,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}

func (m *middleware) clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: "", Path: "/", MaxAge: -1})
}

// signIdentity serializes and HMAC-signs the identity: base64(json).base64(mac).
func (m *middleware) signIdentity(id identity) string {
	payload, _ := json.Marshal(id)
	b64 := base64.RawURLEncoding.EncodeToString(payload)
	return b64 + "." + m.mac(b64)
}

// verifyCookieValue checks signature and expiry and returns the identity.
func (m *middleware) verifyCookieValue(value string) (identity, error) {
	var id identity
	parts := strings.SplitN(value, ".", 2)
	if len(parts) != 2 || value == "" {
		return id, errors.New("malformed cookie")
	}
	if !hmac.Equal([]byte(m.mac(parts[0])), []byte(parts[1])) {
		return id, errors.New("bad signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return id, err
	}
	if err := json.Unmarshal(payload, &id); err != nil {
		return id, err
	}
	if time.Now().Unix() > id.Exp {
		return id, errors.New("expired")
	}
	return id, nil
}

func (m *middleware) mac(data string) string {
	h := hmac.New(sha256.New, m.cfg.CookieSecret)
	h.Write([]byte(data))
	return base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
