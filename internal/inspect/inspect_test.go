package inspect

import (
	"testing"

	"github.com/fxck/rendspec/internal/scene"
)

// ---------------------------------------------------------------------------
// CountFrames
// ---------------------------------------------------------------------------

func TestCountFrames_SingleFrame(t *testing.T) {
	root := scene.NewFrameNode()
	if got := CountFrames(root); got != 1 {
		t.Errorf("CountFrames(single frame) = %d, want 1", got)
	}
}

func TestCountFrames_NestedFrames(t *testing.T) {
	root := scene.NewFrameNode()
	child1 := scene.NewFrameNode()
	child2 := scene.NewFrameNode()
	root.Children = []scene.Node{child1, child2}

	if got := CountFrames(root); got != 3 {
		t.Errorf("CountFrames(root + 2 children) = %d, want 3", got)
	}
}

func TestCountFrames_DeepNesting(t *testing.T) {
	// root -> child -> grandchild -> great-grandchild
	greatGrandchild := scene.NewFrameNode()
	grandchild := scene.NewFrameNode()
	grandchild.Children = []scene.Node{greatGrandchild}
	child := scene.NewFrameNode()
	child.Children = []scene.Node{grandchild}
	root := scene.NewFrameNode()
	root.Children = []scene.Node{child}

	if got := CountFrames(root); got != 4 {
		t.Errorf("CountFrames(4 levels deep) = %d, want 4", got)
	}
}

func TestCountFrames_IgnoresTextNodes(t *testing.T) {
	root := scene.NewFrameNode()
	txt := scene.NewTextNode()
	txt.Content = "hello"
	child := scene.NewFrameNode()
	root.Children = []scene.Node{txt, child}

	// Should count root + child = 2, ignoring the text node.
	if got := CountFrames(root); got != 2 {
		t.Errorf("CountFrames(root + text + child frame) = %d, want 2", got)
	}
}

func TestCountFrames_MixedTreeWithMultipleBranches(t *testing.T) {
	//       root
	//      / | \
	//    f1  t1  f2
	//   / \      |
	//  f3  t2    f4
	root := scene.NewFrameNode()
	f1 := scene.NewFrameNode()
	f2 := scene.NewFrameNode()
	f3 := scene.NewFrameNode()
	f4 := scene.NewFrameNode()
	t1 := scene.NewTextNode()
	t1.Content = "t1"
	t2 := scene.NewTextNode()
	t2.Content = "t2"

	f1.Children = []scene.Node{f3, t2}
	f2.Children = []scene.Node{f4}
	root.Children = []scene.Node{f1, t1, f2}

	// root + f1 + f2 + f3 + f4 = 5
	if got := CountFrames(root); got != 5 {
		t.Errorf("CountFrames(mixed tree) = %d, want 5", got)
	}
}

// ---------------------------------------------------------------------------
// CountTexts
// ---------------------------------------------------------------------------

func TestCountTexts_NoText(t *testing.T) {
	root := scene.NewFrameNode()
	if got := CountTexts(root); got != 0 {
		t.Errorf("CountTexts(no children) = %d, want 0", got)
	}
}

func TestCountTexts_NoTextWithFrameChildren(t *testing.T) {
	root := scene.NewFrameNode()
	root.Children = []scene.Node{scene.NewFrameNode(), scene.NewFrameNode()}

	if got := CountTexts(root); got != 0 {
		t.Errorf("CountTexts(only frame children) = %d, want 0", got)
	}
}

func TestCountTexts_OneText(t *testing.T) {
	root := scene.NewFrameNode()
	txt := scene.NewTextNode()
	txt.Content = "hello"
	root.Children = []scene.Node{txt}

	if got := CountTexts(root); got != 1 {
		t.Errorf("CountTexts(one text child) = %d, want 1", got)
	}
}

