@CLAUDE.md

## Follow-ups

- `syralit.toml` only reads a top-level `port` (`fileConfig.Port` in
  `config_file.go`); `[server] port` is silently ignored. `examples/embed-scroll`,
  `examples/insyra-charts`, `examples/insyra-demo` and `examples/showcase` use the
  `[server]` form, so they all start on the default 8600 instead of their
  declared port. Fix: either move `port` to the top level in those files, or also
  accept `Server.Port` in `applyToConfig`/`applyToDev` (and document it).
- `sy.Embed` inside layout containers (`Columns`, `Container`, `Expander`, …) is
  re-attached on every rerun because the container element is rebuilt, so
  iframes the widget created reload there. Keeping it attached needs in-place
  reconciliation of container children (a keyed DOM diff), out of scope for #5.
