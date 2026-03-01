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

const dslSyntaxHint = ` IMPORTANT: If you are unfamiliar with the rendspec DSL syntax, call the "get_dsl_reference" tool FIRST to get the full syntax documentation and working examples before writing any source.`

const dslReference = `rendspec DSL Reference
======================

EXAMPLE:

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

SYNTAX:

The source is YAML with a custom preprocessor. Hex colors (#...) must be quoted
in standalone properties (fill: "#ff0000") since # is the YAML comment character.
The preprocessor auto-quotes multi-word values for: padding, margin, font, border,
border-top, border-right, border-bottom, border-left, shadow, label-font, gradient, fill.
Children are written directly under their parent with "- " prefix — no "children:" key needed.
Use $$ to produce a literal dollar sign when using design tokens.

TOP-LEVEL STRUCTURE:
  theme: light|dark|blueprint|sketch    # optional
  tokens:                               # optional design tokens
    color:
      primary: "#3b82f6"
  components:                           # optional reusable templates
    button: { ... }
  root:                                 # required — the canvas
    width: 1280
    height: 720
    fill: "#f8fafc"
    ...child frames and text...
  edges:                                # optional connections
    - from: frameA
      to: frameB

FRAME (- frame: name):
  id: custom-id
  width: 200                 height: 100
  min-width: N               max-width: N
  min-height: N              max-height: N
  flex: 1                    # grow to fill available space
  layout: flex|grid          # default: flex
  direction: row|column      # default: column
  align: start|center|end|stretch           # default: stretch
  justify: start|center|end|between|around  # default: start
  gap: 16                    wrap: true
  columns: 3                 rows: 2          # grid mode
  column-gap: 16             row-gap: 12      # grid mode
  padding: 20                # or: 12 24 | 8 16 12 16
  padding-x: 24              padding-y: 12
  margin: 8                  # or: 12 24 | 8 16 12 16
  margin-x: 24               margin-y: 12
  fill: "#2563eb"            # any CSS color (quote hex)
  fill: linear-gradient(135deg, #667eea, #764ba2)
  fill: radial-gradient(circle, #fff, #000)
  gradient: linear-gradient(135deg, #667eea, #764ba2)  # explicit gradient
  opacity: 0.8               radius: 12
  border: 1 solid #e2e8f0    # width style color
  border-top: 1 solid #ccc   # per-side borders
  border-right: 1 solid #ccc
  border-bottom: 1 solid #ccc
  border-left: 1 solid #ccc
  shadow: 0 4 16 rgba(0,0,0,0.08)      # multiple via | separator
  clip: true                 visible: false
  image: "photo.jpg"         image-fit: cover|contain|fill|none
  shape: rect|circle|ellipse|diamond    # default: rect
  position: absolute         x: 20    y: 40    z-index: 2

TEXT (- text: "content"):
  font: 700 24 Inter         # weight size family (no px units)
  color: "#0f172a"
  text-align: left|center|right
  line-height: 1.6           max-width: 400
  letter-spacing: 1.5        truncate: true
  text-decoration: none|underline|strikethrough
  opacity: 0.5

EDGES:
  edges:
    - from: client            # frame name or id
      to: server
      stroke: "#94a3b8"       stroke-width: 2
      style: solid|dashed|dotted
      arrow: none|start|end|both
      curve: straight|orthogonal|bus|vertical
      corner-radius: 8        # curve corner radius
      junction: 100            # bus curve Y position
      label: "HTTPS"
      label-font: 500 11 Inter
      label-color: "#64748b"
      label-position: 0.5     # 0–1 along edge
      from-anchor: auto|top|right|bottom|left
      to-anchor: auto|top|right|bottom|left

COMPONENTS:
  components:
    chip:
      fill: "#eff6ff"
      radius: 20
      padding: 6 16
      - text: "Default"
        font: 500 12 Inter
      variants:
        dark: { fill: "#1e293b" }
  # Usage:
  - use: chip
  - chip: "Custom Label"
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

DESIGN TOKENS:
  tokens:
    color:
      primary: "#3b82f6"
      bg: { card: "#1e293b" }
    radius:
      md: 12
  # Usage: fill: $color.bg.card   radius: $radius.md

THEMES:
  theme: dark                # built-in: light, dark, blueprint, sketch
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
        ...`

func main() {
	s := server.NewMCPServer("rendspec", "0.1.0",
		server.WithToolCapabilities(true),
	)

	// Tool: get_dsl_reference — returns full DSL syntax documentation
	s.AddTool(
		mcp.NewTool("get_dsl_reference",
			mcp.WithDescription("Get the complete rendspec DSL syntax reference with working examples. CALL THIS FIRST before using any other rendspec tool if you are not already familiar with the DSL syntax. Returns the full documentation including: working example, critical rules, all frame/text/edge properties, components, tokens, and themes."),
		),
		handleGetDSLReference,
	)

	// Tool: render_design
	s.AddTool(
		mcp.NewTool("render_design",
			mcp.WithDescription("Render a rendspec .rds design to SVG or PNG. The source is a YAML-based DSL with flexbox/grid layout."+dslSyntaxHint),
			mcp.WithString("source", mcp.Required(), mcp.Description("The .rds DSL source text (YAML). Must start with 'root:' containing width, height, fill, and child frames/text nodes.")),
			mcp.WithString("format", mcp.Description("Output format: svg (default) or png")),
			mcp.WithNumber("scale", mcp.Description("PNG scale factor (default: 2). Only used when format is png.")),
		),
		handleRender,
	)

	// Tool: validate_design
	s.AddTool(
		mcp.NewTool("validate_design",
			mcp.WithDescription("Validate a rendspec .rds DSL source without rendering. Returns canvas size, frame/text/edge counts, and warnings."+dslSyntaxHint),
			mcp.WithString("source", mcp.Required(), mcp.Description("The .rds DSL source text (YAML). Must start with 'root:' containing width, height, fill, and child frames/text nodes.")),
		),
		handleValidate,
	)

	// Tool: inspect_layout
	s.AddTool(
		mcp.NewTool("inspect_layout",
			mcp.WithDescription("Parse and lay out a rendspec .rds DSL source, returning the computed layout tree as JSON with positions."+dslSyntaxHint),
			mcp.WithString("source", mcp.Required(), mcp.Description("The .rds DSL source text (YAML). Must start with 'root:' containing width, height, fill, and child frames/text nodes.")),
		),
		handleInspect,
	)

	// Tool: generate_handover
	s.AddTool(
		mcp.NewTool("generate_handover",
			mcp.WithDescription("Generate a Markdown handover document from a rendspec .rds DSL source with CSS mappings and implementation notes."+dslSyntaxHint),
			mcp.WithString("source", mcp.Required(), mcp.Description("The .rds DSL source text (YAML). Must start with 'root:' containing width, height, fill, and child frames/text nodes.")),
		),
		handleHandover,
	)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func handleGetDSLReference(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return mcp.NewToolResultText(dslReference), nil
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
