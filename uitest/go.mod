// UI test suite for Syralit. A separate module so the heavy chromedp
// dependency tree never touches the core framework's go.mod.
//
// Run with a local Chrome/Chromium installed:
//
//	cd uitest && go test ./...
module github.com/HazelnutParadise/syralit/uitest

go 1.25.11

require (
	github.com/HazelnutParadise/insyra v0.2.19
	github.com/HazelnutParadise/syralit v0.0.0
	github.com/chromedp/cdproto v0.0.0-20250403032234-65de8f5d025b
	github.com/chromedp/chromedp v0.13.6
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/HazelnutParadise/Go-Utils v0.8.2 // indirect
	github.com/Masterminds/squirrel v1.5.4 // indirect
	github.com/chromedp/sysutil v1.1.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/coder/websocket v1.8.15 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/go-json-experiment/json v0.0.0-20250211171154-1ae217ad3535 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.4.0 // indirect
	github.com/goccy/go-json v0.10.6 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/mattn/go-runewidth v0.0.24 // indirect
	github.com/petermattis/goid v0.0.0-20260619124436-7ab4bde3d003 // indirect
	github.com/richardlehane/mscfb v1.0.7 // indirect
	github.com/richardlehane/msoleps v1.0.6 // indirect
	github.com/saintfish/chardet v0.0.0-20230101081208-5e3ef4b5456d // indirect
	github.com/tiendc/go-deepcopy v1.7.2 // indirect
	github.com/xuri/efp v0.0.1 // indirect
	github.com/xuri/excelize/v2 v2.10.1 // indirect
	github.com/xuri/nfp v0.0.2-0.20250530014748-2ddeb826f9a9 // indirect
	github.com/yuin/goldmark v1.8.2 // indirect
	golang.org/x/crypto v0.53.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sys v0.46.0 // indirect
	golang.org/x/term v0.44.0 // indirect
	golang.org/x/text v0.38.0 // indirect
	gonum.org/v1/gonum v0.17.0 // indirect
	gorm.io/gorm v1.31.1 // indirect
)

replace github.com/HazelnutParadise/syralit => ../
