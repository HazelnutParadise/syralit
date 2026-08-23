// OIDC single sign-on: wrap the app with syoidc.Protect and every visitor
// logs in through the provider (Google, Microsoft Entra, Keycloak, Auth0, …)
// before reaching the app. sy.User() then returns the verified claims.
//
// Run with your provider's credentials:
//
//	OIDC_ISSUER=https://accounts.google.com \
//	OIDC_CLIENT_ID=... OIDC_CLIENT_SECRET=... \
//	COOKIE_SECRET=some-32-byte-random-string go run .
package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	sy "github.com/HazelnutParadise/syralit"
	syoidc "github.com/HazelnutParadise/syralit/integrations/oidc"
)

func app() {
	u := sy.User()
	sy.Title("Welcome, " + u["name"])
	sy.Text("Signed in as " + u["email"])
	if u["picture"] != "" {
		sy.Image(u["picture"], sy.Width(80))
	}
	sy.LinkButton("Sign out", "/auth/logout")
}

func main() {
	// This process owns the listener, so sy.Handler ignores Host/Port. Resolve
	// the config ourselves (code > syralit.toml > defaults) to bind the same
	// address a syralit.toml would configure.
	cfg := sy.ResolveConfig(sy.Config{Title: "OIDC Demo"})
	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	handler, err := syoidc.Protect(sy.Handler(cfg, app), syoidc.Config{
		Issuer:       os.Getenv("OIDC_ISSUER"),
		ClientID:     os.Getenv("OIDC_CLIENT_ID"),
		ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		RedirectURL:  fmt.Sprintf("http://localhost:%d/auth/callback", cfg.Port), // must match the URI registered with the provider
		CookieSecret: []byte(os.Getenv("COOKIE_SECRET")),
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("OIDC demo on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, handler))
}
