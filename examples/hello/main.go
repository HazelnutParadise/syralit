package main

import sy "github.com/HazelnutParadise/syralit"

func main() {
	sy.App(func() {
		sy.SetPageConfig(sy.PageTitle("Syralit Demo"), sy.ConfigIcon("🚀"))

		sy.Title("Hello, Syralit")
		sy.Caption("A Go-native Streamlit-style framework")

		name := sy.TextInput("Your name", sy.Placeholder("Enter your name…"))
		if name != "" {
			sy.Success("Hello, " + name + "!")
		}

		sy.Divider()

		cols := sy.Columns(3)
		cols[0](func() {
			sy.Metric("Users", "1,234", sy.Delta("+12%"))
		})
		cols[1](func() {
			sy.Metric("Revenue", "$56K", sy.Delta("+8%"), sy.DeltaColor("normal"))
		})
		cols[2](func() {
			sy.Metric("Errors", "3", sy.Delta("-2"), sy.DeltaColor("inverse"))
		})

		sy.Header("Charts")
		sy.LineChart(map[string][]float64{
			"Revenue": {10, 20, 35, 28, 42, 55, 48},
			"Cost":    {5, 8, 12, 10, 15, 18, 14},
		})

		sy.Header("Interactive Widgets")

		tab := sy.Tabs([]string{"Inputs", "Data", "Layout"})

		tab("Inputs", func() {
			color := sy.ColorPicker("Pick a color")
			sy.Text("Selected: " + color)

			vol := sy.Slider("Volume", 0, 100, sy.DefaultValue(50))
			sy.Textf("Volume: %.0f%%", vol)

			if sy.Toggle("Dark mode") {
				sy.Info("Dark mode enabled (theme switching is a future feature)")
			}
		})

		tab("Data", func() {
			sy.DataFrame(
				[]string{"Name", "Language", "Stars"},
				[][]any{
					{"Syralit", "Go", 0},
					{"Streamlit", "Python", 35000},
					{"Gradio", "Python", 33000},
				},
			)
		})

		tab("Layout", func() {
			sy.Expander("Click to expand", func() {
				sy.Text("Hidden content revealed!")
				sy.Code("fmt.Println(\"Hello from Syralit\")", sy.Language("go"))
			})

			sy.Popover("Show popover", func() {
				sy.Text("This is a floating popover panel.")
				sy.Caption("Click outside to close.")
			})
		})

		count := sy.State("count", 0)
		if sy.Button("Click me") {
			count.Set(count.Get() + 1)
			sy.Toast("Button clicked!", "success")
		}
		sy.Textf("Count: %d", count.Get())

		sy.Progress(float64(count.Get()%10) / 10.0)
	})
}
