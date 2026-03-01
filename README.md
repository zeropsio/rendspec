# rendspec

Visual design DSL that compiles to SVG. Write YAML, get pixel-perfect mockups — no browser, no coordinate math, no Figma.

The core idea: describe *what* your layout should look like (flexbox, grid, tokens, components) and let rendspec compute positions, sizes, and render the result. Works as a CLI tool or as an MCP server so AI agents can generate designs programmatically.

## Why

Design tools are visual-first. That's great for humans dragging boxes around, but terrible for:

- **AI agents** that think in text and have no mouse
- **Rapid iteration** where you want to describe a dashboard in 50 lines, not click through 200 dialogs
- **Version control** — YAML diffs are readable, Figma binary blobs are not
- **Automation** — generate designs from data, templates, or CI pipelines
- **Developer handover** — auto-generate implementation specs from the same source

rendspec fills the gap between "describe a UI in words" and "get a real visual artifact." An agent can write 30 lines of DSL and produce a complete dashboard mockup with proper layout, typography, colors, and spacing. No intermediate tool, no screenshot, no manual cleanup.

## What it actually does

rendspec implements a **real layout engine** — not string templates, not hardcoded coordinates. Under the hood:

- **Flexbox layout** with `direction`, `align`, `justify`, `gap`, `flex` grow, `wrap`, and margins
- **Grid layout** with `columns`, `column-gap`, `row-gap`
- **Font measurement** using per-character width heuristics (50+ font families, narrow/wide character classes, bold adjustments) for accurate text sizing and word wrapping
- **Edge routing** with straight, orthogonal, bus (shared trunk), and vertical curve types, plus auto-anchor selection based on relative frame positions
- **Component system** with parameterized templates (`{{title}}`), variants, and deep-copy instantiation (recursion capped at 32)
- **Design tokens** resolved via `$dotted.path` notation, preserving types (numbers stay numbers)
- **Preprocessor** that auto-quotes multi-word values and infers `children:` blocks from indentation, so the YAML stays clean
- **Two-pass SVG renderer** — first pass collects `<defs>` (gradients, shadows, clips, arrow markers), second pass emits elements in z-index order with matched IDs
- **PNG output** via resvg compiled to WASM (no external binary), with system font loading on macOS/Linux/Windows

The layout engine handles real cases: nested flex containers, absolute positioning overlays, grid-to-flex nesting, text that wraps and resizes its container, edges that route around frames. It's not CSS — it doesn't need to be — but it handles the same layout problems.

## Install

```bash
go install github.com/fxck/rendspec/cmd/rendspec@latest
go install github.com/fxck/rendspec/cmd/rendspec-mcp@latest
```

Or build from source:

```bash
make build    # outputs to bin/
```

## Quick start

```bash
rendspec render design.rds                # SVG to stdout
rendspec render design.rds -o out.svg     # SVG to file
rendspec render design.rds -o out.png     # PNG via resvg-wasm
rendspec validate design.rds              # parse + layout check
rendspec inspect design.rds               # dump computed layout as JSON
rendspec handover design.rds -o out.md    # developer handover doc with specs
```

Reads from stdin too: `cat design.rds | rendspec render -`

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

No quotes needed around `6 16` or `1 solid #334155` — the preprocessor handles it. No `children:` keyword needed — indentation implies it.

## DSL at a glance

**Frames** are the building block. Every rectangle, card, sidebar, or container is a frame:

```yaml
- frame: card
  width: 300
  fill: "#1e293b"
  radius: 12
  padding: 20
  gap: 12
  border: 1 solid #334155
  shadow: 0 4 16 rgba(0,0,0,0.15)
```

**Layout** uses flexbox by default. Switch to grid with `layout: grid`:

```yaml
- frame: stats
  layout: grid
  columns: 3
  column-gap: 16
  row-gap: 16
```

**Text** nodes handle typography:

```yaml
- text: "Revenue"
  font: 700 28 Inter
  color: "#f8fafc"
  text-align: center
  max-width: 400          # triggers word wrapping
```

**Components** are reusable templates with parameters:

