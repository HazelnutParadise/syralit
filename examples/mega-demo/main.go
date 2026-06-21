package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	sy "github.com/HazelnutParadise/syralit"
)

func init() {
	sy.AddPage("Dashboard", pageDashboard, sy.PageIcon("📊"), sy.PageOrder(1))
	sy.AddPage("Widgets", pageWidgets, sy.PageIcon("🎛️"), sy.PageOrder(2))
	sy.AddPage("Charts", pageCharts, sy.PageIcon("📈"), sy.PageOrder(3))
	sy.AddPage("External Charts", pageExternalCharts, sy.PageIcon("🌐"), sy.PageOrder(4))
	sy.AddPage("Layout", pageLayout, sy.PageIcon("🧩"), sy.PageOrder(5))
	sy.AddPage("Forms & Dialogs", pageForms, sy.PageIcon("📝"), sy.PageOrder(6))
	sy.AddPage("Data Tables", pageData, sy.PageIcon("🗃️"), sy.PageOrder(7))
	sy.AddPage("Chat & Stream", pageChat, sy.PageIcon("💬"), sy.PageOrder(8))
	sy.AddPage("Maps & Media", pageMaps, sy.PageIcon("🗺️"), sy.PageOrder(9))
	sy.AddPage("State & Cache", pageState, sy.PageIcon("⚙️"), sy.PageOrder(10))
}

func main() { sy.App(nil) }

// ═══════════════════════════════════════════════════════════════════════════
// Page 1: Dashboard
// ═══════════════════════════════════════════════════════════════════════════

