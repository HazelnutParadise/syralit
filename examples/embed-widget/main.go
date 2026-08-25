// embed-widget demonstrates sy.Embed: third-party markup of the usual
// "mount point + loader script" shape that must run in the main document
// and must not be rebuilt on every rerun. The "vendor" loader here is inline
// (a real one would be <script src="https://vendor.example/loader.js">): it
// counts how many times it has run and fills its slot, so you can see that
// interacting with the app does not reload the widget.
package main

import (
	"fmt"

	sy "github.com/HazelnutParadise/syralit"
)

const widgetHTML = `
<div id="widget-slot" style="padding:12px;border:1px dashed #999;border-radius:8px"></div>
<script>
  window.__loaderRuns = (window.__loaderRuns || 0) + 1;
  var slot = document.getElementById("widget-slot");
  slot.textContent = "Widget loaded " + window.__loaderRuns + " time(s) — "
    + "interact with the app and this number stays put.";
</script>`

func main() {
	sy.App(func() {
		sy.SetPageConfig(sy.PageTitle("sy.Embed"), sy.ConfigIcon("🧩"))
		sy.Title("Third-party widget with sy.Embed")
		sy.Markdown("`sy.Embed` puts markup in the main document, runs its `<script>`s once, " +
			"and keeps the same DOM node across reruns. `sy.HTML` would never run the script; " +
			"`sy.Component` would run it in a sandboxed iframe.")

		sy.Embed(widgetHTML, sy.Key("article-ad-1"))

		count := sy.State("clicks", 0)
		if sy.Button("Rerun the app", sy.Key("rerun")) {
			count.Set(count.Get() + 1)
		}
		sy.Text(fmt.Sprintf("Reruns triggered: %d — the widget above was not rebuilt.", count.Get()))

		sy.Divider()
		sy.Subheader("Inside layout containers")
		sy.Caption("Since #6 the container shell is kept in place around a surviving embed, " +
			"so this one — nested in Columns → Container — doesn't reload either:")
		cols := sy.Columns(2)
		cols[0](func() {
			sy.Container(func() {
				sy.Embed(`<div id="nested-slot" style="padding:8px;border:1px dashed #999;border-radius:8px"></div>
<script>
  window.__nestedRuns = (window.__nestedRuns || 0) + 1;
  document.getElementById("nested-slot").textContent =
    "Nested widget loaded " + window.__nestedRuns + " time(s).";
</script>`, sy.Key("nested-ad"))
			}, sy.Border())
		})
		cols[1](func() {
			sy.Text(fmt.Sprintf("Reruns so far: %d", count.Get()))
		})

		sy.Divider()
		sy.Caption("Changing the html for the same key rebuilds the node and runs the loader again:")
		variant := sy.Toggle("Use alternate widget markup", sy.Key("alt"))
		html := `<p id="v">Variant A</p><script>document.getElementById("v").textContent += " (script ran)";</script>`
		if variant {
			html = `<p id="v">Variant B</p><script>document.getElementById("v").textContent += " (script ran)";</script>`
		}
		sy.Embed(html, sy.Key("variant"))
	})
}
