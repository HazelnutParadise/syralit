// Standalone module: the desktop example needs integrations/desktop, whose
// Wails dependency tree must not leak into the repo's example builds.
module github.com/HazelnutParadise/syralit/examples/desktop-demo

go 1.25.12

require (
	github.com/HazelnutParadise/syralit v0.0.0
	github.com/HazelnutParadise/syralit/integrations/desktop v0.0.0
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/adrg/xdg v0.5.3 // indirect
	github.com/coder/websocket v1.8.15 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/godbus/dbus/v5 v5.2.2 // indirect
	github.com/jchv/go-winloader v0.0.0-20250406163304-c1995be93bd1 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.22 // indirect
	github.com/wailsapp/wails/v3 v3.0.0-alpha2.117 // indirect
	github.com/yuin/goldmark v1.8.2 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace (
	github.com/HazelnutParadise/syralit => ../../
	github.com/HazelnutParadise/syralit/integrations/desktop => ../../integrations/desktop
)
