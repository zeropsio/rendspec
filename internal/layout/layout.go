// Package layout implements a flexbox + grid layout engine.
package layout

import (
	"math"
	"strings"

	"github.com/fxck/rendspec/internal/fonts"
	"github.com/fxck/rendspec/internal/scene"
)

// ComputeLayout computes layout for the entire scene graph.
func ComputeLayout(sg *scene.SceneGraph) {
	root := sg.Root
	w := 1280.0
	h := 720.0
	if root.Width != nil {
		w = *root.Width
	}
	if root.Height != nil {
		h = *root.Height
	}
	layoutNode(root, 0, 0, w, h, false)

	// Resolve edge paths
	frameMap := buildFrameMap(root, 0, 0)

	// Group bus edges by (source, anchor) for shared junction routing
	busGroups := make(map[[2]string][]*scene.EdgeNode)
	var otherEdges []*scene.EdgeNode
	for _, edge := range sg.Edges {
		if edge.Curve == "bus" {
			key := [2]string{edge.FromID, edge.FromAnchor}
			busGroups[key] = append(busGroups[key], edge)
		} else {
			otherEdges = append(otherEdges, edge)
		}
	}

	for _, group := range busGroups {
		if len(group) > 1 {
			resolveBusGroup(group, frameMap)
		} else {
			resolveEdge(group[0], frameMap)
		}
	}

	for _, edge := range otherEdges {
		resolveEdge(edge, frameMap)
	}
}

// MeasureText returns approximate (width, height) of a text node.
// Uses word-based wrapping to match the renderer's wrapTextToLines behavior.
func MeasureText(node *scene.TextNode) (float64, float64) {
	font := node.Font
	textW := fonts.MeasureTextWidth(node.Content, font.Size, font.Family, font.Weight, node.LetterSpacing)
	textH := font.Size * node.LineHeight

	if node.MaxWidth != nil && textW > *node.MaxWidth {
		// Word-based wrapping (matches renderer's wrapTextToLines)
		lines := countWrappedLines(node.Content, *node.MaxWidth, font.Size, font.Family, font.Weight, node.LetterSpacing)
		textW = *node.MaxWidth
		textH = font.Size * node.LineHeight * float64(lines)
	}

	return textW, textH
}

// countWrappedLines counts how many lines text will occupy using word-based wrapping.
// Falls back to character-width estimation for words exceeding maxWidth.
func countWrappedLines(text string, maxWidth, fontSize float64, family string, weight int, letterSpacing float64) int {
	if maxWidth <= 0 {
		return 1
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return 1
	}

	lines := 0
	currentLine := ""

	for _, word := range words {
		wordW := fonts.MeasureTextWidth(word, fontSize, family, weight, letterSpacing)
		if currentLine == "" {
			// Word exceeds maxWidth on its own — use character-based wrapping for this word
			if wordW > maxWidth {
				lines += int(math.Max(1, math.Ceil(wordW/maxWidth)))
				currentLine = ""
				continue
			}
			currentLine = word
			continue
		}

		testLine := currentLine + " " + word
		w := fonts.MeasureTextWidth(testLine, fontSize, family, weight, letterSpacing)
		if w > maxWidth {
			lines++
			// Start new line — check if word itself overflows
			if wordW > maxWidth {
				lines += int(math.Max(1, math.Ceil(wordW/maxWidth)))
				currentLine = ""
			} else {
				currentLine = word
			}
		} else {
			currentLine = testLine
		}
	}
	if currentLine != "" {
		lines++
	}
	return max(1, lines)
}

