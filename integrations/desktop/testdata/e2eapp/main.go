// E2E fixture: a minimal production desktop app with an agent artifact
// endpoint, driven by e2e_test.go. Lives under testdata so ./... skips it.
package main

import (
	"log"

	sy "github.com/HazelnutParadise/syralit"
	sydesktop "github.com/HazelnutParadise/syralit/integrations/desktop"
)

func main() {
	board := sy.NewArtifactStore("main", sy.ArtifactSpec{
		Version: "v1",
		Layout:  sy.ArtifactLayout{Columns: 1, Gap: 8, Padding: 8},
		Nodes: []sy.ArtifactNode{
			{ID: "headline", Component: "text", Props: map[string]any{"text": "e2e"}},
		},
	})
	sy.HandleArtifactEndpoint("/api/agent/artifacts/main", board,
		sy.StaticAgentKey("e2e-agent", "e2e-secret"))

	sydesktop.App(func() {
		// The page function runs only after the webview loaded the shell and
		// its socket connected; e2e_test.go waits for this line before it
		// closes the window, so WebView2 is never torn down mid-embedding.
		log.Println("e2e: rendered")
		sy.Title("E2E")
		sy.ArtifactCanvas(board)
	},
		sydesktop.WindowTitle("SyralitE2EWindow"),
		sydesktop.WindowSize(500, 400),
	)
}
