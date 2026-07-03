// Example: dynamic computation in Syralit with the Insyra DSL.
//
// It shows two ways to drive UI from an Insyra .isr script that runs
// server-side:
//
//   - syidsl.DSL(...)      a Go widget that computes and auto-renders
//   - the "insyra" Artifact component, embeddable in an ArtifactSpec that an
//     agent can POST, so agents can do live computation inside an Artifact
//     Canvas rather than only binding static data.
//
// Importing the insyradsl package enables the "insyra" artifact component.
package main

import (
	sy "github.com/HazelnutParadise/syralit"
	syidsl "github.com/HazelnutParadise/syralit/integrations/insyra/insyradsl"
)

var board = sy.NewArtifactStore("insights", insightsSpec())

func main() {
	sy.App(func() {
		sy.SetPageConfig(sy.PageTitle("Insyra DSL Artifacts"), sy.PageLayout("wide"))
		sy.Title("Insyra DSL Artifacts")
		sy.Markdown("Dynamic data computation with the **Insyra DSL** — scripts run server-side (safe mode) and render live.")

		cols := sy.Columns(2)
		cols[0](func() {
			sy.Header("Go widget — syidsl.DSL")
			sy.Caption("The app author runs a script and auto-renders the result.")
			syidsl.DSL(`
newdl Q1 Q2 Q3 Q4 as quarter
newdl 42 55 61 78 as revenue
newdt quarter revenue as t
setcolnames t quarter revenue
`,
				syidsl.Render("bar_chart"),
				syidsl.Output("t"),
				syidsl.X("quarter"),
				syidsl.Y("revenue"),
				syidsl.Title("Revenue by quarter"),
			)

			sy.Header("Transcript mode")
			sy.Caption("With no render options, DSL prints the script's output.")
			syidsl.DSL("newdl 42 55 61 78 as revenue\nsummary revenue")
		})

		cols[1](func() {
			sy.Header("Artifact Canvas — agent-driveable")
			sy.Caption("The insyra component embeds a DSL script the agent can POST.")
			sy.ArtifactCanvas(board, sy.Height(460))
		})

		sy.Divider()
		sy.Header("Artifact spec")
		sy.JSON(board.Spec(), sy.DefaultValue(false))
	})
}

// insightsSpec builds an artifact whose two data panels are computed live by the
// insyra component instead of being pre-baked into the spec's data map.
func insightsSpec() sy.ArtifactSpec {
	const dataset = "newdl North South North West South North as region\n" +
		"newdl 40 30 50 60 20 30 as deals\n" +
		"newdt region deals as t\n" +
		"setcolnames t region deals\n"

	return sy.ArtifactSpec{
		Version: "v1",
		Layout:  sy.ArtifactLayout{Columns: 2, Gap: 14, Padding: 16},
		Nodes: []sy.ArtifactNode{
			{
				ID:        "headline",
				Component: "markdown",
				Props:     map[string]any{"text": "## Regional performance\nBoth panels below are computed live from an Insyra DSL script."},
				Layout:    sy.ArtifactLayoutItem{ColumnSpan: 2},
			},
			{
				ID:        "chart",
				Component: "insyra",
				Props: map[string]any{
					"script": dataset + "groupby t by region agg deals:sum:total as report",
					"render": "bar_chart",
					"output": "report",
					"x":      "region",
					"y":      "total",
					"title":  "Deals by region (summed)",
				},
				Layout: sy.ArtifactLayoutItem{ColumnSpan: 2},
			},
			{
				ID:        "stats",
				Component: "insyra",
				Props: map[string]any{
					"script": "newdl 40 30 50 60 20 30 as deals\ndescribe deals as summary",
					"render": "dataframe",
					"output": "summary",
				},
				Layout: sy.ArtifactLayoutItem{ColumnSpan: 2},
			},
		},
	}
}
