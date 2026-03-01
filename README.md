# rendspec

Visual design DSL that compiles to SVG. Describe layouts in YAML — flexbox, grid, tokens, components — and rendspec computes positions and renders the result. No browser, no coordinate math.

Works as a CLI or as an [MCP server](#mcp-server) for AI agents.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/zeropsio/rendspec/main/install.sh | sh
```

This downloads the latest release binary for your platform and installs `rendspec` + `rendspec-mcp` to `/usr/local/bin` (or `~/.local/bin` if no write access).

Install a specific version: `curl -fsSL ... | sh -s v0.2.0`

Or install from source:

```bash
go install github.com/zeropsio/rendspec/cmd/rendspec@latest
go install github.com/zeropsio/rendspec/cmd/rendspec-mcp@latest
```

## Usage

```bash
rendspec render design.rds               # SVG to stdout
rendspec render design.rds -o out.svg    # SVG to file
rendspec render design.rds -o out.png    # PNG (resvg-wasm, no external deps)
rendspec validate design.rds             # parse + layout check
rendspec inspect design.rds              # computed layout as JSON
rendspec handover design.rds -o out.md   # developer handover doc
```

Reads from stdin: `cat design.rds | rendspec render -`

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

No quotes needed around `6 16` or `1 solid #334155` — the preprocessor handles it.
No `children:` keyword — indentation implies it.

## Features

**Layout** — Flexbox by default (`direction`, `align`, `justify`, `gap`, `flex`, `wrap`). Grid with `layout: grid` and `columns`. Nested freely.

**Design tokens** — Define colors, radii, fonts once. Reference with `$color.primary` or `$radius.md`. Types are preserved.

**Components** — Reusable templates with `params`, `{{interpolation}}`, and `variants`. Instantiate with `- use: component` or the shorthand `- component: "label"`.

**Edges** — Connect named frames with arrows. Routing modes: `straight`, `orthogonal`, `bus` (shared trunk), `vertical`. Auto-anchor picks the best connection point.

**Themes** — Built-in (`light`, `dark`, `blueprint`, `sketch`) or custom with `background`, `foreground`, `accent`, `border`, `font-family`, etc.

**Visual properties** — Gradients (linear, radial), shadows, borders (per-side), border-radius, opacity, clipping, z-index, shapes (rect, circle, ellipse, diamond), images with fit modes.

**Text** — `font: 700 24 Inter` shorthand. Word wrapping via `max-width`. Alignment, line-height, letter-spacing, decoration, truncation.

**Multi-page** — `pages:` for multi-screen designs, rendered to separate files.

Full DSL reference in [`CLAUDE.md`](CLAUDE.md).

## MCP server

`rendspec-mcp` runs as an MCP stdio server. AI agents write DSL and get back rendered output — no files on disk.

| Tool | Returns |
|---|---|
| `render_design` | SVG |
| `validate_design` | Canvas size, frame/text/edge counts, warnings |
| `inspect_layout` | JSON layout tree with computed positions |
| `generate_handover` | Markdown dev spec with component tree and CSS mappings |

After [installing](#install), add the MCP server:

### Claude Code

```bash
claude mcp add rendspec rendspec-mcp
```

### Claude Desktop / other MCP clients

Add to your MCP config (e.g. `claude_desktop_config.json`):

```json
{
  "mcpServers": {
    "rendspec": {
      "command": "rendspec-mcp"
    }
  }
}
```

## How it compares

rendspec combines a text DSL (like diagram-as-code tools) with a CSS-like layout engine (like Figma). Nothing else does both.

| | Mermaid / D2 | Figma / Penpot | Excalidraw MCP | AI wireframe SaaS | rendspec |
|---|---|---|---|---|---|
| Input | Text DSL | GUI | MCP (x/y coords) | Natural language | YAML DSL |
| Layout engine | Graph algorithms | Flexbox / grid | None (manual) | Opaque | Flexbox + grid |
| UI mockups | No | Yes | Sketchy only | Yes | Yes |
| Design tokens | No | Yes | No | No | Yes |
| Components | No | Yes | No | No | Yes |
| Edge routing | Yes | No | Manual | No | Yes |
| Deterministic | Yes | — | — | No | Yes |
| Headless / CLI | Yes | No | No | No | Yes |
| MCP server | No | No | Yes | No | Yes |
| Version control | Yes | No | No | No | Yes |

**Use Mermaid/D2** for graph diagrams where automatic node placement is the point.
**Use Figma/Penpot** for collaborative human design work.
**Use Excalidraw** for informal sketching.
**Use rendspec** when you need structured UI layouts from text — especially from AI agents.

## Under the hood

The layout engine is real, not templates or hardcoded coordinates:

- Flexbox with flex-grow distribution, justify/align, gap, wrap, margins
- Grid with equal-width column tracks and gap support
- Font measurement via per-character width heuristics across 50+ families (narrow/wide classes, bold adjustment) for accurate text sizing and word wrap
- Edge routing with auto-anchor selection, orthogonal paths, bus topology (shared trunks), and gap-finding
- Two-pass SVG renderer — first pass collects `<defs>` (gradients, shadow filters, clip paths, arrow markers), second pass emits elements in z-index order
- PNG via resvg compiled to WASM with automatic system font loading (macOS, Linux, Windows)

## Showcase

[`showcase/`](showcase/) has 9 example designs with source, SVG, PNG, and handover docs:

| | Source | Handover |
|---|---|---|
| SaaS Dashboard | [`01`](showcase/designs/01-saas-dashboard.rds) | [`md`](showcase/handover/01-saas-dashboard.md) |
| Architecture Diagram | [`02`](showcase/designs/02-architecture-diagram.rds) | [`md`](showcase/handover/02-architecture-diagram.md) |
| Landing Page | [`03`](showcase/designs/03-landing-page.rds) | [`md`](showcase/handover/03-landing-page.md) |
| Mobile App Screens | [`04`](showcase/designs/04-mobile-app-screens.rds) | [`md`](showcase/handover/04-mobile-app-screens.md) |
| Design System | [`05`](showcase/designs/05-design-system.rds) | [`md`](showcase/handover/05-design-system.md) |
| Kanban Board | [`06`](showcase/designs/06-kanban-board.rds) | [`md`](showcase/handover/06-kanban-board.md) |
| Data Flow | [`07`](showcase/designs/07-data-flow.rds) | [`md`](showcase/handover/07-data-flow.md) |
| Z-Index Stress Test | [`08`](showcase/designs/08-z-index-stress.rds) | [`md`](showcase/handover/08-z-index-stress.md) |
| Observability Dashboard | [`09`](showcase/designs/09-observability-v2.rds) | [`md`](showcase/handover/09-observability-v2.md) |

## Development

```bash
make test     # go test ./internal/... -v
make lint     # golangci-lint run ./...
make build    # compile to bin/
make clean    # remove bin/
```

```
cmd/rendspec/         CLI
cmd/rendspec-mcp/     MCP server
internal/
  scene/              data model — frames, text, edges, themes, gradients
  parser/             YAML → scene graph, component expansion, token resolution
  preprocess/         auto-quoting and implicit children transforms
  layout/             flexbox + grid engine, edge routing
  render/             two-pass SVG renderer
  fonts/              per-character width heuristics for 50+ font families
  png/                resvg-wasm PNG conversion with system font loading
  inspect/            layout tree → JSON
  handover/           scene graph → Markdown developer spec
```

## License

MIT
