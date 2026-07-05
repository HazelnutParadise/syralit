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
	handler, err := syoidc.Protect(sy.Handler(sy.Config{Title: "OIDC Demo"}, app), syoidc.Config{
		Issuer:       os.Getenv("OIDC_ISSUER"),
		ClientID:     os.Getenv("OIDC_CLIENT_ID"),
		ClientSecret: os.Getenv("OIDC_CLIENT_SECRET"),
		RedirectURL:  "http://localhost:8600/auth/callback",
		CookieSecret: []byte(os.Getenv("COOKIE_SECRET")),
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println("OIDC demo on http://localhost:8600")
	log.Fatal(http.ListenAndServe("127.0.0.1:8600", handler))
}
