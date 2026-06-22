# CLAUDE.md

Guidance for AI coding agents working in the **Syralit** repository.

## What Syralit is

Syralit is a Go-native framework for building interactive data apps, dashboards, and AI tool UIs — inspired by Streamlit, designed for Go. The goal is to reach Streamlit-level feature completeness while staying idiomatic Go: no JavaScript, no HTML templates, no frontend build step for the app author.

The user import alias is always:

```go
import sy "github.com/HazelnutParadise/syralit"
```

## Architecture

- **Rerun model** — `sy.App(func(){ ... })` re-executes the whole closure on every widget interaction. UI is declared imperatively; there is no virtual DOM. The server diffs and sends a UI tree to the browser over a WebSocket each rerun.
- **Container stack** — layout containers (`Columns`, `Tabs`, `Sidebar`, `Expander`, `Container`, `Form`, `Fragment`, `Status`, `Popover`, `Dialog`) push/pop nodes on `renderContext.stack`. See `context.go`, `layout.go`, `node.go`.
- **Widgets / display / charts / data** live in `widgets.go`, `display.go`, `chart.go`. State and caching in `state.go`, `cache.go`, `session.go`. Multi-page routing in `page.go`. Server and dev/hot-reload in `server.go`, `supervisor.go`, `dev_common.go`. Config/secrets in `config_file.go`.
- **Frontend assets** — `assets/runtime.css` and `assets/runtime.js` are embedded via `//go:embed` (`assets.go`). Edit these files directly; a rebuild picks them up. There is no separate JS toolchain. Theme is driven by CSS variables (`--sy-*`) with light/dark via `prefers-color-scheme` and `html[data-theme]`.
- **CLI** — `cmd/syralit/main.go` implements `syralit new`, `syralit dev`, `syralit run`.

## Insyra integration — keep this boundary

- The **core syralit package must never import Insyra.** All Insyra-specific code lives only in `integrations/insyra/` (`integrations/insyra/insyra.go`). This keeps Insyra an optional dependency for users who don't need it. Before committing, verify no core/`examples` file outside `integrations/insyra/` imports `HazelnutParadise/insyra`.

## Keep Insyra at the latest version

Syralit tracks Insyra closely. **Always keep the Insyra dependency pinned to the latest released version.** When working in this repo:

```bash
go get -u github.com/HazelnutParadise/insyra@latest
go mod tidy
go build ./... && go test ./...
```

Check `go list -m -versions github.com/HazelnutParadise/insyra` for the newest tag and confirm `go.mod` matches it. If a newer version exists, bump it (verify the integration package and examples still build) as part of routine work — don't leave it stale.

## Build, test, run

```bash
go build ./...          # build everything
go test ./...           # run the test suite (server_test.go, etc.)
go vet ./...            # vet

cd examples/hello && go run .   # run an example (defaults to http://localhost:8600)
```

Examples auto-pick the next free port if 8600 is taken (e.g. 8601), and log the actual URL on startup.

## Conventions

- Match existing file/style: option-pattern constructors (`type Option func(*widgetOpts)`, e.g. `sy.Key`, `sy.DefaultValue`, `sy.Height`, `sy.ChartTitle`). New widgets follow the same shape.
- Built-in charts go through `chartProps` in `chart.go` (Chart.js). External chart libs (Vega-Lite, Plotly, Bokeh, deck.gl) are CDN-lazy-loaded from a JSON spec — follow the existing pattern when adding more.
- Keep the public API Streamlit-flavored where a clear equivalent exists.
- Examples live in `examples/`; the agent skill lives in `skills/syralit-dev/SKILL.md`. Update both when adding user-facing features.

## Workflow notes

- Commit and push automatically when work reaches a sensible checkpoint; no need to ask first.
- **Docs are part of "done" for any user-facing change.** Whenever you add, rename, or change the behaviour of a public API (a `sy.*` widget/option, an `integrations/insyra` helper, a CLI command, config key, etc.), update **all** of these in the *same* change — never leave it "for later":
  1. **`skills/syralit-dev/SKILL.md`** — the agent skill is the canonical API reference; add the new function/option with its signature and a one-line description, in the matching section.
  2. **`README.md`** — add it to the relevant section/table.
  3. **`CHANGELOG.md`** — note it under the current unreleased section.
  4. An **example** under `examples/` when the feature benefits from a runnable demo.
  Before committing a feature, re-read SKILL.md and confirm it actually lists what you added — a feature the skill doesn't mention is effectively invisible to AI-assisted users.
