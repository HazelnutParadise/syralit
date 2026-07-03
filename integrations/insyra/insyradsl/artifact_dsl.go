package insyradsl

import (
	"fmt"
	"time"

	sy "github.com/HazelnutParadise/syralit"
)

// Limits for agent-supplied artifact scripts. Artifact specs come from
// (possibly untrusted) agents over the network, so the "insyra" component runs
// under tighter caps than a hand-written Go RunDSL call.
const (
	maxArtifactScriptBytes = 16 << 10 // 16 KiB
	maxArtifactScriptLines = 200
	artifactScriptTimeout  = 10 * time.Second
)

// artifactRenderKinds is the closed set of render targets the "insyra"
// component may produce. Each maps to a Node type the client already renders.
var artifactRenderKinds = map[string]struct{}{
	"dataframe": {}, "table": {}, "line_chart": {}, "bar_chart": {},
	"area_chart": {}, "pie_chart": {}, "metric": {}, "text": {},
}

func init() {
	sy.RegisterArtifactComponent("insyra", compileInsyraArtifactNode)
}

// compileInsyraArtifactNode runs an agent-supplied Insyra DSL script in safe
// mode and renders its output as one Artifact Canvas node. The script computes
// dynamically on the server the moment the artifact is set; the result is baked
// into the node tree that is broadcast to browsers.
//
// Recognised props:
//
//	script   (required) the Insyra DSL (.isr) source
//	render   dataframe|table|line_chart|bar_chart|area_chart|pie_chart|metric|text (default dataframe)
//	output   variable name to render (default: $result, else the sole table/list)
//	x        chart x-axis column (labels)
//	y        chart y-axis column(s): a string, comma-separated string, or array
//	label    pie label column
//	value    pie value column, or a literal metric value
//	metric_label  metric label
//	title    chart title
//	height   chart/table height in px
func compileInsyraArtifactNode(node sy.ArtifactNode, data map[string]any) (*sy.Node, error) {
	script := propString(node.Props, "script")
	if script == "" {
		return nil, fmt.Errorf("insyra: prop %q is required", "script")
	}
	if len(script) > maxArtifactScriptBytes {
		return nil, fmt.Errorf("insyra: script is %d bytes, exceeds limit of %d", len(script), maxArtifactScriptBytes)
	}

	kind := propString(node.Props, "render")
	if kind == "" {
		kind = "dataframe"
	}
	if _, ok := artifactRenderKinds[kind]; !ok {
		return nil, fmt.Errorf("insyra: unsupported render %q", kind)
	}

	rs := renderSpec{
		kind:        kind,
		output:      propString(node.Props, "output"),
		x:           propString(node.Props, "x"),
		y:           propStringSlice(node.Props, "y"),
		label:       propString(node.Props, "label"),
		value:       propString(node.Props, "value"),
		title:       propString(node.Props, "title"),
		metricLabel: propString(node.Props, "metric_label"),
		height:      propInt(node.Props, "height"),
	}

	res := RunDSL(script, DSLTimeout(artifactScriptTimeout), MaxLines(maxArtifactScriptLines))
	if res.Err != nil {
		return nil, res.Err
	}

	return renderArtifactNode(res, rs)
}

// renderArtifactNode converts a DSL result into a single render Node per the
// render spec.
func renderArtifactNode(res DSLResult, rs renderSpec) (*sy.Node, error) {
	if rs.kind == "text" {
		text := res.Output
		if rs.output != "" {
			if v, _, ok := resolveOutputVar(res, rs.output); ok {
				if s, valued := scalarValue(v); valued {
					text = s
				}
			}
		}
		return &sy.Node{Type: "text", Props: map[string]any{"text": text}}, nil
	}

	v, name, ok := resolveOutputVar(res, rs.output)
	if !ok {
		if rs.output != "" {
			return nil, fmt.Errorf("insyra: output variable %q not found", rs.output)
		}
		return nil, fmt.Errorf("insyra: no renderable variable produced (name a var with 'as', or use render \"text\")")
	}
	_ = name

	switch rs.kind {
	case "dataframe":
		headers, rows, tok := tableData(v)
		if !tok {
			return nil, fmt.Errorf("insyra: output is not a table/list")
		}
		props := map[string]any{"headers": headers, "rows": rows}
		if rs.height > 0 {
			props["height"] = rs.height
		}
		return &sy.Node{Type: "dataframe", Props: props}, nil

	case "table":
		headers, rows, tok := tableData(v)
		if !tok {
			return nil, fmt.Errorf("insyra: output is not a table/list")
		}
		return &sy.Node{Type: "table", Props: map[string]any{"headers": headers, "rows": stringRows(rows)}}, nil

	case "line_chart", "bar_chart", "area_chart":
		xLabels, series, err := chartSeries(v, rs.x, rs.y)
		if err != nil {
			return nil, fmt.Errorf("insyra: %w", err)
		}
		props := map[string]any{"series": series}
		if len(xLabels) > 0 {
			props["x_labels"] = xLabels
		}
		if rs.title != "" {
			props["title"] = rs.title
		}
		if rs.height > 0 {
			props["height"] = rs.height
		}
		return &sy.Node{Type: rs.kind, Props: props}, nil

	case "pie_chart":
		pie, err := pieData(v, rs.label, rs.value)
		if err != nil {
			return nil, fmt.Errorf("insyra: %w", err)
		}
		props := map[string]any{"data": pie}
		if rs.title != "" {
			props["title"] = rs.title
		}
		if rs.height > 0 {
			props["height"] = rs.height
		}
		return &sy.Node{Type: "pie_chart", Props: props}, nil

	case "metric":
		value := rs.value
		if s, valued := scalarValue(v); valued {
			value = s
		}
		if value == "" {
			return nil, fmt.Errorf("insyra: metric needs a scalar output or a value prop")
		}
		label := rs.metricLabel
		if label == "" {
			label = "Value"
		}
		return &sy.Node{Type: "metric", Props: map[string]any{"label": label, "value": value}}, nil
	}

	return nil, fmt.Errorf("insyra: unsupported render %q", rs.kind)
}
