// UI test suite for Syralit. A separate module so the heavy chromedp
// dependency tree never touches the core framework's go.mod.
//
// Run with a local Chrome/Chromium installed:
//
//	cd uitest && go test ./...
module github.com/HazelnutParadise/syralit/uitest

go 1.26.9

require (
	github.com/HazelnutParadise/insyra v0.3.5
	github.com/HazelnutParadise/syralit v0.0.0
	github.com/chromedp/cdproto v0.0.0-20260714215040-dc233986426f
	github.com/chromedp/chromedp v0.16.0
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/HazelnutParadise/Go-Utils v0.8.2 // indirect
	github.com/Masterminds/squirrel v1.5.4 // indirect
	github.com/TimLai666/go-decimal v0.1.3 // indirect
	github.com/chromedp/sysutil v1.1.0 // indirect
	github.com/clipperhouse/uax29/v2 v2.7.0 // indirect
	github.com/coder/websocket v1.8.15 // indirect
	github.com/fsnotify/fsnotify v1.10.1 // indirect
	github.com/go-json-experiment/json v0.0.0-20260820222146-c27c302e5fc3 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.4.0 // indirect
	github.com/goccy/go-json v0.11.2 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/lann/builder v0.0.0-20180802200727-47ae307949d0 // indirect
	github.com/lann/ps v0.0.0-20150810152359-62de8c46ede0 // indirect
	github.com/mattn/go-runewidth v0.0.31 // indirect
	github.com/petermattis/goid v0.0.0-20260918085751-abfca077860b // indirect
	github.com/richardlehane/mscfb v1.0.9 // indirect
	github.com/richardlehane/msoleps v1.0.6 // indirect
	github.com/saintfish/chardet v0.0.0-20230101081208-5e3ef4b5456d // indirect
	github.com/tiendc/go-deepcopy v1.7.2 // indirect
	github.com/xuri/efp v0.0.2 // indirect
	github.com/xuri/excelize/v2 v2.11.1-0.20261020033641-611b222fd8f7 // indirect
	github.com/xuri/nfp v0.0.2-0.20250530014748-2ddeb826f9a9 // indirect
	github.com/yuin/goldmark v1.8.2 // indirect
	golang.org/x/crypto v0.58.0 // indirect
	golang.org/x/net v0.61.0 // indirect
	golang.org/x/sys v0.49.0 // indirect
	golang.org/x/term v0.47.0 // indirect
	golang.org/x/text v0.43.0 // indirect
	gonum.org/v1/gonum v0.17.0 // indirect
	gorm.io/gorm v1.31.2 // indirect
)

replace github.com/HazelnutParadise/syralit => ../
