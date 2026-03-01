# rendspec — DSL Primer for AI Agents

rendspec is a CLI tool that renders visual designs from a YAML-based DSL. It produces SVG (and optionally PNG). The DSL is designed to be compact and LLM-friendly — no coordinate math, no browser required.

## Quick Start

```bash
go install ./cmd/rendspec          # install CLI
go install ./cmd/rendspec-mcp      # install MCP server

rendspec render file.rds            # SVG to stdout
rendspec render file.rds -o out.svg # SVG to file
rendspec render file.rds -o out.png # PNG (built-in, resvg via WASM)
rendspec validate file.rds          # parse + layout check
rendspec inspect file.rds           # dump layout as JSON
```

## DSL Structure

Every `.rds` file is valid YAML with this top-level shape:

```yaml
theme: light          # optional: light | dark | blueprint | sketch | {custom}
tokens:               # optional: design token definitions
  color:
    primary: "#3b82f6"
components:           # optional: reusable frame templates
  button: { ... }
root:                 # required: the canvas frame
  width: 1280
  height: 720
  ...
edges:                # optional: connections between frames
  - from: a
    to: b
```

## Preprocessor (runs before YAML parsing)

Two automatic transformations let you write cleaner DSL:

**1. Auto-quoting** — Multi-word shorthand values don't need quotes:
```yaml
padding: 12 24              # auto-quoted to "12 24"
font: 700 20 Inter          # auto-quoted to "700 20 Inter"
border: 1 solid #e2e8f0     # auto-quoted to "1 solid #e2e8f0"
shadow: 0 4 16 rgba(0,0,0,0.08)
fill: linear-gradient(135deg, #667eea, #764ba2)
```
Properties: `padding`, `margin`, `font`, `border`, `border-top/right/bottom/left`, `shadow`, `label-font`, `gradient`, `fill`

**2. Implicit children** — No need to write `children:`:
```yaml
- frame: card
  fill: white
  padding: 20
  - text: "Hello"       # ← automatically becomes a child
    font: 700 20 Inter
  - frame: inner         # ← also a child
    fill: "#f0f0f0"
```

## Frame Properties

Frames are the core building block. Use `- frame: name` for named frames or `- frame:` for anonymous.

```yaml
- frame: card-name       # name becomes the id (unless explicit id: is set)
  id: custom-id          # explicit id (for edge connections)

  # Sizing
  width: 200             # explicit size in pixels
  height: 100
  min-width: 100         # constraints
  max-width: 400
  min-height: 50
  max-height: 300
  flex: 1                # flex grow factor (fills available space)

  # Layout (how children are arranged)
  layout: flex           # flex (default) | grid
  direction: row         # row | column (default: column)
  align: center          # start | center | end | stretch (default: stretch)
  justify: between       # start | center | end | between | around (default: start)
  gap: 16                # space between children in px
  wrap: true             # flex wrapping (default: false)

  # Grid layout (when layout: grid)
  columns: 3             # number of columns
  rows: 2                # optional number of rows
  column-gap: 16
  row-gap: 12

  # Spacing (CSS shorthand)
  padding: 12 24         # single | "vert horiz" | "top right bottom left"
  padding-x: 24
  padding-y: 12
  margin: 8              # same shorthand as padding
  margin-x: 24
  margin-y: 12

  # Visual
  fill: "#2563eb"        # any CSS color
  fill: linear-gradient(135deg, #667eea, #764ba2)
  fill: radial-gradient(circle, #fff, #000)
  gradient: linear-gradient(135deg, #667eea, #764ba2)  # explicit gradient
  opacity: 0.8
  radius: 12
  border: 1 solid #e2e8f0    # width style color
  border-top: 1.5 dashed #ccc
  border-right: 1.5 dashed #ccc
  border-bottom: 1.5 dashed #ccc
  border-left: 1.5 dashed #ccc
  shadow: 0 4 16 rgba(0,0,0,0.08)  # multiple via | separator
  clip: true
  visible: false

  # Image
  image: "photo.jpg"
  image-fit: cover       # cover | contain | fill | none

  # Shape
  shape: circle          # rect (default) | circle | ellipse | diamond

  # Z-ordering
  z-index: 2

  # Absolute positioning
  position: absolute
  x: 20
  y: 40
```

## Text Nodes

```yaml
- text: "Hello World"
  font: 700 24 Inter           # "weight size family"
  color: "#0f172a"
  text-align: center           # left | center | right
  line-height: 1.6
  max-width: 400               # triggers word wrapping
  letter-spacing: 1.5
  text-decoration: underline   # none | underline | strikethrough
  truncate: true
  opacity: 0.5
```

## Edges (Connections)

```yaml
edges:
  - from: client
    to: server
    stroke: "#94a3b8"
    stroke-width: 2
    style: dashed              # solid | dashed | dotted
    arrow: end                 # none | start | end | both
    curve: orthogonal          # straight | orthogonal | bus | vertical
    corner-radius: 8          # curve corner radius
    junction: 100              # bus curve Y position
    label: "HTTPS"
    label-font: 500 11 Inter
    label-color: "#64748b"
    label-position: 0.5
    from-anchor: bottom        # auto | top | right | bottom | left
    to-anchor: top
```

## Components (Reusable Templates)

```yaml
components:
  chip:
    fill: "#eff6ff"
    radius: 20
    padding: 6 16
    border: 1 solid #bfdbfe
    - text: "Default"
      font: 500 12 Inter
    variants:
      dark:
        fill: "#1e293b"

root:
  width: 400
  height: 200
  direction: row
  gap: 8
  - use: chip
  - chip: "Custom Label"
  - chip: "Dark Mode"
    variant: dark
```

### Parameterized Components

```yaml
components:
  stat-card:
    fill: "#1e293b"
    radius: 12
    padding: 20
    gap: 8
    params:
      title: { default: "Metric" }
      value: { default: "0" }
    - text: "{{title}}"
      font: 500 12 Inter
    - text: "{{value}}"
      font: 700 28 Inter

root:
  width: 800
  height: 200
  direction: row
  gap: 16
  - use: stat-card
    title: "Revenue"
    value: "$12,450"
```

## Design Tokens

```yaml
tokens:
  color:
    primary: "#3b82f6"
    bg:
      card: "#1e293b"
  radius:
    md: 12
  font:
    heading: "700 24 Inter"

root:
  width: 800
  height: 600
  fill: $color.bg.card         # resolves to "#1e293b"
  - text: "Hello"
    font: $font.heading
    color: $color.primary
  - text: "Price: $$9.99"      # $$ escapes to literal $
```

## Multi-Page Documents

```yaml
pages:
  - name: Login
    root:
      width: 400
      height: 600
      - text: "Login"
  - name: Dashboard
    root:
      width: 1280
      height: 800
      - text: "Dashboard"
```

## Themes

```yaml
theme: dark                      # built-in: light | dark | blueprint | sketch

# or custom:
theme:
  background: "#0f172a"
  foreground: "#f8fafc"
  muted: "#94a3b8"
  accent: "#3b82f6"
  border: "#334155"
  radius: 8
  font-family: Inter
  font-size: 14
  font-weight: 400
```

## Development Notes

- Source: `internal/` — `scene/` (data), `parser/` (YAML→graph), `preprocess/` (text transforms), `fonts/` (metrics), `layout/` (flexbox+grid), `render/` (SVG)
- Tests: `go test ./internal/... -v`
- Build: `make build` produces `bin/rendspec` and `bin/rendspec-mcp`