// intrinsicSize computes the intrinsic (content-driven) size of a node.
func intrinsicSize(node scene.Node, availW, availH float64) (float64, float64) {
	if tn, ok := node.(*scene.TextNode); ok {
		return MeasureText(tn)
	}

	fn, ok := node.(*scene.FrameNode)
	if !ok {
		return 0, 0
	}

	w := optFloat(fn.Width)
	h := optFloat(fn.Height)

	padH := fn.Padding.Horizontal()
	padV := fn.Padding.Vertical()

	var flow []scene.Node
	for _, c := range fn.Children {
		if f, ok := c.(*scene.FrameNode); ok && f.Position == "absolute" {
			continue
		}
		flow = append(flow, c)
	}

	if len(flow) == 0 {
		rw := w
		rh := h
		if rw == 0 {
			rw = padH
		}
		if rh == 0 {
			rh = padV
		}
		return rw, rh
	}

	contentW := availW - padH
	if w > 0 {
		contentW = w - padH
	}
	if contentW < 0 {
		contentW = 0
	}

	// Grid layout
	if fn.LayoutMode == "grid" && fn.Columns != nil {
		return intrinsicSizeGrid(fn, flow, contentW, availH, padH, padV, w, h)
	}

	isRow := fn.Direction == "row"
	childSizes := make([][2]float64, len(flow))
	for i, c := range flow {
		cw, ch := intrinsicSize(c, contentW, availH)
		childSizes[i] = [2]float64{cw, ch}
	}

	totalGap := fn.Gap * math.Max(0, float64(len(flow)-1))

	// Include child margins
	marginMain := 0.0
	marginCross := 0.0
	for _, c := range flow {
		if f, ok := c.(*scene.FrameNode); ok {
			if isRow {
				marginMain += f.Margin.Left + f.Margin.Right
				marginCross = math.Max(marginCross, f.Margin.Top+f.Margin.Bottom)
			} else {
				marginMain += f.Margin.Top + f.Margin.Bottom
				marginCross = math.Max(marginCross, f.Margin.Left+f.Margin.Right)
			}
		}
	}

	var iw, ih float64
	if isRow {
		mainTotal := totalGap + marginMain
		crossMax := marginCross
		for _, s := range childSizes {
			mainTotal += s[0]
			if s[1] > crossMax-marginCross {
				crossMax = s[1] + marginCross
			}
		}
		iw = mainTotal + padH
		ih = crossMax + padV
	} else {
		crossMax := marginCross
		mainTotal := totalGap + marginMain
		for _, s := range childSizes {
			mainTotal += s[1]
			if s[0] > crossMax-marginCross {
				crossMax = s[0] + marginCross
			}
		}
		iw = crossMax + padH
		ih = mainTotal + padV
	}

	if w > 0 {
		iw = w
	}
	if h > 0 {
		ih = h
	}
	return iw, ih
}

func intrinsicSizeGrid(node *scene.FrameNode, flow []scene.Node, contentW, availH, padH, padV, w, h float64) (float64, float64) {
	cols := 1
	if node.Columns != nil {
		cols = *node.Columns
	}
	if cols < 1 {
		cols = 1
	}
	colGap := node.Gap
	if node.ColumnGap != nil {
		colGap = *node.ColumnGap
	}
	rowGap := node.Gap
	if node.RowGap != nil {
		rowGap = *node.RowGap
	}

	n := len(flow)
	numRows := (n + cols - 1) / cols

	colW := 0.0
	if contentW > 0 {
		colW = (contentW - colGap*math.Max(0, float64(cols-1))) / float64(cols)
	}

	childSizes := make([][2]float64, n)
	for i, c := range flow {
		cw, ch := intrinsicSize(c, colW, availH)
		childSizes[i] = [2]float64{cw, ch}
	}

	rowHeights := make([]float64, numRows)
	for r := range numRows {
		rowStart := r * cols
		rowEnd := min(rowStart+cols, n)
		for i := rowStart; i < rowEnd; i++ {
			if childSizes[i][1] > rowHeights[r] {
				rowHeights[r] = childSizes[i][1]
			}
		}
	}

	totalH := padV + rowGap*math.Max(0, float64(numRows-1))
	for _, rh := range rowHeights {
		totalH += rh
	}

	totalW := contentW + padH
	if contentW <= 0 {
		maxCW := 0.0
		for i := range n {
			if childSizes[i][0] > maxCW {
				maxCW = childSizes[i][0]
			}
		}
		totalW = maxCW*float64(cols) + colGap*math.Max(0, float64(cols-1)) + padH
	}

	if w > 0 {
		totalW = w
	}
	if h > 0 {
		totalH = h
	}
	return totalW, totalH
}