func pageDashboard() {
	sy.SetPageConfig(sy.PageTitle("Syralit Mega Demo"), sy.ConfigIcon("🚀"))

	sy.Title("Syralit Mega Demo")
	sy.Markdown("A **comprehensive showcase** of every Syralit feature. Navigate pages from the sidebar.")

	sy.Divider()

	// KPI row
	cols := sy.Columns(4)
	cols[0](func() { sy.Metric("Users", "24,891", sy.Delta("+12.3%")) })
	cols[1](func() { sy.Metric("Revenue", "$182.4K", sy.Delta("+18%"), sy.DeltaColor("normal")) })
	cols[2](func() { sy.Metric("Latency", "38ms", sy.Delta("-5ms"), sy.DeltaColor("inverse")) })
	cols[3](func() { sy.Metric("Error Rate", "0.03%", sy.Delta("-40%"), sy.DeltaColor("inverse")) })

	sy.Divider()

	// Charts row
	lr := sy.WeightedColumns(3, 2)
	lr[0](func() {
		sy.Header("Weekly Revenue")
		sy.LineChart(map[string][]float64{
			"Revenue": {52, 58, 61, 55, 67, 72, 80},
			"Target":  {55, 55, 60, 60, 65, 65, 70},
		}, sy.ChartTitle("Revenue vs Target (K)"))
	})
	lr[1](func() {
		sy.Header("Traffic Sources")
		sy.DoughnutChart(map[string]float64{
			"Organic":  42,
			"Direct":   28,
			"Referral": 18,
			"Social":   12,
		})
	})

	sy.Divider()

	// Activity table
	sy.Header("Recent Activity")
	sy.DataFrame(
		[]string{"Time", "User", "Action", "Status"},
		[][]any{
			{"09:42", "alice@acme.com", "Deployed v3.1.0", "Success"},
			{"09:35", "bob@acme.com", "Merged PR #287", "Complete"},
			{"09:28", "carol@acme.com", "CI Pipeline", "Running"},
			{"09:15", "dave@acme.com", "Alert triggered", "Resolved"},
			{"09:02", "eve@acme.com", "Scaled replicas 3→5", "Success"},
		},
	)

	sy.Sidebar(func() {
		sy.Caption("Syralit Mega Demo")
		sy.Text("All features in one app.")
		sy.Divider()
		sy.Textf("Rendered at: %s", time.Now().Format("15:04:05"))
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// Page 2: All Input Widgets
// ═══════════════════════════════════════════════════════════════════════════

func pageWidgets() {
	sy.Title("Input Widgets")
	sy.Caption("Every input widget Syralit offers.")

	tab := sy.Tabs([]string{"Text", "Selection", "Numbers", "Date/Time", "Buttons", "Media", "Special"})

	tab("Text", func() {
		sy.Subheader("Text Inputs")
		name := sy.TextInput("Name", sy.Placeholder("Enter your name"), sy.Key("w_name"))
		pw := sy.PasswordInput("Password", sy.Placeholder("••••••••"), sy.Key("w_pw"))
		bio := sy.TextArea("Bio", sy.Placeholder("Tell us about yourself..."), sy.Height(100), sy.Key("w_bio"))

		if name != "" {
			sy.Success("Hello, " + name + "!")
		}
		if pw != "" {
			sy.Caption(fmt.Sprintf("Password length: %d", len(pw)))
		}
		if bio != "" {
			sy.Caption(fmt.Sprintf("Bio: %d chars", len(bio)))
		}
	})

	tab("Selection", func() {
		sy.Subheader("Selection Widgets")

		lang := sy.SelectBox("Language", []string{"Go", "Python", "Rust", "TypeScript", "Java", "C++", "Ruby", "Swift"}, sy.Key("w_lang"))
		sy.Text("Selected: " + lang)

		features := sy.MultiSelect("Features", []string{"Charts", "Tables", "Forms", "Chat", "Auth", "Maps", "Streaming"}, sy.Key("w_feat"))
		if len(features) > 0 {
			sy.Info("Features: " + strings.Join(features, ", "))
		}

		level := sy.Radio("Experience", []string{"Beginner", "Intermediate", "Advanced", "Expert"}, sy.Key("w_level"))
		sy.Text("Level: " + level)

		size := sy.SelectSlider("T-Shirt Size", []string{"XS", "S", "M", "L", "XL", "XXL"}, sy.DefaultValue("M"), sy.Key("w_size"))
		sy.Text("Size: " + size)

		seg := sy.SegmentedControl("View Mode", []string{"Grid", "List", "Table"}, sy.Key("w_seg"))
		sy.Text("View: " + seg)

		pill := sy.Pills("Category", []string{"All", "Active", "Archived", "Draft"}, sy.Key("w_pill"))
		sy.Text("Filter: " + pill)
	})

	tab("Numbers", func() {
		sy.Subheader("Numeric Inputs")

		age := sy.NumberInput("Age", sy.Min(0), sy.Max(150), sy.Step(1), sy.Key("w_age"))
		sy.Textf("Age: %.0f", age)

		temp := sy.Slider("Temperature (°C)", 0, 100, sy.DefaultValue(37.0), sy.Key("w_temp"))
		sy.Textf("Temperature: %.1f°C", temp)

		rating := sy.Slider("Rating", 1, 5, sy.DefaultValue(3.0), sy.Step(0.5), sy.Key("w_rating"))
		sy.Textf("Rating: %.1f / 5.0", rating)

		lo, hi := sy.RangeSlider("Price range", 0, 1000, sy.DefaultValue([2]float64{200, 800}), sy.Step(10), sy.Key("w_price"))
		sy.Textf("Price: $%.0f – $%.0f", lo, hi)

		d := sy.DateSlider("Release date", "2026-01-01", "2026-12-31", sy.DefaultValue("2026-06-15"), sy.Key("w_dslider"))
		sy.Text("Release: " + d)

		page := sy.Pagination(20, sy.Key("w_page"))
		sy.Textf("Page: %d / 20", page)
	})

	tab("Date/Time", func() {
		sy.Subheader("Date & Time")

		date := sy.DateInput("Start Date", sy.MinDate("2026-01-01"), sy.MaxDate("2026-12-31"), sy.Key("w_date"))
		if date != "" {
			sy.Text("Date: " + date)
		}
		sy.Caption("Bounded to 2026 via MinDate/MaxDate.")

		start, end := sy.DateRangeInput("Booking period", sy.Key("w_daterange"))
		if start != "" || end != "" {
			sy.Textf("From %s to %s", start, end)
		}

		t := sy.TimeInput("Meeting Time", sy.Key("w_time"))
		if t != "" {
			sy.Text("Time: " + t)
		}

		color := sy.ColorPicker("Brand Color", sy.Key("w_color"))
		sy.Text("Color: " + color)
	})

	tab("Buttons", func() {
		sy.Subheader("Buttons & Actions")

		bcols := sy.Columns(3)
		bcols[0](func() {
			count := sy.State("w_btn_count", 0)
			if sy.Button("Click me", sy.Key("w_btn")) {
				count.Set(count.Get() + 1)
				sy.Toast(fmt.Sprintf("Clicked %d times!", count.Get()), "success")
			}
			sy.Textf("Count: %d", count.Get())
		})
		bcols[1](func() {
			sy.LinkButton("GitHub", "https://github.com/HazelnutParadise/syralit", sy.Icon("🔗"), sy.ButtonType("secondary"), sy.Key("w_link"))
			sy.DownloadButton("Download CSV",
				[]byte("name,score\nAlice,95\nBob,82\nCarol,78"),
				"scores.csv",
				sy.MimeType("text/csv"),
				sy.Icon("⬇️"),
				sy.Key("w_dl"),
			)
		})
		bcols[2](func() {
			if sy.Button("Balloons!", sy.Key("w_bal")) {
				sy.Balloons()
			}
			if sy.Button("Snow!", sy.Key("w_snow")) {
				sy.Snow()
			}
		})

		sy.Divider()

		sy.Subheader("Button Variants")
		sy.Button("Primary", sy.Key("w_bt_pri"), sy.Icon("🚀"))
		sy.Button("Secondary", sy.Key("w_bt_sec"), sy.ButtonType("secondary"))
		sy.Button("Tertiary", sy.Key("w_bt_ter"), sy.ButtonType("tertiary"))
		sy.Button("Full-width Action", sy.Key("w_bt_full"), sy.Icon("✅"), sy.UseContainerWidth())

		sy.Divider()

		sy.Subheader("Toggles & Checks")
		dark := sy.Toggle("Dark mode preference", sy.Key("w_dark"))
		if dark {
			sy.Info("Dark mode preferred")
		}
		agreed := sy.Checkbox("I agree to the terms", sy.Key("w_agree"))
		if agreed {
			sy.Success("Thank you!")
		}
	})

	tab("Media", func() {
		sy.Subheader("Media Inputs")

		sy.Text("File Uploader")
		file := sy.FileUploader("Upload a file", sy.Key("w_file"))
		if file != nil {
			sy.Success(fmt.Sprintf("Uploaded: %s (%d bytes)", file.Name, len(file.Data)))
		}

		sy.Divider()

		sy.Text("Camera Input")
		photo := sy.CameraInput("Take a photo", sy.Key("w_cam"))
		if photo != "" {
			sy.Success("Photo captured!")
		}

		sy.Divider()

		sy.Text("Audio Input")
		audio := sy.AudioInput("Record audio", sy.Key("w_audio"))
		if audio != "" {
			sy.Success("Audio recorded!")
		}
	})

	tab("Special", func() {
		sy.Subheader("Special Widgets")

		sy.Text("Chat Input")
		msg := sy.ChatInput("Type a message...", sy.Key("w_chat"))
		if msg != "" {
			sy.Info("You said: " + msg)
		}

		sy.Divider()

		sy.Text("Feedback — thumbs / stars / faces")
		fcols := sy.Columns(3)
		fcols[0](func() {
			if fb := sy.Feedback(sy.Key("w_fb")); fb != "" {
				sy.Caption("Thumbs: " + fb)
			}
		})
		fcols[1](func() {
			if fb := sy.Feedback(sy.Key("w_fb_stars"), sy.FeedbackStyle("stars")); fb != "" {
				sy.Caption("Stars: " + fb)
			}
		})
		fcols[2](func() {
			if fb := sy.Feedback(sy.Key("w_fb_faces"), sy.FeedbackStyle("faces")); fb != "" {
				sy.Caption("Faces: " + fb)
			}
		})

		sy.Divider()

		sy.Text("Badges")
		bcols := sy.Columns(4)
		bcols[0](func() { sy.Badge("Production", sy.Color("green")) })
		bcols[1](func() { sy.Badge("Beta", sy.Color("orange")) })
		bcols[2](func() { sy.Badge("Deprecated", sy.Color("red")) })
		bcols[3](func() { sy.Badge("New", sy.Color("blue")) })
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// Page 3: Built-in Charts
// ═══════════════════════════════════════════════════════════════════════════

func pageCharts() {
	sy.Title("Built-in Charts (Chart.js)")

	months := map[string][]float64{
		"Sales":    {120, 150, 180, 140, 200, 230, 210, 250, 270, 290, 310, 350},
		"Expenses": {80, 90, 100, 95, 110, 105, 115, 120, 130, 125, 140, 145},
	}

	tab := sy.Tabs([]string{"Line", "Bar", "Area", "Scatter", "Pie", "Doughnut", "Histogram", "Radar", "Graphviz"})

	tab("Line", func() {
		sy.LineChart(months, sy.ChartTitle("Monthly Sales vs Expenses"))
	})

	tab("Bar", func() {
		sy.BarChart(months, sy.ChartTitle("Sales vs Expenses (Bar)"))
	})

	tab("Area", func() {
		sy.AreaChart(months, sy.ChartTitle("Filled Area"))
	})

	tab("Scatter", func() {
		points := make([][2]float64, 80)
		for i := range points {
			x := float64(i) * 0.4
			points[i] = [2]float64{x, math.Sin(x)*15 + 30 + float64(i%7)*2}
		}
		points2 := make([][2]float64, 80)
		for i := range points2 {
			x := float64(i) * 0.4
			points2[i] = [2]float64{x, math.Cos(x)*12 + 25 + float64(i%5)*3}
		}
		sy.ScatterChart(map[string][][2]float64{
			"Sensor A": points,
			"Sensor B": points2,
		}, sy.ChartTitle("Sensor Readings"))
	})

	tab("Pie", func() {
		sy.PieChart(map[string]float64{
			"Go":         38,
			"Python":     28,
			"TypeScript": 18,
			"Rust":       10,
			"Other":      6,
		}, sy.ChartTitle("Language Popularity"))
	})

	tab("Doughnut", func() {
		sy.DoughnutChart(map[string]float64{
			"API":      42,
			"Web":      28,
			"Mobile":   18,
			"Desktop":  8,
			"CLI":      4,
		}, sy.ChartTitle("Traffic by Platform"))
	})

	tab("Histogram", func() {
		data := make([]float64, 500)
		for i := range data {
			data[i] = 50 + math.Sin(float64(i)*0.2)*20 + float64(i%11)*2.5
		}
		sy.HistogramChart(data, 20, sy.ChartTitle("Response Time Distribution (ms)"))
	})

	tab("Radar", func() {
		sy.RadarChart(
			[]string{"Speed", "Reliability", "Cost", "Features", "Support", "Security"},
			map[string][]float64{
				"Product A": {85, 92, 60, 78, 88, 75},
				"Product B": {72, 80, 90, 85, 65, 95},
				"Product C": {90, 70, 75, 92, 78, 80},
			},
			sy.ChartTitle("Product Comparison"),
		)
	})

	tab("Graphviz", func() {
		sy.GraphvizChart(`digraph {
  rankdir=TB;
  node [shape=box, style=filled, fillcolor="#e8f4f8"];

  Client -> "Load Balancer";
  "Load Balancer" -> "App Server 1";
  "Load Balancer" -> "App Server 2";
  "App Server 1" -> Database;
  "App Server 2" -> Database;
  "App Server 1" -> Cache;
  "App Server 2" -> Cache;
  Cache -> Database [style=dashed, label="miss"];
  Database -> "Backup DB" [style=dotted];
}`, sy.Height(400))
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// Page 4: External Charts (CDN)
// ═══════════════════════════════════════════════════════════════════════════

func pageExternalCharts() {
	sy.Title("External Charts (CDN)")
	sy.Caption("Vega-Lite, Plotly, Bokeh, deck.gl — all rendered client-side via CDN.")

	tab := sy.Tabs([]string{"Vega-Lite", "Plotly", "PydeckChart"})

	tab("Vega-Lite", func() {
		sy.Subheader("Vega-Lite (Altair equivalent)")
		sy.VegaLiteChart(map[string]any{
			"mark": map[string]any{"type": "bar", "cornerRadiusTopLeft": 4, "cornerRadiusTopRight": 4},
			"encoding": map[string]any{
				"x":     map[string]any{"field": "language", "type": "nominal", "axis": map[string]any{"labelAngle": 0}},
				"y":     map[string]any{"field": "users", "type": "quantitative", "title": "Users (millions)"},
				"color": map[string]any{"field": "language", "type": "nominal", "legend": nil},
			},
			"data": map[string]any{
				"values": []map[string]any{
					{"language": "Go", "users": 3.2},
					{"language": "Python", "users": 15.7},
					{"language": "JavaScript", "users": 18.5},
					{"language": "Rust", "users": 1.8},
					{"language": "TypeScript", "users": 9.3},
					{"language": "Java", "users": 12.1},
				},
			},
		}, sy.Height(400))

		sy.Divider()
		sy.Subheader("Vega-Lite Scatter Plot")
		values := make([]map[string]any, 60)
		for i := range values {
			x := float64(i) + 1
			values[i] = map[string]any{
				"x":        x,
				"y":        math.Log(x)*15 + float64(i%5)*3,
				"category": []string{"A", "B", "C"}[i%3],
			}
		}
		sy.VegaLiteChart(map[string]any{
			"mark": "point",
			"encoding": map[string]any{
				"x":     map[string]any{"field": "x", "type": "quantitative"},
				"y":     map[string]any{"field": "y", "type": "quantitative"},
				"color": map[string]any{"field": "category", "type": "nominal"},
			},
			"data": map[string]any{"values": values},
		}, sy.Height(350))
	})

	tab("Plotly", func() {
		sy.Subheader("Plotly Line + Markers")
		sy.PlotlyChart(map[string]any{
			"data": []map[string]any{
				{
					"x":      []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"},
					"y":      []float64{120, 150, 180, 140, 200, 230, 210, 250, 270, 290, 310, 350},
					"type":   "scatter",
					"mode":   "lines+markers",
					"name":   "Revenue",
					"marker": map[string]any{"size": 8},
				},
				{
					"x":    []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"},
					"y":    []float64{80, 90, 100, 95, 110, 105, 115, 120, 130, 125, 140, 145},
					"type": "scatter",
					"mode": "lines+markers",
					"name": "Expenses",
				},
			},
			"layout": map[string]any{
				"title":  "Revenue vs Expenses (Plotly)",
				"height": 450,
			},
		})

		sy.Divider()
		sy.Subheader("Plotly Bar Chart")
		sy.PlotlyChart(map[string]any{
			"data": []map[string]any{
				{
					"x":    []string{"Go", "Python", "Rust", "TypeScript"},
					"y":    []float64{85, 72, 90, 78},
					"type": "bar",
					"name": "Performance",
					"marker": map[string]any{
						"color": []string{"#00ADD8", "#3776AB", "#CE422B", "#3178C6"},
					},
				},
			},
			"layout": map[string]any{"title": "Language Performance Score"},
		})
	})

	tab("PydeckChart", func() {
		sy.Subheader("deck.gl 3D Visualization")
		sy.Caption("HexagonLayer over San Francisco bike parking data.")
		sy.PydeckChart(map[string]any{
			"initialViewState": map[string]any{
				"latitude":  37.7749,
				"longitude": -122.4194,
				"zoom":      11,
				"pitch":     50,
				"bearing":   0,
			},
			"layers": []map[string]any{
				{
					"@@type":         "HexagonLayer",
					"data":           "https://raw.githubusercontent.com/visgl/deck.gl-data/master/website/sf-bike-parking.json",
					"getPosition":    "@@=[lng, lat]",
					"radius":         200,
					"elevationScale": 4,
					"extruded":       true,
					"coverage":       0.8,
				},
			},
		}, sy.Height(500))
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// Page 5: Layout & Containers
// ═══════════════════════════════════════════════════════════════════════════

func pageLayout() {
	sy.Title("Layout & Containers")

	sy.Header("Equal Columns")
	cols3 := sy.Columns(3)
	cols3[0](func() {
		sy.Container(func() {
			sy.Subheader("Column 1")
			sy.Text("Equal width column.")
			sy.Progress(0.33)
		}, sy.Border())
	})
	cols3[1](func() {
		sy.Container(func() {
			sy.Subheader("Column 2")
			sy.Text("Second column.")
			sy.Progress(0.66)
		}, sy.Border())
	})
	cols3[2](func() {
		sy.Container(func() {
			sy.Subheader("Column 3")
			sy.Text("Third column.")
			sy.Progress(1.0)
		}, sy.Border())
	})

	sy.Divider()

	sy.Header("Weighted Columns with Vertical Alignment")
	wc := sy.WeightedColumns(3, 1, 1)
	wc[0](func() {
		sy.Container(func() {
			sy.Subheader("Wide (3fr)")
			sy.Markdown("This column takes up **3x** the space of the narrow columns. Great for main content with a sidebar.")
			sy.LineChart(map[string][]float64{
				"Data": {10, 25, 18, 42, 35},
			})
		}, sy.Border())
	})
	wc[1](func() {
		sy.Container(func() {
			sy.Subheader("Narrow")
			sy.Info("Sidebar-like")
			sy.Metric("Score", "92")
		}, sy.Border())
	})
	wc[2](func() {
		sy.Container(func() {
			sy.Subheader("Narrow")
			sy.Warning("Alert zone")
			sy.Metric("Risk", "Low")
		}, sy.Border())
	})

	sy.Divider()

	sy.Header("Tabs")
	ltab := sy.Tabs([]string{"Preview", "Code", "Output"})
	ltab("Preview", func() {
		sy.Text("This is the preview pane.")
		sy.Image("https://placehold.co/600x200/e8f4f8/333?text=Preview+Content", sy.ImageCaption("Placeholder preview"))
	})
	ltab("Code", func() {
		sy.Code(`func main() {
    sy.App(func() {
        sy.Title("Tabs Demo")
        tab := sy.Tabs([]string{"A", "B"})
        tab("A", func() { sy.Text("Tab A") })
        tab("B", func() { sy.Text("Tab B") })
    })
}`, sy.Language("go"))
	})
	ltab("Output", func() {
		sy.JSON(map[string]any{
			"status": "ok",
			"tabs":   []string{"Preview", "Code", "Output"},
			"active": "Output",
		})
	})

	sy.Divider()

	sy.Header("Expander")
	sy.Expander("System Details (click to expand)", func() {
		sy.Markdown(`
### System Information

| Key | Value |
|-----|-------|
| Go Version | 1.25 |
| Framework | Syralit |
| Server | WebSocket |
| Theme | Auto |
`)
		sy.LaTeX(`\sum_{i=1}^{n} x_i = x_1 + x_2 + \cdots + x_n`)
	})

	sy.Expander("More Details", func() {
		sy.Text("This expander is collapsed by default.")
		sy.Code(`sy.Expander("Title", func() { ... })`, sy.Language("go"))
	}, sy.Expanded())

	sy.Divider()

	sy.Header("Popover")
	sy.Popover("Hover for details", func() {
		sy.Text("This content floats above the page.")
		sy.Caption("Click outside to dismiss.")
		sy.Badge("Popover", sy.Color("blue"))
	})

	sy.Divider()

	sy.Header("Status Container")
	scols := sy.Columns(3)
	scols[0](func() {
		sy.Status("Data loaded", "complete", func() {
			sy.Text("All 5 datasets ready.")
			sy.Progress(1.0)
		})
	})
	scols[1](func() {
		sy.Status("Processing", "running", func() {
			sy.Text("Crunching numbers...")
			sy.Progress(0.65)
		})
	})
	scols[2](func() {
		sy.Status("Connection lost", "error", func() {
			sy.Text("Retrying in 5s...")
			sy.Progress(0.0)
		})
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// Page 6: Forms & Dialogs
// ═══════════════════════════════════════════════════════════════════════════

func pageForms() {
	sy.Title("Forms & Dialogs")

	sy.Header("Form (Batched Inputs)")
	sy.Caption("Widgets inside a form don't trigger reruns until submit.")

	sy.Form("contact_form", func() {
		fcols := sy.Columns(2)
		var fname, femail string
		fcols[0](func() {
			fname = sy.TextInput("Name", sy.Key("f_name"), sy.Placeholder("Jane Doe"))
		})
		fcols[1](func() {
			femail = sy.TextInput("Email", sy.Key("f_email"), sy.Placeholder("jane@example.com"))
		})

		fsubject := sy.SelectBox("Subject", []string{"General", "Bug Report", "Feature Request", "Support"}, sy.Key("f_subject"))
		fmsg := sy.TextArea("Message", sy.Key("f_msg"), sy.Placeholder("Your message..."), sy.Height(100))
		fpriority := sy.Slider("Priority", 1, 5, sy.DefaultValue(3.0), sy.Key("f_priority"))
		fagree := sy.Checkbox("Send me a copy", sy.Key("f_copy"))

		if sy.FormSubmitButton("Send Message") {
			if fname == "" || femail == "" {
				sy.Toast("Name and email are required!", "error")
			} else {
				sy.Toast(fmt.Sprintf("Message sent! Subject: %s, Priority: %.0f", fsubject, fpriority), "success")
				if fagree {
					sy.Toast("Copy will be sent to "+femail, "info")
				}
				_ = fmsg
			}
		}
	})

	sy.Divider()

	sy.Header("Form Batching: Range + Date Range")
	sy.Caption("RangeSlider and DateRangeInput are batched here — values only commit on submit.")
	applied := sy.State("mega_filter", "")
	sy.Form("filter_form", func() {
		lo, hi := sy.RangeSlider("Budget range", 0, 5000, sy.DefaultValue([2]float64{1000, 4000}), sy.Step(100), sy.Key("ff_budget"))
		start, end := sy.DateRangeInput("Travel dates", sy.Key("ff_dates"))
		if sy.FormSubmitButton("Apply filters") {
			applied.Set(fmt.Sprintf("Budget $%.0f–$%.0f · Dates %s→%s", lo, hi, dash(start), dash(end)))
		}
	})
	if applied.Get() != "" {
		mcols := sy.Columns(2)
		mcols[0](func() { sy.Metric("Applied filter", "✓", sy.Border()) })
		mcols[1](func() { sy.Metric("Summary", applied.Get(), sy.Border()) })
	}

	sy.Divider()

	sy.Header("Dialog (Modal)")
	sy.Dialog("settings_dialog", func() {
		sy.Subheader("App Settings")
		sy.TextInput("App Name", sy.Key("d_appname"), sy.DefaultValue("My App"))
		sy.SelectBox("Theme", []string{"Light", "Dark", "Auto"}, sy.Key("d_theme"))
		sy.Toggle("Enable Notifications", sy.Key("d_notif"))
		sy.Slider("Font Size", 12, 24, sy.DefaultValue(16.0), sy.Key("d_font"))

		dcols := sy.Columns(2)
		dcols[0](func() {
			if sy.Button("Save", sy.Key("d_save")) {
				sy.CloseDialog("settings_dialog")
				sy.Toast("Settings saved!", "success")
			}
		})
		dcols[1](func() {
			if sy.Button("Cancel", sy.Key("d_cancel")) {
				sy.CloseDialog("settings_dialog")
			}
		})
	}, sy.Key("settings_dialog"))

	if sy.Button("Open Settings Dialog", sy.Key("open_settings")) {
		sy.ShowDialog("settings_dialog")
	}

	sy.Divider()

	sy.Header("Status Messages")
	mcols := sy.Columns(2)
	mcols[0](func() {
		sy.Success("Operation completed successfully.")
		sy.Info("This is an informational message.")
	})
	mcols[1](func() {
		sy.Warning("Careful! This action cannot be undone.")
		sy.Error("Something went wrong. Please try again.")
	})

	sy.Subheader("Exception")
	sy.Caption("sy.Exception renders a Go error in a monospace box.")
	sy.Exception(fmt.Errorf("dial tcp 10.0.0.5:5432: connect: connection refused"))

	sy.Divider()

	sy.Header("Toast Notifications")
	tcols := sy.Columns(4)
	tcols[0](func() {
		if sy.Button("Success Toast", sy.Key("t_success")) {
			sy.Toast("All good!", "success")
		}
	})
	tcols[1](func() {
		if sy.Button("Info Toast", sy.Key("t_info")) {
			sy.Toast("FYI: New update available.", "info")
		}
	})
	tcols[2](func() {
		if sy.Button("Warning Toast", sy.Key("t_warn")) {
			sy.Toast("Disk space is running low.", "warning")
		}
	})
	tcols[3](func() {
		if sy.Button("Error Toast", sy.Key("t_err")) {
			sy.Toast("Connection failed!", "error")
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════
// Page 7: Data Tables
// ═══════════════════════════════════════════════════════════════════════════

func pageData() {
	sy.Title("Data Tables")

	sy.Header("Static Table")
	sy.Table(
		[]string{"Feature", "Status", "Priority", "Owner"},
		[][]string{
			{"Input Widgets", "Complete", "P0", "Core"},
			{"Charts", "Complete", "P0", "Core"},
			{"Layout", "Complete", "P0", "Core"},
			{"Auth", "Complete", "P1", "Security"},
			{"Themes", "In Progress", "P2", "Design"},
			{"Plugins", "Planned", "P3", "Community"},
		},
	)

	sy.Divider()

	sy.Header("DataFrame (Sortable)")
	sy.DataFrame(
		[]string{"Product", "Category", "Price", "Stock", "Rating"},
		[][]any{
			{"Laptop Pro", "Electronics", 1299.99, 45, 4.8},
			{"Wireless Mouse", "Electronics", 29.99, 230, 4.5},
			{"Ergonomic Chair", "Furniture", 449.00, 18, 4.7},
			{"Desk Lamp", "Furniture", 34.99, 92, 4.2},
			{"Go in Action", "Books", 44.99, 156, 4.6},
			{"Clean Code", "Books", 39.99, 203, 4.9},
			{"Coffee Mug", "Kitchen", 12.99, 500, 4.1},
			{"Water Bottle", "Kitchen", 24.99, 312, 4.4},
		},
		sy.Height(300),
	)

	sy.Divider()

	sy.Header("DataFrame — Selectable + Column Config")
	sy.Caption("Click rows to select; columns render by type via ColConfig.")
	tasks := [][]any{
		{"Ship v3.1", "Done", 100, "https://example.com/1"},
		{"Write docs", "In Progress", 60, "https://example.com/2"},
		{"Fix flaky test", "Todo", 0, "https://example.com/3"},
		{"Plan Q3", "In Progress", 35, "https://example.com/4"},
	}
	sel := sy.DataFrame(
		[]string{"Task", "Status", "Progress", "Link"},
		tasks,
		sy.Selectable(),
		sy.ColConfig(map[string]sy.ColumnConfig{
			"Progress": {Type: "progress", Max: 100},
			"Link":     {Type: "link"},
		}),
		sy.Key("task_df"),
	)
	if len(sel) > 0 {
		names := make([]string, 0, len(sel))
		for _, i := range sel {
			if i < len(tasks) {
				names = append(names, fmt.Sprint(tasks[i][0]))
			}
		}
		sy.Success("Selected: " + strings.Join(names, ", "))
	} else {
		sy.Caption("No rows selected.")
	}

	sy.Divider()

	sy.Header("DataEditor — All Column Types")
	sy.Caption("11 different column types demonstrating full DataEditor capabilities.")
	edited := sy.DataEditor(
		[]string{"Name", "Score", "Approved", "Grade", "Due Date", "Check-in", "Updated", "Website", "Avatar", "Progress", "Tags"},
		[][]any{
			{"Alice", 95, true, "A", "2026-07-01", "09:00", "2026-06-20T09:00", "https://example.com", "https://placehold.co/32", 90, "go,web"},
			{"Bob", 82, true, "B", "2026-07-15", "10:30", "2026-06-19T14:30", "https://example.org", "https://placehold.co/32", 65, "python"},
			{"Carol", 67, false, "D", "2026-08-01", "14:00", "2026-06-18T11:00", "https://example.net", "https://placehold.co/32", 40, "rust,cli"},
		},
		sy.Key("mega_editor"),
		sy.ColConfig(map[string]sy.ColumnConfig{
			"Score":    {Type: "number", Min: 0, Max: 100},
			"Approved": {Type: "checkbox"},
			"Grade":    {Type: "select", Options: []string{"A", "B", "C", "D", "F"}},
			"Due Date": {Type: "date"},
			"Check-in": {Type: "time"},
			"Updated":  {Type: "datetime"},
			"Website":  {Type: "link"},
			"Avatar":   {Type: "image"},
			"Progress": {Type: "progress", Max: 100},
			"Tags":     {Type: "list"},
		}),
		sy.DynamicRows(),
	)
	if len(edited) > 0 {
		sy.Caption(fmt.Sprintf("Editor has %d rows. First row: %v", len(edited), edited[0]))
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Page 8: Chat & Streaming
// ═══════════════════════════════════════════════════════════════════════════

func pageChat() {
	sy.Title("Chat & Streaming")

	sy.Header("Chat Interface")
	msgs := sy.State("mega_msgs", []map[string]string{})

	for _, m := range msgs.Get() {
		role := m["role"]
		content := m["content"]
		sy.ChatMessage(role, func() {
			sy.Markdown(content)
		})
	}

	prompt := sy.ChatInput("Ask me anything...", sy.Key("mega_chat"))

	if prompt != "" {
		history := msgs.Get()
		history = append(history, map[string]string{"role": "user", "content": prompt})
		msgs.Set(history)

		sy.ChatMessage("user", func() {
			sy.Markdown(prompt)
		})

		reply := chatReply(prompt)

		sy.ChatMessage("assistant", func() {
			sy.WriteStream(func(yield func(string)) {
				words := strings.Fields(reply)
				for i, word := range words {
					if i > 0 {
						yield(" ")
					}
					yield(word)
					time.Sleep(40 * time.Millisecond)
				}
			}, sy.Key(fmt.Sprintf("mega_stream_%d", len(msgs.Get()))))
		})

		history = msgs.Get()
		history = append(history, map[string]string{"role": "assistant", "content": reply})
		msgs.Set(history)
	}

	sy.Sidebar(func() {
		sy.Caption("Chat Controls")
		if sy.Button("Clear Chat", sy.Key("mega_clear_chat")) {
			msgs.Set([]map[string]string{})
			sy.Rerun()
		}
		sy.Divider()
		sy.Textf("Messages: %d", len(msgs.Get()))
	})

	sy.Divider()

	sy.Header("WriteStream Demo")
	sy.Caption("Token-by-token streaming output — ideal for LLM responses.")
	if sy.Button("Stream a paragraph", sy.Key("mega_stream_demo")) {
		sy.WriteStream(func(yield func(string)) {
			text := "Syralit is a Go-native framework for building interactive data apps. It provides a rich set of widgets, charts, layout containers, and state management primitives. The rerun model re-executes the entire app function on each interaction, making it simple to reason about state. All UI is declared imperatively — no templates, no virtual DOM, no frontend build step."
			for _, word := range strings.Fields(text) {
				yield(word + " ")
				time.Sleep(35 * time.Millisecond)
			}
		}, sy.Key("mega_stream_para"))
	}
}

func dash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

func chatReply(prompt string) string {
	lower := strings.ToLower(prompt)
	switch {
	case strings.Contains(lower, "hello") || strings.Contains(lower, "hi"):
		return "Hello! I'm a demo chatbot built with Syralit's chat widgets and WriteStream. Try asking about features, charts, or state management!"
	case strings.Contains(lower, "chart"):
		return "Syralit includes 9 built-in Chart.js chart types (line, bar, area, scatter, pie, doughnut, histogram, radar, graphviz) plus 5 external library integrations (Vega-Lite, Plotly, Bokeh, PyplotChart, deck.gl). All charts are interactive with tooltips and hover highlights."
	case strings.Contains(lower, "widget"):
		return "Syralit has 24+ input widgets: TextInput, PasswordInput, NumberInput, Slider, SelectSlider, Checkbox, Toggle, Radio, SelectBox, MultiSelect, DateInput, TimeInput, ColorPicker, FileUploader, CameraInput, AudioInput, ChatInput, Feedback, SegmentedControl, Pills, Pagination, and various button types."
	case strings.Contains(lower, "state"):
		return "State in Syralit is managed with `sy.State[T](key, default)` which returns a `*StateVar[T]`. Use `.Get()` to read and `.Set()` to write. State persists across reruns within the same session. For caching expensive computations, use `sy.CacheData` or `sy.CacheResource`."
	case strings.Contains(lower, "layout"):
		return "Syralit provides Columns (equal or weighted), Tabs, Sidebar, Expander, Container, Form, Status, Popover, Dialog, Empty, Fragment, and Divider. Containers can have borders, fixed heights, and vertical alignment."
	default:
		return fmt.Sprintf("You said: \"%s\". I'm a simple demo bot. Try asking about charts, widgets, state, or layout!", prompt)
	}
}

// ═══════════════════════════════════════════════════════════════════════════
// Page 9: Maps & Media
// ═══════════════════════════════════════════════════════════════════════════

func pageMaps() {
	sy.Title("Maps & Media")

	sy.Header("Interactive Map (Leaflet.js)")
	sy.Map([]sy.MapPoint{
		{Lat: 25.0330, Lon: 121.5654, Text: "Taipei 101"},
		{Lat: 25.0478, Lon: 121.5170, Text: "Taipei Main Station"},
		{Lat: 25.0340, Lon: 121.5645, Text: "Xinyi District"},
		{Lat: 25.0524, Lon: 121.5207, Text: "Zhongshan District"},
		{Lat: 25.0418, Lon: 121.5075, Text: "Wanhua (Longshan Temple)"},
	}, sy.Height(450))

	sy.Divider()

	sy.Header("Code with Syntax Highlighting")
	sy.Code(`package main

import (
    "fmt"
    sy "github.com/HazelnutParadise/syralit"
)

func main() {
    sy.App(func() {
        sy.Title("Hello, World!")
        name := sy.TextInput("Your name")
        if name != "" {
            sy.Success(fmt.Sprintf("Hello, %s!", name))
        }
    })
}`, sy.Language("go"))

	sy.Divider()

	sy.Header("LaTeX (KaTeX)")
	lcols := sy.Columns(2)
	lcols[0](func() {
		sy.Caption("Einstein's Energy-Mass Equivalence")
		sy.LaTeX(`E = mc^2`)
		sy.Caption("Euler's Identity")
		sy.LaTeX(`e^{i\pi} + 1 = 0`)
	})
	lcols[1](func() {
		sy.Caption("Gaussian Integral")
		sy.LaTeX(`\int_{-\infty}^{\infty} e^{-x^2}\,dx = \sqrt{\pi}`)
		sy.Caption("Quadratic Formula")
		sy.LaTeX(`x = \frac{-b \pm \sqrt{b^2 - 4ac}}{2a}`)
	})

	sy.Divider()

	sy.Header("JSON Viewer")
	sy.JSON(map[string]any{
		"project": "Syralit",
		"version": "0.1.0",
		"features": map[string]any{
			"widgets": 24,
			"charts":  14,
			"layouts": 12,
		},
		"dependencies": []string{
			"websocket", "goldmark", "toml", "fsnotify",
		},
	})

	sy.Divider()

	sy.Header("HTML (Raw)")
	sy.HTML(`<div style="padding:16px;border:2px solid #4CAF50;border-radius:8px;background:#f0fff0">
  <h3 style="margin:0 0 8px;color:#2E7D32">Custom HTML Block</h3>
  <p style="margin:0;color:#555">This is raw HTML rendered inside a Syralit app. You can embed any HTML content.</p>
</div>`)
}

// ═══════════════════════════════════════════════════════════════════════════
// Page 10: State & Cache
// ═══════════════════════════════════════════════════════════════════════════

func pageState() {
	sy.Title("State & Cache")

	sy.Header("Session State")
	sy.Caption("State persists across reruns within the same session.")

	counter := sy.State("mega_counter", 0)
	message := sy.State("mega_message", "")

	ccols := sy.Columns(3)
	ccols[0](func() {
		sy.Metric("Counter", fmt.Sprintf("%d", counter.Get()))
		if sy.Button("+1", sy.Key("s_inc")) {
			counter.Set(counter.Get() + 1)
		}
		if sy.Button("-1", sy.Key("s_dec")) {
			counter.Set(counter.Get() - 1)
		}
		if sy.Button("Reset", sy.Key("s_reset")) {
			counter.Set(0)
		}
	})
	ccols[1](func() {
		input := sy.TextInput("Set message", sy.Key("s_msg_input"))
		if sy.Button("Save", sy.Key("s_msg_save")) {
			message.Set(input)
			sy.Toast("Message saved!", "success")
		}
	})
	ccols[2](func() {
		sy.Subheader("Current State")
		sy.JSON(map[string]any{
			"counter": counter.Get(),
			"message": message.Get(),
		})
	})

	sy.Divider()

	sy.Header("Fragment (Partial Rerun)")
	sy.Caption("Only the fragment re-renders when its widgets change — not the whole page.")

	sy.Fragment("mega_fragment", func() {
		fcount := sy.State("mega_frag_count", 0)
		sy.Container(func() {
			sy.Subheader("Isolated Fragment")
			fcols := sy.Columns(2)
			fcols[0](func() {
				if sy.Button("Fragment +1", sy.Key("frag_inc")) {
					fcount.Set(fcount.Get() + 1)
				}
				if sy.Button("Fragment Reset", sy.Key("frag_reset")) {
					fcount.Set(0)
				}
			})
			fcols[1](func() {
				sy.Metric("Fragment Counter", fmt.Sprintf("%d", fcount.Get()))
			})
			sy.Caption("Clicking these buttons only re-renders this fragment, not the entire page.")
		}, sy.Border())
	})

	sy.Divider()

	sy.Header("CacheData")
	sy.Caption("Expensive computations are cached and reused across reruns.")

	expensive := sy.CacheData("mega_expensive", func() map[string]float64 {
		result := map[string]float64{}
		for i := 0; i < 1000; i++ {
			result[fmt.Sprintf("key_%d", i)] = math.Sin(float64(i)) * 100
		}
		return result
	})

	sy.Textf("Cached map has %d entries.", len(expensive))
	sy.Textf("Sample: key_42 = %.2f, key_100 = %.2f", expensive["key_42"], expensive["key_100"])

	sy.Divider()

	sy.Header("Render Info")
	sy.JSON(map[string]any{
		"rendered_at": time.Now().Format("2006-01-02 15:04:05"),
		"page":        "State & Cache",
		"session":     "active",
	})
}
