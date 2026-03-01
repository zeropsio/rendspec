// rendspec-mcp — MCP stdio server for rendspec design rendering.
package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	"github.com/zeropsio/rendspec/internal/handover"
	"github.com/zeropsio/rendspec/internal/inspect"
	"github.com/zeropsio/rendspec/internal/layout"
	"github.com/zeropsio/rendspec/internal/parser"
	pngrender "github.com/zeropsio/rendspec/internal/png"
	"github.com/zeropsio/rendspec/internal/render"
	"github.com/zeropsio/rendspec/internal/scene"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

const renderDesignDesc = `Render a rendspec .rds design to SVG (or PNG). The source parameter takes a YAML-based DSL that describes visual designs — UI mockups, diagrams, architecture charts, etc. No coordinate math or browser required; the engine handles layout automatically via flexbox and grid.

IMPORTANT SYNTAX NOTES:
- The source is YAML. String values containing special characters (#, :, etc.) MUST be quoted.
- Multi-word shorthand values for padding, font, border, shadow, fill are auto-quoted by a preprocessor, so you can write: padding: 12 24 (no quotes needed for these specific properties).
- Children can be written inline without an explicit "children:" key — just indent child frames/text under the parent.
- Use "flex: 1" on a child to make it fill remaining space.
- Default direction is column. Default align is stretch. Default justify is start.

DSL STRUCTURE:
  theme: light|dark|blueprint|sketch|{custom}  # optional
  tokens:          # optional design tokens
    color:
      primary: "#3b82f6"
  components:      # optional reusable templates
    button: { ... }
  root:            # REQUIRED — the canvas
    width: 1280
    height: 720
    ...children...
  edges:           # optional connections
    - from: a
      to: b

FRAME PROPERTIES (- frame: name):
  # Sizing
  width: 200               # px
  height: 100
  min-width/max-width/min-height/max-height: N
  flex: 1                  # grow factor

  # Layout (children arrangement)
  layout: flex|grid        # default: flex
  direction: row|column    # default: column
  align: start|center|end|stretch
  justify: start|center|end|between|around
  gap: 16
  wrap: true

  # Grid (when layout: grid)
  columns: 3
  rows: 2
  column-gap: 16
  row-gap: 12

  # Spacing (CSS-like shorthand)
  padding: 20              # single value
  padding: 12 24           # vert horiz (auto-quoted)
  padding: 8 16 12 16      # top right bottom left
  padding-x: 24
  padding-y: 12
  margin: 8

  # Visual
  fill: "#2563eb"          # any CSS color
  fill: linear-gradient(135deg, #667eea, #764ba2)
  fill: radial-gradient(circle, #fff, #000)
  opacity: 0.8
  radius: 12
  border: 1 solid #e2e8f0  # width style color (auto-quoted)
  border-top/right/bottom/left: 1.5 dashed #ccc
  shadow: 0 4 16 rgba(0,0,0,0.08)
  clip: true
  visible: false

  # Image
  image: "photo.jpg"
  image-fit: cover|contain|fill|none

  # Shape
  shape: rect|circle|ellipse|diamond

  # Absolute positioning
  position: absolute
  x: 20
  y: 40
  z-index: 2

TEXT NODES (- text: "content"):
  font: 700 24 Inter       # "weight size family" (auto-quoted)
  color: "#0f172a"
  text-align: left|center|right
  line-height: 1.6
  max-width: 400           # word wrapping
  letter-spacing: 1.5
  text-decoration: none|underline|strikethrough
  truncate: true
  opacity: 0.5

EDGES (connections between named frames):
  edges:
    - from: client          # frame name or id
      to: server
      stroke: "#94a3b8"
      stroke-width: 2
      style: solid|dashed|dotted
      arrow: none|start|end|both
      curve: straight|orthogonal|bus|vertical
      label: "HTTPS"
      label-font: 500 11 Inter
      label-color: "#64748b"
      label-position: 0.5  # 0-1 along edge
      from-anchor: top|right|bottom|left
      to-anchor: top|right|bottom|left

COMPONENTS (reusable templates):
  components:
    chip:
      fill: "#eff6ff"
      radius: 20
      padding: 6 16
      - text: "Default"
        font: 500 12 Inter
      variants:
        dark:
          fill: "#1e293b"
  # Usage in root:
  - use: chip
  - chip: "Custom Label"   # shorthand sets first text child
  - chip: "Dark"
    variant: dark

PARAMETERIZED COMPONENTS:
  components:
    stat-card:
      params:
        title: { default: "Metric" }
        value: { default: "0" }
      fill: "#1e293b"
      padding: 20
      - text: "{{title}}"
      - text: "{{value}}"
        font: 700 28 Inter
  # Usage:
  - use: stat-card
    title: "Revenue"
    value: "$12,450"

DESIGN TOKENS (referenced with $ prefix):
  tokens:
    color:
      primary: "#3b82f6"
      bg:
        card: "#1e293b"
    radius:
      md: 12
  # Usage:
  fill: $color.bg.card     # resolves to "#1e293b"
  radius: $radius.md       # resolves to 12

THEMES:
  theme: dark              # built-in: light, dark, blueprint, sketch
  # Or custom:
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

MULTI-PAGE:
  pages:
    - name: Login
      root:
        width: 400
        height: 600
        ...
    - name: Dashboard
      root:
        width: 1280
        height: 800
        ...

MINIMAL EXAMPLE:
  root:
    width: 400
    height: 300
    fill: "#f8fafc"
    padding: 24
    gap: 16
    - text: "Hello World"
      font: 700 24 Inter
      color: "#0f172a"
    - frame: card
      fill: white
      radius: 12
      padding: 16
      border: 1 solid #e2e8f0
      shadow: 0 2 8 rgba(0,0,0,0.06)
      - text: "A simple card"
        font: 400 14 Inter
        color: "#475569"`