func layoutNode(node scene.Node, x, y, availW, availH float64, stretched bool) {
	if tn, ok := node.(*scene.TextNode); ok {
		tw, th := MeasureText(tn)
		tn.Layout.X = x
		tn.Layout.Y = y
		if availW > tw {
			tn.Layout.Width = availW
		} else {
			tn.Layout.Width = tw
		}
		tn.Layout.Height = th
		return
	}

	fn, ok := node.(*scene.FrameNode)
	if !ok {
		return
	}

	w := availW
	if fn.Width != nil {
		w = *fn.Width
	}
	h := availH
	if fn.Height != nil {
		h = *fn.Height
	}

	// Clamp to min/max (CSS spec: min wins over max when they conflict)
	if fn.MaxWidth != nil && w > *fn.MaxWidth {
		w = *fn.MaxWidth
	}
	if fn.MinWidth != nil && w < *fn.MinWidth {
		w = *fn.MinWidth
	}
	if fn.MaxHeight != nil && h > *fn.MaxHeight {
		h = *fn.MaxHeight
	}
	if fn.MinHeight != nil && h < *fn.MinHeight {
		h = *fn.MinHeight
	}

	fn.Layout.X = x
	fn.Layout.Y = y
	fn.Layout.Width = w
	fn.Layout.Height = h

	if len(fn.Children) == 0 {
		if fn.Height == nil && !stretched {
			h := fn.Padding.Vertical()
			if fn.MinHeight != nil && h < *fn.MinHeight {
				h = *fn.MinHeight
			}
			fn.Layout.Height = h
		}
		return
	}

	pad := fn.Padding
	contentX := pad.Left
	contentY := pad.Top
	contentW := math.Max(0, w-pad.Horizontal())
	contentH := math.Max(0, h-pad.Vertical())

	var flow []scene.Node
	var absolute []*scene.FrameNode
	for _, c := range fn.Children {
		if f, ok := c.(*scene.FrameNode); ok && f.Position == "absolute" {
			absolute = append(absolute, f)
		} else {
			flow = append(flow, c)
		}
	}

	if len(flow) > 0 {
		if fn.LayoutMode == "grid" && fn.Columns != nil {
			layoutGridChildren(fn, flow, contentX, contentY, contentW, contentH)
		} else {
			layoutFlowChildren(fn, flow, contentX, contentY, contentW, contentH)
		}

		// If auto height and NOT stretched by parent, recompute from children
		if fn.Height == nil && !stretched {
			recomputeAutoHeight(fn, flow)
			// Re-apply min/max constraints after auto-height computation
			if fn.MinHeight != nil && fn.Layout.Height < *fn.MinHeight {
				fn.Layout.Height = *fn.MinHeight
			}
			if fn.MaxHeight != nil && fn.Layout.Height > *fn.MaxHeight {
				fn.Layout.Height = *fn.MaxHeight
			}
		}
	}

	// Absolute children
	for _, child := range absolute {
		absX := contentX
		absY := contentY
		if child.X != nil {
			absX += *child.X
		}
		if child.Y != nil {
			absY += *child.Y
		}
		absW := contentW
		if child.Width != nil {
			absW = *child.Width
		}
		absH := contentH
		if child.Height != nil {
			absH = *child.Height
		}
		layoutNode(child, absX, absY, absW, absH, false)
	}
}

