package syralit

// LineChart renders a line chart from named series.
//
//	sy.LineChart(map[string][]float64{
//	    "Revenue": {10, 20, 30, 25, 35},
//	    "Cost":    {5, 8, 12, 10, 15},
//	})
func LineChart(data map[string][]float64, opts ...Option) {
	o := applyOpts(opts)
	props := map[string]any{"series": data}
	if o.height > 0 {
		props["height"] = o.height
	}
	if o.width > 0 {
		props["width"] = o.width
	}
	current().add(&Node{Type: "line_chart", Props: props})
}

// BarChart renders a bar chart from named series.
func BarChart(data map[string][]float64, opts ...Option) {
	o := applyOpts(opts)
	props := map[string]any{"series": data}
	if o.height > 0 {
		props["height"] = o.height
	}
	if o.width > 0 {
		props["width"] = o.width
	}
	current().add(&Node{Type: "bar_chart", Props: props})
}

// AreaChart renders an area chart (filled line chart) from named series.
func AreaChart(data map[string][]float64, opts ...Option) {
	o := applyOpts(opts)
	props := map[string]any{"series": data}
	if o.height > 0 {
		props["height"] = o.height
	}
	if o.width > 0 {
		props["width"] = o.width
	}
	current().add(&Node{Type: "area_chart", Props: props})
}