func TestCountTexts_MultipleTextsNestedFrames(t *testing.T) {
	// root has 1 text, child frame has 2 texts
	root := scene.NewFrameNode()
	t1 := scene.NewTextNode()
	t1.Content = "root text"
	child := scene.NewFrameNode()
	t2 := scene.NewTextNode()
	t2.Content = "child text 1"
	t3 := scene.NewTextNode()
	t3.Content = "child text 2"
	child.Children = []scene.Node{t2, t3}
	root.Children = []scene.Node{t1, child}

	if got := CountTexts(root); got != 3 {
		t.Errorf("CountTexts(3 texts across nesting) = %d, want 3", got)
	}
}

func TestCountTexts_DeeplyNested(t *testing.T) {
	// root -> frame -> frame -> text
	inner := scene.NewFrameNode()
	txt := scene.NewTextNode()
	txt.Content = "deep"
	inner.Children = []scene.Node{txt}
	mid := scene.NewFrameNode()
	mid.Children = []scene.Node{inner}
	root := scene.NewFrameNode()
	root.Children = []scene.Node{mid}

	if got := CountTexts(root); got != 1 {
		t.Errorf("CountTexts(deeply nested single text) = %d, want 1", got)
	}
}

// ---------------------------------------------------------------------------
// Round1
// ---------------------------------------------------------------------------

