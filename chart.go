package syralit

func chartProps(o widgetOpts, data any) map[string]any {
	props := map[string]any{"series": data}
	if o.height > 0 {
		props["height"] = o.height
	}
	if o.width > 0 {
		props["width"] = o.width
	}
	if o.title != "" {
		props["title"] = o.title
	}
	if len(o.xLabels) > 0 {
		props["x_labels"] = o.xLabels
	}
	return props
}

// LineChart renders a line chart from named series.
//
//	sy.LineChart(map[string][]float64{
//	    "Revenue": {10, 20, 30, 25, 35},
//	    "Cost":    {5, 8, 12, 10, 15},
//	})
func LineChart(data map[string][]float64, opts ...Option) {
	o := applyOpts(opts)
	current().add(&Node{Type: "line_chart", Props: chartProps(o, data)})
}

// BarChart renders a bar chart from named series.
func BarChart(data map[string][]float64, opts ...Option) {
	o := applyOpts(opts)
	current().add(&Node{Type: "bar_chart", Props: chartProps(o, data)})
}

// ScatterChart renders a scatter plot. Each series maps to a slice of [x, y]
// coordinate pairs.
//
//	sy.ScatterChart(map[string][][2]float64{
//	    "Group A": {{1, 2}, {3, 4}, {5, 6}},
//	})
func ScatterChart(data map[string][][2]float64, opts ...Option) {
	o := applyOpts(opts)
	series := make(map[string]any, len(data))
	for name, points := range data {
		ps := make([][]float64, len(points))
		for i, p := range points {
			ps[i] = []float64{p[0], p[1]}
		}
		series[name] = ps
	}
	current().add(&Node{Type: "scatter_chart", Props: chartProps(o, series)})
}

// PieChart renders a pie chart from labelled values.
//
//	sy.PieChart(map[string]float64{
//	    "Go": 45,
//	    "Python": 35,
//	    "Rust": 20,
//	})
func PieChart(data map[string]float64, opts ...Option) {
	o := applyOpts(opts)
	props := chartProps(o, nil)
	props["data"] = data
	delete(props, "series")
	current().add(&Node{Type: "pie_chart", Props: props})
}

// AreaChart renders an area chart (filled line chart) from named series.
func AreaChart(data map[string][]float64, opts ...Option) {
	o := applyOpts(opts)
	current().add(&Node{Type: "area_chart", Props: chartProps(o, data)})
}

// HistogramChart renders a histogram from raw data values. The bins parameter
// sets the number of bins (defaults to 10 if zero).
func HistogramChart(data []float64, bins int, opts ...Option) {
	if bins <= 0 {
		bins = 10
	}
	o := applyOpts(opts)
	props := chartProps(o, nil)
	props["data"] = data
	props["bins"] = bins
	delete(props, "series")
	current().add(&Node{Type: "histogram_chart", Props: props})
}

// DoughnutChart renders a doughnut chart (pie with hole). Labels map to values.
func DoughnutChart(data map[string]float64, opts ...Option) {
	o := applyOpts(opts)
	props := chartProps(o, nil)
	props["data"] = data
	delete(props, "series")
	current().add(&Node{Type: "doughnut_chart", Props: props})
}

// RadarChart renders a radar/spider chart from named series.
func RadarChart(labels []string, data map[string][]float64, opts ...Option) {
	o := applyOpts(opts)
	props := chartProps(o, data)
	props["labels"] = labels
	current().add(&Node{Type: "radar_chart", Props: props})
}

// GraphvizChart renders a Graphviz DOT graph using viz.js (CDN).
//
//	sy.GraphvizChart(`digraph { A -> B -> C; B -> D; }`)
func GraphvizChart(dot string, opts ...Option) {
	o := applyOpts(opts)
	props := map[string]any{"dot": dot}
	if o.height > 0 {
		props["height"] = o.height
	}
	if o.width > 0 {
		props["width"] = o.width
	}
	current().add(&Node{Type: "graphviz_chart", Props: props})
}
