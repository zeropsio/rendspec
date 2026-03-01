// Package handover generates structured Markdown handover documents from rendspec scene graphs.
// These documents bridge the gap between visual designs and code implementation by providing
// component trees with computed layout, CSS mappings, design tokens, and edge relationships.
package handover

import (
	"fmt"
	"sort"
	"strings"

	"github.com/zeropsio/rendspec/internal/scene"
)

// Options configures handover generation.
type Options struct {
	// ImagePaths are relative paths to rendered PNG images to embed in the document.
	ImagePaths []string
}

// Generate produces a Markdown handover document from a single-page SceneGraph.
func Generate(sg *scene.SceneGraph, opts ...Options) string {
	var o Options
	if len(opts) > 0 {
		o = opts[0]
	}

	var b strings.Builder

	b.WriteString("# Design Handover Document\n\n")

	writeImages(&b, o.ImagePaths)
	writeOverview(&b, sg)
	writeTokens(&b, sg.Tokens)
	writeComponents(&b, sg.Components)
	writeTree(&b, sg.Root)
	writeEdges(&b, sg.Edges)
	writeImplementationNotes(&b)

	return b.String()
}

// GenerateDocument produces a Markdown handover document from a multi-page Document.
// Each page gets its own component tree and edges section.
// pageImages maps page index to relative PNG path.
func GenerateDocument(doc *scene.Document, pages []scene.Page, pageImages map[int]string) string {
	var b strings.Builder

	b.WriteString("# Design Handover Document\n\n")

	// Overview — aggregate counts across ALL pages
	totalFrames, totalTexts, totalEdges := 0, 0, 0
	for _, page := range pages {
		f, t := countAllNodes(page.Root)
		totalFrames += f
		totalTexts += t
		totalEdges += len(page.Edges)
	}
	sg := &scene.SceneGraph{
		Root:       pages[0].Root,
		Edges:      make([]*scene.EdgeNode, totalEdges),
		Theme:      doc.Theme,
		Components: doc.Components,
		Tokens:     doc.Tokens,
	}
	writeOverviewMultiPage(&b, sg, len(pages), totalFrames, totalTexts, totalEdges)
	writeTokens(&b, doc.Tokens)
	writeComponents(&b, doc.Components)

	for i, page := range pages {
		b.WriteString(fmt.Sprintf("## Page: %s\n\n", page.Name))
		if img, ok := pageImages[i]; ok {
			b.WriteString(fmt.Sprintf("![%s](%s)\n\n", page.Name, img))
		}
		writeTree(&b, page.Root)
		writeEdges(&b, page.Edges)
	}

	writeImplementationNotes(&b)

	return b.String()
}

// --- Preview Image ---

func writeImages(b *strings.Builder, paths []string) {
	if len(paths) == 0 {
		return
	}
	for _, p := range paths {
		b.WriteString(fmt.Sprintf("![Design Preview](%s)\n\n", p))
	}
}

// --- Overview ---

func writeOverview(b *strings.Builder, sg *scene.SceneGraph) {
	b.WriteString("## Overview\n\n")
	b.WriteString("| Property | Value |\n")
	b.WriteString("|----------|-------|\n")

	w := sg.Root.Layout.Width
	h := sg.Root.Layout.Height
	b.WriteString(fmt.Sprintf("| Canvas | %.0f x %.0f |\n", w, h))
	b.WriteString(fmt.Sprintf("| Theme | %s |\n", describeTheme(sg.Theme)))

	if sg.Root.Fill != nil {
		b.WriteString(fmt.Sprintf("| Background | `%s` |\n", *sg.Root.Fill))
	} else if sg.Root.Gradient != nil {
		b.WriteString(fmt.Sprintf("| Background | gradient |\n"))
	}

	fontDesc := fmt.Sprintf("%d %gpx %s", sg.Theme.FontWeight, sg.Theme.FontSize, sg.Theme.FontFamily)
	b.WriteString(fmt.Sprintf("| Default Font | `%s` |\n", fontDesc))

	frameCount, textCount := countAllNodes(sg.Root)
	b.WriteString(fmt.Sprintf("| Frames | %d |\n", frameCount))
	b.WriteString(fmt.Sprintf("| Text Nodes | %d |\n", textCount))
	b.WriteString(fmt.Sprintf("| Edges | %d |\n", len(sg.Edges)))
	b.WriteString("\n")
}

