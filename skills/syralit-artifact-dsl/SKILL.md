---
name: syralit-artifact-dsl
description: Generate valid Syralit Artifact DSL for agent-updatable canvases. Use when an AI agent must output JSON for `ArtifactSpec`, call an artifact endpoint, or transform structured data into safe reusable Syralit components.
---

# Syralit Artifact DSL

Use this skill when the task is not "build a whole Syralit app", but "produce a
valid Artifact DSL payload" for `ArtifactCanvas` / `HandleArtifactEndpoint`.

This skill is narrower than `syralit-dev` on purpose. It should help an agent
generate safe, valid JSON instead of freeform UI descriptions.

## Agent Connection Workflow

When an endpoint URL and bearer token are available, do not ask the user for
canvas names. Discover them:

```http
GET /api/agent/artifacts
Authorization: Bearer <token>
```

The response lists every store explicitly exposed by the app:

```json
{
  "artifacts": [
    {
      "id": "main",
      "revision": 3,
      "placements": [
        {
          "page": "Home",
          "url": "http://localhost:8600/",
          "canvas_id": "artifact:main",
          "selector": "[data-artifact-id=\"artifact:main\"]"
        }
      ]
    }
  ]
}
```

If a multi-page app has not rendered every page yet, open top-level `app_url`,
visit its navigation pages once, and repeat discovery. Placement metadata comes
from actual renders so it does not invent pages by executing application code
inside the API request. A separately mounted API may omit placement URLs because
it cannot safely infer the Syralit UI origin; use the app URL supplied by the
app owner instead of guessing from the API port.

Before replacing an existing artifact, read its current DSL:

```http
GET /api/agent/artifacts?artifact=main
Authorization: Bearer <token>
```

Update through the same unified URL:

```json
{
  "artifact": "main",
  "expected_revision": 3,
  "spec": {
    "version": "v1",
    "nodes": []
  }
}
```

Use the `artifact` ID as the update target. Never send or invent a CSS selector
to choose what the API modifies. Selectors returned by the server are read-only
preview locators.

`expected_revision` is required and must come from the latest discovery/detail
response. On `409 revision_conflict`, fetch the current spec again and merge the
user's intended change into that version. Never retry a stale full replacement
unchanged.

After a successful update:

1. Read `revision` and `preview` from the response.
2. Open `preview.url` and navigate to `preview.page` when present.
3. Locate `preview.selector`.
4. Wait until `data-artifact-revision` equals the returned revision.
5. Wait until `data-artifact-state` is `settled`.
6. Require `data-artifact-readiness="complete"` for full visual verification.
   Report `partial` (failed image/chart/font) or `timeout` honestly.
7. Capture that element and return the screenshot with the result.
8. Inspect the screenshot. If content overlaps, clips, fails to render, or does
   not answer the user's request, revise the DSL and repeat the cycle.

Do not claim visual verification when no browser/screenshot capability is
available. In that case, report that the API update succeeded but visual
verification was not performed.

Per-store endpoints created with `HandleArtifactEndpoint` use
`GET/POST /configured/path` and omit the top-level `artifact` field. The app may
also mount `ArtifactAPIHandler` or `ArtifactHandler` on a different port; always
use the URL supplied by the app or discovery response instead of assuming port
8600.

## What This DSL Is

Artifact DSL is a JSON-shaped spec that compiles into a controlled subset of
Syralit UI components. It is designed for live, agent-driven updates.

It is **not**:

- raw HTML
- a JavaScript execution surface
- a custom component system
- the internal `Node` protocol
- a drag-and-drop whiteboard document model

## Output Contract

Always produce JSON matching this shape:

```json
{
  "version": "v1",
  "layout": {
    "columns": 1,
    "gap": 12,
    "padding": 16
  },
  "data": {},
  "nodes": []
}
```

When returning a unified API payload, wrap it as:

```json
{
  "artifact": "main",
  "expected_revision": 1,
  "spec": {
    "version": "v1",
    "nodes": []
  }
}
```

## Required Rules

1. Every node must have a stable, unique string `id`.
2. `component` must be one of the allowlisted components below.
3. Only `container` may have `children`.
4. `bind` targets must start with `props.` and point to one direct prop only.
5. JSON Pointer values in `bind` must start with `/`.
6. Do not invent props outside the component's allowlist.
7. Do not output HTML, JS, iframe URLs, or script-like content.

