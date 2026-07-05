// Standalone module: the OIDC example needs integrations/oidc, whose
// dependency tree must not leak into the repo's example builds.
module oidc-login

go 1.25.11

require (
	github.com/HazelnutParadise/syralit v0.0.0
	github.com/HazelnutParadise/syralit/integrations/oidc v0.0.0
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/coder/websocket v1.8.15 // indirect
	github.com/coreos/go-oidc/v3 v3.14.1 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/go-jose/go-jose/v4 v4.0.5 // indirect
	github.com/yuin/goldmark v1.8.2 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/oauth2 v0.30.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
)

replace (
	github.com/HazelnutParadise/syralit => ../../
	github.com/HazelnutParadise/syralit/integrations/oidc => ../../integrations/oidc
)