func writeOverviewMultiPage(b *strings.Builder, sg *scene.SceneGraph, pageCount, frameCount, textCount, edgeCount int) {
	b.WriteString("## Overview\n\n")
	b.WriteString("| Property | Value |\n")
	b.WriteString("|----------|-------|\n")

	b.WriteString(fmt.Sprintf("| Pages | %d |\n", pageCount))
	b.WriteString(fmt.Sprintf("| Theme | %s |\n", describeTheme(sg.Theme)))

	fontDesc := fmt.Sprintf("%d %gpx %s", sg.Theme.FontWeight, sg.Theme.FontSize, sg.Theme.FontFamily)
	b.WriteString(fmt.Sprintf("| Default Font | `%s` |\n", fontDesc))

	b.WriteString(fmt.Sprintf("| Total Frames | %d |\n", frameCount))
	b.WriteString(fmt.Sprintf("| Total Text Nodes | %d |\n", textCount))
	b.WriteString(fmt.Sprintf("| Total Edges | %d |\n", edgeCount))
	b.WriteString("\n")
}

func describeTheme(theme scene.Theme) string {
	for name, builtin := range scene.BuiltinThemes {
		if theme == builtin {
			return name
		}
	}
	return "custom"
}

func countAllNodes(root *scene.FrameNode) (frames, texts int) {
	frames = 1 // count root
	for _, child := range root.Children {
		switch c := child.(type) {
		case *scene.FrameNode:
			f, t := countAllNodes(c)
			frames += f
			texts += t
		case *scene.TextNode:
			texts++
		}
	}
	return
}

// --- Design Tokens ---

func writeTokens(b *strings.Builder, tokens map[string]interface{}) {
	if len(tokens) == 0 {
		return
	}

	flat := FlattenTokens(tokens)
	if len(flat) == 0 {
		return
	}

	b.WriteString("## Design Tokens\n\n")

	// Token table
	b.WriteString("| Token | Value |\n")
	b.WriteString("|-------|-------|\n")

	keys := sortedKeys(flat)
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("| `$%s` | `%v` |\n", k, flat[k]))
	}
	b.WriteString("\n")

	// CSS variables
	b.WriteString("### CSS Variables\n\n")
	b.WriteString("```css\n:root {\n")
	for _, k := range keys {
		cssVar := strings.ReplaceAll(k, ".", "-")
		b.WriteString(fmt.Sprintf("  --%s: %v;\n", cssVar, flat[k]))
	}
	b.WriteString("}\n```\n\n")
}

// FlattenTokens recursively flattens a nested token map into dot-path keys.
func FlattenTokens(tokens map[string]interface{}) map[string]string {
	result := make(map[string]string)
	flattenTokensRecursive("", tokens, result)
	return result
}

func flattenTokensRecursive(prefix string, m map[string]interface{}, result map[string]string) {
	for k, v := range m {
		path := k
		if prefix != "" {
			path = prefix + "." + k
		}
		switch val := v.(type) {
		case map[string]interface{}:
			flattenTokensRecursive(path, val, result)
		default:
			result[path] = fmt.Sprintf("%v", val)
		}
	}
}

// --- Components ---

