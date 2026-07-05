package syralit

// ChartSelection is the point a user clicked on a selectable chart (see
// sy.Selectable() on LineChart/BarChart/AreaChart/ScatterChart/PieChart).
// Series is the series (or pie slice) name, Index the point's position along
// the x axis (or slice index), X its label, and Value the y (or slice) value.
// The selection persists across reruns until the user clicks another point.
type ChartSelection struct {
	Series string
	Index  int
	X      string
	Value  float64
}

// chartSelection wires a selectable chart's widget ID into props and reads the
// stored click. Returns ("", nil) when the chart is not selectable.
func chartSelection(o widgetOpts, typ string, props map[string]any) (string, *ChartSelection) {
	if !o.selectable {
		return "", nil
	}
	rc := current()
	id := rc.widgetID(typ, o.key)
	props["selectable"] = true
	val, _ := rc.sess.widgetValue(id)
	m, ok := val.(map[string]any)
	if !ok {
		return id, nil
	}
	sel := &ChartSelection{
		Index: int(toFloat64(m["index"])),
		Value: toFloat64(m["value"]),
	}
	sel.Series, _ = m["series"].(string)
	sel.X, _ = m["x"].(string)
	return id, sel
}

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
	if o.horizontal {
		props["horizontal"] = true
	}
	if o.stacked {
		props["stacked"] = true
	}
	if len(o.colors) > 0 {
		props["colors"] = o.colors
	}
	return props
}

// LineChart renders a line chart from named series.
//
//	sy.LineChart(map[string][]float64{
//	    "Revenue": {10, 20, 30, 25, 35},
//	    "Cost":    {5, 8, 12, 10, 15},
//	})
//
// With sy.Selectable(), the chart reports the clicked point (nil until then).
func LineChart(data map[string][]float64, opts ...Option) *ChartSelection {
	o := applyOpts(opts)
	props := chartProps(o, data)
	id, sel := chartSelection(o, "line_chart", props)
	current().add(&Node{ID: id, Type: "line_chart", Props: props})
	return sel
}

// BarChart renders a bar chart from named series. With sy.Selectable(), the
// chart reports the clicked bar (nil until then).
func BarChart(data map[string][]float64, opts ...Option) *ChartSelection {
	o := applyOpts(opts)
	props := chartProps(o, data)
	id, sel := chartSelection(o, "bar_chart", props)
	current().add(&Node{ID: id, Type: "bar_chart", Props: props})
	return sel
}

// ScatterChart renders a scatter plot. Each series maps to a slice of [x, y]
// coordinate pairs.
//
//	sy.ScatterChart(map[string][][2]float64{
//	    "Group A": {{1, 2}, {3, 4}, {5, 6}},
//	})
func ScatterChart(data map[string][][2]float64, opts ...Option) *ChartSelection {
	o := applyOpts(opts)
	series := make(map[string]any, len(data))
	for name, points := range data {
		ps := make([][]float64, len(points))
		for i, p := range points {
			ps[i] = []float64{p[0], p[1]}
		}
		series[name] = ps
	}
	props := chartProps(o, series)
	id, sel := chartSelection(o, "scatter_chart", props)
	current().add(&Node{ID: id, Type: "scatter_chart", Props: props})
	return sel
}

// PieChart renders a pie chart from labelled values.
//
//	sy.PieChart(map[string]float64{
//	    "Go": 45,
//	    "Python": 35,
//	    "Rust": 20,
//	})
func PieChart(data map[string]float64, opts ...Option) *ChartSelection {
	o := applyOpts(opts)
	props := chartProps(o, nil)
	props["data"] = data
	delete(props, "series")
	id, sel := chartSelection(o, "pie_chart", props)
	current().add(&Node{ID: id, Type: "pie_chart", Props: props})
	return sel
}

