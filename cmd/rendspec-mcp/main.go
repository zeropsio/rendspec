// rendspec-mcp — MCP stdio server for rendspec design rendering.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/fxck/rendspec/internal/handover"
	"github.com/fxck/rendspec/internal/inspect"
	"github.com/fxck/rendspec/internal/layout"
	"github.com/fxck/rendspec/internal/parser"
	"github.com/fxck/rendspec/internal/render"
	"github.com/fxck/rendspec/internal/scene"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	s := server.NewMCPServer("rendspec", "0.1.0",
		server.WithToolCapabilities(true),
	)

	// Tool: render_design
	s.AddTool(
		mcp.NewTool("render_design",
			mcp.WithDescription("Render a .rds DSL source to SVG. Returns the SVG as text."),
			mcp.WithString("source", mcp.Required(), mcp.Description("The .rds DSL source text")),
		),
		handleRender,
	)

	// Tool: validate_design
	s.AddTool(
		mcp.NewTool("validate_design",
			mcp.WithDescription("Validate a .rds DSL source. Returns validation stats or error."),
			mcp.WithString("source", mcp.Required(), mcp.Description("The .design DSL source text")),
		),
		handleValidate,
	)

	// Tool: inspect_layout
	s.AddTool(
		mcp.NewTool("inspect_layout",
			mcp.WithDescription("Parse and lay out a .rds DSL source. Returns JSON layout tree with computed positions."),
			mcp.WithString("source", mcp.Required(), mcp.Description("The .rds DSL source text")),
		),
		handleInspect,
	)

	// Tool: generate_handover
	s.AddTool(
		mcp.NewTool("generate_handover",
			mcp.WithDescription("Generate a structured Markdown handover document from a .rds DSL source. Returns component tree, CSS mappings, design tokens, and implementation notes."),
			mcp.WithString("source", mcp.Required(), mcp.Description("The .rds DSL source text")),
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

	// Check for multi-page
	doc, err := parser.ParseDocument(source)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Parse error: %v", err)), nil
	}

	if len(doc.Pages) > 1 {
		var results []string
		if len(doc.Warnings) > 0 {
			results = append(results, "Warnings:\n"+joinStr(prefixEach(doc.Warnings, "  - "), "\n"))
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
			results = append(results, fmt.Sprintf("--- Page: %s ---\n%s", page.Name, svg))
		}
		return mcp.NewToolResultText(fmt.Sprintf("Rendered %d pages:\n\n%s", len(doc.Pages), joinStr(results, "\n\n"))), nil
	}

	sg, err := parser.ParseString(source)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("Parse error: %v", err)), nil
	}

	layout.ComputeLayout(sg)
	svg := render.RenderSVG(sg)

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
