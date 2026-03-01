// Package render converts a laid-out scene graph to SVG.
package render

import (
	"fmt"
	"html"
	"math"
	"sort"
	"strings"

	"github.com/fxck/rendspec/internal/fonts"
	"github.com/fxck/rendspec/internal/scene"
)

// fmtN formats a float cleanly for SVG output.
func fmtN(v float64) string {
	r := math.Round(v*100) / 100
	if r == math.Trunc(r) {
		return fmt.Sprintf("%d", int(r))
	}
	return fmt.Sprintf("%.2f", r)
}

// RenderSVG renders a fully laid-out scene graph to an SVG string.
func RenderSVG(sg *scene.SceneGraph) string {
	root := sg.Root
	w := root.Layout.Width
	h := root.Layout.Height

	var b strings.Builder
	var defs strings.Builder
	counter := 0

	// Collect all defs first.
	// INVARIANT: The defs pass and render pass must traverse nodes in identical
	// order and increment the counter identically so that generated IDs (shadow-N,
	// grad-N, arrow-N, clip-N, etc.) match between <defs> and usage sites.
	collectDefs(root, &defs, &counter)
	for _, edge := range sg.Edges {
		collectEdgeDefs(edge, &defs, &counter)
	}

	b.WriteString(fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="%s" height="%s" viewBox="0 0 %s %s">`+"\n",
		fmtN(w), fmtN(h), fmtN(w), fmtN(h),
	))

	fontFamily := sg.Theme.FontFamily
	if fontFamily == "" {
		fontFamily = "Inter"
	}
	b.WriteString("<style>\n")
	fmt.Fprintf(&b, "  text { font-family: \"%s\", -apple-system, BlinkMacSystemFont, \"Segoe UI\", Roboto, sans-serif; }\n", fontFamily)
	b.WriteString("</style>\n")

	if defs.Len() > 0 {
		b.WriteString("<defs>\n")
		b.WriteString(defs.String())
		b.WriteString("</defs>\n")
	}

	// Render root
	counter = 0
	renderFrame(root, &b, 0, 0, &counter)

	// Render edges — counter continues from the defs pass so marker IDs match
	for _, edge := range sg.Edges {
		renderEdge(edge, &b, &counter)
	}

	b.WriteString("</svg>")
	return b.String()
}

// --- Defs ---

func collectDefs(node *scene.FrameNode, defs *strings.Builder, counter *int) {
	if !node.Visible {
		return
	}
	// Clip counter (emits nothing to defs, but keeps counter in sync with renderFrame)
	if node.Clip {
		*counter++
	}
	if len(node.Shadow) > 0 {
		*counter++
		makeCompositeShadowFilter(node.Shadow, *counter, defs)
	}
	if node.Gradient != nil {
		*counter++
		makeGradientDef(node.Gradient, *counter, defs)
	}
	// Image clip counter (emits nothing to defs, but keeps counter in sync with renderImage)
	if node.Image != nil && node.Radius > 0 {
		*counter++
	}
	for _, child := range sortedChildren(node) {
		if fn, ok := child.(*scene.FrameNode); ok {
			collectDefs(fn, defs, counter)
		}
	}
}

// makeCompositeShadowFilter emits a single <filter> containing all shadows as merged layers.
// This ensures one counter increment per frame regardless of shadow count.
func makeCompositeShadowFilter(shadows []scene.Shadow, idx int, defs *strings.Builder) {
	fmt.Fprintf(defs, "  <filter id=\"shadow-%d\" x=\"-50%%\" y=\"-50%%\" width=\"200%%\" height=\"200%%\">\n", idx)

	if len(shadows) == 1 && shadows[0].Spread == 0 {
		// Simple case: single shadow without spread
		s := shadows[0]
		stdDev := s.Blur / 2
		fmt.Fprintf(defs, "    <feDropShadow dx=\"%g\" dy=\"%g\" stdDeviation=\"%g\" flood-color=\"%s\" flood-opacity=\"1\"/>\n",
			s.X, s.Y, stdDev, s.Color)
	} else {
		// Multiple shadows or spread: composite via feMerge
		for i, s := range shadows {
			stdDev := s.Blur / 2
			blurResult := fmt.Sprintf("blur%d", i)
			shadowResult := fmt.Sprintf("shadow%d", i)

			fmt.Fprintf(defs, "    <feGaussianBlur in=\"SourceAlpha\" stdDeviation=\"%g\" result=\"%s\"/>\n", stdDev, blurResult)

			offsetIn := blurResult
			if s.Spread > 0 {
				spreadResult := fmt.Sprintf("spread%d", i)
				fmt.Fprintf(defs, "    <feMorphology in=\"%s\" operator=\"dilate\" radius=\"%g\" result=\"%s\"/>\n", blurResult, s.Spread, spreadResult)
				offsetIn = spreadResult
			}

			offsetResult := fmt.Sprintf("offset%d", i)
			fmt.Fprintf(defs, "    <feOffset in=\"%s\" dx=\"%g\" dy=\"%g\" result=\"%s\"/>\n", offsetIn, s.X, s.Y, offsetResult)

			colorResult := fmt.Sprintf("color%d", i)
			fmt.Fprintf(defs, "    <feFlood flood-color=\"%s\" flood-opacity=\"1\" result=\"%s\"/>\n", s.Color, colorResult)

			fmt.Fprintf(defs, "    <feComposite in=\"%s\" in2=\"%s\" operator=\"in\" result=\"%s\"/>\n", colorResult, offsetResult, shadowResult)
		}

		defs.WriteString("    <feMerge>\n")
		for i := range shadows {
			fmt.Fprintf(defs, "      <feMergeNode in=\"shadow%d\"/>\n", i)
		}
		defs.WriteString("      <feMergeNode in=\"SourceGraphic\"/>\n")
		defs.WriteString("    </feMerge>\n")
	}

	defs.WriteString("  </filter>\n")
}

func makeGradientDef(gradient *scene.Gradient, idx int, defs *strings.Builder) {
	if gradient.Type == "radial" {
		fmt.Fprintf(defs,
			"  <radialGradient id=\"grad-%d\" cx=\"%s\" cy=\"%s\" r=\"0.5\" fx=\"%s\" fy=\"%s\" gradientUnits=\"objectBoundingBox\">\n",
			idx, fmtN(gradient.CX), fmtN(gradient.CY), fmtN(gradient.CX), fmtN(gradient.CY),
		)
	} else {
		angleRad := gradient.Angle * math.Pi / 180
		x1 := 0.5 - 0.5*math.Sin(angleRad)
		y1 := 0.5 + 0.5*math.Cos(angleRad)
		x2 := 0.5 + 0.5*math.Sin(angleRad)
		y2 := 0.5 - 0.5*math.Cos(angleRad)
		fmt.Fprintf(defs,
			"  <linearGradient id=\"grad-%d\" x1=\"%s\" y1=\"%s\" x2=\"%s\" y2=\"%s\" gradientUnits=\"objectBoundingBox\">\n",
			idx, fmtN(x1), fmtN(y1), fmtN(x2), fmtN(y2),
		)
	}

	for _, stop := range gradient.Stops {
		fmt.Fprintf(defs, "    <stop offset=\"%s%%\" stop-color=\"%s\"/>\n",
			fmtN(stop.Position*100), stop.Color)
	}

	if gradient.Type == "radial" {
		defs.WriteString("  </radialGradient>\n")
	} else {
		defs.WriteString("  </linearGradient>\n")
	}
}

func collectEdgeDefs(edge *scene.EdgeNode, defs *strings.Builder, counter *int) {
	if edge.Arrow == "end" || edge.Arrow == "both" {
		*counter++
		fmt.Fprintf(defs,
			"  <marker id=\"arrow-%d\" viewBox=\"0 0 10 10\" refX=\"10\" refY=\"5\" markerWidth=\"8\" markerHeight=\"8\" orient=\"auto-start-reverse\">\n"+
				"    <path d=\"M 0 0 L 10 5 L 0 10 z\" fill=\"%s\"/>\n"+
				"  </marker>\n",
			*counter, edge.Stroke,
		)
	}
	if edge.Arrow == "start" || edge.Arrow == "both" {
		*counter++
		fmt.Fprintf(defs,
			"  <marker id=\"arrow-start-%d\" viewBox=\"0 0 10 10\" refX=\"0\" refY=\"5\" markerWidth=\"8\" markerHeight=\"8\" orient=\"auto-start-reverse\">\n"+
				"    <path d=\"M 10 0 L 0 5 L 10 10 z\" fill=\"%s\"/>\n"+
				"  </marker>\n",
			*counter, edge.Stroke,
		)
	}
}

// --- Z-index sorting ---

func sortedChildren(node *scene.FrameNode) []scene.Node {
	if len(node.Children) == 0 {
		return nil
	}
	hasZ := false
	for _, c := range node.Children {
		if fn, ok := c.(*scene.FrameNode); ok && fn.ZIndex != 0 {
			hasZ = true
			break
		}
	}
	if !hasZ {
		return node.Children
	}
	sorted := make([]scene.Node, len(node.Children))
	copy(sorted, node.Children)
	sort.SliceStable(sorted, func(i, j int) bool {
		zi, zj := 0, 0
		if fn, ok := sorted[i].(*scene.FrameNode); ok {
			zi = fn.ZIndex
		}
		if fn, ok := sorted[j].(*scene.FrameNode); ok {
			zj = fn.ZIndex
		}
		return zi < zj
	})
	return sorted
}

// --- Frame Rendering ---

// We need a global counter for unique IDs that persists across the render pass.
// The defs pass uses its own counter, so we re-use counter for the render pass.

func renderFrame(node *scene.FrameNode, b *strings.Builder, parentX, parentY float64, counter *int) {
	if !node.Visible {
		return
	}

	absX := parentX + node.Layout.X
	absY := parentY + node.Layout.Y
	w := node.Layout.Width
	h := node.Layout.Height

	opacityAttr := ""
	if node.Opacity < 1 {
		opacityAttr = fmt.Sprintf(" opacity=\"%g\"", node.Opacity)
	}

	clipAttr := ""
	if node.Clip {
		*counter++
		clipID := fmt.Sprintf("clip-%d", *counter)
		fmt.Fprintf(b, "<clipPath id=\"%s\"><rect x=\"%s\" y=\"%s\" width=\"%s\" height=\"%s\" rx=\"%s\"/></clipPath>\n",
			clipID, fmtN(absX), fmtN(absY), fmtN(w), fmtN(h), fmtN(node.Radius))
		clipAttr = fmt.Sprintf(" clip-path=\"url(#%s)\"", clipID)
	}

	fmt.Fprintf(b, "<g%s%s>\n", opacityAttr, clipAttr)

	// Shadow filter
	shadowFilter := ""
	if len(node.Shadow) > 0 {
		*counter++
		shadowFilter = fmt.Sprintf(" filter=\"url(#shadow-%d)\"", *counter)
	}

	// Determine fill value
	fillValue := "none"
	if node.Gradient != nil {
		*counter++
		fillValue = fmt.Sprintf("url(#grad-%d)", *counter)
	} else if node.Fill != nil {
		fillValue = *node.Fill
	}

	// Render the frame shape
	if fillValue != "none" || node.Border != nil || len(node.Shadow) > 0 {
		stroke := ""
		if node.Border != nil {
			stroke = fmt.Sprintf(" stroke=\"%s\" stroke-width=\"%s\"", node.Border.Color, fmtN(node.Border.Width))
			if node.Border.Style == "dashed" {
				stroke += " stroke-dasharray=\"8 4\""
			} else if node.Border.Style == "dotted" {
				stroke += " stroke-dasharray=\"3 3\""
			}
		}

		switch node.Shape {
		case "circle", "ellipse":
			cx := absX + w/2
			cy := absY + h/2
			if node.Shape == "circle" {
				r := math.Min(w, h) / 2
				fmt.Fprintf(b, "<circle cx=\"%s\" cy=\"%s\" r=\"%s\" fill=\"%s\"%s%s/>\n",
					fmtN(cx), fmtN(cy), fmtN(r), fillValue, stroke, shadowFilter)
			} else {
				fmt.Fprintf(b, "<ellipse cx=\"%s\" cy=\"%s\" rx=\"%s\" ry=\"%s\" fill=\"%s\"%s%s/>\n",
					fmtN(cx), fmtN(cy), fmtN(w/2), fmtN(h/2), fillValue, stroke, shadowFilter)
			}
		case "diamond":
			cx := absX + w/2
			cy := absY + h/2
			pts := fmt.Sprintf("%s,%s %s,%s %s,%s %s,%s",
				fmtN(cx), fmtN(absY), fmtN(absX+w), fmtN(cy),
				fmtN(cx), fmtN(absY+h), fmtN(absX), fmtN(cy))
			fmt.Fprintf(b, "<polygon points=\"%s\" fill=\"%s\"%s%s/>\n",
				pts, fillValue, stroke, shadowFilter)
		default: // rect
			rx := ""
			if node.Radius > 0 {
				rx = fmt.Sprintf(" rx=\"%s\"", fmtN(node.Radius))
			}
			fmt.Fprintf(b, "<rect x=\"%s\" y=\"%s\" width=\"%s\" height=\"%s\" fill=\"%s\"%s%s%s/>\n",
				fmtN(absX), fmtN(absY), fmtN(w), fmtN(h), fillValue, rx, stroke, shadowFilter)
		}
	}

	// Render image
	if node.Image != nil {
		renderImage(node, absX, absY, w, h, b, counter)
	}

	// Render side borders (only for rect shapes)
	if node.Shape == "" || node.Shape == "rect" {
		renderSideBorders(node, absX, absY, w, h, b)
	}

	// Render children
	for _, child := range sortedChildren(node) {
		if tn, ok := child.(*scene.TextNode); ok {
			renderText(tn, b, absX, absY)
		} else if fn, ok := child.(*scene.FrameNode); ok {
			renderFrame(fn, b, absX, absY, counter)
		}
	}

	b.WriteString("</g>\n")
}

func renderImage(node *scene.FrameNode, x, y, w, h float64, b *strings.Builder, counter *int) {
	preserve := "xMidYMid slice" // cover
	switch node.ImageFit {
	case "contain":
		preserve = "xMidYMid meet"
	case "fill":
		preserve = "none"
	case "none":
		preserve = "xMinYMin meet" // no scaling, top-left aligned
	}

	clip := ""
	if node.Radius > 0 {
		*counter++
		clipID := fmt.Sprintf("img-clip-%d", *counter)
		fmt.Fprintf(b, "<clipPath id=\"%s\"><rect x=\"%s\" y=\"%s\" width=\"%s\" height=\"%s\" rx=\"%s\"/></clipPath>\n",
			clipID, fmtN(x), fmtN(y), fmtN(w), fmtN(h), fmtN(node.Radius))
		clip = fmt.Sprintf(" clip-path=\"url(#%s)\"", clipID)
	}

	fmt.Fprintf(b, "<image x=\"%s\" y=\"%s\" width=\"%s\" height=\"%s\" href=\"%s\" preserveAspectRatio=\"%s\"%s/>\n",
		fmtN(x), fmtN(y), fmtN(w), fmtN(h), html.EscapeString(*node.Image), preserve, clip)
}

func renderSideBorders(node *scene.FrameNode, x, y, w, h float64, b *strings.Builder) {
	borders := []struct {
		border     *scene.Border
		x1, y1, x2, y2 float64
	}{
		{node.BorderTop, x, y, x + w, y},
		{node.BorderRight, x + w, y, x + w, y + h},
		{node.BorderBottom, x, y + h, x + w, y + h},
		{node.BorderLeft, x, y, x, y + h},
	}

	for _, bd := range borders {
		if bd.border == nil {
			continue
		}
		dash := ""
		if bd.border.Style == "dashed" {
			dash = " stroke-dasharray=\"8 4\""
		} else if bd.border.Style == "dotted" {
			dash = " stroke-dasharray=\"3 3\""
		}
		fmt.Fprintf(b, "<line x1=\"%s\" y1=\"%s\" x2=\"%s\" y2=\"%s\" stroke=\"%s\" stroke-width=\"%s\"%s/>\n",
			fmtN(bd.x1), fmtN(bd.y1), fmtN(bd.x2), fmtN(bd.y2),
			bd.border.Color, fmtN(bd.border.Width), dash)
	}
}

func renderText(node *scene.TextNode, b *strings.Builder, parentX, parentY float64) {
	absX := parentX + node.Layout.X
	absY := parentY + node.Layout.Y

	family := node.Font.Family
	if strings.Contains(family, " ") {
		family = "\"" + family + "\""
	}

	opacityAttr := ""
	if node.Opacity < 1 {
		opacityAttr = fmt.Sprintf(" opacity=\"%g\"", node.Opacity)
	}

	anchor := "start"
	textX := absX
	switch node.TextAlign {
	case "center":
		anchor = "middle"
		textX = absX + node.Layout.Width/2
	case "right":
		anchor = "end"
		textX = absX + node.Layout.Width
	}

	decoration := ""
	switch node.TextDecoration {
	case "underline":
		decoration = " text-decoration=\"underline\""
	case "strikethrough":
		decoration = " text-decoration=\"line-through\""
	}

	fallback := "sans-serif"
	fl := strings.ToLower(node.Font.Family)
	if strings.Contains(fl, "mono") || strings.Contains(fl, "code") ||
		strings.Contains(fl, "courier") || strings.Contains(fl, "consolas") ||
		strings.Contains(fl, "inconsolata") || strings.Contains(fl, "menlo") ||
		strings.Contains(fl, "hack") {
		fallback = "monospace"
	} else if strings.Contains(fl, "serif") && !strings.Contains(fl, "sans") {
		fallback = "serif"
	} else if strings.Contains(fl, "georgia") || strings.Contains(fl, "times") ||
		strings.Contains(fl, "garamond") || strings.Contains(fl, "merriweather") ||
		strings.Contains(fl, "playfair") || strings.Contains(fl, "lora") {
		fallback = "serif"
	}

	letterSp := ""
	if node.LetterSpacing != 0 {
		letterSp = fmt.Sprintf(" letter-spacing=\"%s\"", fmtN(node.LetterSpacing))
	}

	content := node.Content
	fontSize := node.Font.Size
	lineHeight := node.LineHeight
	if lineHeight == 0 {
		lineHeight = 1.4
	}

	// Handle truncation: if truncate is set and text exceeds max-width, clip with ellipsis
	if node.Truncate && node.MaxWidth != nil {
		textW := fonts.MeasureTextWidth(content, fontSize, node.Font.Family, node.Font.Weight, node.LetterSpacing)
		if textW > *node.MaxWidth {
			content = truncateText(content, *node.MaxWidth, fontSize, node.Font.Family, node.Font.Weight, node.LetterSpacing)
		}
	}

	// Handle multi-line text: if max-width is set and text exceeds it, wrap into lines
	if node.MaxWidth != nil && !node.Truncate {
		textW := fonts.MeasureTextWidth(content, fontSize, node.Font.Family, node.Font.Weight, node.LetterSpacing)
		if textW > *node.MaxWidth {
			lines := wrapTextToLines(content, *node.MaxWidth, fontSize, node.Font.Family, node.Font.Weight, node.LetterSpacing)
			if len(lines) > 1 {
				baselineY := absY + fontSize*0.82
				fmt.Fprintf(b, "<text x=\"%s\" y=\"%s\" font-family='%s, %s' font-size=\"%s\" font-weight=\"%d\" fill=\"%s\" text-anchor=\"%s\"%s%s%s>",
					fmtN(textX), fmtN(baselineY), family, fallback,
					fmtN(fontSize), node.Font.Weight, node.Color, anchor,
					opacityAttr, decoration, letterSp)
				for i, line := range lines {
					escaped := html.EscapeString(line)
					if i == 0 {
						fmt.Fprintf(b, "<tspan x=\"%s\">%s</tspan>", fmtN(textX), escaped)
					} else {
						fmt.Fprintf(b, "<tspan x=\"%s\" dy=\"%s\">%s</tspan>",
							fmtN(textX), fmtN(fontSize*lineHeight), escaped)
					}
				}
				b.WriteString("</text>\n")
				return
			}
		}
	}

	// Single-line rendering
	baselineY := absY + fontSize*0.82
	escaped := html.EscapeString(content)
	fmt.Fprintf(b, "<text x=\"%s\" y=\"%s\" font-family='%s, %s' font-size=\"%s\" font-weight=\"%d\" fill=\"%s\" text-anchor=\"%s\"%s%s%s>%s</text>\n",
		fmtN(textX), fmtN(baselineY), family, fallback,
		fmtN(fontSize), node.Font.Weight, node.Color, anchor,
		opacityAttr, decoration, letterSp, escaped)
}

// truncateText truncates text to fit within maxWidth, appending "..." if needed.
func truncateText(text string, maxWidth, fontSize float64, family string, weight int, letterSpacing float64) string {
	ellipsis := "..."
	ellipsisW := fonts.MeasureTextWidth(ellipsis, fontSize, family, weight, letterSpacing)
	availW := maxWidth - ellipsisW
	if availW <= 0 {
		return ellipsis
	}

	runes := []rune(text)
	for i := len(runes); i > 0; i-- {
		sub := string(runes[:i])
		w := fonts.MeasureTextWidth(sub, fontSize, family, weight, letterSpacing)
		if w <= availW {
			return sub + ellipsis
		}
	}
	return ellipsis
}

// wrapTextToLines splits text into lines that fit within maxWidth.
func wrapTextToLines(text string, maxWidth, fontSize float64, family string, weight int, letterSpacing float64) []string {
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}

	var lines []string
	currentLine := words[0]

	for _, word := range words[1:] {
		testLine := currentLine + " " + word
		w := fonts.MeasureTextWidth(testLine, fontSize, family, weight, letterSpacing)
		if w > maxWidth && currentLine != "" {
			lines = append(lines, currentLine)
			currentLine = word
		} else {
			currentLine = testLine
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}
	return lines
}

// interpolatePath returns the (x, y) at parameter t (0-1) along a polyline path,
// using actual segment lengths for accurate positioning.
func interpolatePath(path []scene.Point, t float64) (float64, float64) {
	if len(path) < 2 {
		if len(path) == 1 {
			return path[0].X, path[0].Y
		}
		return 0, 0
	}
	if len(path) == 2 {
		return path[0].X + t*(path[1].X-path[0].X),
			path[0].Y + t*(path[1].Y-path[0].Y)
	}

	// Compute total path length
	totalLen := 0.0
	for i := 1; i < len(path); i++ {
		totalLen += math.Hypot(path[i].X-path[i-1].X, path[i].Y-path[i-1].Y)
	}
	if totalLen == 0 {
		return path[0].X, path[0].Y
	}

	// Walk segments to find the point at t * totalLen
	target := t * totalLen
	walked := 0.0
	for i := 1; i < len(path); i++ {
		segLen := math.Hypot(path[i].X-path[i-1].X, path[i].Y-path[i-1].Y)
		if walked+segLen >= target {
			frac := 0.0
			if segLen > 0 {
				frac = (target - walked) / segLen
			}
			return path[i-1].X + frac*(path[i].X-path[i-1].X),
				path[i-1].Y + frac*(path[i].Y-path[i-1].Y)
		}
		walked += segLen
	}

	last := path[len(path)-1]
	return last.X, last.Y
}

// --- Edge Rendering ---

const edgeCornerRadius = 12

func buildRoundedPath(points []scene.Point, radius float64) string {
	n := len(points)
	if n < 2 {
		return ""
	}
	if n == 2 {
		return fmt.Sprintf("M %s %s L %s %s",
			fmtN(points[0].X), fmtN(points[0].Y),
			fmtN(points[1].X), fmtN(points[1].Y))
	}

	var parts []string
	parts = append(parts, fmt.Sprintf("M %s %s", fmtN(points[0].X), fmtN(points[0].Y)))

	for i := 1; i < n-1; i++ {
		prev := points[i-1]
		curr := points[i]
		nxt := points[i+1]

		dx1, dy1 := curr.X-prev.X, curr.Y-prev.Y
		dx2, dy2 := nxt.X-curr.X, nxt.Y-curr.Y
		len1 := math.Hypot(dx1, dy1)
		len2 := math.Hypot(dx2, dy2)

		if len1 == 0 || len2 == 0 {
			parts = append(parts, fmt.Sprintf("L %s %s", fmtN(curr.X), fmtN(curr.Y)))
			continue
		}

		r := math.Min(radius, math.Min(len1/2, len2/2))

		bx := curr.X - r*dx1/len1
		by := curr.Y - r*dy1/len1
		ax := curr.X + r*dx2/len2
		ay := curr.Y + r*dy2/len2

		parts = append(parts, fmt.Sprintf("L %s %s", fmtN(bx), fmtN(by)))
		parts = append(parts, fmt.Sprintf("Q %s %s %s %s", fmtN(curr.X), fmtN(curr.Y), fmtN(ax), fmtN(ay)))
	}

	last := points[n-1]
	parts = append(parts, fmt.Sprintf("L %s %s", fmtN(last.X), fmtN(last.Y)))
	return strings.Join(parts, " ")
}

// renderEdge renders an edge path with optional arrow markers and label.
// The counter must be the same pointer used during collectEdgeDefs so that
// marker IDs in <defs> match the references in the rendered path.
func renderEdge(edge *scene.EdgeNode, b *strings.Builder, counter *int) {
	if len(edge.ResolvedPath) < 2 {
		return
	}

	radius := float64(edgeCornerRadius)
	if edge.CornerRadius != nil {
		radius = *edge.CornerRadius
	}
	d := buildRoundedPath(edge.ResolvedPath, radius)

	dash := ""
	if edge.Style == "dashed" {
		dash = " stroke-dasharray=\"8 4\""
	} else if edge.Style == "dotted" {
		dash = " stroke-dasharray=\"3 3\""
	}

	markerEnd := ""
	markerStart := ""
	if edge.Arrow == "end" || edge.Arrow == "both" {
		*counter++
		markerEnd = fmt.Sprintf(" marker-end=\"url(#arrow-%d)\"", *counter)
	}
	if edge.Arrow == "start" || edge.Arrow == "both" {
		*counter++
		markerStart = fmt.Sprintf(" marker-start=\"url(#arrow-start-%d)\"", *counter)
	}

	fmt.Fprintf(b, "<path d=\"%s\" fill=\"none\" stroke=\"%s\" stroke-width=\"%s\" stroke-linecap=\"round\" stroke-linejoin=\"round\"%s%s%s/>\n",
		d, edge.Stroke, fmtN(edge.StrokeWidth), dash, markerEnd, markerStart)

	// Render label
	if edge.Label != nil && *edge.Label != "" {
		t := edge.LabelPosition
		path := edge.ResolvedPath
		lx, ly := interpolatePath(path, t)

		font := edge.LabelFont
		labelText := html.EscapeString(*edge.Label)
		fmt.Fprintf(b, "<text x=\"%s\" y=\"%s\" font-family=\"%s, sans-serif\" font-size=\"%s\" font-weight=\"%d\" fill=\"%s\" text-anchor=\"middle\">%s</text>\n",
			fmtN(lx), fmtN(ly-font.Size*0.5), font.Family, fmtN(font.Size), font.Weight, edge.LabelColor, labelText)
	}
}
