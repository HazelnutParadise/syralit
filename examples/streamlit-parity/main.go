// Demonstrates the Streamlit-parity additions: PDF viewer, multi-file upload,
// collapsed initial sidebar, app menu items, MultiSelect with free entry,
// toast duration, and media playback clipping.
package main

import (
	"fmt"
	"strings"

	sy "github.com/HazelnutParadise/syralit"
)

func main() {
	sy.App(func() {
		sy.SetPageConfig(
			sy.PageTitle("Parity Demo"),
			sy.InitialSidebarState("collapsed"),
			sy.ConfigMenuItems(
				"https://github.com/HazelnutParadise/syralit#readme",
				"https://github.com/HazelnutParadise/syralit/issues",
				"**Parity Demo**\n\nBuilt with Syralit.",
			),
		)

		sy.Sidebar(func() {
			sy.Header("Sidebar")
			sy.Text("Starts collapsed; reopen with the floating button.")
		})

		sy.Title("Streamlit-parity additions")

		sy.Header("PDF viewer")
		sy.PDF("https://www.w3.org/WAI/ER/tests/xhtml/testfiles/resources/pdf/dummy.pdf", sy.Height(360))

		sy.Header("Multi-file upload")
		files := sy.FileUploaderMultiple("Upload several files")
		for _, f := range files {
			sy.Textf("%s (%d bytes, %s)", f.Name, f.Size, f.Type)
		}

		sy.Header("MultiSelect with free entry")
		tags := sy.MultiSelect("Tags", []string{"go", "data", "ui"}, sy.AcceptNewOptions())
		if len(tags) > 0 {
			sy.Text("Selected: " + strings.Join(tags, ", "))
		}

		sy.Header("Toast with duration")
		if sy.Button("Show 8-second toast") {
			sy.Toast(fmt.Sprintf("Visible for 8s (files uploaded: %d)", len(files)), "success", "⏱️", "8s")
		}

		sy.Header("Clipped video")
		sy.Video("https://interactive-examples.mdn.mozilla.net/media/cc0-videos/flower.mp4",
			sy.StartTime(2), sy.EndTime(5), sy.Muted())
	})
}