func writeComponents(b *strings.Builder, components map[string]interface{}) {
	if len(components) == 0 {
		return
	}

	b.WriteString("## Components\n\n")

	names := make([]string, 0, len(components))
	for name := range components {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		def, ok := components[name].(map[string]interface{})
		if !ok {
			continue
		}

		b.WriteString(fmt.Sprintf("### `%s`\n\n", name))

		// Parameters
		if params, ok := def["params"].(map[string]interface{}); ok && len(params) > 0 {
			b.WriteString("**Parameters:**\n\n")
			b.WriteString("| Param | Default |\n")
			b.WriteString("|-------|---------|\n")

			paramNames := make([]string, 0, len(params))
			for pn := range params {
				paramNames = append(paramNames, pn)
			}
			sort.Strings(paramNames)

			for _, pn := range paramNames {
				defVal := "-"
				if spec, ok := params[pn].(map[string]interface{}); ok {
					if d, ok := spec["default"]; ok {
						defVal = fmt.Sprintf("`%v`", d)
					}
				}
				b.WriteString(fmt.Sprintf("| `%s` | %s |\n", pn, defVal))
			}
			b.WriteString("\n")
		}

		// Base CSS from component definition
		css := cssFromComponentDef(def)
		if css != "" {
			b.WriteString("**Base CSS:**\n\n")
			b.WriteString("```css\n")
			b.WriteString(css)
			b.WriteString("\n```\n\n")
		}
	}
}

func cssFromComponentDef(def map[string]interface{}) string {
	var props []string

	if fill, ok := def["fill"].(string); ok {
		if isGradient(fill) {
			props = append(props, fmt.Sprintf("background: %s;", fill))
		} else {
			props = append(props, fmt.Sprintf("background-color: %s;", fill))
		}
	}
	if r, ok := toFloat(def["radius"]); ok && r > 0 {
		props = append(props, fmt.Sprintf("border-radius: %gpx;", r))
	}
	if p, ok := def["padding"]; ok {
		props = append(props, fmt.Sprintf("padding: %s;", spacingToCSS(p)))
	}
	if g, ok := toFloat(def["gap"]); ok && g > 0 {
		props = append(props, fmt.Sprintf("gap: %gpx;", g))
	}
	if border, ok := def["border"].(string); ok {
		props = append(props, fmt.Sprintf("border: %s;", borderToCSS(border)))
	}

	if len(props) == 0 {
		return ""
	}
	return strings.Join(props, "\n")
}

// --- Component Tree ---

func writeTree(b *strings.Builder, root *scene.FrameNode) {
	b.WriteString("## Component Tree\n\n")
	b.WriteString("```\n")
	writeTreeNode(b, root, "", true, true)
	b.WriteString("```\n\n")
}

func writeTreeNode(b *strings.Builder, node scene.Node, prefix string, isLast bool, isRoot bool) {
	switch n := node.(type) {
	case *scene.FrameNode:
		writeFrameTreeNode(b, n, prefix, isLast, isRoot)
	case *scene.TextNode:
		writeTextTreeNode(b, n, prefix, isLast)
	}
}

func writeFrameTreeNode(b *strings.Builder, fn *scene.FrameNode, prefix string, isLast bool, isRoot bool) {
	// Node header line
	connector := "+-- "
	if isRoot {
		connector = ""
	}

	label := "frame"
	if isRoot {
		label = "root"
	}
	if fn.ID != "" {
		label = fmt.Sprintf("frame#%s", fn.ID)
	}
	if fn.ComponentName != "" {
		label = fmt.Sprintf("[%s]", fn.ComponentName)
		if fn.ID != "" {
			label = fmt.Sprintf("[%s]#%s", fn.ComponentName, fn.ID)
		}
	}

	b.WriteString(fmt.Sprintf("%s%s%s (%.0f x %.0f @ %.0f, %.0f)\n",
		prefix, connector, label,
		fn.Layout.Width, fn.Layout.Height, fn.Layout.X, fn.Layout.Y))

	// Properties line
	var childPrefix string
	if isRoot {
		childPrefix = ""
	} else if isLast {
		childPrefix = prefix + "      "
	} else {
		childPrefix = prefix + "|     "
	}

	propsLine := framePropsLine(fn)
	if propsLine != "" {
		b.WriteString(fmt.Sprintf("%s  %s\n", childPrefix, propsLine))
	}

	// CSS line
	cssLine := frameCSSLine(fn)
	if cssLine != "" {
		b.WriteString(fmt.Sprintf("%s  css: { %s }\n", childPrefix, cssLine))
	}

	// Children
	if len(fn.Children) > 0 {
		b.WriteString(fmt.Sprintf("%s  |\n", childPrefix))
		for i, child := range fn.Children {
			last := i == len(fn.Children)-1
			writeTreeNode(b, child, childPrefix, last, false)
		}
	}
}

