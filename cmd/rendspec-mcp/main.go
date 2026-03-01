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

const dslReference = `

COMPLETE WORKING EXAMPLE (use this as your starting template):

root:
  width: 1280
  height: 720
  fill: "#f8fafc"
  padding: 24
  gap: 16
  - text: "Dashboard"
    font: 700 28 Inter
    color: "#0f172a"
  - frame: cards
    direction: row
    gap: 16
    - frame: card1
      fill: white
      radius: 12
      padding: 20
      flex: 1
      border: 1 solid #e2e8f0
      shadow: 0 2 8 rgba(0,0,0,0.06)
      gap: 8
      - text: "Revenue"
        font: 500 14 Inter
        color: "#64748b"
      - text: "$12,450"
        font: 700 32 Inter
        color: "#0f172a"
    - frame: card2
      fill: white
      radius: 12
      padding: 20
      flex: 1
      border: 1 solid #e2e8f0
      gap: 8
      - text: "Users"
        font: 500 14 Inter
        color: "#64748b"
      - text: "1,234"
        font: 700 32 Inter
        color: "#0f172a"

CRITICAL RULES — read these before writing ANY source:
- The top-level key MUST be "root:" (NOT canvas, NOT config, NOT scene, NOT layout)
- Rectangles/containers are "- frame: name" (NOT type: rect, NOT - rect, NOT layers)
- Text is "- text: \"content\"" (NOT - label, NOT type: text)
- Background color is "fill:" (NOT background, NOT bg, NOT color on frames)
- Font shorthand is "font: 700 24 Inter" — weight(number) size(number, NO px) family (NOT fontSize, NOT fontWeight, NOT font-size)
- Children go directly under their parent frame, indented by 2 spaces with "- " prefix
- There is NO "children:" key, NO "layers:" key, NO "elements:" key
- Hex colors in standalone properties MUST be quoted: fill: "#ff0000" (the # starts a YAML comment otherwise)
- Hex colors inside shorthand properties like border/shadow don't need quotes: border: 1 solid #e2e8f0

TOP-LEVEL STRUCTURE:
  theme: light|dark|blueprint|sketch  # optional
  tokens:                             # optional design tokens
    color:
      primary: "#3b82f6"
  components:                         # optional reusable templates
    button: { ... }
  root:                               # REQUIRED — the canvas
    width: 1280
    height: 720
    fill: "#f8fafc"
    ...child frames and text nodes...
  edges:                              # optional connections between frames
    - from: frameA
      to: frameB

FRAME PROPERTIES (- frame: name):
  width: 200                 height: 100
  min-width: N               max-width: N
  min-height: N              max-height: N
  flex: 1                    # grow to fill available space
  layout: flex|grid          # default: flex
  direction: row|column      # default: column
  align: start|center|end|stretch     # default: stretch
  justify: start|center|end|between|around  # default: start
  gap: 16                    wrap: true
  columns: 3                 rows: 2        # grid mode
  column-gap: 16             row-gap: 12    # grid mode
  padding: 20                # or "12 24" or "8 16 12 16"
  padding-x: 24              padding-y: 12
  margin: 8
  fill: "#2563eb"            # any CSS color, MUST quote hex
  fill: linear-gradient(135deg, #667eea, #764ba2)
  opacity: 0.8               radius: 12
  border: 1 solid #e2e8f0    # auto-quoted, no quotes needed
  shadow: 0 4 16 rgba(0,0,0,0.08)
  clip: true                 visible: false
  image: "photo.jpg"         image-fit: cover|contain|fill|none
  shape: rect|circle|ellipse|diamond
  position: absolute         x: 20    y: 40    z-index: 2

TEXT NODE PROPERTIES (- text: "content"):
  font: 700 24 Inter         # weight size family (NO px units!)
  color: "#0f172a"           # MUST quote hex
  text-align: left|center|right
  line-height: 1.6           max-width: 400
  letter-spacing: 1.5        truncate: true
  text-decoration: none|underline|strikethrough
  opacity: 0.5

EDGES (connections between named frames):
  edges:
    - from: client
      to: server
      stroke: "#94a3b8"      stroke-width: 2
      style: solid|dashed|dotted
      arrow: none|start|end|both
      curve: straight|orthogonal|bus|vertical
      label: "HTTPS"
      label-font: 500 11 Inter
      label-color: "#64748b"
      label-position: 0.5
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
        dark: { fill: "#1e293b" }
  # Usage: - use: chip   OR   - chip: "Label"   OR   - chip: "Dark" + variant: dark

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
  # Usage: - use: stat-card + title: "Revenue" + value: "$12,450"

DESIGN TOKENS ($ prefix):
  tokens:
    color:
      primary: "#3b82f6"
      bg: { card: "#1e293b" }
  # Usage: fill: $color.bg.card   radius: $radius.md

THEMES:
  theme: dark    # built-in: light, dark, blueprint, sketch
  # Custom: theme: { background: "#0f172a", foreground: "#f8fafc", accent: "#3b82f6", border: "#334155", radius: 8, font-family: Inter }

MULTI-PAGE:
  pages:
    - name: Login
      root: { width: 400, height: 600, ... }
    - name: Dashboard
      root: { width: 1280, height: 800, ... }`

const renderDesignDesc = `Render a rendspec .rds design to SVG (or PNG). The source parameter takes a YAML-based DSL that describes visual designs — UI mockups, diagrams, architecture charts, etc. No coordinate math or browser required; the engine handles layout automatically via flexbox and grid.` + dslReference

