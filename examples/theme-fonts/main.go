// Demonstrates font theming: built-in font keywords ("sans-serif" / "serif" /
// "monospace" map to the embedded Source Sans 3 / Source Serif 4 / Source Code
// Pro), custom fonts loaded via [[theme.font_faces]], and size/weight tuning.
// All theme values live in syralit.toml next to this file.
package main

import sy "github.com/HazelnutParadise/syralit"

func main() {
	sy.App(func() {
		sy.Title("Font Theming")
		sy.Caption("Body uses the built-in serif (Source Serif 4); headings use a custom font loaded via [[theme.font_faces]]; code uses Source Code Pro.")

		sy.Header("Body text")
		sy.Markdown("Regular paragraph text, *italic*, **bold**, and `inline code`. " +
			"The base font size is bumped to 17px via `base_font_size`.")

		sy.Header("Code")
		sy.Code(`fmt.Println("Source Code Pro, sized by code_font_size")`, sy.Language("go"))

		sy.Header("Colors & charts")
		sy.Markdown("Links use `link_color`: [Syralit](https://github.com/HazelnutParadise/syralit). Buttons use `button_radius`.")
		sy.Button("Pill button")
		sy.LineChart(map[string][]float64{
			"A": {3, 6, 4, 8, 7},
			"B": {2, 3, 5, 4, 6},
		}) // series colors come from chart_categorical_colors

		sy.Sidebar(func() {
			sy.Header("Sidebar")
			sy.Text("The sidebar can use its own fonts via [theme.sidebar].")
		})
	})
}