func writeTextTreeNode(b *strings.Builder, tn *scene.TextNode, prefix string, isLast bool) {
	connector := "+-- "
	content := tn.Content
	if len(content) > 40 {
		content = content[:37] + "..."
	}

	b.WriteString(fmt.Sprintf("%s%stext \"%s\" (%.0f x %.0f @ %.0f, %.0f)\n",
		prefix, connector, content,
		tn.Layout.Width, tn.Layout.Height, tn.Layout.X, tn.Layout.Y))

	var childPrefix string
	if isLast {
		childPrefix = prefix + "      "
	} else {
		childPrefix = prefix + "|     "
	}

	var props []string
	props = append(props, fmt.Sprintf("font: %d %gpx %s", tn.Font.Weight, tn.Font.Size, tn.Font.Family))
	if tn.Color != "" {
		props = append(props, fmt.Sprintf("color: %s", tn.Color))
	}
	if tn.TextAlign != "left" {
		props = append(props, fmt.Sprintf("text-align: %s", tn.TextAlign))
	}
	if tn.LineHeight != 0 && tn.LineHeight != 1.4 {
		props = append(props, fmt.Sprintf("line-height: %g", tn.LineHeight))
	}
	if tn.MaxWidth != nil {
		props = append(props, fmt.Sprintf("max-width: %gpx", *tn.MaxWidth))
	}
	if tn.LetterSpacing != 0 {
		props = append(props, fmt.Sprintf("letter-spacing: %gpx", tn.LetterSpacing))
	}
	if tn.TextDecoration != "" && tn.TextDecoration != "none" {
		props = append(props, fmt.Sprintf("text-decoration: %s", tn.TextDecoration))
	}
	if tn.Truncate {
		props = append(props, "truncate: true")
	}
	if tn.Opacity < 1.0 {
		props = append(props, fmt.Sprintf("opacity: %g", tn.Opacity))
	}

	b.WriteString(fmt.Sprintf("%s  %s\n", childPrefix, strings.Join(props, " | ")))
}