```yaml
components:
  metric:
    fill: "#1e293b"
    radius: 12
    padding: 16
    params:
      label: { default: "Metric" }
      value: { default: "0" }
    - text: "{{label}}"
      font: 500 12 Inter
    - text: "{{value}}"
      font: 700 24 Inter

root:
  - use: metric
    label: "Revenue"
    value: "$12,450"
```

**Edges** connect named frames with automatic routing:

```yaml
edges:
  - from: api-gateway
    to: auth-service
    label: "JWT"
    arrow: end
    curve: orthogonal
```

**Tokens** define design constants once:

```yaml
tokens:
  color:
    primary: "#3b82f6"
    bg:
      card: "#1e293b"
  radius:
    md: 12

root:
  fill: $color.bg.card
  radius: $radius.md
```

**Themes** set global defaults — built-in (`light`, `dark`, `blueprint`, `sketch`) or custom:

```yaml
theme:
  background: "#0f172a"
  foreground: "#f8fafc"
  accent: "#3b82f6"
  font-family: Inter
```

**Multi-page** for multi-screen designs:

```yaml
pages:
  - name: Login
    root:
      width: 400
      height: 600
      - text: "Sign In"
  - name: Dashboard
    root:
      width: 1280
      height: 800
      - text: "Welcome back"
```

See [`CLAUDE.md`](CLAUDE.md) for the full DSL reference with every property documented.

## MCP server

`rendspec-mcp` exposes rendspec as an MCP tool server. AI agents call it to render designs without touching the filesystem.

| Tool | Description |
|---|---|
| `render_design` | Render DSL source → SVG |
| `validate_design` | Parse + layout check, return stats |
| `inspect_layout` | Computed layout tree as JSON (positions, sizes) |
| `generate_handover` | Markdown handover doc with component tree and CSS mappings |

Add to Claude Desktop or Claude Code MCP config:

```json
{
  "mcpServers": {
    "rendspec": {
      "command": "rendspec-mcp"
    }
  }
}
```

Then an agent can write DSL inline and get back rendered SVG, validation results, or a full developer handover document — all without files on disk.

## Showcase

The [`showcase/`](showcase/) directory has 9 example designs covering dashboards, architecture diagrams, landing pages, mobile screens, design systems, and more. Each includes the `.rds` source, rendered SVG, PNG, and a generated handover doc.

| Design | Source | Handover |
|--------|--------|----------|
| SaaS Dashboard | [`01-saas-dashboard.rds`](showcase/designs/01-saas-dashboard.rds) | [`handover`](showcase/handover/01-saas-dashboard.md) |
| Architecture Diagram | [`02-architecture-diagram.rds`](showcase/designs/02-architecture-diagram.rds) | [`handover`](showcase/handover/02-architecture-diagram.md) |
| Landing Page | [`03-landing-page.rds`](showcase/designs/03-landing-page.rds) | [`handover`](showcase/handover/03-landing-page.md) |
| Mobile App Screens | [`04-mobile-app-screens.rds`](showcase/designs/04-mobile-app-screens.rds) | [`handover`](showcase/handover/04-mobile-app-screens.md) |
| Design System | [`05-design-system.rds`](showcase/designs/05-design-system.rds) | [`handover`](showcase/handover/05-design-system.md) |
| Kanban Board | [`06-kanban-board.rds`](showcase/designs/06-kanban-board.rds) | [`handover`](showcase/handover/06-kanban-board.md) |
| Data Flow | [`07-data-flow.rds`](showcase/designs/07-data-flow.rds) | [`handover`](showcase/handover/07-data-flow.md) |
| Z-Index Stress Test | [`08-z-index-stress.rds`](showcase/designs/08-z-index-stress.rds) | [`handover`](showcase/handover/08-z-index-stress.md) |
| Observability Dashboard | [`09-observability-v2.rds`](showcase/designs/09-observability-v2.rds) | [`handover`](showcase/handover/09-observability-v2.md) |

## How it compares

There's no direct equivalent. rendspec sits at the intersection of diagram-as-code tools and GUI design tools, combining a text DSL with a real layout engine. Here's how it relates to things you might already know:

