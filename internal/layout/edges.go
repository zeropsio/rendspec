package layout

import (
	"math"
	"sort"

	"github.com/zeropsio/rendspec/internal/scene"
)

// Rect represents a frame's absolute position and size.
type Rect struct {
	X, Y, W, H float64
}

const busSpread = 20 // px between stacked junction lines

var vAnchors = map[string]bool{"top": true, "bottom": true}

func buildFrameMap(node *scene.FrameNode, parentX, parentY float64) map[string]Rect {
	result := make(map[string]Rect)
	absX := parentX + node.Layout.X
	absY := parentY + node.Layout.Y
	if node.ID != "" {
		result[node.ID] = Rect{absX, absY, node.Layout.Width, node.Layout.Height}
	}
	for _, child := range node.Children {
		if fn, ok := child.(*scene.FrameNode); ok {
			for k, v := range buildFrameMap(fn, absX, absY) {
				result[k] = v
			}
		}
	}
	return result
}

func anchorPoint(rect Rect, anchor string) scene.Point {
	cx := rect.X + rect.W/2
	cy := rect.Y + rect.H/2
	switch anchor {
	case "top":
		return scene.Point{X: cx, Y: rect.Y}
	case "bottom":
		return scene.Point{X: cx, Y: rect.Y + rect.H}
	case "left":
		return scene.Point{X: rect.X, Y: cy}
	case "right":
		return scene.Point{X: rect.X + rect.W, Y: cy}
	default:
		return scene.Point{X: cx, Y: cy}
	}
}

func autoAnchors(src, dst Rect) (string, string) {
	scx := src.X + src.W/2
	scy := src.Y + src.H/2
	dcx := dst.X + dst.W/2
	dcy := dst.Y + dst.H/2

	hdist := math.Abs(dcx - scx)
	vdist := math.Abs(dcy - scy)

	if hdist > vdist {
		if dcx > scx {
			return "right", "left"
		}
		return "left", "right"
	}
	if dcy > scy {
		return "bottom", "top"
	}
	return "top", "bottom"
}

func computeJunctionY(srcY, dstY float64, junction *float64, frameMap map[string]Rect) float64 {
	if junction != nil {
		if *junction <= 1 {
			return srcY + *junction*(dstY-srcY)
		}
		if dstY >= srcY {
			return srcY + *junction
		}
		return srcY - *junction
	}

	// Auto mode
	if len(frameMap) > 0 {
		if jy := findBestGapY(srcY, dstY, frameMap); jy != nil {
			return *jy
		}
	}

	return (srcY + dstY) / 2
}

func findBestGapY(srcY, dstY float64, frameMap map[string]Rect) *float64 {
	lo := math.Min(srcY, dstY)
	hi := math.Max(srcY, dstY)
	if hi-lo < 4 {
		return nil
	}

	var intervals [][2]float64
	for _, r := range frameMap {
		ftop := r.Y
		fbot := r.Y + r.H
		if ftop <= lo && fbot >= hi {
			continue
		}
		if fbot > lo && ftop < hi {
			intervals = append(intervals, [2]float64{
				math.Max(ftop, lo),
				math.Min(fbot, hi),
			})
		}
	}

	if len(intervals) == 0 {
		mid := (lo + hi) / 2
		return &mid
	}

	sort.Slice(intervals, func(i, j int) bool {
		return intervals[i][0] < intervals[j][0]
	})

	// Merge overlapping
	merged := [][2]float64{intervals[0]}
	for _, iv := range intervals[1:] {
		last := &merged[len(merged)-1]
		if iv[0] <= last[1] {
			if iv[1] > last[1] {
				last[1] = iv[1]
			}
		} else {
			merged = append(merged, iv)
		}
	}

	// Find gaps
	var gaps [][2]float64
	if merged[0][0] > lo {
		gaps = append(gaps, [2]float64{lo, merged[0][0]})
	}
	for i := 0; i < len(merged)-1; i++ {
		gapStart := merged[i][1]
		gapEnd := merged[i+1][0]
		if gapEnd > gapStart {
			gaps = append(gaps, [2]float64{gapStart, gapEnd})
		}
	}
	if merged[len(merged)-1][1] < hi {
		gaps = append(gaps, [2]float64{merged[len(merged)-1][1], hi})
	}

	if len(gaps) == 0 {
		return nil
	}

	// Pick the largest gap
	best := gaps[0]
	for _, g := range gaps[1:] {
		if g[1]-g[0] > best[1]-best[0] {
			best = g
		}
	}
	mid := (best[0] + best[1]) / 2
	return &mid
}