func framePropsLine(fn *scene.FrameNode) string {
	var parts []string

	if fn.Fill != nil {
		parts = append(parts, fmt.Sprintf("fill: %s", *fn.Fill))
	}
	if fn.Gradient != nil {
		parts = append(parts, "fill: gradient")
	}
	if hasPadding(fn.Padding) {
		parts = append(parts, fmt.Sprintf("padding: %s", spacingToStr(fn.Padding)))
	}
	if hasSpacing(fn.Margin) {
		parts = append(parts, fmt.Sprintf("margin: %s", spacingToStr(fn.Margin)))
	}
	if fn.Gap > 0 {
		parts = append(parts, fmt.Sprintf("gap: %gpx", fn.Gap))
	}
	if fn.Direction != "column" {
		parts = append(parts, fmt.Sprintf("direction: %s", fn.Direction))
	}
	if fn.Justify != "start" {
		parts = append(parts, fmt.Sprintf("justify: %s", fn.Justify))
	}
	if fn.Align != "stretch" {
		parts = append(parts, fmt.Sprintf("align: %s", fn.Align))
	}
	if fn.Wrap {
		parts = append(parts, "wrap: true")
	}
	if fn.LayoutMode == "grid" && fn.Columns != nil {
		cols := fmt.Sprintf("grid: %d cols", *fn.Columns)
		if fn.Rows != nil {
			cols += fmt.Sprintf(" x %d rows", *fn.Rows)
		}
		parts = append(parts, cols)
	}
	if fn.MinWidth != nil {
		parts = append(parts, fmt.Sprintf("min-width: %gpx", *fn.MinWidth))
	}
	if fn.MaxWidth != nil {
		parts = append(parts, fmt.Sprintf("max-width: %gpx", *fn.MaxWidth))
	}
	if fn.MinHeight != nil {
		parts = append(parts, fmt.Sprintf("min-height: %gpx", *fn.MinHeight))
	}
	if fn.MaxHeight != nil {
		parts = append(parts, fmt.Sprintf("max-height: %gpx", *fn.MaxHeight))
	}
	if fn.Radius > 0 {
		parts = append(parts, fmt.Sprintf("radius: %gpx", fn.Radius))
	}
	if fn.Border != nil {
		parts = append(parts, fmt.Sprintf("border: %gpx %s %s", fn.Border.Width, fn.Border.Style, fn.Border.Color))
	}
	if fn.BorderTop != nil {
		parts = append(parts, fmt.Sprintf("border-top: %gpx %s %s", fn.BorderTop.Width, fn.BorderTop.Style, fn.BorderTop.Color))
	}
	if fn.BorderRight != nil {
		parts = append(parts, fmt.Sprintf("border-right: %gpx %s %s", fn.BorderRight.Width, fn.BorderRight.Style, fn.BorderRight.Color))
	}
	if fn.BorderBottom != nil {
		parts = append(parts, fmt.Sprintf("border-bottom: %gpx %s %s", fn.BorderBottom.Width, fn.BorderBottom.Style, fn.BorderBottom.Color))
	}
	if fn.BorderLeft != nil {
		parts = append(parts, fmt.Sprintf("border-left: %gpx %s %s", fn.BorderLeft.Width, fn.BorderLeft.Style, fn.BorderLeft.Color))
	}
	if len(fn.Shadow) > 0 {
		parts = append(parts, "shadow: yes")
	}
	if fn.Clip {
		parts = append(parts, "clip: true")
	}
	if fn.Shape != "" && fn.Shape != "rect" {
		parts = append(parts, fmt.Sprintf("shape: %s", fn.Shape))
	}
	if fn.Image != nil {
		img := fmt.Sprintf("image: %s", *fn.Image)
		if fn.ImageFit != "cover" {
			img += fmt.Sprintf(" (%s)", fn.ImageFit)
		}
		parts = append(parts, img)
	}
	if fn.Flex != nil {
		parts = append(parts, fmt.Sprintf("flex: %g", *fn.Flex))
	}
	if fn.ZIndex != 0 {
		parts = append(parts, fmt.Sprintf("z-index: %d", fn.ZIndex))
	}
	if !fn.Visible {
		parts = append(parts, "visible: false")
	}
	if fn.Position == "absolute" {
		parts = append(parts, "position: absolute")
	}

	return strings.Join(parts, " | ")
}

func hasSpacing(s scene.Spacing) bool {
	return s.Top > 0 || s.Right > 0 || s.Bottom > 0 || s.Left > 0
}