### vs. diagram-as-code (Mermaid, D2, PlantUML, Graphviz)

These tools are great at what they do — **graph layout**. You define nodes and edges, and a layout algorithm (dagre, ELK, Graphviz dot) minimizes crossings and produces a readable diagram.

But they have no concept of UI layout. There's no flexbox, no grid, no padding, no `direction: row` with `gap: 16`. You can't build a dashboard mockup, a card with aligned text, or a sidebar with navigation items. They position nodes to optimize a graph, not to compose a user interface.

rendspec does both: it has a CSS-like layout engine for UI composition *and* edge routing for connecting frames. You can build a dashboard with stat cards in a grid layout and draw data-flow arrows between services — in the same file.

| | Mermaid | D2 | PlantUML | rendspec |
|---|---|---|---|---|
| Layout engine | Graph (dagre) | Graph (dagre/ELK/TALA) | Graph (Graphviz) | Flexbox + grid |
| UI mockups | No | No | Salt (crude) | Yes |
| Design tokens | No | No | No | Yes |
| Components with params | No | No | No | Yes |
| Edge routing | Yes | Yes | Yes | Yes |
| MCP server | No | No | No | Yes |
| Headless CLI | Yes | Yes | Yes | Yes |

**Use Mermaid/D2/PlantUML** for UML, sequence diagrams, entity-relationship diagrams, or any graph where automatic node placement is the point. **Use rendspec** when you need to control the visual layout — cards, grids, sidebars, dashboards, wireframes.

### vs. GUI design tools (Figma, Penpot)

Figma and Penpot have real layout engines (auto-layout / flexbox / grid), design tokens, components with variants — the same primitives rendspec offers. The difference is the interface.

Figma is GUI-first. You drag, click, and visually compose. That's ideal for human designers but unusable for AI agents, CI pipelines, or version-controlled design artifacts. There's no way to write a text file and get a rendered design — you need the app.

Penpot is open-source and has a proper CSS grid/flexbox engine, but it's also GUI-first. No CLI, no text DSL, no headless rendering.

rendspec is text-first. Same layout concepts, but the input is a YAML file and the output is an SVG. That makes it trivially usable by AI agents (via MCP), diffable in git, and runnable in any pipeline.

**Use Figma/Penpot** for collaborative human design work. **Use rendspec** when the "designer" is an AI agent, a CI job, or anyone who'd rather type than click.

### vs. canvas MCP tools (Excalidraw, tldraw)

Both Excalidraw and tldraw have MCP servers, so AI agents can draw on them. But neither has a layout engine — the agent must compute every x/y coordinate itself. Want three cards in a row with 16px gap? The agent has to calculate `x=0`, `x=216`, `x=432` (assuming 200px cards). Resize a card and everything downstream breaks.

With rendspec, the agent writes `direction: row, gap: 16` and the engine computes positions. The agent describes structure, not coordinates.

Excalidraw also locks you into a hand-drawn aesthetic. tldraw's MCP implementations are fragmented across multiple repos. Neither supports design tokens, components, or theming.

**Use Excalidraw** for informal sketching and brainstorming. **Use rendspec** when the agent needs to produce clean, structured layouts without doing coordinate math.

### vs. AI wireframe generators (Visily, Uizard, Motiff)

These are SaaS products where you type a prompt and get a wireframe. They're the AI, not tools for an AI agent. The output isn't deterministic (same prompt, different result), isn't version-controllable (no text source), and can't be embedded in a development workflow.

rendspec is deterministic — same YAML always produces the same SVG. It's a tool that agents use, not an agent itself. The YAML is the source of truth, lives in your repo, and diffs cleanly.

**Use AI wireframe generators** for quick human exploration of ideas. **Use rendspec** when you need reproducible, version-controlled design artifacts in an automated pipeline.

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
  fonts/              character-level width heuristics for 50+ font families
  png/                resvg-wasm PNG conversion with system font loading
  inspect/            layout tree → JSON
  handover/           scene graph → Markdown developer spec
```

## License

MIT
