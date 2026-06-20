package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	sy "github.com/HazelnutParadise/syralit"
)

func init() {
	sy.AddPage("Dashboard", dashboard, sy.PageIcon("📊"), sy.PageOrder(1))
	sy.AddPage("Widgets", widgets, sy.PageIcon("🎛️"), sy.PageOrder(2))
	sy.AddPage("Charts", charts, sy.PageIcon("📈"), sy.PageOrder(3))
	sy.AddPage("Layout", layout, sy.PageIcon("🧩"), sy.PageOrder(4))
	sy.AddPage("Data", data, sy.PageIcon("🗃️"), sy.PageOrder(5))
	sy.AddPage("Map & Media", mapMedia, sy.PageIcon("🗺️"), sy.PageOrder(6))
}

func main() {
	sy.App(nil)
}

func dashboard() {
	sy.SetPageConfig(sy.PageTitle("Syralit Showcase"), sy.ConfigIcon("🚀"))

	sy.Title("Syralit Showcase")
	sy.Markdown("A **comprehensive demo** of all Syralit features. Use the sidebar to explore different sections.")

	sy.Divider()

	cols := sy.Columns(4)
	cols[0](func() { sy.Metric("Users", "12,345", sy.Delta("+8.2%")) })
	cols[1](func() { sy.Metric("Revenue", "$89.4K", sy.Delta("+15%"), sy.DeltaColor("normal")) })
	cols[2](func() { sy.Metric("Latency", "42ms", sy.Delta("-3ms"), sy.DeltaColor("inverse")) })
	cols[3](func() { sy.Metric("Errors", "0.02%", sy.Delta("-50%"), sy.DeltaColor("inverse")) })

	sy.Header("Weekly Revenue")
	sy.LineChart(map[string][]float64{
		"Revenue": {45, 52, 48, 61, 55, 67, 72},
		"Target":  {50, 50, 50, 60, 60, 60, 70},
	})

	lr := sy.WeightedColumns(2, 1)
	lr[0](func() {
		sy.Subheader("Recent Activity")
		sy.DataFrame(
			[]string{"Time", "Event", "Status"},
			[][]any{
				{"09:15", "Deploy v2.4.1", "Success"},
				{"09:02", "CI pipeline", "Running"},
				{"08:45", "PR #142 merged", "Complete"},
				{"08:30", "Alert resolved", "OK"},
			},
		)
	})
	lr[1](func() {
		sy.Subheader("Distribution")
		sy.PieChart(map[string]float64{
			"API":     45,
			"Web":     30,
			"Mobile":  20,
			"Desktop": 5,
		})
	})
}

func widgets() {
	sy.Title("Input Widgets")

	tab := sy.Tabs([]string{"Text", "Selection", "Numbers", "Other"})

	tab("Text", func() {
		name := sy.TextInput("Name", sy.Placeholder("Enter your name..."))
		bio := sy.TextArea("Bio", sy.Placeholder("Tell us about yourself..."), sy.Height(120))
		if name != "" {
			sy.Success(fmt.Sprintf("Hello, %s!", name))
		}
		if bio != "" {
			sy.Caption(fmt.Sprintf("Bio length: %d characters", len(bio)))
		}
	})

	tab("Selection", func() {
		lang := sy.SelectBox("Language", []string{"Go", "Python", "Rust", "TypeScript"})
		sy.Text("Selected: " + lang)

		features := sy.MultiSelect("Features", []string{"Charts", "Tables", "Forms", "Chat"})
		if len(features) > 0 {
			sy.Info("Selected: " + strings.Join(features, ", "))
		}

		level := sy.Radio("Experience", []string{"Beginner", "Intermediate", "Expert"})
		sy.Text("Level: " + level)

		size := sy.SelectSlider("Size", []string{"XS", "S", "M", "L", "XL"}, sy.DefaultValue("M"))
		sy.Text("Size: " + size)
	})

	tab("Numbers", func() {
		age := sy.NumberInput("Age", sy.Min(0), sy.Max(150), sy.Step(1))
		sy.Textf("Age: %.0f", age)

		temp := sy.Slider("Temperature", 0, 100, sy.DefaultValue(37.0))
		sy.Textf("Temperature: %.1f°C", temp)

		color := sy.ColorPicker("Accent Color")
		sy.Text("Color: " + color)
	})

	tab("Other", func() {
		date := sy.DateInput("Start Date")
		if date != "" {
			sy.Text("Date: " + date)
		}

		t := sy.TimeInput("Meeting Time")
		if t != "" {
			sy.Text("Time: " + t)
		}

		dark := sy.Toggle("Dark mode preference")
		if dark {
			sy.Info("Dark mode preferred")
		}

		agreed := sy.Checkbox("I agree to the terms")
		if agreed {
			sy.Success("Thank you for agreeing!")
		}
	})

	sy.Divider()
	sy.Header("Buttons & Actions")

	bcols := sy.Columns(3)
	bcols[0](func() {
		count := sy.State("btn_count", 0)
		if sy.Button("Click me") {
			count.Set(count.Get() + 1)
			sy.Toast("Clicked!", "success")
		}
		sy.Textf("Count: %d", count.Get())
	})
	bcols[1](func() {
		sy.LinkButton("Syralit on GitHub", "https://github.com/HazelnutParadise/syralit")
		sy.DownloadButton("Download CSV",
			[]byte("name,value\nfoo,42\nbar,99"),
			"data.csv",
			sy.MimeType("text/csv"),
		)
	})
	bcols[2](func() {
		if sy.Button("Celebrate!", sy.Key("celebrate")) {
			sy.Balloons()
		}
		if sy.Button("Snow!", sy.Key("snow")) {
			sy.Snow()
		}
	})
}