// AreaChart renders an area chart (filled line chart) from named series. With
// sy.Selectable(), the chart reports the clicked point (nil until then).
func AreaChart(data map[string][]float64, opts ...Option) *ChartSelection {
	o := applyOpts(opts)
	props := chartProps(o, data)
	id, sel := chartSelection(o, "area_chart", props)
	current().add(&Node{ID: id, Type: "area_chart", Props: props})
	return sel
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

// VegaLiteChart renders a Vega-Lite chart from a spec. The spec is a
// map[string]any matching the Vega-Lite JSON schema. The chart is rendered
// client-side using the Vega-Lite JavaScript library (CDN).
//
// This is the Go equivalent of Streamlit's st.altair_chart — Altair is a
// Python API that generates Vega-Lite JSON specs, so accepting the spec
// directly provides the same capability.
//
//	sy.VegaLiteChart(map[string]any{
//	    "mark": "bar",
//	    "encoding": map[string]any{
//	        "x": map[string]any{"field": "category", "type": "nominal"},
//	        "y": map[string]any{"field": "value", "type": "quantitative"},
//	    },
//	    "data": map[string]any{
//	        "values": []map[string]any{
//	            {"category": "A", "value": 28},
//	            {"category": "B", "value": 55},
//	        },
//	    },
//	})
func VegaLiteChart(spec map[string]any, opts ...Option) {
	o := applyOpts(opts)
	props := map[string]any{"spec": spec}
	if o.height > 0 {
		props["height"] = o.height
	}
	if o.width > 0 {
		props["width"] = o.width
	}
	if o.title != "" {
		props["title"] = o.title
	}
	current().add(&Node{Type: "vega_lite_chart", Props: props})
}

// PlotlyChart renders a Plotly chart from a figure spec. The spec is a
// map[string]any matching the Plotly JSON schema (data + layout). Rendered
// client-side using Plotly.js (CDN).
//
// This is the Go equivalent of Streamlit's st.plotly_chart — Plotly figures
// in Python serialize to JSON, so accepting the JSON spec directly provides
// the same capability.
//
//	sy.PlotlyChart(map[string]any{
//	    "data": []map[string]any{{
//	        "x": []string{"giraffes", "orangutans", "monkeys"},
//	        "y": []float64{20, 14, 23},
//	        "type": "bar",
//	    }},
//	    "layout": map[string]any{"title": "Animals"},
//	})
func PlotlyChart(spec map[string]any, opts ...Option) {
	o := applyOpts(opts)
	props := map[string]any{"spec": spec}
	if o.height > 0 {
		props["height"] = o.height
	}
	if o.width > 0 {
		props["width"] = o.width
	}
	current().add(&Node{Type: "plotly_chart", Props: props})
}

// PyplotChart renders a static chart image from raw PNG/SVG bytes. This is
// the Go equivalent of Streamlit's st.pyplot — in Python, matplotlib figures
// are exported as images. In Go, any charting library that can export PNG or
// SVG can be used.
//
// The format is auto-detected: if data starts with "<svg" it is treated as
// inline SVG; otherwise it is treated as a base64-encoded PNG data URI.
//
//	// From SVG string:
//	sy.PyplotChart(svgString)
//	// From PNG bytes:
//	sy.PyplotChart(base64.StdEncoding.EncodeToString(pngBytes))
func PyplotChart(data string, opts ...Option) {
	o := applyOpts(opts)
	props := map[string]any{"data": data}
	if o.height > 0 {
		props["height"] = o.height
	}
	if o.width > 0 {
		props["width"] = o.width
	}
	if o.caption != "" {
		props["caption"] = o.caption
	}
	current().add(&Node{Type: "pyplot_chart", Props: props})
}

// BokehChart renders a Bokeh chart from a JSON spec. The spec is a
// map[string]any matching Bokeh's JSON document format. Rendered client-side
// using BokehJS (CDN).
//
// This is the Go equivalent of Streamlit's st.bokeh_chart.
func BokehChart(spec map[string]any, opts ...Option) {
	o := applyOpts(opts)
	props := map[string]any{"spec": spec}
	if o.height > 0 {
		props["height"] = o.height
	}
	if o.width > 0 {
		props["width"] = o.width
	}
	current().add(&Node{Type: "bokeh_chart", Props: props})
}

// PydeckChart renders a deck.gl 3D map visualization from a JSON spec.
// The spec is a map[string]any matching deck.gl's JSON schema. Rendered
// client-side using deck.gl (CDN).
//
// This is the Go equivalent of Streamlit's st.pydeck_chart — PyDeck is a
// Python API that generates deck.gl JSON specs.
//
//	sy.PydeckChart(map[string]any{
//	    "initialViewState": map[string]any{
//	        "latitude": 37.76, "longitude": -122.4,
//	        "zoom": 11, "pitch": 50,
//	    },
//	    "layers": []map[string]any{{
//	        "@@type": "HexagonLayer",
//	        "data": "https://raw.githubusercontent.com/visgl/deck.gl-data/master/website/sf-bike-parking.json",
//	        "getPosition": "@@=[lng, lat]",
//	        "radius": 200,
//	        "elevationScale": 4,
//	        "extruded": true,
//	    }},
//	})
func PydeckChart(spec map[string]any, opts ...Option) {
	o := applyOpts(opts)
	props := map[string]any{"spec": spec}
	if o.height > 0 {
		props["height"] = o.height
	}
	if o.width > 0 {
		props["width"] = o.width
	}
	current().add(&Node{Type: "pydeck_chart", Props: props})
}