func main() {
	s := server.NewMCPServer("rendspec", "0.1.0",
		server.WithToolCapabilities(true),
	)

	// Tool: render_design
	s.AddTool(
		mcp.NewTool("render_design",
			mcp.WithDescription(renderDesignDesc),
			mcp.WithString("source", mcp.Required(), mcp.Description("The .rds DSL source text (YAML format). See the tool description for full DSL reference.")),
			mcp.WithString("format", mcp.Description("Output format: svg (default) or png")),
			mcp.WithNumber("scale", mcp.Description("PNG scale factor (default: 2). Only used when format is png.")),
		),
		handleRender,
	)

	// Tool: validate_design
	s.AddTool(
		mcp.NewTool("validate_design",
			mcp.WithDescription("Validate a rendspec .rds DSL source without rendering. Returns canvas size, frame/text/edge counts, and any warnings or parse errors. Use this to check DSL syntax before rendering. The DSL format is documented in the render_design tool description."),
			mcp.WithString("source", mcp.Required(), mcp.Description("The .rds DSL source text (YAML format)")),
		),
		handleValidate,
	)

	// Tool: inspect_layout
	s.AddTool(
		mcp.NewTool("inspect_layout",
			mcp.WithDescription("Parse and lay out a rendspec .rds DSL source, then return the computed layout tree as JSON. Each node includes type, id, x, y, width, height, and children. Use this to debug layout issues or extract positioning data. The DSL format is documented in the render_design tool description."),
			mcp.WithString("source", mcp.Required(), mcp.Description("The .rds DSL source text (YAML format)")),
		),
		handleInspect,
	)

	// Tool: generate_handover
	s.AddTool(
		mcp.NewTool("generate_handover",
			mcp.WithDescription("Generate a structured Markdown handover document from a rendspec .rds DSL source. Returns component tree, CSS property mappings, design tokens as CSS variables, and implementation notes for developers. Useful for converting a visual design into a development specification. The DSL format is documented in the render_design tool description."),
			mcp.WithString("source", mcp.Required(), mcp.Description("The .rds DSL source text (YAML format)")),
		),
		handleHandover,
	)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleRender(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	source, _ := args["source"].(string)
	if source == "" {
		return mcp.NewToolResultError("source is required"), nil
	}

	format := "svg"
	if f, ok := args["format"].(string); ok && f != "" {
		format = f
	}
	scale := 2.0
	if s, ok := args["scale"].(float64); ok && s > 0 {
		scale = s
	}

	wantPNG := format == "png"

	// Check for multi-page
	doc, err := parser.ParseDocument(source)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Parse error: %v", err)), nil
	}

	if len(doc.Pages) > 1 {
		var results []mcp.Content
		if len(doc.Warnings) > 0 {
			results = append(results, mcp.NewTextContent("Warnings:\n"+joinStr(prefixEach(doc.Warnings, "  - "), "\n")))
		}
		for _, page := range doc.Pages {
			sg := &scene.SceneGraph{
				Root:       page.Root,
				Edges:      page.Edges,
				Theme:      doc.Theme,
				Components: doc.Components,
				Tokens:     doc.Tokens,
			}
			layout.ComputeLayout(sg)
			svg := render.RenderSVG(sg)
			if wantPNG {
				pngBytes, pngErr := pngrender.RenderPNG(svg, scale)
				if pngErr != nil {
					return mcp.NewToolResultError(fmt.Sprintf("PNG render error (page %s): %v", page.Name, pngErr)), nil
				}
				results = append(results, mcp.NewTextContent(fmt.Sprintf("--- Page: %s ---", page.Name)))
				results = append(results, mcp.NewImageContent(base64.StdEncoding.EncodeToString(pngBytes), "image/png"))
			} else {
				results = append(results, mcp.NewTextContent(fmt.Sprintf("--- Page: %s ---\n%s", page.Name, svg)))
			}
		}
		return &mcp.CallToolResult{Content: results}, nil
	}

	sg, err := parser.ParseString(source)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Parse error: %v", err)), nil
	}

	layout.ComputeLayout(sg)
	svg := render.RenderSVG(sg)

	if wantPNG {
		pngBytes, pngErr := pngrender.RenderPNG(svg, scale)
		if pngErr != nil {
			return mcp.NewToolResultError(fmt.Sprintf("PNG render error: %v", pngErr)), nil
		}
		var content []mcp.Content
		if len(sg.Warnings) > 0 {
			content = append(content, mcp.NewTextContent("Warnings:\n"+joinStr(prefixEach(sg.Warnings, "  - "), "\n")))
		}
		content = append(content, mcp.NewImageContent(base64.StdEncoding.EncodeToString(pngBytes), "image/png"))
		return &mcp.CallToolResult{Content: content}, nil
	}

	result := svg
	if len(sg.Warnings) > 0 {
		result = "Warnings:\n" + joinStr(prefixEach(sg.Warnings, "  - "), "\n") + "\n\n" + svg
	}
	return mcp.NewToolResultText(result), nil
}