func layoutFlowChildren(parent *scene.FrameNode, children []scene.Node, cx, cy, cw, ch float64) {
	isRow := parent.Direction == "row"
	mainAvail := cw
	crossAvail := ch
	if !isRow {
		mainAvail = ch
		crossAvail = cw
	}

	// Step 1: measure each child's desired main & cross sizes
	childMains := make([]float64, len(children))
	childCrosses := make([]float64, len(children))

	for i, child := range children {
		if tn, ok := child.(*scene.TextNode); ok {
			tw, th := MeasureText(tn)
			if isRow {
				childMains[i] = tw
				childCrosses[i] = th
			} else {
				childMains[i] = th
				childCrosses[i] = tw
			}
		} else if fn, ok := child.(*scene.FrameNode); ok {
			if isRow {
				if fn.Width != nil {
					childMains[i] = *fn.Width
				} else if fn.Flex != nil {
					childMains[i] = 0
				} else {
					iw, _ := intrinsicSize(fn, cw, ch)
					childMains[i] = iw
				}
				if fn.Height != nil {
					childCrosses[i] = *fn.Height
				} else if parent.Align == "stretch" && !parent.Wrap {
					childCrosses[i] = crossAvail
				} else {
					_, ih := intrinsicSize(fn, cw, ch)
					childCrosses[i] = ih
				}
			} else {
				if fn.Height != nil {
					childMains[i] = *fn.Height
				} else if fn.Flex != nil {
					childMains[i] = 0
				} else {
					_, ih := intrinsicSize(fn, cw, ch)
					childMains[i] = ih
				}
				if fn.Width != nil {
					childCrosses[i] = *fn.Width
				} else if parent.Align == "stretch" && !parent.Wrap {
					childCrosses[i] = crossAvail
				} else {
					iw, _ := intrinsicSize(fn, cw, ch)
					childCrosses[i] = iw
				}
			}
		}
	}

	// If wrapping is enabled, split into lines and lay out each line
	if parent.Wrap {
		layoutWrapLines(parent, children, childMains, childCrosses, cx, cy, cw, ch, mainAvail, isRow)
		return
	}

	n := len(children)
	totalGap := parent.Gap * math.Max(0, float64(n-1))

	// Step 2: distribute flex space (include main-axis margins)
	totalFixed := totalGap
	for i, cm := range childMains {
		totalFixed += cm
		if fn, ok := children[i].(*scene.FrameNode); ok {
			if isRow {
				totalFixed += fn.Margin.Left + fn.Margin.Right
			} else {
				totalFixed += fn.Margin.Top + fn.Margin.Bottom
			}
		}
	}
	remaining := math.Max(0, mainAvail-totalFixed)
	totalFlex := 0.0
	for _, c := range children {
		if fn, ok := c.(*scene.FrameNode); ok && fn.Flex != nil {
			totalFlex += *fn.Flex
		}
	}
	if totalFlex > 0 {
		for i, c := range children {
			if fn, ok := c.(*scene.FrameNode); ok && fn.Flex != nil {
				share := remaining * (*fn.Flex / totalFlex)
				childMains[i] = share
			}
		}
	}

	// Recalculate after flex
	totalUsed := totalGap
	for _, cm := range childMains {
		totalUsed += cm
	}

	// Step 3: justify
	mainOffset := 0.0
	gap := parent.Gap
	switch parent.Justify {
	case "center":
		mainOffset = math.Max(0, mainAvail-totalUsed) / 2
	case "end":
		mainOffset = math.Max(0, mainAvail-totalUsed)
	case "between":
		if n > 1 {
			sumMains := 0.0
			for _, cm := range childMains {
				sumMains += cm
			}
			gap = math.Max(0, (mainAvail-sumMains)/float64(n-1))
			mainOffset = 0
		}
	case "around":
		if n > 0 {
			sumMains := 0.0
			for _, cm := range childMains {
				sumMains += cm
			}
			space := math.Max(0, mainAvail-sumMains) / float64(n)
			gap = space
			mainOffset = space / 2
		}
	}

	// Step 4: position each child
	pos := mainOffset
	for i, child := range children {
		cm := childMains[i]
		cc := childCrosses[i]

		// Apply margin offsets
		ml, mt, mr, mb := 0.0, 0.0, 0.0, 0.0
		if fn, ok := child.(*scene.FrameNode); ok {
			ml = fn.Margin.Left
			mt = fn.Margin.Top
			mr = fn.Margin.Right
			mb = fn.Margin.Bottom
		}

		// Add leading margin
		if isRow {
			pos += ml
		} else {
			pos += mt
		}

		// Cross-axis alignment
		crossOffset := 0.0
		switch parent.Align {
		case "center":
			crossOffset = math.Max(0, crossAvail-cc) / 2
		case "end":
			crossOffset = math.Max(0, crossAvail-cc)
		case "stretch":
			cc = crossAvail
		}

		// Determine if child is being stretched on the cross axis
		isCrossStretched := parent.Align == "stretch"
		if fn, ok := child.(*scene.FrameNode); ok && isCrossStretched {
			if isRow && fn.Height != nil {
				isCrossStretched = false
			}
			if !isRow && fn.Width != nil {
				isCrossStretched = false
			}
		} else if _, ok := child.(*scene.TextNode); ok {
			isCrossStretched = false
		}

		var childX, childY, childW, childH float64
		if isRow {
			childX = cx + pos
			childY = cy + crossOffset + mt
			childW = cm
			childH = cc
		} else {
			childX = cx + crossOffset + ml
			childY = cy + pos
			childW = cc
			childH = cm
		}

		// Only pass stretched=true when HEIGHT is cross-stretched (parent is row)
		heightStretched := isCrossStretched && isRow
		layoutNode(child, childX, childY, childW, childH, heightStretched)

		pos += cm + gap
		// Trailing margin
		if isRow {
			pos += mr
		} else {
			pos += mb
		}
	}
}

