# Artifact DSL

`Artifact Canvas` is Syralit's safe, agent-updatable canvas surface. The DSL is
JSON-shaped and compiles into a curated subset of Syralit's internal node tree.
It is designed for dynamic UI generation without exposing raw HTML, custom JS,
iframes, or the internal `Node.Type/Props` protocol.

## Core Shape

```json
{
  "version": "v1",
  "layout": {
    "columns": 2,
    "gap": 14,
    "padding": 16
  },
  "data": {},
  "nodes": []
}
```

### Top-Level Fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `version` | `string` | yes | Currently `v1`. |
| `layout` | `object` | no | Canvas-wide responsive grid settings. |
| `data` | `object` | no | Arbitrary data bag used by `bind`. |
| `nodes` | `array` | yes | Ordered list of artifact nodes. |

### `layout`

| Field | Type | Required | Notes |
|---|---|---|---|
| `columns` | `number` | no | Grid column count. Defaults to `1`. |
| `gap` | `number` | no | Grid gap in pixels. |
| `padding` | `number` | no | Inner canvas padding in pixels. |
| `mode` | `string` | no | Reserved for future layout modes. |

## Nodes

Each entry in `nodes` is an `ArtifactNode`:

```json
{
  "id": "revenue",
  "component": "metric",
  "props": {
    "label": "Revenue"
  },
  "bind": {
    "props.value": "/summary/revenue"
  },
  "layout": {
    "column_span": 1,
    "row_span": 1
  },
  "children": []
}
```

### Node Fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `id` | `string` | yes | Must be stable and unique within the artifact. Animations rely on it. |
| `component` | `string` | yes | Must be one of the allowed components below. |
| `props` | `object` | no | Static props for the component. |
| `bind` | `object` | no | JSON Pointer bindings into `data`. |
| `layout` | `object` | no | Per-node grid sizing. |
| `children` | `array` | no | Only allowed on `container`. |

### Node Layout

| Field | Type | Required | Notes |
|---|---|---|---|
| `column_span` | `number` | no | Grid column span. |
| `row_span` | `number` | no | Grid row span. |

## Data Binding

`bind` copies values from `data` into direct props using JSON Pointer.

Example:

```json
{
  "data": {
    "summary": {
      "revenue": "$42k",
      "delta": "+9%"
    }
  },
  "nodes": [
    {
      "id": "revenue",
      "component": "metric",
      "props": {
        "label": "Revenue"
      },
      "bind": {
        "props.value": "/summary/revenue",
        "props.delta": "/summary/delta"
      }
    }
  ]
}
```

### Binding Rules

- Binding targets must start with `props.`
- Bindings can only write one direct prop at a time, e.g. `props.value`
- JSON Pointer must start with `/`
- Arrays are addressed by numeric index, e.g. `/rows/0/1`
- If a pointer is missing or crosses a non-container value, validation fails

## Allowed Components

Only these components are currently accepted:

| Component | Typical Use |
|---|---|
| `text` | Plain text paragraph |
| `markdown` | Rich text from Markdown |
| `metric` | Headline KPI with optional delta |
| `table` | Static table |
| `dataframe` | Rich read-only data grid |
| `line_chart` | Multi-series line chart |
| `bar_chart` | Multi-series bar chart |
| `pie_chart` | Pie chart |
| `image` | Image with optional caption |
| `progress` | Progress bar |
| `container` | Group children and/or draw a bordered panel |

## Allowed Props by Component

### `text`

```json
{ "text": "Hello" }
```

Allowed props: `text`

### `markdown`

```json
{ "text": "## Heading\nBody copy" }
```

Allowed props: `text`

### `metric`

```json
{
  "label": "Revenue",
  "value": "$42k",
  "delta": "+9%",
  "delta_color": "normal",
  "border": true,
  "help": "Month to date"
}
```

Allowed props: `label`, `value`, `delta`, `delta_color`, `border`, `help`

### `table`

```json
{
  "headers": ["Region", "Deals"],
  "rows": [
    ["North", 12],
    ["South", 18]
  ]
}
```

