@CLAUDE.md

## Follow-ups

- `sy.Embed` inside layout containers (`Columns`, `Container`, `Expander`, …) is
  re-attached on every rerun because the container element is rebuilt, so
  iframes the widget created reload there. Keeping it attached needs in-place
  reconciliation of container children (a keyed DOM diff), out of scope for #5.