const validateDesignDesc = `Validate a rendspec .rds DSL source without rendering. Returns canvas size, frame/text/edge counts, and any warnings or parse errors. Use this to check DSL syntax before rendering.` + dslReference

const inspectLayoutDesc = `Parse and lay out a rendspec .rds DSL source, then return the computed layout tree as JSON. Each node includes type, id, x, y, width, height, and children. Use this to debug layout issues or extract positioning data.` + dslReference

const generateHandoverDesc = `Generate a structured Markdown handover document from a rendspec .rds DSL source. Returns component tree, CSS property mappings, design tokens as CSS variables, and implementation notes for developers. Useful for converting a visual design into a development specification.` + dslReference

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
			mcp.WithDescription(validateDesignDesc),
			mcp.WithString("source", mcp.Required(), mcp.Description("The .rds DSL source text (YAML format). See the tool description for full DSL reference.")),
		),
		handleValidate,
	)

	// Tool: inspect_layout
	s.AddTool(
		mcp.NewTool("inspect_layout",
			mcp.WithDescription(inspectLayoutDesc),
			mcp.WithString("source", mcp.Required(), mcp.Description("The .rds DSL source text (YAML format). See the tool description for full DSL reference.")),
		),
		handleInspect,
	)

	// Tool: generate_handover
	s.AddTool(
		mcp.NewTool("generate_handover",
			mcp.WithDescription(generateHandoverDesc),
			mcp.WithString("source", mcp.Required(), mcp.Description("The .rds DSL source text (YAML format). See the tool description for full DSL reference.")),
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

	// Single page — reuse already-parsed document instead of re-parsing
	page := doc.Pages[0]
	sg := &scene.SceneGraph{
		Root:       page.Root,
		Edges:      page.Edges,
		Theme:      doc.Theme,
		Components: doc.Components,
		Tokens:     doc.Tokens,
		Warnings:   doc.Warnings,
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

	doc, err := parser.ParseDocument(source)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Invalid: %v", err)), nil
	}

	var results []string
	for _, page := range doc.Pages {
		sg := &scene.SceneGraph{
			Root:       page.Root,
			Edges:      page.Edges,
			Theme:      doc.Theme,
			Components: doc.Components,
			Tokens:     doc.Tokens,
		}
		layout.ComputeLayout(sg)

		frameCount := inspect.CountFrames(sg.Root)
		textCount := inspect.CountTexts(sg.Root)
		edgeCount := len(sg.Edges)

		if len(doc.Pages) > 1 {
			results = append(results, fmt.Sprintf("Page '%s':", page.Name))
		}
		results = append(results, fmt.Sprintf("Valid\n  Canvas: %.0f x %.0f\n  Frames: %d\n  Text nodes: %d\n  Edges: %d",
			sg.Root.Layout.Width, sg.Root.Layout.Height, frameCount, textCount, edgeCount))
	}

	result := joinStr(results, "\n\n")
	if len(doc.Warnings) > 0 {
		result += "\n\nWarnings:\n" + joinStr(prefixEach(doc.Warnings, "  - "), "\n")
	}

	return mcp.NewToolResultText(result), nil
}

func handleInspect(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.GetArguments()
	source, _ := args["source"].(string)
	if source == "" {
		return mcp.NewToolResultError("source is required"), nil
	}

	doc, err := parser.ParseDocument(source)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Parse error: %v", err)), nil
	}

	if len(doc.Pages) > 1 {
		var pages []map[string]interface{}
		for _, page := range doc.Pages {
			sg := &scene.SceneGraph{
				Root:       page.Root,
				Edges:      page.Edges,
				Theme:      doc.Theme,
				Components: doc.Components,
				Tokens:     doc.Tokens,
			}
			layout.ComputeLayout(sg)
			data := inspect.NodeToMap(sg.Root)
			edges := make([]map[string]interface{}, len(sg.Edges))
			for i, e := range sg.Edges {
				edges[i] = inspect.EdgeToDict(e)
			}
			data["edges"] = edges
			data["page"] = page.Name
			pages = append(pages, data)
		}
		b, _ := json.MarshalIndent(map[string]interface{}{"pages": pages}, "", "  ")
		return mcp.NewToolResultText(string(b)), nil
	}

	page := doc.Pages[0]
	sg := &scene.SceneGraph{
		Root:       page.Root,
		Edges:      page.Edges,
		Theme:      doc.Theme,
		Components: doc.Components,
		Tokens:     doc.Tokens,
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

	doc, err := parser.ParseDocument(source)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Parse error: %v", err)), nil
	}

	if len(doc.Pages) > 1 {
		for i := range doc.Pages {
			sg := &scene.SceneGraph{
				Root:       doc.Pages[i].Root,
				Edges:      doc.Pages[i].Edges,
				Theme:      doc.Theme,
				Components: doc.Components,
				Tokens:     doc.Tokens,
			}
			layout.ComputeLayout(sg)
		}
		md := handover.GenerateDocument(doc, doc.Pages, nil)
		return mcp.NewToolResultText(md), nil
	}

	page := doc.Pages[0]
	sg := &scene.SceneGraph{
		Root:       page.Root,
		Edges:      page.Edges,
		Theme:      doc.Theme,
		Components: doc.Components,
		Tokens:     doc.Tokens,
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