func charts() {
	sy.Title("Charts")

	months := map[string][]float64{
		"Sales":    {120, 150, 180, 140, 200, 230},
		"Expenses": {80, 90, 100, 95, 110, 105},
	}

	tab := sy.Tabs([]string{"Line", "Bar", "Area", "Scatter", "Pie", "Doughnut", "Histogram", "Radar", "Graphviz", "Vega-Lite", "Plotly"})

	tab("Line", func() {
		sy.Subheader("Line Chart")
		sy.LineChart(months, sy.ChartTitle("Monthly Trends"))
	})

	tab("Bar", func() {
		sy.Subheader("Bar Chart")
		sy.BarChart(months, sy.ChartTitle("Sales vs Expenses"))
	})

	tab("Area", func() {
		sy.Subheader("Area Chart")
		sy.AreaChart(months)
	})

	tab("Scatter", func() {
		sy.Subheader("Scatter Chart")
		points := make([][2]float64, 50)
		for i := range points {
			x := float64(i) * 0.5
			points[i] = [2]float64{x, math.Sin(x)*10 + 20 + float64(i%5)}
		}
		sy.ScatterChart(map[string][][2]float64{
			"Measurements": points,
		})
	})

	tab("Pie", func() {
		sy.Subheader("Pie Chart")
		sy.PieChart(map[string]float64{
			"Go":         45,
			"Python":     25,
			"TypeScript": 20,
			"Other":      10,
		})
	})

	tab("Doughnut", func() {
		sy.Subheader("Doughnut Chart")
		sy.DoughnutChart(map[string]float64{
			"API":    45,
			"Web":    30,
			"Mobile": 20,
			"CLI":    5,
		}, sy.ChartTitle("Traffic Sources"))
	})

	tab("Histogram", func() {
		sy.Subheader("Histogram")
		data := make([]float64, 200)
		for i := range data {
			data[i] = 50 + math.Sin(float64(i)*0.3)*20 + float64(i%7)*2
		}
		sy.HistogramChart(data, 15, sy.ChartTitle("Value Distribution"))
	})

	tab("Radar", func() {
		sy.Subheader("Radar Chart")
		sy.RadarChart(
			[]string{"Speed", "Reliability", "Cost", "Features", "Support"},
			map[string][]float64{
				"Product A": {85, 90, 60, 75, 80},
				"Product B": {70, 80, 90, 85, 65},
			},
			sy.ChartTitle("Product Comparison"),
		)
	})

	tab("Graphviz", func() {
		sy.Subheader("Graphviz Chart")
		sy.GraphvizChart(`digraph {
  rankdir=LR;
  Client -> "Load Balancer" -> "App Server" -> Database;
  "App Server" -> Cache;
  Cache -> "App Server" [style=dashed];
}`, sy.Height(300))
	})

	tab("Vega-Lite", func() {
		sy.Subheader("Vega-Lite Chart (Altair equivalent)")
		sy.VegaLiteChart(map[string]any{
			"mark": "bar",
			"encoding": map[string]any{
				"x": map[string]any{"field": "category", "type": "nominal",
					"axis": map[string]any{"labelAngle": 0}},
				"y": map[string]any{"field": "value", "type": "quantitative"},
				"color": map[string]any{"field": "category", "type": "nominal"},
			},
			"data": map[string]any{
				"values": []map[string]any{
					{"category": "Go", "value": 45},
					{"category": "Python", "value": 35},
					{"category": "Rust", "value": 25},
					{"category": "TypeScript", "value": 30},
				},
			},
		}, sy.Height(350))
	})

	tab("Plotly", func() {
		sy.Subheader("Plotly Chart")
		sy.PlotlyChart(map[string]any{
			"data": []map[string]any{
				{
					"x":    []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"},
					"y":    []float64{120, 150, 180, 140, 200, 230},
					"type": "scatter",
					"mode": "lines+markers",
					"name": "Sales",
				},
				{
					"x":    []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun"},
					"y":    []float64{80, 90, 100, 95, 110, 105},
					"type": "scatter",
					"mode": "lines+markers",
					"name": "Expenses",
				},
			},
			"layout": map[string]any{
				"title": "Sales vs Expenses (Plotly)",
			},
		}, sy.Height(400))
	})
}