func frameCSSLine(fn *scene.FrameNode) string {
	var parts []string

	// Display/layout
	if fn.LayoutMode == "grid" && fn.Columns != nil {
		parts = append(parts, "display: grid;")
		parts = append(parts, fmt.Sprintf("grid-template-columns: repeat(%d, 1fr);", *fn.Columns))
	} else if len(fn.Children) > 0 {
		parts = append(parts, "display: flex;")
		parts = append(parts, fmt.Sprintf("flex-direction: %s;", fn.Direction))
	}

	// Justify
	if fn.Justify != "start" {
		parts = append(parts, fmt.Sprintf("justify-content: %s;", justifyToCSS(fn.Justify)))
	}

	// Align
	if fn.Align != "stretch" {
		parts = append(parts, fmt.Sprintf("align-items: %s;", alignToCSS(fn.Align)))
	}

	// Gap
	if fn.Gap > 0 {
		if fn.LayoutMode == "grid" {
			if fn.ColumnGap != nil && fn.RowGap != nil && *fn.ColumnGap != *fn.RowGap {
				parts = append(parts, fmt.Sprintf("column-gap: %gpx;", *fn.ColumnGap))
				parts = append(parts, fmt.Sprintf("row-gap: %gpx;", *fn.RowGap))
			} else {
				parts = append(parts, fmt.Sprintf("gap: %gpx;", fn.Gap))
			}
		} else {
			parts = append(parts, fmt.Sprintf("gap: %gpx;", fn.Gap))
		}
	}
	if fn.LayoutMode == "grid" && fn.Gap == 0 {
		if fn.ColumnGap != nil {
			parts = append(parts, fmt.Sprintf("column-gap: %gpx;", *fn.ColumnGap))
		}
		if fn.RowGap != nil {
			parts = append(parts, fmt.Sprintf("row-gap: %gpx;", *fn.RowGap))
		}
	}

	// Wrap
	if fn.Wrap {
		parts = append(parts, "flex-wrap: wrap;")
	}

	// Padding
	if hasPadding(fn.Padding) {
		parts = append(parts, fmt.Sprintf("padding: %s;", spacingToCSSValue(fn.Padding)))
	}

	// Margin
	if hasSpacing(fn.Margin) {
		parts = append(parts, fmt.Sprintf("margin: %s;", spacingToCSSValue(fn.Margin)))
	}

	// Fill
	if fn.Fill != nil {
		if isGradient(*fn.Fill) {
			parts = append(parts, fmt.Sprintf("background: %s;", *fn.Fill))
		} else {
			parts = append(parts, fmt.Sprintf("background-color: %s;", *fn.Fill))
		}
	}
	if fn.Gradient != nil {
		parts = append(parts, fmt.Sprintf("background: %s;", gradientToCSS(fn.Gradient)))
	}

	// Radius
	if fn.Radius > 0 {
		parts = append(parts, fmt.Sprintf("border-radius: %gpx;", fn.Radius))
	}

	// Border
	if fn.Border != nil {
		parts = append(parts, fmt.Sprintf("border: %s;", borderStructToCSS(fn.Border)))
	}
	if fn.BorderTop != nil {
		parts = append(parts, fmt.Sprintf("border-top: %s;", borderStructToCSS(fn.BorderTop)))
	}
	if fn.BorderRight != nil {
		parts = append(parts, fmt.Sprintf("border-right: %s;", borderStructToCSS(fn.BorderRight)))
	}
	if fn.BorderBottom != nil {
		parts = append(parts, fmt.Sprintf("border-bottom: %s;", borderStructToCSS(fn.BorderBottom)))
	}
	if fn.BorderLeft != nil {
		parts = append(parts, fmt.Sprintf("border-left: %s;", borderStructToCSS(fn.BorderLeft)))
	}

	// Shadow
	if len(fn.Shadow) > 0 {
		parts = append(parts, fmt.Sprintf("box-shadow: %s;", shadowToCSS(fn.Shadow)))
	}

	// Clip
	if fn.Clip {
		parts = append(parts, "overflow: hidden;")
	}

	// Opacity
	if fn.Opacity < 1.0 {
		parts = append(parts, fmt.Sprintf("opacity: %g;", fn.Opacity))
	}

	// Width/Height if explicit
	if fn.Width != nil {
		parts = append(parts, fmt.Sprintf("width: %gpx;", *fn.Width))
	}
	if fn.Height != nil {
		parts = append(parts, fmt.Sprintf("height: %gpx;", *fn.Height))
	}
	if fn.MinWidth != nil {
		parts = append(parts, fmt.Sprintf("min-width: %gpx;", *fn.MinWidth))
	}
	if fn.MaxWidth != nil {
		parts = append(parts, fmt.Sprintf("max-width: %gpx;", *fn.MaxWidth))
	}
	if fn.MinHeight != nil {
		parts = append(parts, fmt.Sprintf("min-height: %gpx;", *fn.MinHeight))
	}
	if fn.MaxHeight != nil {
		parts = append(parts, fmt.Sprintf("max-height: %gpx;", *fn.MaxHeight))
	}

	// Z-index
	if fn.ZIndex != 0 {
		parts = append(parts, fmt.Sprintf("z-index: %d;", fn.ZIndex))
	}

	// Visibility
	if !fn.Visible {
		parts = append(parts, "visibility: hidden;")
	}

	// Flex grow
	if fn.Flex != nil {
		parts = append(parts, fmt.Sprintf("flex: %g;", *fn.Flex))
	}

	// Absolute positioning
	if fn.Position == "absolute" {
		parts = append(parts, "position: absolute;")
		if fn.X != nil {
			parts = append(parts, fmt.Sprintf("left: %gpx;", *fn.X))
		}
		if fn.Y != nil {
			parts = append(parts, fmt.Sprintf("top: %gpx;", *fn.Y))
		}
	}

	return strings.Join(parts, " ")
}

