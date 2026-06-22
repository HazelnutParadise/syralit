// embed-scroll demonstrates that scrollbars inside embedded content (the
// Component widget, which renders custom HTML in a sandboxed iframe, and
// same-origin IFrames) follow the Syralit light/dark theme — a thin, rounded,
// transparent-track scrollbar instead of the default OS one. Toggle the theme
// (top-right) and the embedded scrollbars recolor with the page.
package main

import (
	"fmt"
	"strings"

	sy "github.com/HazelnutParadise/syralit"
)

func main() {
	sy.App(func() {
		sy.SetPageConfig(sy.PageTitle("Themed Embedded Scrollbars"), sy.ConfigIcon("📜"))
		sy.Title("Themed Embedded Scrollbars")
		sy.Markdown("A `Component` embeds custom HTML in a sandboxed iframe — a separate document the page's CSS can't reach. Syralit injects a theme-matched scrollbar so embedded content scrolls in keeping with the rest of the app.")

		sy.Divider()

		sy.Subheader("Scrolls both ways")
		sy.Caption("Content taller and wider than the frame — vertical and horizontal scrollbars, both themed.")
		sy.Component(overflowHTML(), sy.Height(260))

		sy.Divider()

		sy.Subheader("Vertical only")
		sy.Caption("A long list inside a short frame.")
		sy.Component(listHTML(), sy.Height(200))

		sy.Divider()
		sy.Markdown("Native **go-echarts** charts (`integrations/insyra/eplot`) render through the same `Component`, so they inherit the themed scrollbar too — though charts auto-resize to fit and rarely need one.")
	})
}

func overflowHTML() string {
	var b strings.Builder
	b.WriteString(`<div style="font-family:system-ui,sans-serif;padding:12px 14px;width:1300px;color:inherit">`)
	for i := 1; i <= 30; i++ {
		fmt.Fprintf(&b, `<p style="margin:6px 0;white-space:nowrap">Row %02d — this line is wider than the frame and the block is taller than it, so both scrollbars appear, each themed to match Syralit.</p>`, i)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func listHTML() string {
	var b strings.Builder
	b.WriteString(`<ul style="font-family:system-ui,sans-serif;padding:8px 28px;margin:0;color:inherit">`)
	for i := 1; i <= 40; i++ {
		fmt.Fprintf(&b, `<li style="margin:4px 0">Item %02d</li>`, i)
	}
	b.WriteString(`</ul>`)
	return b.String()
}
