package insyradsl

import (
	"hash/fnv"
	"strconv"
	"strings"

	sy "github.com/HazelnutParadise/syralit"
)

// widgetConfig holds the render + input options for the DSL widget.
type widgetConfig struct {
	rs     renderSpec
	inject map[string]any
}

// WidgetOption configures the DSL widget.
type WidgetOption func(*widgetConfig)

// Render selects how the output is drawn: dataframe|table|line_chart|bar_chart|
// area_chart|pie_chart|metric|text. With no Render and no Output, DSL prints the
// script's textual transcript.
func Render(kind string) WidgetOption { return func(c *widgetConfig) { c.rs.kind = kind } }

// Output names the variable to render (default: $result, else the sole
// table/list the script produced).
func Output(name string) WidgetOption { return func(c *widgetConfig) { c.rs.output = name } }

// X sets the chart x-axis (label) column.
func X(col string) WidgetOption { return func(c *widgetConfig) { c.rs.x = col } }

// Y sets the chart y-axis (series) columns. Omit to chart every numeric column
// except X.
func Y(cols ...string) WidgetOption { return func(c *widgetConfig) { c.rs.y = cols } }

// Label sets the pie-chart label column.
func Label(col string) WidgetOption { return func(c *widgetConfig) { c.rs.label = col } }

// Value sets the pie-chart value column (or a literal metric value).
func Value(col string) WidgetOption { return func(c *widgetConfig) { c.rs.value = col } }

// MetricLabel sets the metric label.
func MetricLabel(label string) WidgetOption {
	return func(c *widgetConfig) { c.rs.metricLabel = label }
}

// Title sets the chart title.
func Title(title string) WidgetOption { return func(c *widgetConfig) { c.rs.title = title } }

// Height sets the chart/table height in pixels.
func Height(px int) WidgetOption { return func(c *widgetConfig) { c.rs.height = px } }

// Input injects variables into the script (e.g. an in-memory *insyra.DataTable).
// When Input is used the result is not cached, since the data may change between
// reruns.
func Input(vars map[string]any) WidgetOption { return func(c *widgetConfig) { c.inject = vars } }

// DSL runs an Insyra DSL script in safe mode and renders its result as a Syralit
// widget. Without Input, the result is cached by script hash so the enclosing
// closure's reruns don't recompute. With no render options it prints the
// script's textual transcript (handy for show/summary/mean).
//
//	syidsl.DSL(`
//	    newdl North South West as region
//	    newdl 12 18 9 as deals
//	    newdt region deals as t
//	`, syidsl.Render("bar_chart"), syidsl.Output("t"), syidsl.X("region"), syidsl.Y("deals"))
func DSL(script string, opts ...WidgetOption) {
	c := widgetConfig{}
	for _, o := range opts {
		o(&c)
	}

	var res DSLResult
	if c.inject != nil {
		res = RunDSL(script, WithVars(c.inject))
	} else {
		res = sy.CacheData("insyradsl:"+hashScript(script), func() DSLResult {
			return RunDSL(script)
		})
	}
	renderWidget(res, c.rs)
}

// renderWidget emits the DSL result as Syralit widgets, mirroring the Artifact
// component's node rendering.
func renderWidget(res DSLResult, rs renderSpec) {
	if res.Err != nil {
		sy.Error(res.Err.Error())
		return
	}

	// Default with no render intent: show the textual transcript.
	if rs.kind == "" && rs.output == "" {
		if strings.TrimSpace(res.Output) != "" {
			sy.Code(res.Output)
		}
		return
	}

	kind := rs.kind
	if kind == "" {
		kind = "dataframe"
	}

	if kind == "text" {
		text := res.Output
		if rs.output != "" {
			if v, _, ok := resolveOutputVar(res, rs.output); ok {
				if s, valued := scalarValue(v); valued {
					text = s
				}
			}
		}
		sy.Text(text)
		return
	}

	v, _, ok := resolveOutputVar(res, rs.output)
	if !ok {
		sy.Warning("insyra dsl: no renderable variable (name a var with 'as', or use Render(\"text\"))")
		return
	}

	switch kind {
	case "dataframe":
		headers, rows, tok := tableData(v)
		if !tok {
			sy.Warning("insyra dsl: output is not a table/list")
			return
		}
		if rs.height > 0 {
			sy.DataFrame(headers, rows, sy.Height(rs.height))
		} else {
			sy.DataFrame(headers, rows)
		}

	case "table":
		headers, rows, tok := tableData(v)
		if !tok {
			sy.Warning("insyra dsl: output is not a table/list")
			return
		}
		sy.Table(headers, stringRows(rows))

	case "line_chart", "bar_chart", "area_chart":
		xLabels, series, err := chartSeries(v, rs.x, rs.y)
		if err != nil {
			sy.Error("insyra dsl: " + err.Error())
			return
		}
		opts := chartOpts(rs, xLabels)
		switch kind {
		case "line_chart":
			sy.LineChart(series, opts...)
		case "bar_chart":
			sy.BarChart(series, opts...)
		case "area_chart":
			sy.AreaChart(series, opts...)
		}

	case "pie_chart":
		pie, err := pieData(v, rs.label, rs.value)
		if err != nil {
			sy.Error("insyra dsl: " + err.Error())
			return
		}
		var opts []sy.Option
		if rs.title != "" {
			opts = append(opts, sy.ChartTitle(rs.title))
		}
		if rs.height > 0 {
			opts = append(opts, sy.Height(rs.height))
		}
		sy.PieChart(pie, opts...)

	case "metric":
		value := rs.value
		if s, valued := scalarValue(v); valued {
			value = s
		}
		label := rs.metricLabel
		if label == "" {
			label = "Value"
		}
		sy.Metric(label, value)

	default:
		sy.Warning("insyra dsl: unsupported render " + kind)
	}
}

func chartOpts(rs renderSpec, xLabels []string) []sy.Option {
	var opts []sy.Option
	if len(xLabels) > 0 {
		opts = append(opts, sy.XLabels(xLabels))
	}
	if rs.title != "" {
		opts = append(opts, sy.ChartTitle(rs.title))
	}
	if rs.height > 0 {
		opts = append(opts, sy.Height(rs.height))
	}
	return opts
}

func hashScript(s string) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return strconv.FormatUint(h.Sum64(), 16)
}
