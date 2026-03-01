# rendspec

A CLI tool and MCP server that renders visual designs from a compact YAML DSL. Produces SVG and PNG — no browser, no coordinate math, no external dependencies.

Designed to be used by AI agents (via MCP) or humans (via CLI).

## Install

```bash
go install github.com/fxck/rendspec/cmd/rendspec@latest
go install github.com/fxck/rendspec/cmd/rendspec-mcp@latest
```

Or build from source:

```bash
make build    # outputs to bin/
```

## Quick Start

```bash
rendspec render design.rds                # SVG to stdout
rendspec render design.rds -o out.svg     # SVG to file
rendspec render design.rds -o out.png     # PNG (uses resvg via wasm)
rendspec validate design.rds              # parse + layout check
rendspec inspect design.rds               # dump layout tree as JSON
rendspec handover design.rds -o out.md    # generate developer handover doc
```

## Example

```yaml
theme: dark

tokens:
  color:
    primary: "#3b82f6"
    bg: "#0f172a"

components:
  chip:
    fill: "#1e293b"
    radius: 20
    padding: 6 16
    border: 1 solid #334155
    - text: "Default"
      font: 500 12 Inter

root:
  width: 600
  height: 400
  fill: $color.bg
  padding: 32
  gap: 16

  - text: "Hello, rendspec"
    font: 700 28 Inter
    color: $color.primary

  - frame: row
    direction: row
    gap: 8
    - chip: "Design"
    - chip: "Tokens"
    - chip: "Components"
```

## DSL Features

- **Flexbox & grid layout** — `direction`, `align`, `justify`, `gap`, `flex`, `columns`
- **Design tokens** — define once, reference with `$token.path`
- **Reusable components** — `components:` + `use:` with params and variants
- **Gradients** — `fill: linear-gradient(...)` and `radial-gradient(...)`
- **Edges/connections** — arrows between named frames with labels and routing
- **Multi-page documents** — `pages:` for multi-screen designs
- **Themes** — built-in (`light`, `dark`, `blueprint`, `sketch`) or custom
- **Shapes** — `rect`, `circle`, `ellipse`, `diamond`
- **Z-index & absolute positioning** — layered compositions
- **Shadows, borders, opacity, clipping** — standard visual properties
- **Auto-quoting preprocessor** — write `padding: 12 24` without quotes
- **Implicit children** — no need to write `children:`

See [`CLAUDE.md`](CLAUDE.md) for the full DSL reference.

## MCP Server

`rendspec-mcp` exposes four tools over MCP stdio:

| Tool | Description |
|---|---|
| `render_design` | Render `.rds` source to SVG |
| `validate_design` | Validate source, return stats |
| `inspect_layout` | Return computed layout tree as JSON |
| `generate_handover` | Generate a Markdown handover document |

### Claude Desktop / Claude Code

Add to your MCP config:

```json
{
  "mcpServers": {
    "rendspec": {
      "command": "rendspec-mcp"
    }
  }
}
```

## Showcase

The [`showcase/`](showcase/) directory contains example designs with rendered SVGs, PNGs, and handover docs:

| # | Design | Source |
|---|--------|--------|
| 01 | SaaS Dashboard | [`01-saas-dashboard.rds`](showcase/designs/01-saas-dashboard.rds) |
| 02 | Architecture Diagram | [`02-architecture-diagram.rds`](showcase/designs/02-architecture-diagram.rds) |
| 03 | Landing Page | [`03-landing-page.rds`](showcase/designs/03-landing-page.rds) |
| 04 | Mobile App Screens | [`04-mobile-app-screens.rds`](showcase/designs/04-mobile-app-screens.rds) |
| 05 | Design System | [`05-design-system.rds`](showcase/designs/05-design-system.rds) |
| 06 | Kanban Board | [`06-kanban-board.rds`](showcase/designs/06-kanban-board.rds) |
| 07 | Data Flow | [`07-data-flow.rds`](showcase/designs/07-data-flow.rds) |
| 08 | Z-Index Stress Test | [`08-z-index-stress.rds`](showcase/designs/08-z-index-stress.rds) |
| 09 | Observability Dashboard | [`09-observability-v2.rds`](showcase/designs/09-observability-v2.rds) |

## Development

```bash
make test     # run tests
make lint     # golangci-lint
make build    # compile to bin/
make clean    # remove bin/
```

Project structure:

```
cmd/rendspec/       CLI entry point
cmd/rendspec-mcp/   MCP server entry point
internal/
  scene/            data model (frames, edges, text)
  parser/           YAML → scene graph
  preprocess/       auto-quoting, implicit children
  tokens/           design token resolution
  layout/           flexbox + grid layout engine
  render/           SVG renderer
  fonts/            font metrics
  png/              PNG via resvg-wasm
  inspect/          layout tree inspection
  handover/         Markdown handover generation
```

## License

MIT