Allowed props: `headers`, `rows`

### `dataframe`

```json
{
  "headers": ["Name", "Score"],
  "rows": [
    ["Ada", 98],
    ["Linus", 94]
  ],
  "height": 320
}
```

Allowed props: `headers`, `rows`, `height`, `column_config`

### `line_chart`

```json
{
  "series": {
    "Revenue": [10, 20, 30],
    "Cost": [6, 9, 12]
  },
  "title": "Trend",
  "x_labels": ["Jan", "Feb", "Mar"],
  "height": 300
}
```

Allowed props: `series`, `height`, `width`, `title`, `x_labels`, `stacked`,
`colors`

### `bar_chart`

```json
{
  "series": {
    "Deals": [12, 18, 9]
  },
  "x_labels": ["North", "South", "West"],
  "horizontal": true
}
```

Allowed props: `series`, `height`, `width`, `title`, `x_labels`,
`horizontal`, `stacked`, `colors`

### `pie_chart`

```json
{
  "data": {
    "Direct": 52,
    "Partner": 31,
    "Other": 17
  },
  "title": "Channel mix"
}
```

Allowed props: `data`, `height`, `width`, `title`, `colors`

### `image`

```json
{
  "src": "https://example.com/chart.png",
  "alt": "Chart preview",
  "caption": "Latest snapshot"
}
```

Allowed props: `src`, `alt`, `width`, `caption`, `containerWidth`

### `progress`

```json
{
  "value": 0.72,
  "text": "Plan confidence"
}
```

Allowed props: `value`, `text`

### `container`

```json
{
  "border": true,
  "height": 240
}
```

Allowed props: `border`, `height`

`container` is the only component that may contain `children`.

## Validation Rules

Validation fails when any of these happen:

- missing `id`
- duplicate `id`
- unsupported `component`
- unsupported prop for the chosen component
- invalid JSON Pointer
- non-`container` node with `children`
- missing required props such as `metric.label`, `metric.value`, `image.src`

## Example

```json
{
  "version": "v1",
  "layout": {
    "columns": 2,
    "gap": 14,
    "padding": 16
  },
  "data": {
    "summary": {
      "revenue": "$58k",
      "delta": "+18%",
      "rows": [
        ["North", 22],
        ["South", 15],
        ["West", 19]
      ]
    }
  },
  "nodes": [
    {
      "id": "title",
      "component": "markdown",
      "props": {
        "text": "## Agent-updated board\nGenerated from a safe DSL."
      },
      "layout": {
        "column_span": 2
      }
    },
    {
      "id": "revenue",
      "component": "metric",
      "props": {
        "label": "Revenue"
      },
      "bind": {
        "props.value": "/summary/revenue",
        "props.delta": "/summary/delta"
      }
    },
    {
      "id": "progress",
      "component": "progress",
      "props": {
        "value": 0.72,
        "text": "Plan confidence"
      }
    },
    {
      "id": "table",
      "component": "table",
      "props": {
        "headers": ["Region", "Deals"]
      },
      "bind": {
        "props.rows": "/summary/rows"
      },
      "layout": {
        "column_span": 2
      }
    }
  ]
}
```

## Endpoint Usage

Apps opt in to an update endpoint explicitly:

```go
board := sy.NewArtifactStore("main", initialSpec)

sy.HandleArtifactEndpoint(
    "/api/agent/artifacts/main",
    board,
    sy.StaticAgentKey("local-agent", sy.Secrets("AGENT_KEY")),
)

sy.App(func() {
    sy.ArtifactCanvas(board, sy.Height(520))
})
```

Update payload:

```json
{
  "spec": {
    "version": "v1",
    "nodes": [
      {
        "id": "msg",
        "component": "text",
        "props": {
          "text": "Updated"
        }
      }
    ]
  }
}
```

## Non-Goals

This DSL is intentionally not:

- a raw HTML transport
- a custom component SDK
- a freeform script container
- a drag-and-drop whiteboard document model
- the public form of Syralit's internal `Node` protocol
