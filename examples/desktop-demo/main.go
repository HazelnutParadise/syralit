// Desktop demo: the same Syralit code as any webapp, shipped as a native
// desktop window. Only the entry point differs — sydesktop.App instead of
// sy.App. Because the Go code runs on the user's machine, it can read local
// files directly (shown below with os.ReadDir) instead of round-tripping
// through sy.FileUploader.
//
// Build requirements are Wails v3's: nothing extra on Windows (WebView2 is
// preinstalled on 10/11), Xcode CLT on macOS, webkit2gtk on Linux.
package main

import (
	"os"

	sy "github.com/HazelnutParadise/syralit"
	sydesktop "github.com/HazelnutParadise/syralit/integrations/desktop"
)

func main() {
	sydesktop.App(func() {
		sy.Title("Syralit Desktop")
		sy.Caption("A native window, the same Go code as a webapp")

		cols := sy.Columns(3)
		cols[0](func() { sy.Metric("Mode", "Desktop") })
		cols[1](func() { sy.Metric("Server", "127.0.0.1") })
		cols[2](func() { sy.Metric("Frontend", "WebView") })

		sy.Divider()

		sy.Header("Local file access")
		sy.Text("The app runs on this machine, so it can browse local paths directly:")
		dir := sy.TextInput("Directory", sy.DefaultValue("."))
		if entries, err := os.ReadDir(dir); err != nil {
			sy.Warning(err.Error())
		} else {
			rows := make([][]any, 0, len(entries))
			for _, e := range entries {
				kind := "file"
				if e.IsDir() {
					kind = "dir"
				}
				rows = append(rows, []any{e.Name(), kind})
			}
			sy.DataFrame([]string{"Name", "Type"}, rows)
		}

		sy.Header("Everything else just works")
		n := sy.Slider("Points", 3, 50, sy.DefaultValue(12))
		series := make([]float64, int(n))
		for i := range series {
			series[i] = float64((i*7)%13) + float64(i)
		}
		sy.LineChart(map[string][]float64{"demo": series})

		count := sy.State("count", 0)
		if sy.Button("Click me") {
			count.Set(count.Get() + 1)
		}
		sy.Textf("Count: %d", count.Get())
	},
		sydesktop.WindowSize(1100, 800),
		sydesktop.MinSize(640, 480),
	)
}
