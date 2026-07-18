// OIDC login middleware for Syralit. A separate module so the go-oidc/oauth2
// dependency tree stays out of the core framework's go.mod — apps that don't
// use SSO never pull it.
module github.com/HazelnutParadise/syralit/integrations/oidc

go 1.25.12

require (
	github.com/HazelnutParadise/syralit v0.0.0
	github.com/coreos/go-oidc/v3 v3.14.1
	golang.org/x/oauth2 v0.30.0
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/coder/websocket v1.8.15 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/go-jose/go-jose/v4 v4.0.5 // indirect
	github.com/yuin/goldmark v1.8.2 // indirect
	golang.org/x/crypto v0.54.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace github.com/HazelnutParadise/syralit => ../../