// wrapLine holds a line of children for flex-wrap layout.
type wrapLine struct {
	start     int     // index of first child in this line
	end       int     // index past last child
	crossSize float64 // max cross size of children in this line
}

func layoutWrapLines(parent *scene.FrameNode, children []scene.Node,
	childMains, childCrosses []float64, cx, cy, cw, ch, mainAvail float64, isRow bool) {

	gap := parent.Gap

	// Group children into lines
	var lines []wrapLine
	lineStart := 0
	lineMain := 0.0

	for i := range children {
		childMain := childMains[i]
		// Include child margins in the main-axis size
		if fn, ok := children[i].(*scene.FrameNode); ok {
			if isRow {
				childMain += fn.Margin.Left + fn.Margin.Right
			} else {
				childMain += fn.Margin.Top + fn.Margin.Bottom
			}
		}
		// Add gap if not first item in line
		needed := childMain
		if i > lineStart {
			needed += gap
		}

		if i > lineStart && lineMain+needed > mainAvail {
			// Start new line
			crossMax := 0.0
			for j := lineStart; j < i; j++ {
				if childCrosses[j] > crossMax {
					crossMax = childCrosses[j]
				}
			}
			lines = append(lines, wrapLine{start: lineStart, end: i, crossSize: crossMax})
			lineStart = i
			lineMain = childMain
		} else {
			lineMain += needed
		}
	}
	// Last line
	if lineStart < len(children) {
		crossMax := 0.0
		for j := lineStart; j < len(children); j++ {
			if childCrosses[j] > crossMax {
				crossMax = childCrosses[j]
			}
		}
		lines = append(lines, wrapLine{start: lineStart, end: len(children), crossSize: crossMax})
	}

	// Position each line along the cross axis
	crossPos := 0.0
	for _, line := range lines {
		lineChildren := children[line.start:line.end]
		lineMains := childMains[line.start:line.end]
		n := len(lineChildren)

		// Justify within this line
		totalUsed := gap * math.Max(0, float64(n-1))
		for _, cm := range lineMains {
			totalUsed += cm
		}

		mainOffset := 0.0
		lineGap := gap
		switch parent.Justify {
		case "center":
			mainOffset = math.Max(0, mainAvail-totalUsed) / 2
		case "end":
			mainOffset = math.Max(0, mainAvail-totalUsed)
		case "between":
			if n > 1 {
				sumMains := 0.0
				for _, cm := range lineMains {
					sumMains += cm
				}
				lineGap = math.Max(0, (mainAvail-sumMains)/float64(n-1))
			}
		case "around":
			if n > 0 {
				sumMains := 0.0
				for _, cm := range lineMains {
					sumMains += cm
				}
				space := math.Max(0, mainAvail-sumMains) / float64(n)
				lineGap = space
				mainOffset = space / 2
			}
		}

		// Position children in this line
		pos := mainOffset
		for li, child := range lineChildren {
			cm := lineMains[li]
			cc := childCrosses[line.start+li]

			// Apply margin offsets
			ml, mt, mr, mb := 0.0, 0.0, 0.0, 0.0
			if fn, ok := child.(*scene.FrameNode); ok {
				ml = fn.Margin.Left
				mt = fn.Margin.Top
				mr = fn.Margin.Right
				mb = fn.Margin.Bottom
			}

			// Add leading margin
			if isRow {
				pos += ml
			} else {
				pos += mt
			}

			// Cross-axis alignment within line
			lineCross := line.crossSize
			crossOffset := 0.0
			switch parent.Align {
			case "center":
				crossOffset = math.Max(0, lineCross-cc) / 2
			case "end":
				crossOffset = math.Max(0, lineCross-cc)
			case "stretch":
				cc = lineCross
			}

			var childX, childY, childW, childH float64
			if isRow {
				childX = cx + pos
				childY = cy + crossPos + crossOffset + mt
				childW = cm
				childH = cc
			} else {
				childX = cx + crossPos + crossOffset + ml
				childY = cy + pos
				childW = cc
				childH = cm
			}

			heightStretched := parent.Align == "stretch" && isRow
			if fn, ok := child.(*scene.FrameNode); ok && heightStretched && fn.Height != nil {
				heightStretched = false
			}
			if _, ok := child.(*scene.TextNode); ok {
				heightStretched = false
			}

			layoutNode(child, childX, childY, childW, childH, heightStretched)
			pos += cm + lineGap

			// Trailing margin
			if isRow {
				pos += mr
			} else {
				pos += mb
			}
		}

		crossPos += line.crossSize + gap
	}
}

