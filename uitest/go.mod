// UI test suite for Syralit. A separate module so the heavy chromedp
// dependency tree never touches the core framework's go.mod.
//
// Run with a local Chrome/Chromium installed:
//
//	cd uitest && go test ./...
module github.com/HazelnutParadise/syralit/uitest

go 1.25.11

require (
	github.com/HazelnutParadise/syralit v0.0.0
	github.com/chromedp/chromedp v0.13.6
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/chromedp/cdproto v0.0.0-20250403032234-65de8f5d025b // indirect
	github.com/chromedp/sysutil v1.1.0 // indirect
	github.com/coder/websocket v1.8.15 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/go-json-experiment/json v0.0.0-20250211171154-1ae217ad3535 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.4.0 // indirect
	github.com/yuin/goldmark v1.8.2 // indirect
	golang.org/x/sys v0.46.0 // indirect
)

replace github.com/HazelnutParadise/syralit => ../