func resolveBusGroup(edges []*scene.EdgeNode, frameMap map[string]Rect) {
	first := edges[0]
	src, ok := frameMap[first.FromID]
	if !ok {
		return
	}

	fromAnchor := first.FromAnchor
	p1 := anchorPoint(src, fromAnchor)

	if !vAnchors[fromAnchor] {
		for _, edge := range edges {
			resolveEdge(edge, frameMap)
		}
		return
	}

	type destInfo struct {
		edge     *scene.EdgeNode
		dst      Rect
		toAnchor string
		p2       scene.Point
	}
	var destEdges []destInfo

	for _, edge := range edges {
		dst, ok := frameMap[edge.ToID]
		if !ok {
			continue
		}
		toAnchor := edge.ToAnchor
		if toAnchor == "auto" {
			_, toAnchor = autoAnchors(src, dst)
		}
		p2 := anchorPoint(dst, toAnchor)
		destEdges = append(destEdges, destInfo{edge, dst, toAnchor, p2})
	}

	if len(destEdges) == 0 {
		return
	}

	junctionVal := first.Junction

	var nearestDestY float64
	if fromAnchor == "bottom" {
		nearestDestY = destEdges[0].p2.Y
		for _, de := range destEdges[1:] {
			if de.p2.Y < nearestDestY {
				nearestDestY = de.p2.Y
			}
		}
	} else {
		nearestDestY = destEdges[0].p2.Y
		for _, de := range destEdges[1:] {
			if de.p2.Y > nearestDestY {
				nearestDestY = de.p2.Y
			}
		}
	}

	baseJY := computeJunctionY(p1.Y, nearestDestY, junctionVal, frameMap)

	// Sort by distance from trunk X (outermost first)
	sort.Slice(destEdges, func(i, j int) bool {
		return math.Abs(destEdges[i].p2.X-p1.X) > math.Abs(destEdges[j].p2.X-p1.X)
	})

	goingDown := fromAnchor == "bottom"
	n := len(destEdges)

	for i, de := range destEdges {
		var edgeJY float64
		if de.edge.Junction != nil && (junctionVal == nil || *de.edge.Junction != *junctionVal) {
			edgeJY = computeJunctionY(p1.Y, de.p2.Y, de.edge.Junction, frameMap)
		} else if n > 1 {
			offset := float64(i) * busSpread
			if goingDown {
				edgeJY = baseJY + offset
			} else {
				edgeJY = baseJY - offset
			}
		} else {
			edgeJY = baseJY
		}

		if math.Abs(p1.X-de.p2.X) < 2 {
			de.edge.ResolvedPath = []scene.Point{p1, de.p2}
		} else {
			de.edge.ResolvedPath = []scene.Point{
				p1,
				{X: p1.X, Y: edgeJY},
				{X: de.p2.X, Y: edgeJY},
				de.p2,
			}
		}
	}
}

func resolveEdge(edge *scene.EdgeNode, frameMap map[string]Rect) {
	src, srcOK := frameMap[edge.FromID]
	dst, dstOK := frameMap[edge.ToID]
	if !srcOK || !dstOK {
		return
	}

	fromAnchor := edge.FromAnchor
	toAnchor := edge.ToAnchor
	if fromAnchor == "auto" && toAnchor == "auto" {
		fromAnchor, toAnchor = autoAnchors(src, dst)
	} else if fromAnchor == "auto" {
		fromAnchor, _ = autoAnchors(src, dst)
	} else if toAnchor == "auto" {
		_, toAnchor = autoAnchors(src, dst)
	}

	p1 := anchorPoint(src, fromAnchor)
	p2 := anchorPoint(dst, toAnchor)

	switch edge.Curve {
	case "orthogonal":
		edge.ResolvedPath = orthogonalWaypoints(p1, p2, fromAnchor, toAnchor)
	case "bus":
		if vAnchors[fromAnchor] && vAnchors[toAnchor] {
			jy := computeJunctionY(p1.Y, p2.Y, edge.Junction, frameMap)
			edge.ResolvedPath = []scene.Point{p1, {X: p1.X, Y: jy}, {X: p2.X, Y: jy}, p2}
		} else if vAnchors[fromAnchor] {
			edge.ResolvedPath = []scene.Point{p1, {X: p1.X, Y: p2.Y}, p2}
		} else {
			edge.ResolvedPath = []scene.Point{p1, {X: p2.X, Y: p1.Y}, p2}
		}
	case "vertical":
		edge.ResolvedPath = []scene.Point{p1, {X: p1.X, Y: p2.Y}}
	default: // straight
		edge.ResolvedPath = []scene.Point{p1, p2}
	}
}

func orthogonalWaypoints(p1, p2 scene.Point, fromAnchor, toAnchor string) []scene.Point {
	fromV := vAnchors[fromAnchor]
	toV := vAnchors[toAnchor]

	if fromV && toV {
		midY := (p1.Y + p2.Y) / 2
		return []scene.Point{p1, {X: p1.X, Y: midY}, {X: p2.X, Y: midY}, p2}
	}
	if !fromV && !toV {
		midX := (p1.X + p2.X) / 2
		return []scene.Point{p1, {X: midX, Y: p1.Y}, {X: midX, Y: p2.Y}, p2}
	}
	if fromV {
		return []scene.Point{p1, {X: p1.X, Y: p2.Y}, p2}
	}
	return []scene.Point{p1, {X: p2.X, Y: p1.Y}, p2}
}