// --- Edges ---

func writeEdges(b *strings.Builder, edges []*scene.EdgeNode) {
	if len(edges) == 0 {
		return
	}

	b.WriteString("## Edges\n\n")
	b.WriteString("| From | To | Stroke | Width | Style | Arrow | Curve | Label |\n")
	b.WriteString("|------|----|--------|-------|-------|-------|-------|-------|\n")

	for _, e := range edges {
		label := "-"
		if e.Label != nil {
			label = fmt.Sprintf("`%s`", *e.Label)
		}
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | `%s` | %gpx | %s | %s | %s | %s |\n",
			e.FromID, e.ToID, e.Stroke, e.StrokeWidth, e.Style, e.Arrow, e.Curve, label))
	}

	b.WriteString("\n")

	// Path details
	b.WriteString("### Edge Paths\n\n")
	for _, e := range edges {
		labelStr := ""
		if e.Label != nil {
			labelStr = fmt.Sprintf(" (%s)", *e.Label)
		}
		b.WriteString(fmt.Sprintf("**%s → %s**%s\n", e.FromID, e.ToID, labelStr))
		b.WriteString(fmt.Sprintf("- Stroke: `%s` %gpx %s\n", e.Stroke, e.StrokeWidth, e.Style))
		if len(e.ResolvedPath) > 0 {
			points := make([]string, len(e.ResolvedPath))
			for i, p := range e.ResolvedPath {
				points[i] = fmt.Sprintf("(%.0f, %.0f)", p.X, p.Y)
			}
			b.WriteString(fmt.Sprintf("- Path: %s\n", strings.Join(points, " → ")))
		}
		b.WriteString("\n")
	}
}

// --- Implementation Notes ---

func writeImplementationNotes(b *strings.Builder) {
	b.WriteString("## Implementation Notes\n\n")

	b.WriteString("### DSL → CSS Property Mapping\n\n")
	b.WriteString("| DSL Property | CSS Equivalent |\n")
	b.WriteString("|-------------|----------------|\n")
	b.WriteString("| `direction: row` | `flex-direction: row` |\n")
	b.WriteString("| `direction: column` | `flex-direction: column` |\n")
	b.WriteString("| `justify: start` | `justify-content: flex-start` |\n")
	b.WriteString("| `justify: center` | `justify-content: center` |\n")
	b.WriteString("| `justify: end` | `justify-content: flex-end` |\n")
	b.WriteString("| `justify: between` | `justify-content: space-between` |\n")
	b.WriteString("| `justify: around` | `justify-content: space-around` |\n")
	b.WriteString("| `align: start` | `align-items: flex-start` |\n")
	b.WriteString("| `align: center` | `align-items: center` |\n")
	b.WriteString("| `align: end` | `align-items: flex-end` |\n")
	b.WriteString("| `align: stretch` | `align-items: stretch` |\n")
	b.WriteString("| `layout: grid` + `columns: N` | `display: grid; grid-template-columns: repeat(N, 1fr)` |\n")
	b.WriteString("| `fill: #color` | `background-color: #color` |\n")
	b.WriteString("| `fill: linear-gradient(...)` | `background: linear-gradient(...)` |\n")
	b.WriteString("| `border: W solid C` | `border: Wpx solid C` |\n")
	b.WriteString("| `shadow: X Y B C` | `box-shadow: Xpx Ypx Bpx C` |\n")
	b.WriteString("| `radius: N` | `border-radius: Npx` |\n")
	b.WriteString("| `clip: true` | `overflow: hidden` |\n")
	b.WriteString("| `truncate: true` | `overflow: hidden; text-overflow: ellipsis; white-space: nowrap` |\n")
	b.WriteString("| `gap: N` | `gap: Npx` |\n")
	b.WriteString("| `flex: N` | `flex: N` |\n")
	b.WriteString("| `opacity: N` | `opacity: N` |\n")
	b.WriteString("\n")
}