func layout() {
	sy.Title("Layout & Containers")

	sy.Header("Columns")
	cols := sy.WeightedColumns(2, 1, 1)
	cols[0](func() {
		sy.Container(func() {
			sy.Text("Wide column (2fr)")
			sy.Progress(0.75)
		}, sy.Border())
	})
	cols[1](func() {
		sy.Container(func() {
			sy.Text("Narrow (1fr)")
			sy.Info("Info box")
		}, sy.Border())
	})
	cols[2](func() {
		sy.Container(func() {
			sy.Text("Narrow (1fr)")
			sy.Warning("Warning")
		}, sy.Border())
	})

	sy.Divider()

	sy.Header("Expander")
	sy.Expander("Click to expand details", func() {
		sy.Markdown(`
### Markdown Rendering

This is **rendered Markdown** with:

- Bullet points
- **Bold** and *italic*
- [Links](https://github.com/HazelnutParadise/syralit)

` + "```go\nfmt.Println(\"Hello\")\n```")
	})

	sy.Header("Popover")
	sy.Popover("Show info", func() {
		sy.Text("This floats above the content.")
		sy.Caption("Click outside to close.")
	})

	sy.Header("Dialog")
	sy.Dialog("Settings", func() {
		sy.TextInput("App Name", sy.Key("dialog_name"))
		sy.Toggle("Notifications", sy.Key("dialog_notif"))
		if sy.Button("Save", sy.Key("dialog_save")) {
			sy.CloseDialog("Settings")
			sy.Toast("Settings saved!", "success")
		}
	}, sy.Key("Settings"))

	if sy.Button("Open Dialog", sy.Key("open_dialog")) {
		sy.ShowDialog("Settings")
	}

	sy.Header("Form")
	sy.Form("contact", func() {
		sy.TextInput("Email", sy.Key("form_email"), sy.Placeholder("you@example.com"))
		sy.TextArea("Message", sy.Key("form_msg"), sy.Placeholder("Your message..."))
		if sy.FormSubmitButton("Send") {
			sy.Toast("Form submitted!", "success")
		}
	})
}