func layoutGridChildren(parent *scene.FrameNode, children []scene.Node, cx, cy, cw, ch float64) {
	cols := 1
	if parent.Columns != nil {
		cols = *parent.Columns
	}
	if cols < 1 {
		cols = 1
	}
	colGap := parent.Gap
	if parent.ColumnGap != nil {
		colGap = *parent.ColumnGap
	}
	rowGap := parent.Gap
	if parent.RowGap != nil {
		rowGap = *parent.RowGap
	}

	n := len(children)
	numRows := (n + cols - 1) / cols

	// Equal column widths
	totalColGap := colGap * math.Max(0, float64(cols-1))
	colW := math.Max(0, (cw-totalColGap)/float64(cols))

	// First pass: compute row heights
	rowHeights := make([]float64, numRows)
	for r := range numRows {
		for c := range cols {
			idx := r*cols + c
			if idx >= n {
				break
			}
			child := children[idx]
			_, ih := intrinsicSize(child, colW, ch)
			if fn, ok := child.(*scene.FrameNode); ok && fn.Height != nil {
				ih = *fn.Height
			}
			if ih > rowHeights[r] {
				rowHeights[r] = ih
			}
		}
	}

	// Second pass: position children.
	// Items are placed in uniform grid tracks (CSS Grid behavior). A child with
	// an explicit width renders at that width but is positioned within the
	// uniform column track, not resizing the track.
	yOffset := 0.0
	for r := range numRows {
		xOffset := 0.0
		for c := range cols {
			idx := r*cols + c
			if idx >= n {
				break
			}
			child := children[idx]
			childW := colW
			childH := rowHeights[r]

			// Apply margin offsets
			ml, mt := 0.0, 0.0
			if fn, ok := child.(*scene.FrameNode); ok {
				ml = fn.Margin.Left
				mt = fn.Margin.Top
				if fn.Width != nil {
					childW = *fn.Width
				}
				if fn.Height != nil {
					childH = *fn.Height
				}
			}

			isStretched := true
			if fn, ok := child.(*scene.FrameNode); ok && fn.Height != nil {
				isStretched = false
			}

			layoutNode(child, cx+xOffset+ml, cy+yOffset+mt, childW, childH, isStretched)
			xOffset += colW + colGap
		}
		yOffset += rowHeights[r] + rowGap
	}
}

func recomputeAutoHeight(parent *scene.FrameNode, flow []scene.Node) {
	isRow := parent.Direction == "row"

	if parent.LayoutMode == "grid" {
		maxBottom := 0.0
		for _, c := range flow {
			l := c.GetLayout()
			bottom := l.Y + l.Height
			if bottom > maxBottom {
				maxBottom = bottom
			}
		}
		parent.Layout.Height = maxBottom + parent.Padding.Bottom
		return
	}

	if isRow {
		maxBottom := 0.0
		for _, c := range flow {
			l := c.GetLayout()
			bottom := l.Y + l.Height
			if bottom > maxBottom {
				maxBottom = bottom
			}
		}
		parent.Layout.Height = maxBottom + parent.Padding.Bottom
	} else {
		if len(flow) > 0 {
			last := flow[len(flow)-1]
			l := last.GetLayout()
			parent.Layout.Height = l.Y + l.Height + parent.Padding.Bottom
		} else {
			parent.Layout.Height = parent.Padding.Vertical()
		}
	}
}

func optFloat(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}
