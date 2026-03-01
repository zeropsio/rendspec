// Package inspect provides shared helpers for introspecting scene graphs.
// Used by the CLI and MCP server to avoid code duplication.
package inspect

import (
	"github.com/zeropsio/rendspec/internal/scene"
	"math"
)

// CountFrames counts all frame nodes in the tree rooted at node.
func CountFrames(node *scene.FrameNode) int {
	count := 1
	for _, child := range node.Children {
		if fn, ok := child.(*scene.FrameNode); ok {
			count += CountFrames(fn)
		}
	}
	return count
}

// CountTexts counts all text nodes in the tree rooted at node.
func CountTexts(node *scene.FrameNode) int {
	count := 0
	for _, child := range node.Children {
		switch c := child.(type) {
		case *scene.TextNode:
			count++
		case *scene.FrameNode:
			count += CountTexts(c)
		}
	}
	return count
}

// Round1 rounds a float to 1 decimal place.
func Round1(v float64) float64 {
	return math.Round(v*10) / 10
}

// NodeToDict converts a scene node to a map for JSON serialization.
// Layout coordinates are rounded to 1 decimal place.
func NodeToDict(node scene.Node) map[string]interface{} {
	if tn, ok := node.(*scene.TextNode); ok {
		return map[string]interface{}{
			"type":    "text",
			"content": tn.Content,
			"layout": map[string]interface{}{
				"x":      Round1(tn.Layout.X),
				"y":      Round1(tn.Layout.Y),
				"width":  Round1(tn.Layout.Width),
				"height": Round1(tn.Layout.Height),
			},
		}
	}
	fn, ok := node.(*scene.FrameNode)
	if !ok {
		return nil
	}

	d := map[string]interface{}{
		"type": "frame",
		"layout": map[string]interface{}{
			"x":      Round1(fn.Layout.X),
			"y":      Round1(fn.Layout.Y),
			"width":  Round1(fn.Layout.Width),
			"height": Round1(fn.Layout.Height),
		},
	}
	if fn.ID != "" {
		d["id"] = fn.ID
	}
	if len(fn.Children) > 0 {
		children := make([]map[string]interface{}, 0, len(fn.Children))
		for _, c := range fn.Children {
			children = append(children, NodeToDict(c))
		}
		d["children"] = children
	}
	return d
}

// EdgeToDict converts an edge node to a map for JSON serialization.
func EdgeToDict(edge *scene.EdgeNode) map[string]interface{} {
	path := make([][]float64, len(edge.ResolvedPath))
	for i, p := range edge.ResolvedPath {
		path[i] = []float64{Round1(p.X), Round1(p.Y)}
	}
	d := map[string]interface{}{
		"from":         edge.FromID,
		"to":           edge.ToID,
		"stroke":       edge.Stroke,
		"stroke_width": edge.StrokeWidth,
		"style":        edge.Style,
		"arrow":        edge.Arrow,
		"curve":        edge.Curve,
		"path":         path,
	}
	if edge.Label != nil {
		d["label"] = *edge.Label
	}
	return d
}

// NodeToMap converts a scene node to a flat map (no nested layout key).
// Used by the MCP inspect handler. Coordinates are rounded to 1 decimal place.
func NodeToMap(node scene.Node) map[string]interface{} {
	if tn, ok := node.(*scene.TextNode); ok {
		return map[string]interface{}{
			"type":    "text",
			"content": tn.Content,
			"x":       Round1(tn.Layout.X),
			"y":       Round1(tn.Layout.Y),
			"width":   Round1(tn.Layout.Width),
			"height":  Round1(tn.Layout.Height),
		}
	}
	fn, ok := node.(*scene.FrameNode)
	if !ok {
		return nil
	}
	d := map[string]interface{}{
		"type":   "frame",
		"x":      Round1(fn.Layout.X),
		"y":      Round1(fn.Layout.Y),
		"width":  Round1(fn.Layout.Width),
		"height": Round1(fn.Layout.Height),
	}
	if fn.ID != "" {
		d["id"] = fn.ID
	}
	if len(fn.Children) > 0 {
		children := make([]map[string]interface{}, 0, len(fn.Children))
		for _, c := range fn.Children {
			children = append(children, NodeToMap(c))
		}
		d["children"] = children
	}
	return d
}