func handleValidate(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	source, _ := args["source"].(string)
	if source == "" {
		return mcp.NewToolResultError("source is required"), nil
	}

	sg, err := parser.ParseString(source)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid: %v", err)), nil
	}

	layout.ComputeLayout(sg)

	frameCount := inspect.CountFrames(sg.Root)
	textCount := inspect.CountTexts(sg.Root)
	edgeCount := len(sg.Edges)

	result := fmt.Sprintf("Valid\n  Canvas: %.0f x %.0f\n  Frames: %d\n  Text nodes: %d\n  Edges: %d",
		sg.Root.Layout.Width, sg.Root.Layout.Height, frameCount, textCount, edgeCount)

	if len(sg.Warnings) > 0 {
		result += "\n\nWarnings:\n" + joinStr(prefixEach(sg.Warnings, "  - "), "\n")
	}

	return mcp.NewToolResultText(result), nil
}

func handleInspect(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	source, _ := args["source"].(string)
	if source == "" {
		return mcp.NewToolResultError("source is required"), nil
	}

	sg, err := parser.ParseString(source)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Parse error: %v", err)), nil
	}

	layout.ComputeLayout(sg)

	data := inspect.NodeToMap(sg.Root)
	edges := make([]map[string]interface{}, len(sg.Edges))
	for i, e := range sg.Edges {
		edges[i] = inspect.EdgeToDict(e)
	}
	data["edges"] = edges
	b, _ := json.MarshalIndent(data, "", "  ")
	return mcp.NewToolResultText(string(b)), nil
}

func handleHandover(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	source, _ := args["source"].(string)
	if source == "" {
		return mcp.NewToolResultError("source is required"), nil
	}

	sg, err := parser.ParseString(source)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Parse error: %v", err)), nil
	}

	layout.ComputeLayout(sg)
	md := handover.Generate(sg)

	return mcp.NewToolResultText(md), nil
}

func joinStr(ss []string, sep string) string {
	result := ""
	for i, s := range ss {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

func prefixEach(ss []string, prefix string) []string {
	out := make([]string, len(ss))
	for i, s := range ss {
		out[i] = prefix + s
	}
	return out
}