## Allowed Components

Only use:

- `text`
- `markdown`
- `metric`
- `table`
- `dataframe`
- `line_chart`
- `bar_chart`
- `pie_chart`
- `image`
- `progress`
- `container`

## Allowed Props

### `text`

Allowed props: `text`

Example:

```json
{
  "id": "intro",
  "component": "text",
  "props": {
    "text": "Hello"
  }
}
```

### `markdown`

Allowed props: `text`

Example:

```json
{
  "id": "headline",
  "component": "markdown",
  "props": {
    "text": "## Weekly summary\nRevenue improved this week."
  }
}
```

### `metric`

Allowed props: `label`, `value`, `delta`, `delta_color`, `border`, `help`

Minimum useful shape:

```json
{
  "id": "revenue",
  "component": "metric",
  "props": {
    "label": "Revenue",
    "value": "$42k"
  }
}
```

### `table`

Allowed props: `headers`, `rows`

### `dataframe`

Allowed props: `headers`, `rows`, `height`, `column_config`

### `line_chart`

Allowed props: `series`, `height`, `width`, `title`, `x_labels`, `stacked`,
`colors`

### `bar_chart`

Allowed props: `series`, `height`, `width`, `title`, `x_labels`,
`horizontal`, `stacked`, `colors`

### `pie_chart`

Allowed props: `data`, `height`, `width`, `title`, `colors`

### `image`

Allowed props: `src`, `alt`, `width`, `caption`, `containerWidth`

### `progress`

Allowed props: `value`, `text`

### `container`

Allowed props: `border`, `height`

`container` may also include `children`.

## Layout Rules

Top-level `layout` controls the whole canvas:

```json
{
  "layout": {
    "columns": 2,
    "gap": 14,
    "padding": 16
  }
}
```

Per-node `layout` controls grid spans:

```json
{
  "layout": {
    "column_span": 2,
    "row_span": 1
  }
}
```

## Binding Rules

Use `bind` when UI values should come from `data`:

```json
{
  "data": {
    "summary": {
      "revenue": "$58k",
      "delta": "+18%"
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

Use static `props` for fixed display text and `bind` for dynamic values.

## Preferred Patterns

### KPI Board

```json
{
  "version": "v1",
  "layout": {
    "columns": 3,
    "gap": 14,
    "padding": 16
  },
  "data": {
    "summary": {
      "revenue": "$58k",
      "margin": "32%",
      "deals": "19"
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
        "props.value": "/summary/revenue"
      }
    },
    {
      "id": "margin",
      "component": "metric",
      "props": {
        "label": "Margin"
      },
      "bind": {
        "props.value": "/summary/margin"
      }
    },
    {
      "id": "deals",
      "component": "metric",
      "props": {
        "label": "Deals"
      },
      "bind": {
        "props.value": "/summary/deals"
      }
    }
  ]
}
```

### Narrative + Table

```json
{
  "version": "v1",
  "layout": {
    "columns": 2,
    "gap": 14,
    "padding": 16
  },
  "data": {
    "report": {
      "rows": [
        ["North", 22],
        ["South", 15]
      ]
    }
  },
  "nodes": [
    {
      "id": "headline",
      "component": "markdown",
      "props": {
        "text": "## Pipeline update\nNorth is leading this week."
      },
      "layout": {
        "column_span": 2
      }
    },
    {
      "id": "table",
      "component": "table",
      "props": {
        "headers": ["Region", "Deals"]
      },
      "bind": {
        "props.rows": "/report/rows"
      },
      "layout": {
        "column_span": 2
      }
    }
  ]
}
```

## Things To Avoid

Do not output:

- `component: "html"`
- `component: "component"`
- `component: "iframe"`
- props like `onClick`, `script`, `style`, `className`
- unstable ids like random UUIDs generated per request unless the node is truly new
- prose like "put a nice chart here" instead of real JSON

## Response Style

When the task is "give me the DSL", output only:

- the JSON spec, or
- the wrapped endpoint payload JSON

When the task includes applying the DSL, also perform discovery, update,
settled-state waiting, screenshot capture, and visual inspection as described
above.

Keep explanation short unless the user explicitly asks for reasoning.

## Cross-Reference

For the framework API, examples, and endpoint registration, also use
`skills/syralit-dev/SKILL.md`.