func data() {
	sy.Title("Data & Display")

	sy.Header("Code with Syntax Highlighting")
	sy.Code(`package main

import (
    "fmt"
    sy "github.com/HazelnutParadise/syralit"
)

func main() {
    sy.App(func() {
        sy.Title("Hello, Syralit")
        name := sy.TextInput("Name")
        if name != "" {
            sy.Success(fmt.Sprintf("Hello, %s!", name))
        }
    })
}`, sy.Language("go"))

	sy.Header("JSON Viewer")
	sy.JSON(map[string]any{
		"name":    "Syralit",
		"version": "0.1.0",
		"features": []string{
			"widgets", "charts", "layout", "state",
		},
	})

	sy.Header("Table")
	sy.Table(
		[]string{"Feature", "Status", "Priority"},
		[][]string{
			{"Widgets", "Complete", "P0"},
			{"Charts", "Complete", "P0"},
			{"Multi-page", "Complete", "P1"},
			{"Themes", "Basic", "P2"},
		},
	)

	sy.Header("Badges")
	bcols := sy.Columns(4)
	bcols[0](func() { sy.Badge("Production", sy.Color("green")) })
	bcols[1](func() { sy.Badge("Beta", sy.Color("orange")) })
	bcols[2](func() { sy.Badge("Deprecated", sy.Color("red")) })
	bcols[3](func() { sy.Badge("New", sy.Color("blue")) })

	sy.Header("Feedback Widget")
	fb := sy.Feedback(sy.Key("showcase_feedback"))
	if fb != "" {
		sy.Textf("You voted: %s", fb)
	}

	sy.Header("Editable Table")
	edited := sy.DataEditor(
		[]string{"Name", "Score", "Grade"},
		[][]any{
			{"Alice", 95, "A"},
			{"Bob", 82, "B"},
			{"Carol", 78, "C+"},
		},
		sy.Key("grade_editor"),
		sy.DynamicRows(),
	)
	if len(edited) > 0 {
		sy.Caption(fmt.Sprintf("First row: %v", edited[0]))
	}

	sy.Header("Column Config Table")
	sy.DataEditor(
		[]string{"Name", "Score", "Pass", "Grade"},
		[][]any{
			{"Alice", 95, true, "A"},
			{"Bob", 82, true, "B"},
		},
		sy.Key("config_editor"),
		sy.ColConfig(map[string]sy.ColumnConfig{
			"Score": {Type: "number", Min: 0, Max: 100},
			"Pass":  {Type: "checkbox"},
			"Grade": {Type: "select", Options: []string{"A", "B", "C", "D", "F"}},
		}),
	)

	sy.Header("Status Container")
	sy.Status("Processing data", "complete", func() {
		sy.Text("All 3 datasets loaded successfully.")
		sy.Progress(1.0)
	})

	sy.Header("LaTeX")
	sy.LaTeX(`E = mc^2`)
	sy.LaTeX(`\int_{-\infty}^{\infty} e^{-x^2} dx = \sqrt{\pi}`)

	sy.Header("Fragment (Partial Rerun)")
	sy.Caption("The counter below re-renders independently without affecting the rest of the page.")
	sy.Fragment("data-counter", func() {
		count := sy.State("frag_demo_count", 0)
		cols := sy.Columns(2)
		cols[0](func() {
			if sy.Button("Increment", sy.Key("frag_inc")) {
				count.Set(count.Get() + 1)
			}
		})
		cols[1](func() {
			sy.Metric("Fragment Counter", fmt.Sprintf("%d", count.Get()))
		})
	})

	sy.Sidebar(func() {
		sy.Caption("Sidebar Content")
		sy.Text("This appears in the sidebar below the page links.")
		sy.Textf("Current time: %s", time.Now().Format("15:04:05"))
	})
}

func mapMedia() {
	sy.Title("Map & Media")

	sy.Header("Interactive Map")
	sy.Map([]sy.MapPoint{
		{Lat: 25.0330, Lon: 121.5654, Text: "Taipei 101"},
		{Lat: 25.0478, Lon: 121.5170, Text: "Taipei Main Station"},
		{Lat: 25.0340, Lon: 121.5645, Text: "Xinyi District"},
		{Lat: 25.0524, Lon: 121.5207, Text: "Zhongshan District"},
	}, sy.Height(450))

	sy.Divider()

	sy.Header("Camera Input")
	sy.Caption("Click 'Take Photo' to capture from your webcam.")
	photo := sy.CameraInput("Camera", sy.Key("cam"))
	if photo != "" {
		sy.Success("Photo captured!")
		sy.Image(photo, sy.ImageCaption("Captured photo"))
	}
}