func TestRound1(t *testing.T) {
	tests := []struct {
		name string
		in   float64
		want float64
	}{
		{"zero", 0, 0},
		{"integer", 5, 5},
		{"already one decimal", 3.1, 3.1},
		{"rounds down", 1.24, 1.2},
		{"rounds up", 1.25, 1.3},
		{"rounds up high", 1.26, 1.3},
		{"negative rounds", -2.35, -2.4},
		{"large number", 12345.678, 12345.7},
		{"small fraction", 0.04, 0.0},
		{"exactly half", 0.05, 0.1},
		{"many decimals", 3.14159, 3.1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Round1(tt.in)
			if got != tt.want {
				t.Errorf("Round1(%v) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// NodeToDict
// ---------------------------------------------------------------------------

func TestNodeToDict_TextNode(t *testing.T) {
	tn := scene.NewTextNode()
	tn.Content = "Hello"
	tn.Layout = scene.ComputedLayout{X: 10.123, Y: 20.456, Width: 100.789, Height: 30.999}

	d := NodeToDict(tn)
	if d == nil {
		t.Fatal("NodeToDict returned nil for TextNode")
	}
	if d["type"] != "text" {
		t.Errorf("type = %v, want text", d["type"])
	}
	if d["content"] != "Hello" {
		t.Errorf("content = %v, want Hello", d["content"])
	}
	layout, ok := d["layout"].(map[string]interface{})
	if !ok {
		t.Fatal("layout is not a map")
	}
	if layout["x"] != 10.1 {
		t.Errorf("layout.x = %v, want 10.1", layout["x"])
	}
	if layout["y"] != 20.5 {
		t.Errorf("layout.y = %v, want 20.5", layout["y"])
	}
	if layout["width"] != 100.8 {
		t.Errorf("layout.width = %v, want 100.8", layout["width"])
	}
	if layout["height"] != 31.0 {
		t.Errorf("layout.height = %v, want 31.0", layout["height"])
	}
}

func TestNodeToDict_FrameNode_NoChildren_NoID(t *testing.T) {
	fn := scene.NewFrameNode()
	fn.Layout = scene.ComputedLayout{X: 0, Y: 0, Width: 200, Height: 100}

	d := NodeToDict(fn)
	if d == nil {
		t.Fatal("NodeToDict returned nil for FrameNode")
	}
	if d["type"] != "frame" {
		t.Errorf("type = %v, want frame", d["type"])
	}
	if _, exists := d["id"]; exists {
		t.Error("id should not be present when empty")
	}
	if _, exists := d["children"]; exists {
		t.Error("children should not be present when empty")
	}
	layout, ok := d["layout"].(map[string]interface{})
	if !ok {
		t.Fatal("layout is not a map")
	}
	if layout["width"] != 200.0 {
		t.Errorf("layout.width = %v, want 200.0", layout["width"])
	}
}

func TestNodeToDict_FrameNode_WithID(t *testing.T) {
	fn := scene.NewFrameNode()
	fn.ID = "card"
	fn.Layout = scene.ComputedLayout{X: 5, Y: 10, Width: 300, Height: 150}

	d := NodeToDict(fn)
	if d["id"] != "card" {
		t.Errorf("id = %v, want card", d["id"])
	}
}

func TestNodeToDict_FrameNode_WithChildren(t *testing.T) {
	parent := scene.NewFrameNode()
	parent.ID = "parent"
	parent.Layout = scene.ComputedLayout{X: 0, Y: 0, Width: 400, Height: 300}

	child := scene.NewFrameNode()
	child.ID = "child"
	child.Layout = scene.ComputedLayout{X: 10, Y: 10, Width: 100, Height: 50}

	txt := scene.NewTextNode()
	txt.Content = "label"
	txt.Layout = scene.ComputedLayout{X: 20, Y: 20, Width: 60, Height: 14}

	parent.Children = []scene.Node{child, txt}

	d := NodeToDict(parent)
	children, ok := d["children"].([]map[string]interface{})
	if !ok {
		t.Fatal("children is not a []map[string]interface{}")
	}
	if len(children) != 2 {
		t.Fatalf("len(children) = %d, want 2", len(children))
	}
	if children[0]["type"] != "frame" {
		t.Errorf("first child type = %v, want frame", children[0]["type"])
	}
	if children[0]["id"] != "child" {
		t.Errorf("first child id = %v, want child", children[0]["id"])
	}
	if children[1]["type"] != "text" {
		t.Errorf("second child type = %v, want text", children[1]["type"])
	}
	if children[1]["content"] != "label" {
		t.Errorf("second child content = %v, want label", children[1]["content"])
	}
}

func TestNodeToDict_UnknownNodeType(t *testing.T) {
	// EdgeNode implements Node but is neither FrameNode nor TextNode.
	edge := scene.NewEdgeNode()
	d := NodeToDict(edge)
	if d != nil {
		t.Errorf("NodeToDict(EdgeNode) = %v, want nil", d)
	}
}

func TestNodeToDict_RoundsLayoutValues(t *testing.T) {
	fn := scene.NewFrameNode()
	fn.Layout = scene.ComputedLayout{X: 1.55, Y: 2.44, Width: 99.99, Height: 50.05}

	d := NodeToDict(fn)
	layout := d["layout"].(map[string]interface{})
	if layout["x"] != 1.6 {
		t.Errorf("x = %v, want 1.6", layout["x"])
	}
	if layout["y"] != 2.4 {
		t.Errorf("y = %v, want 2.4", layout["y"])
	}
	if layout["width"] != 100.0 {
		t.Errorf("width = %v, want 100.0", layout["width"])
	}
	if layout["height"] != 50.1 {
		t.Errorf("height = %v, want 50.1", layout["height"])
	}
}

// ---------------------------------------------------------------------------
// EdgeToDict
// ---------------------------------------------------------------------------

func TestEdgeToDict_Basic(t *testing.T) {
	edge := scene.NewEdgeNode()
	edge.FromID = "a"
	edge.ToID = "b"
	edge.ResolvedPath = []scene.Point{
		{X: 10.123, Y: 20.456},
		{X: 30.789, Y: 40.999},
	}

	d := EdgeToDict(edge)
	if d["from"] != "a" {
		t.Errorf("from = %v, want a", d["from"])
	}
	if d["to"] != "b" {
		t.Errorf("to = %v, want b", d["to"])
	}
	path, ok := d["path"].([][]float64)
	if !ok {
		t.Fatal("path is not [][]float64")
	}
	if len(path) != 2 {
		t.Fatalf("len(path) = %d, want 2", len(path))
	}
	if path[0][0] != 10.1 || path[0][1] != 20.5 {
		t.Errorf("path[0] = %v, want [10.1, 20.5]", path[0])
	}
	if path[1][0] != 30.8 || path[1][1] != 41.0 {
		t.Errorf("path[1] = %v, want [30.8, 41.0]", path[1])
	}
}

func TestEdgeToDict_EmptyPath(t *testing.T) {
	edge := scene.NewEdgeNode()
	edge.FromID = "x"
	edge.ToID = "y"
	edge.ResolvedPath = nil

	d := EdgeToDict(edge)
	path, ok := d["path"].([][]float64)
	if !ok {
		t.Fatal("path is not [][]float64")
	}
	if len(path) != 0 {
		t.Errorf("len(path) = %d, want 0", len(path))
	}
}

func TestEdgeToDict_SinglePoint(t *testing.T) {
	edge := scene.NewEdgeNode()
	edge.FromID = "start"
	edge.ToID = "end"
	edge.ResolvedPath = []scene.Point{{X: 5.55, Y: 9.99}}

	d := EdgeToDict(edge)
	path := d["path"].([][]float64)
	if len(path) != 1 {
		t.Fatalf("len(path) = %d, want 1", len(path))
	}
	if path[0][0] != 5.6 || path[0][1] != 10.0 {
		t.Errorf("path[0] = %v, want [5.6, 10.0]", path[0])
	}
}

// ---------------------------------------------------------------------------
// NodeToMap
// ---------------------------------------------------------------------------

func TestNodeToMap_TextNode(t *testing.T) {
	tn := scene.NewTextNode()
	tn.Content = "World"
	tn.Layout = scene.ComputedLayout{X: 5, Y: 10, Width: 80, Height: 16}

	d := NodeToMap(tn)
	if d == nil {
		t.Fatal("NodeToMap returned nil for TextNode")
	}
	if d["type"] != "text" {
		t.Errorf("type = %v, want text", d["type"])
	}
	if d["content"] != "World" {
		t.Errorf("content = %v, want World", d["content"])
	}
	// NodeToMap uses flat layout (no nested "layout" key)
	if d["x"] != 5.0 {
		t.Errorf("x = %v, want 5.0", d["x"])
	}
	if d["y"] != 10.0 {
		t.Errorf("y = %v, want 10.0", d["y"])
	}
	if d["width"] != 80.0 {
		t.Errorf("width = %v, want 80.0", d["width"])
	}
	if d["height"] != 16.0 {
		t.Errorf("height = %v, want 16.0", d["height"])
	}
	// Should not have a nested "layout" key
	if _, exists := d["layout"]; exists {
		t.Error("NodeToMap should not have nested layout key")
	}
}

func TestNodeToMap_FrameNode_NoChildren_NoID(t *testing.T) {
	fn := scene.NewFrameNode()
	fn.Layout = scene.ComputedLayout{X: 0, Y: 0, Width: 500, Height: 400}

	d := NodeToMap(fn)
	if d == nil {
		t.Fatal("NodeToMap returned nil for FrameNode")
	}
	if d["type"] != "frame" {
		t.Errorf("type = %v, want frame", d["type"])
	}
	if d["x"] != 0.0 {
		t.Errorf("x = %v, want 0.0", d["x"])
	}
	if d["width"] != 500.0 {
		t.Errorf("width = %v, want 500.0", d["width"])
	}
	if _, exists := d["id"]; exists {
		t.Error("id should not be present when empty")
	}
	if _, exists := d["children"]; exists {
		t.Error("children should not be present when empty")
	}
}

func TestNodeToMap_FrameNode_WithID(t *testing.T) {
	fn := scene.NewFrameNode()
	fn.ID = "sidebar"
	fn.Layout = scene.ComputedLayout{X: 0, Y: 0, Width: 250, Height: 600}

	d := NodeToMap(fn)
	if d["id"] != "sidebar" {
		t.Errorf("id = %v, want sidebar", d["id"])
	}
}

func TestNodeToMap_FrameNode_WithChildren(t *testing.T) {
	parent := scene.NewFrameNode()
	parent.ID = "container"
	parent.Layout = scene.ComputedLayout{X: 0, Y: 0, Width: 600, Height: 400}

	child := scene.NewFrameNode()
	child.ID = "box"
	child.Layout = scene.ComputedLayout{X: 10, Y: 10, Width: 200, Height: 100}

	txt := scene.NewTextNode()
	txt.Content = "info"
	txt.Layout = scene.ComputedLayout{X: 20, Y: 120, Width: 50, Height: 14}

	parent.Children = []scene.Node{child, txt}

	d := NodeToMap(parent)
	children, ok := d["children"].([]map[string]interface{})
	if !ok {
		t.Fatal("children is not []map[string]interface{}")
	}
	if len(children) != 2 {
		t.Fatalf("len(children) = %d, want 2", len(children))
	}
	// First child is frame
	if children[0]["type"] != "frame" {
		t.Errorf("first child type = %v, want frame", children[0]["type"])
	}
	if children[0]["id"] != "box" {
		t.Errorf("first child id = %v, want box", children[0]["id"])
	}
	if children[0]["x"] != 10.0 {
		t.Errorf("first child x = %v, want 10.0", children[0]["x"])
	}
	// Second child is text
	if children[1]["type"] != "text" {
		t.Errorf("second child type = %v, want text", children[1]["type"])
	}
	if children[1]["content"] != "info" {
		t.Errorf("second child content = %v, want info", children[1]["content"])
	}
}

func TestNodeToMap_UnknownNodeType(t *testing.T) {
	edge := scene.NewEdgeNode()
	d := NodeToMap(edge)
	if d != nil {
		t.Errorf("NodeToMap(EdgeNode) = %v, want nil", d)
	}
}

func TestNodeToMap_RoundsValues(t *testing.T) {
	// NodeToMap now rounds layout values to 1 decimal place, matching NodeToDict.
	fn := scene.NewFrameNode()
	fn.Layout = scene.ComputedLayout{X: 1.55, Y: 2.44, Width: 99.99, Height: 50.05}

	d := NodeToMap(fn)
	if d["x"] != 1.6 {
		t.Errorf("x = %v, want 1.6 (rounded)", d["x"])
	}
	if d["y"] != 2.4 {
		t.Errorf("y = %v, want 2.4 (rounded)", d["y"])
	}
	if d["width"] != 100.0 {
		t.Errorf("width = %v, want 100 (rounded)", d["width"])
	}
	if d["height"] != 50.1 {
		t.Errorf("height = %v, want 50.1 (rounded)", d["height"])
	}
}

func TestNodeToMap_RecursiveChildren(t *testing.T) {
	// Verify children are recursively converted with NodeToMap (flat layout).
	root := scene.NewFrameNode()
	root.Layout = scene.ComputedLayout{X: 0, Y: 0, Width: 800, Height: 600}
	child := scene.NewFrameNode()
	child.Layout = scene.ComputedLayout{X: 10, Y: 10, Width: 200, Height: 100}
	grandchild := scene.NewTextNode()
	grandchild.Content = "deep"
	grandchild.Layout = scene.ComputedLayout{X: 20, Y: 20, Width: 60, Height: 14}
	child.Children = []scene.Node{grandchild}
	root.Children = []scene.Node{child}

	d := NodeToMap(root)
	children := d["children"].([]map[string]interface{})
	if len(children) != 1 {
		t.Fatalf("root children count = %d, want 1", len(children))
	}
	grandChildren := children[0]["children"].([]map[string]interface{})
	if len(grandChildren) != 1 {
		t.Fatalf("child children count = %d, want 1", len(grandChildren))
	}
	gc := grandChildren[0]
	if gc["type"] != "text" {
		t.Errorf("grandchild type = %v, want text", gc["type"])
	}
	if gc["content"] != "deep" {
		t.Errorf("grandchild content = %v, want deep", gc["content"])
	}
	// Flat layout, not nested
	if gc["x"] != 20.0 {
		t.Errorf("grandchild x = %v, want 20.0", gc["x"])
	}
}