// --- Helpers ---

func justifyToCSS(j string) string {
	switch j {
	case "start":
		return "flex-start"
	case "end":
		return "flex-end"
	case "center":
		return "center"
	case "between":
		return "space-between"
	case "around":
		return "space-around"
	default:
		return j
	}
}

func alignToCSS(a string) string {
	switch a {
	case "start":
		return "flex-start"
	case "end":
		return "flex-end"
	case "center":
		return "center"
	case "stretch":
		return "stretch"
	default:
		return a
	}
}

func isGradient(fill string) bool {
	return strings.HasPrefix(fill, "linear-gradient(") || strings.HasPrefix(fill, "radial-gradient(")
}

func gradientToCSS(g *scene.Gradient) string {
	if g.Type == "radial" {
		stops := make([]string, len(g.Stops))
		for i, s := range g.Stops {
			stops[i] = fmt.Sprintf("%s %.0f%%", s.Color, s.Position*100)
		}
		return fmt.Sprintf("radial-gradient(circle, %s)", strings.Join(stops, ", "))
	}
	stops := make([]string, len(g.Stops))
	for i, s := range g.Stops {
		stops[i] = fmt.Sprintf("%s %.0f%%", s.Color, s.Position*100)
	}
	return fmt.Sprintf("linear-gradient(%gdeg, %s)", g.Angle, strings.Join(stops, ", "))
}

func borderToCSS(border string) string {
	parts := strings.Fields(border)
	if len(parts) >= 3 {
		return fmt.Sprintf("%spx %s %s", parts[0], parts[1], parts[2])
	}
	return border
}

func borderStructToCSS(b *scene.Border) string {
	return fmt.Sprintf("%gpx %s %s", b.Width, b.Style, b.Color)
}

func shadowToCSS(shadows []scene.Shadow) string {
	parts := make([]string, len(shadows))
	for i, s := range shadows {
		if s.Spread != 0 {
			parts[i] = fmt.Sprintf("%gpx %gpx %gpx %gpx %s", s.X, s.Y, s.Blur, s.Spread, s.Color)
		} else {
			parts[i] = fmt.Sprintf("%gpx %gpx %gpx %s", s.X, s.Y, s.Blur, s.Color)
		}
	}
	return strings.Join(parts, ", ")
}

func hasPadding(s scene.Spacing) bool {
	return s.Top > 0 || s.Right > 0 || s.Bottom > 0 || s.Left > 0
}

func spacingToStr(s scene.Spacing) string {
	if s.Top == s.Right && s.Right == s.Bottom && s.Bottom == s.Left {
		return fmt.Sprintf("%gpx", s.Top)
	}
	if s.Top == s.Bottom && s.Left == s.Right {
		return fmt.Sprintf("%gpx %gpx", s.Top, s.Left)
	}
	return fmt.Sprintf("%gpx %gpx %gpx %gpx", s.Top, s.Right, s.Bottom, s.Left)
}

func spacingToCSSValue(s scene.Spacing) string {
	return spacingToStr(s)
}

func spacingToCSS(p interface{}) string {
	switch v := p.(type) {
	case float64:
		return fmt.Sprintf("%gpx", v)
	case int:
		return fmt.Sprintf("%dpx", v)
	case string:
		parts := strings.Fields(v)
		for i, p := range parts {
			parts[i] = p + "px"
		}
		return strings.Join(parts, " ")
	default:
		return fmt.Sprintf("%v", p)
	}
}

func toFloat(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case float32:
		return float64(n), true
	default:
		return 0, false
	}
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
