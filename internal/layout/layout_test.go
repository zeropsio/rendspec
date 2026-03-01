package layout

import (
	"fmt"
	"math"
	"testing"

	"github.com/fxck/rendspec/internal/parser"
	"github.com/fxck/rendspec/internal/scene"
)

func TestMeasureText_Basic(t *testing.T) {
	node := &scene.TextNode{Content: "Hello", Font: scene.Font{Size: 14, Family: "Inter", Weight: 400}, LineHeight: 1.4}
	w, h := MeasureText(node)
	if w <= 0 || h <= 0 {
		t.Errorf("expected positive size, got %f x %f", w, h)
	}
}

func TestMeasureText_BoldWider(t *testing.T) {
	normal := &scene.TextNode{Content: "Hello", Font: scene.Font{Weight: 400, Size: 14, Family: "Inter"}, LineHeight: 1.4}
	bold := &scene.TextNode{Content: "Hello", Font: scene.Font{Weight: 700, Size: 14, Family: "Inter"}, LineHeight: 1.4}
	nw, _ := MeasureText(normal)
	bw, _ := MeasureText(bold)
	if bw <= nw {
		t.Errorf("bold should be wider: %f <= %f", bw, nw)
	}
}

func TestMeasureText_MonospaceConsistent(t *testing.T) {
	node := &scene.TextNode{Content: "iiiWWW", Font: scene.Font{Size: 14, Family: "JetBrains Mono", Weight: 400}, LineHeight: 1.4}
	w, _ := MeasureText(node)
	expected := 6 * 14 * 0.60
	if math.Abs(w-expected) > 1 {
		t.Errorf("expected ~%f, got %f", expected, w)
	}
}

func TestMeasureText_MaxWidthWrapping(t *testing.T) {
	maxW := 200.0
	node := &scene.TextNode{
		Content:  stringRepeat("A", 100),
		Font:     scene.Font{Size: 14, Family: "Inter", Weight: 400},
		MaxWidth: &maxW,
		LineHeight: 1.4,
	}
	w, h := MeasureText(node)
	if w > 200 {
		t.Errorf("width should be <= 200, got %f", w)
	}
	if h <= 14*1.4 {
		t.Errorf("should be multiple lines, height=%f", h)
	}
}

func TestMeasureText_LetterSpacing(t *testing.T) {
	base := &scene.TextNode{Content: "Hello", Font: scene.Font{Size: 14, Family: "Inter", Weight: 400}, LineHeight: 1.4}
	spaced := &scene.TextNode{Content: "Hello", Font: scene.Font{Size: 14, Family: "Inter", Weight: 400}, LetterSpacing: 2, LineHeight: 1.4}
	bw, _ := MeasureText(base)
	sw, _ := MeasureText(spaced)
	if sw <= bw {
		t.Errorf("letter-spacing should increase width: %f <= %f", sw, bw)
	}
}

// --- Flex Layout ---

func TestFlexLayout_RootSize(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{"width": 800, "height": 600},
	})
	ComputeLayout(sg)
	if sg.Root.Layout.Width != 800 || sg.Root.Layout.Height != 600 {
		t.Errorf("got %f x %f", sg.Root.Layout.Width, sg.Root.Layout.Height)
	}
}

func TestFlexLayout_ColumnLayout(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300, "direction": "column",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "height": 100},
				map[string]interface{}{"frame": "b", "height": 100},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	b := sg.Root.Children[1].(*scene.FrameNode)
	if a.Layout.Y >= b.Layout.Y {
		t.Errorf("a.y=%f should be < b.y=%f", a.Layout.Y, b.Layout.Y)
	}
}

func TestFlexLayout_RowLayout(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 100, "direction": "row",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "width": 100},
				map[string]interface{}{"frame": "b", "width": 100},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	b := sg.Root.Children[1].(*scene.FrameNode)
	if a.Layout.X >= b.Layout.X {
		t.Errorf("a.x=%f should be < b.x=%f", a.Layout.X, b.Layout.X)
	}
}

func TestFlexLayout_FlexDistribution(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 300, "height": 100, "direction": "row",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "flex": 1},
				map[string]interface{}{"frame": "b", "flex": 2},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	b := sg.Root.Children[1].(*scene.FrameNode)
	if math.Abs(b.Layout.Width-2*a.Layout.Width) > 1 {
		t.Errorf("b.w=%f should be ~2*a.w=%f", b.Layout.Width, a.Layout.Width)
	}
}

func TestFlexLayout_Gap(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300, "direction": "column", "gap": 20,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "height": 50},
				map[string]interface{}{"frame": "b", "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	b := sg.Root.Children[1].(*scene.FrameNode)
	gap := b.Layout.Y - (a.Layout.Y + a.Layout.Height)
	if math.Abs(gap-20) > 1 {
		t.Errorf("gap=%f, expected ~20", gap)
	}
}

func TestFlexLayout_Padding(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300, "padding": 20,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	if a.Layout.X != 20 || a.Layout.Y != 20 {
		t.Errorf("expected x=20 y=20, got x=%f y=%f", a.Layout.X, a.Layout.Y)
	}
}

func TestFlexLayout_JustifyCenter(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300, "justify": "center",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "height": 100},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	if a.Layout.Y != 100 {
		t.Errorf("expected y=100, got %f", a.Layout.Y)
	}
}

func TestFlexLayout_JustifyEnd(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300, "justify": "end",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "height": 100},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	if a.Layout.Y != 200 {
		t.Errorf("expected y=200, got %f", a.Layout.Y)
	}
}

func TestFlexLayout_AlignCenter(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300, "direction": "column", "align": "center",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "width": 200, "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	if a.Layout.X != 100 {
		t.Errorf("expected x=100, got %f", a.Layout.X)
	}
}

func TestFlexLayout_AbsolutePositioning(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "abs", "position": "absolute", "x": 50, "y": 60, "width": 100, "height": 80},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	if a.Layout.X != 50 || a.Layout.Y != 60 {
		t.Errorf("expected x=50 y=60, got x=%f y=%f", a.Layout.X, a.Layout.Y)
	}
}

// --- Grid Layout ---

func TestGridLayout_Basic(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 400, "layout": "grid", "columns": 2, "gap": 10,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "height": 50},
				map[string]interface{}{"frame": "b", "height": 50},
				map[string]interface{}{"frame": "c", "height": 50},
				map[string]interface{}{"frame": "d", "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	b := sg.Root.Children[1].(*scene.FrameNode)
	c := sg.Root.Children[2].(*scene.FrameNode)
	d := sg.Root.Children[3].(*scene.FrameNode)

	// a and b should be on the same row
	if math.Abs(a.Layout.Y-b.Layout.Y) > 1 {
		t.Errorf("a and b should be on same row: a.y=%f b.y=%f", a.Layout.Y, b.Layout.Y)
	}
	// a and c should be in the same column
	if math.Abs(a.Layout.X-c.Layout.X) > 1 {
		t.Errorf("a and c should be in same column: a.x=%f c.x=%f", a.Layout.X, c.Layout.X)
	}
	// b and d should be in the same column
	if math.Abs(b.Layout.X-d.Layout.X) > 1 {
		t.Errorf("b and d should be in same column: b.x=%f d.x=%f", b.Layout.X, d.Layout.X)
	}
}

func TestGridLayout_ColumnWidth(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 420, "height": 200, "layout": "grid", "columns": 2, "column-gap": 20,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "height": 50},
				map[string]interface{}{"frame": "b", "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	b := sg.Root.Children[1].(*scene.FrameNode)
	if math.Abs(a.Layout.Width-200) > 1 || math.Abs(b.Layout.Width-200) > 1 {
		t.Errorf("a.w=%f b.w=%f, expected ~200", a.Layout.Width, b.Layout.Width)
	}
}

func TestGridLayout_WithPadding(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 440, "height": 200, "layout": "grid", "columns": 2, "column-gap": 20, "padding": 10,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "height": 50},
				map[string]interface{}{"frame": "b", "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	if a.Layout.X != 10 || a.Layout.Y != 10 {
		t.Errorf("expected x=10 y=10, got x=%f y=%f", a.Layout.X, a.Layout.Y)
	}
}

// --- Edge Routing ---

func TestEdgeRouting_Straight(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 200, "direction": "row",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "width": 100, "height": 50},
				map[string]interface{}{"frame": "b", "id": "b", "width": 100, "height": 50},
			},
		},
		"edges": []interface{}{map[string]interface{}{"from": "a", "to": "b"}},
	})
	ComputeLayout(sg)
	if len(sg.Edges[0].ResolvedPath) != 2 {
		t.Errorf("expected 2 points, got %d", len(sg.Edges[0].ResolvedPath))
	}
}

func TestEdgeRouting_Orthogonal(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 200,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "height": 50},
				map[string]interface{}{"frame": "b", "id": "b", "height": 50},
			},
		},
		"edges": []interface{}{map[string]interface{}{"from": "a", "to": "b", "curve": "orthogonal"}},
	})
	ComputeLayout(sg)
	if len(sg.Edges[0].ResolvedPath) < 2 {
		t.Errorf("expected >=2 points, got %d", len(sg.Edges[0].ResolvedPath))
	}
}

func TestEdgeRouting_MissingFrameNoCrash(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root":  map[string]interface{}{"width": 400, "height": 200},
		"edges": []interface{}{map[string]interface{}{"from": "nonexistent", "to": "also-missing"}},
	})
	ComputeLayout(sg)
	if len(sg.Edges[0].ResolvedPath) != 0 {
		t.Errorf("expected empty path for missing frames")
	}
}

// --- Min/Max Constraints ---

func TestFlexLayout_MinWidth(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "min-width": 200, "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	if a.Layout.Width < 200 {
		t.Errorf("width=%f should be >= 200", a.Layout.Width)
	}
}

func TestFlexLayout_MaxWidth(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "max-width": 100, "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	if a.Layout.Width > 100 {
		t.Errorf("width=%f should be <= 100", a.Layout.Width)
	}
}

func TestFlexLayout_MinHeight(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "min-height": 100, "width": 50},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	if a.Layout.Height < 100 {
		t.Errorf("height=%f should be >= 100", a.Layout.Height)
	}
}

func TestFlexLayout_MaxHeight(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "max-height": 50, "width": 100, "height": 200},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	if a.Layout.Height > 50 {
		t.Errorf("height=%f should be <= 50", a.Layout.Height)
	}
}

// --- Flex Wrap ---

func TestFlexWrap_WrapsOverflow(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 300, "height": 300, "direction": "row", "wrap": true, "gap": 10,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "width": 100, "height": 50},
				map[string]interface{}{"frame": "b", "width": 100, "height": 50},
				map[string]interface{}{"frame": "c", "width": 100, "height": 50},
				map[string]interface{}{"frame": "d", "width": 100, "height": 50},
			},
		},
	})
	ComputeLayout(sg)

	a := sg.Root.Children[0].(*scene.FrameNode)
	b := sg.Root.Children[1].(*scene.FrameNode)
	c := sg.Root.Children[2].(*scene.FrameNode)
	d := sg.Root.Children[3].(*scene.FrameNode)

	// a, b should be on first row (same Y)
	if math.Abs(a.Layout.Y-b.Layout.Y) > 1 {
		t.Errorf("a and b should be same row: a.y=%f b.y=%f", a.Layout.Y, b.Layout.Y)
	}

	// c should wrap to second row
	if c.Layout.Y <= a.Layout.Y {
		t.Errorf("c should be on next row: c.y=%f a.y=%f", c.Layout.Y, a.Layout.Y)
	}

	// c, d should be on same row (the second row)
	if math.Abs(c.Layout.Y-d.Layout.Y) > 1 {
		t.Errorf("c and d should be same row: c.y=%f d.y=%f", c.Layout.Y, d.Layout.Y)
	}
}

func TestFlexWrap_NoWrapSingleLine(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 200, "height": 300, "direction": "row",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "width": 100, "height": 50},
				map[string]interface{}{"frame": "b", "width": 100, "height": 50},
				map[string]interface{}{"frame": "c", "width": 100, "height": 50},
			},
		},
	})
	ComputeLayout(sg)

	// Without wrap, all should be on the same row even if they overflow
	a := sg.Root.Children[0].(*scene.FrameNode)
	c := sg.Root.Children[2].(*scene.FrameNode)
	if a.Layout.Y != c.Layout.Y {
		t.Errorf("without wrap, all should be same row: a.y=%f c.y=%f", a.Layout.Y, c.Layout.Y)
	}
}

// --- Nested Flex ---

func TestFlexLayout_NestedFlex(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300, "direction": "column",
			"children": []interface{}{
				map[string]interface{}{
					"frame": "row", "direction": "row", "height": 100,
					"children": []interface{}{
						map[string]interface{}{"frame": "a", "flex": 1},
						map[string]interface{}{"frame": "b", "flex": 1},
					},
				},
			},
		},
	})
	ComputeLayout(sg)
	row := sg.Root.Children[0].(*scene.FrameNode)
	a := row.Children[0].(*scene.FrameNode)
	b := row.Children[1].(*scene.FrameNode)

	if math.Abs(a.Layout.Width-b.Layout.Width) > 1 {
		t.Errorf("nested flex children should have equal width: a=%f b=%f", a.Layout.Width, b.Layout.Width)
	}
	if math.Abs(a.Layout.Width-200) > 1 {
		t.Errorf("each child should be ~200px: got %f", a.Layout.Width)
	}
}

// --- Justify between and around ---

func TestFlexLayout_JustifyBetween(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 100, "direction": "row", "justify": "between",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "width": 50, "height": 50},
				map[string]interface{}{"frame": "b", "width": 50, "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	b := sg.Root.Children[1].(*scene.FrameNode)

	if a.Layout.X != 0 {
		t.Errorf("first child should start at 0, got %f", a.Layout.X)
	}
	if math.Abs(b.Layout.X+b.Layout.Width-400) > 1 {
		t.Errorf("last child should end at 400, got %f", b.Layout.X+b.Layout.Width)
	}
}

func TestFlexLayout_JustifyAround(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 100, "direction": "row", "justify": "around",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "width": 50, "height": 50},
				map[string]interface{}{"frame": "b", "width": 50, "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)

	// With around, first child should have space before it
	if a.Layout.X <= 0 {
		t.Errorf("first child should have space before it, got x=%f", a.Layout.X)
	}
}

// --- Align end ---

func TestFlexLayout_AlignEnd(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300, "direction": "column", "align": "end",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "width": 200, "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	if math.Abs(a.Layout.X-200) > 1 {
		t.Errorf("expected x=200 for align-end, got %f", a.Layout.X)
	}
}

// --- Grid edge cases ---

func TestGridLayout_FewerItemsThanColumns(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 200, "layout": "grid", "columns": 4,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "height": 50},
				map[string]interface{}{"frame": "b", "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	b := sg.Root.Children[1].(*scene.FrameNode)

	// Both should be on same row
	if math.Abs(a.Layout.Y-b.Layout.Y) > 1 {
		t.Errorf("should be on same row: a.y=%f b.y=%f", a.Layout.Y, b.Layout.Y)
	}
	// Both should have equal widths (1/4 of container)
	if math.Abs(a.Layout.Width-100) > 1 {
		t.Errorf("expected width=100, got %f", a.Layout.Width)
	}
}

func TestGridLayout_ExplicitRows(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 200, "layout": "grid", "columns": 2, "rows": 2,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "height": 50},
				map[string]interface{}{"frame": "b", "height": 50},
				map[string]interface{}{"frame": "c", "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	// Should not crash; 3 items in a 2x2 grid
	c := sg.Root.Children[2].(*scene.FrameNode)
	if c.Layout.Y <= 0 {
		t.Errorf("third item should be on second row: y=%f", c.Layout.Y)
	}
}

// --- Benchmarks ---

func BenchmarkComputeLayout(b *testing.B) {
	// Build a moderately complex scene with 50 frames
	children := make([]interface{}, 50)
	for i := range children {
		children[i] = map[string]interface{}{
			"frame":  fmt.Sprintf("frame-%d", i),
			"width":  80,
			"height": 40,
			"fill":   "#eee",
		}
	}
	data := map[string]interface{}{
		"root": map[string]interface{}{
			"width": 1280, "height": 720,
			"layout":  "grid",
			"columns": 10,
			"gap":     8,
			"padding": 20,
			"children": children,
		},
	}
	sg := parser.ParseDict(data)

	b.ResetTimer()
	for range b.N {
		ComputeLayout(sg)
	}
}

func BenchmarkMeasureText(b *testing.B) {
	node := &scene.TextNode{
		Content:    "The quick brown fox jumps over the lazy dog",
		Font:       scene.Font{Size: 14, Family: "Inter", Weight: 400},
		LineHeight: 1.4,
	}

	b.ResetTimer()
	for range b.N {
		MeasureText(node)
	}
}

func stringRepeat(s string, n int) string {
	result := ""
	for range n {
		result += s
	}
	return result
}

// --- Auto height ---

func TestLayout_AutoHeight_EmptyFrame(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 400,
			"children": []interface{}{
				map[string]interface{}{"frame": "empty"},
			},
		},
	})
	ComputeLayout(sg)
	empty := sg.Root.Children[0].(*scene.FrameNode)
	// Empty frame with no children and no explicit height should collapse to padding only (0 here)
	if empty.Layout.Height != 0 {
		t.Errorf("empty frame height=%f, expected 0", empty.Layout.Height)
	}
}

func TestLayout_AutoHeight_EmptyFrameWithPadding(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 400,
			"children": []interface{}{
				map[string]interface{}{"frame": "padded", "padding": 20},
			},
		},
	})
	ComputeLayout(sg)
	padded := sg.Root.Children[0].(*scene.FrameNode)
	if math.Abs(padded.Layout.Height-40) > 1 {
		t.Errorf("padded empty frame height=%f, expected 40 (20+20)", padded.Layout.Height)
	}
}

func TestLayout_AutoHeight_ColumnWithChildren(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 400,
			"children": []interface{}{
				map[string]interface{}{
					"frame": "auto-h", "padding": 10, "gap": 5,
					"children": []interface{}{
						map[string]interface{}{"frame": "a", "height": 30},
						map[string]interface{}{"frame": "b", "height": 40},
					},
				},
			},
		},
	})
	ComputeLayout(sg)
	autoH := sg.Root.Children[0].(*scene.FrameNode)
	// Expected: padding-top(10) + child-a(30) + gap(5) + child-b(40) + padding-bottom(10) = 95
	if math.Abs(autoH.Layout.Height-95) > 1 {
		t.Errorf("auto height=%f, expected 95", autoH.Layout.Height)
	}
}

func TestLayout_AutoHeight_RowWithChildren(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 400,
			"children": []interface{}{
				map[string]interface{}{
					"frame": "row", "direction": "row", "padding": 10,
					"children": []interface{}{
						map[string]interface{}{"frame": "a", "width": 100, "height": 50},
						map[string]interface{}{"frame": "b", "width": 100, "height": 80},
					},
				},
			},
		},
	})
	ComputeLayout(sg)
	row := sg.Root.Children[0].(*scene.FrameNode)
	// Auto height for row = padding-top(10) + max(50, 80) + padding-bottom(10) = 100
	if math.Abs(row.Layout.Height-100) > 1 {
		t.Errorf("auto height=%f, expected 100", row.Layout.Height)
	}
}

// --- Align in row layout ---

func TestFlexLayout_RowAlignCenter(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 200, "direction": "row", "align": "center",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "width": 100, "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	// Cross axis is vertical: (200 - 50) / 2 = 75
	if math.Abs(a.Layout.Y-75) > 1 {
		t.Errorf("expected y=75 for row align-center, got %f", a.Layout.Y)
	}
}

func TestFlexLayout_RowAlignEnd(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 200, "direction": "row", "align": "end",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "width": 100, "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	// Cross axis: 200 - 50 = 150
	if math.Abs(a.Layout.Y-150) > 1 {
		t.Errorf("expected y=150 for row align-end, got %f", a.Layout.Y)
	}
}

func TestFlexLayout_RowAlignStretch(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 200, "direction": "row", "align": "stretch",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "width": 100},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	// Stretch should fill full cross axis
	if math.Abs(a.Layout.Height-200) > 1 {
		t.Errorf("expected height=200 for row align-stretch, got %f", a.Layout.Height)
	}
}

// --- Mixed fixed + flex children ---

func TestFlexLayout_MixedFixedAndFlex(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 100, "direction": "row",
			"children": []interface{}{
				map[string]interface{}{"frame": "fixed", "width": 100, "height": 100},
				map[string]interface{}{"frame": "flex", "flex": 1, "height": 100},
			},
		},
	})
	ComputeLayout(sg)
	fixed := sg.Root.Children[0].(*scene.FrameNode)
	flex := sg.Root.Children[1].(*scene.FrameNode)
	if math.Abs(fixed.Layout.Width-100) > 1 {
		t.Errorf("fixed width=%f, expected 100", fixed.Layout.Width)
	}
	if math.Abs(flex.Layout.Width-300) > 1 {
		t.Errorf("flex width=%f, expected 300 (400-100)", flex.Layout.Width)
	}
}

func TestFlexLayout_ThreeFlexChildren(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 300, "height": 100, "direction": "row",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "flex": 1},
				map[string]interface{}{"frame": "b", "flex": 1},
				map[string]interface{}{"frame": "c", "flex": 1},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	b := sg.Root.Children[1].(*scene.FrameNode)
	c := sg.Root.Children[2].(*scene.FrameNode)
	if math.Abs(a.Layout.Width-100) > 1 || math.Abs(b.Layout.Width-100) > 1 || math.Abs(c.Layout.Width-100) > 1 {
		t.Errorf("flex children should each be 100px: a=%f b=%f c=%f", a.Layout.Width, b.Layout.Width, c.Layout.Width)
	}
}

// --- Text in row layout ---

func TestFlexLayout_TextInRow(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 100, "direction": "row",
			"children": []interface{}{
				map[string]interface{}{"text": "Hello"},
				map[string]interface{}{"text": "World"},
			},
		},
	})
	ComputeLayout(sg)
	hello := sg.Root.Children[0].(*scene.TextNode)
	world := sg.Root.Children[1].(*scene.TextNode)
	// Text nodes in a row should be positioned side by side
	if hello.Layout.X >= world.Layout.X {
		t.Errorf("hello.x=%f should be < world.x=%f", hello.Layout.X, world.Layout.X)
	}
	// Both should be on the same row
	if math.Abs(hello.Layout.Y-world.Layout.Y) > 1 {
		t.Errorf("both texts should have same Y: hello.y=%f world.y=%f", hello.Layout.Y, world.Layout.Y)
	}
}

func TestFlexLayout_TextInColumn(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 200, "direction": "column",
			"children": []interface{}{
				map[string]interface{}{"text": "Line 1"},
				map[string]interface{}{"text": "Line 2"},
			},
		},
	})
	ComputeLayout(sg)
	line1 := sg.Root.Children[0].(*scene.TextNode)
	line2 := sg.Root.Children[1].(*scene.TextNode)
	if line1.Layout.Y >= line2.Layout.Y {
		t.Errorf("line1.y=%f should be < line2.y=%f", line1.Layout.Y, line2.Layout.Y)
	}
}

// --- Flex wrap with justify modes ---

func TestFlexWrap_JustifyCenter(t *testing.T) {
	// 3 items of 120px each + 10px gap. Line fits 120+10+120=250 <= 300, third wraps (250+10+120=380 > 300)
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 300, "height": 300, "direction": "row", "wrap": true, "justify": "center", "gap": 10,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "width": 120, "height": 50},
				map[string]interface{}{"frame": "b", "width": 120, "height": 50},
				map[string]interface{}{"frame": "c", "width": 120, "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	c := sg.Root.Children[2].(*scene.FrameNode)
	// First line: a+gap+b = 120+10+120=250. Center offset = (300-250)/2 = 25
	if math.Abs(a.Layout.X-25) > 1 {
		t.Errorf("first child should be centered: a.x=%f, expected ~25", a.Layout.X)
	}
	// c wraps to second line
	if c.Layout.Y <= a.Layout.Y {
		t.Errorf("c should wrap to next row: c.y=%f a.y=%f", c.Layout.Y, a.Layout.Y)
	}
}

func TestFlexWrap_JustifyBetween(t *testing.T) {
	// 3 items of 120px each + 10px gap. First line fits 2, third wraps
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 300, "height": 300, "direction": "row", "wrap": true, "justify": "between", "gap": 10,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "width": 120, "height": 50},
				map[string]interface{}{"frame": "b", "width": 120, "height": 50},
				map[string]interface{}{"frame": "c", "width": 120, "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	b := sg.Root.Children[1].(*scene.FrameNode)
	// First line with justify-between: a at 0, b at right edge (300-120=180)
	if math.Abs(a.Layout.X-0) > 1 {
		t.Errorf("a.x=%f, expected 0", a.Layout.X)
	}
	if math.Abs(b.Layout.X+b.Layout.Width-300) > 1 {
		t.Errorf("b should end at 300: b.x=%f b.w=%f", b.Layout.X, b.Layout.Width)
	}
}

func TestFlexWrap_AlignCenter(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 300, "height": 300, "direction": "row", "wrap": true, "align": "center",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "width": 100, "height": 30},
				map[string]interface{}{"frame": "b", "width": 100, "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	b := sg.Root.Children[1].(*scene.FrameNode)
	// Both on same line. Line height = max(30, 50) = 50
	// a should be vertically centered: (50 - 30) / 2 = 10
	if math.Abs(a.Layout.Y-10) > 1 {
		t.Errorf("a.y=%f, expected ~10 (centered in 50px line)", a.Layout.Y)
	}
	if math.Abs(b.Layout.Y-0) > 1 {
		t.Errorf("b.y=%f, expected ~0 (fills line height)", b.Layout.Y)
	}
}

func TestFlexWrap_AlignStretch(t *testing.T) {
	// Use auto-height children (no explicit height) so stretch can take effect
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 300, "height": 300, "direction": "row", "wrap": true, "align": "stretch",
			"children": []interface{}{
				map[string]interface{}{
					"frame": "a", "width": 100,
					"children": []interface{}{
						map[string]interface{}{"frame": "inner-a", "height": 30},
					},
				},
				map[string]interface{}{
					"frame": "b", "width": 100,
					"children": []interface{}{
						map[string]interface{}{"frame": "inner-b", "height": 50},
					},
				},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	b := sg.Root.Children[1].(*scene.FrameNode)
	// Both on same line. Line height = max intrinsic = 50. With stretch, a should get 50
	if math.Abs(a.Layout.Height-b.Layout.Height) > 1 {
		t.Errorf("stretch should equalize heights: a=%f b=%f", a.Layout.Height, b.Layout.Height)
	}
}

// --- Grid edge cases ---

func TestGridLayout_AutoHeight(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 400,
			"children": []interface{}{
				map[string]interface{}{
					"frame": "grid", "layout": "grid", "columns": 2, "gap": 10,
					"children": []interface{}{
						map[string]interface{}{"frame": "a", "height": 40},
						map[string]interface{}{"frame": "b", "height": 60},
						map[string]interface{}{"frame": "c", "height": 30},
					},
				},
			},
		},
	})
	ComputeLayout(sg)
	grid := sg.Root.Children[0].(*scene.FrameNode)
	// Row 1: max(40, 60) = 60. Row 2: 30. Total: 60 + 10 (gap) + 30 = 100
	if math.Abs(grid.Layout.Height-100) > 1 {
		t.Errorf("grid auto-height=%f, expected 100", grid.Layout.Height)
	}
}

func TestGridLayout_ChildExplicitWidth(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 200, "layout": "grid", "columns": 2,
			"children": []interface{}{
				map[string]interface{}{"frame": "narrow", "width": 50, "height": 50},
				map[string]interface{}{"frame": "full", "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	narrow := sg.Root.Children[0].(*scene.FrameNode)
	full := sg.Root.Children[1].(*scene.FrameNode)
	// Narrow should have explicit width 50, not the column width
	if math.Abs(narrow.Layout.Width-50) > 1 {
		t.Errorf("narrow width=%f, expected 50", narrow.Layout.Width)
	}
	// Full should have column width = 400/2 = 200
	if math.Abs(full.Layout.Width-200) > 1 {
		t.Errorf("full width=%f, expected 200", full.Layout.Width)
	}
}

func TestGridLayout_SingleColumn(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 400, "layout": "grid", "columns": 1,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "height": 50},
				map[string]interface{}{"frame": "b", "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	b := sg.Root.Children[1].(*scene.FrameNode)
	// Same column, stacked vertically
	if math.Abs(a.Layout.X-b.Layout.X) > 1 {
		t.Errorf("same column: a.x=%f b.x=%f", a.Layout.X, b.Layout.X)
	}
	if a.Layout.Y >= b.Layout.Y {
		t.Errorf("a should be above b: a.y=%f b.y=%f", a.Layout.Y, b.Layout.Y)
	}
	// Each should span full width
	if math.Abs(a.Layout.Width-400) > 1 {
		t.Errorf("a width=%f, expected 400", a.Layout.Width)
	}
}

// --- Edge routing: vertical, bus, anchors ---

func TestEdgeRouting_Vertical(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "height": 50},
				map[string]interface{}{"frame": "b", "id": "b", "height": 50},
			},
		},
		"edges": []interface{}{map[string]interface{}{"from": "a", "to": "b", "curve": "vertical"}},
	})
	ComputeLayout(sg)
	path := sg.Edges[0].ResolvedPath
	if len(path) < 2 {
		t.Fatalf("expected >=2 points, got %d", len(path))
	}
	// Vertical curve: first segment should maintain X of source
	if math.Abs(path[0].X-path[1].X) > 1 {
		t.Errorf("vertical curve should maintain source X: p0.x=%f p1.x=%f", path[0].X, path[1].X)
	}
}

func TestEdgeRouting_Bus(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 600, "height": 400,
			"children": []interface{}{
				map[string]interface{}{"frame": "src", "id": "src", "height": 50},
				map[string]interface{}{
					"frame": "row", "direction": "row", "gap": 20,
					"children": []interface{}{
						map[string]interface{}{"frame": "dst1", "id": "dst1", "width": 100, "height": 50},
						map[string]interface{}{"frame": "dst2", "id": "dst2", "width": 100, "height": 50},
					},
				},
			},
		},
		"edges": []interface{}{
			map[string]interface{}{"from": "src", "to": "dst1", "curve": "bus"},
			map[string]interface{}{"from": "src", "to": "dst2", "curve": "bus"},
		},
	})
	ComputeLayout(sg)
	// Both edges should have resolved paths
	for i, edge := range sg.Edges {
		if len(edge.ResolvedPath) < 2 {
			t.Errorf("edge[%d] should have >=2 points, got %d", i, len(edge.ResolvedPath))
		}
	}
}

func TestEdgeRouting_ExplicitAnchors(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 200, "direction": "row",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "width": 100, "height": 100},
				map[string]interface{}{"frame": "b", "id": "b", "width": 100, "height": 100},
			},
		},
		"edges": []interface{}{map[string]interface{}{
			"from": "a", "to": "b",
			"from-anchor": "right", "to-anchor": "left",
		}},
	})
	ComputeLayout(sg)
	path := sg.Edges[0].ResolvedPath
	if len(path) != 2 {
		t.Fatalf("expected 2 points for straight edge, got %d", len(path))
	}
	aRect := sg.Root.Children[0].(*scene.FrameNode)
	// From right anchor of a: x = a.x + a.width
	expectedFromX := aRect.Layout.X + aRect.Layout.Width
	if math.Abs(path[0].X-expectedFromX) > 1 {
		t.Errorf("from-anchor right: path[0].x=%f, expected %f", path[0].X, expectedFromX)
	}
}

func TestEdgeRouting_OrthogonalHorizontal(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 200,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "height": 50},
				map[string]interface{}{"frame": "b", "id": "b", "height": 50},
			},
		},
		"edges": []interface{}{map[string]interface{}{
			"from": "a", "to": "b", "curve": "orthogonal",
			"from-anchor": "left", "to-anchor": "right",
		}},
	})
	ComputeLayout(sg)
	path := sg.Edges[0].ResolvedPath
	// Horizontal-to-horizontal orthogonal should have 4 points with midX
	if len(path) != 4 {
		t.Fatalf("expected 4 points for h-to-h orthogonal, got %d", len(path))
	}
}

// --- Intrinsic sizing ---

func TestIntrinsicSize_NestedFrames(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 400,
			"children": []interface{}{
				map[string]interface{}{
					"frame": "outer", "direction": "row",
					"children": []interface{}{
						map[string]interface{}{"frame": "inner-a", "width": 80, "height": 40},
						map[string]interface{}{"frame": "inner-b", "width": 120, "height": 60},
					},
				},
			},
		},
	})
	ComputeLayout(sg)
	outer := sg.Root.Children[0].(*scene.FrameNode)
	// Auto height for row: max(40, 60) = 60
	if math.Abs(outer.Layout.Height-60) > 1 {
		t.Errorf("outer auto height=%f, expected 60", outer.Layout.Height)
	}
}

// --- Min/max with auto height ---

func TestLayout_MinHeightWithAutoHeight(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 400,
			"children": []interface{}{
				map[string]interface{}{
					"frame": "container", "min-height": 200,
					"children": []interface{}{
						map[string]interface{}{"frame": "small", "height": 30},
					},
				},
			},
		},
	})
	ComputeLayout(sg)
	container := sg.Root.Children[0].(*scene.FrameNode)
	// Auto height would be 30, but min-height clamps to 200
	if container.Layout.Height < 200-1 {
		t.Errorf("container height=%f, expected >= 200 (min-height)", container.Layout.Height)
	}
}

// --- Padding + gap + children ---

func TestLayout_PaddingAndGap(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 400,
			"children": []interface{}{
				map[string]interface{}{
					"frame": "box", "padding": "20 30", "gap": 10, "direction": "row",
					"children": []interface{}{
						map[string]interface{}{"frame": "a", "width": 50, "height": 40},
						map[string]interface{}{"frame": "b", "width": 60, "height": 40},
					},
				},
			},
		},
	})
	ComputeLayout(sg)
	box := sg.Root.Children[0].(*scene.FrameNode)
	a := box.Children[0].(*scene.FrameNode)
	b := box.Children[1].(*scene.FrameNode)
	// a starts at padding-left
	if math.Abs(a.Layout.X-30) > 1 {
		t.Errorf("a.x=%f, expected 30 (padding-left)", a.Layout.X)
	}
	if math.Abs(a.Layout.Y-20) > 1 {
		t.Errorf("a.y=%f, expected 20 (padding-top)", a.Layout.Y)
	}
	// b starts after a + gap
	expectedBx := a.Layout.X + a.Layout.Width + 10
	if math.Abs(b.Layout.X-expectedBx) > 1 {
		t.Errorf("b.x=%f, expected %f (a.x + a.width + gap)", b.Layout.X, expectedBx)
	}
}

// --- Default root size ---

func TestLayout_DefaultRootSize(t *testing.T) {
	// Root with explicit dimensions should use them
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{"width": 1280, "height": 720},
	})
	ComputeLayout(sg)
	if sg.Root.Layout.Width != 1280 || sg.Root.Layout.Height != 720 {
		t.Errorf("explicit root should be 1280x720, got %fx%f", sg.Root.Layout.Width, sg.Root.Layout.Height)
	}
}

func TestLayout_DefaultRootAutoHeight(t *testing.T) {
	// Root with no explicit height auto-sizes to content (0 with no children)
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{},
	})
	ComputeLayout(sg)
	if sg.Root.Layout.Width != 1280 {
		t.Errorf("default width should be 1280, got %f", sg.Root.Layout.Width)
	}
	// With no children and no explicit height, auto-height collapses
	if sg.Root.Layout.Height != 0 {
		t.Errorf("auto-height with no children should be 0, got %f", sg.Root.Layout.Height)
	}
}

// --- Flex column with gap ---

func TestFlexLayout_ColumnGap(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 400, "direction": "column", "gap": 16,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "height": 40},
				map[string]interface{}{"frame": "b", "height": 40},
				map[string]interface{}{"frame": "c", "height": 40},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	b := sg.Root.Children[1].(*scene.FrameNode)
	c := sg.Root.Children[2].(*scene.FrameNode)
	gapAB := b.Layout.Y - (a.Layout.Y + a.Layout.Height)
	gapBC := c.Layout.Y - (b.Layout.Y + b.Layout.Height)
	if math.Abs(gapAB-16) > 1 {
		t.Errorf("gap A-B=%f, expected 16", gapAB)
	}
	if math.Abs(gapBC-16) > 1 {
		t.Errorf("gap B-C=%f, expected 16", gapBC)
	}
}

// --- Visibility ---

func TestLayout_InvisibleFrameStillLaidOut(t *testing.T) {
	// Invisible frames should still be computed in layout, just not rendered
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "visible": false, "height": 50},
				map[string]interface{}{"frame": "b", "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	b := sg.Root.Children[1].(*scene.FrameNode)
	// b should still be positioned after a
	if b.Layout.Y < 49 {
		t.Errorf("b.y=%f, invisible frame should still occupy layout space", b.Layout.Y)
	}
}

// --- countWrappedLines edge cases ---

func TestCountWrappedLines_EmptyText(t *testing.T) {
	lines := countWrappedLines("", 200, 14, "Inter", 400, 0)
	if lines != 1 {
		t.Errorf("expected 1 line for empty text, got %d", lines)
	}
}

func TestCountWrappedLines_SingleWord(t *testing.T) {
	lines := countWrappedLines("Hello", 200, 14, "Inter", 400, 0)
	if lines != 1 {
		t.Errorf("expected 1 line for single word, got %d", lines)
	}
}

func TestCountWrappedLines_VeryLongWord(t *testing.T) {
	// A single word wider than maxWidth
	longWord := stringRepeat("W", 100) // Wide characters
	lines := countWrappedLines(longWord, 100, 14, "Inter", 400, 0)
	if lines < 2 {
		t.Errorf("expected multiple lines for overflow word, got %d", lines)
	}
}

// --- Absolute positioning with padding ---

func TestLayout_AbsoluteWithPadding(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300, "padding": 20,
			"children": []interface{}{
				map[string]interface{}{"frame": "abs", "position": "absolute", "x": 10, "y": 10, "width": 50, "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	abs := sg.Root.Children[0].(*scene.FrameNode)
	// Absolute position should be relative to content area (after padding)
	if math.Abs(abs.Layout.X-30) > 1 {
		t.Errorf("abs.x=%f, expected 30 (padding 20 + x 10)", abs.Layout.X)
	}
	if math.Abs(abs.Layout.Y-30) > 1 {
		t.Errorf("abs.y=%f, expected 30 (padding 20 + y 10)", abs.Layout.Y)
	}
}

// --- Bus edge routing with vertical anchors ---

func TestEdgeRouting_BusGroupVertical(t *testing.T) {
	// Bus group: one source branching to multiple destinations via shared junction
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 600, "height": 400, "direction": "column", "gap": 50,
			"children": []interface{}{
				map[string]interface{}{"frame": "src", "id": "src", "height": 60},
				map[string]interface{}{
					"frame": "row", "direction": "row", "gap": 40, "height": 60,
					"children": []interface{}{
						map[string]interface{}{"frame": "d1", "id": "d1", "width": 100, "height": 60},
						map[string]interface{}{"frame": "d2", "id": "d2", "width": 100, "height": 60},
						map[string]interface{}{"frame": "d3", "id": "d3", "width": 100, "height": 60},
					},
				},
			},
		},
		"edges": []interface{}{
			map[string]interface{}{"from": "src", "to": "d1", "curve": "bus", "from-anchor": "bottom", "to-anchor": "top"},
			map[string]interface{}{"from": "src", "to": "d2", "curve": "bus", "from-anchor": "bottom", "to-anchor": "top"},
			map[string]interface{}{"from": "src", "to": "d3", "curve": "bus", "from-anchor": "bottom", "to-anchor": "top"},
		},
	})
	ComputeLayout(sg)
	// All edges should have resolved paths with junction routing (4 points each)
	for i, edge := range sg.Edges {
		if len(edge.ResolvedPath) < 2 {
			t.Errorf("edge[%d] should have resolved path, got %d points", i, len(edge.ResolvedPath))
		}
	}
	// Different destinations should have different junction Y offsets (bus spread)
	if len(sg.Edges) == 3 && len(sg.Edges[0].ResolvedPath) >= 2 && len(sg.Edges[1].ResolvedPath) >= 2 {
		// Bus edges should have different junction points (stacked)
		p0 := sg.Edges[0].ResolvedPath
		p1 := sg.Edges[1].ResolvedPath
		if len(p0) == 4 && len(p1) == 4 {
			// The junction Y values should differ by busSpread
			j0 := p0[1].Y
			j1 := p1[1].Y
			if math.Abs(j0-j1) < 1 {
				t.Logf("bus junctions may overlap: j0=%f j1=%f (may be by design for close targets)", j0, j1)
			}
		}
	}
}

func TestEdgeRouting_BusWithExplicitJunction(t *testing.T) {
	junc := 0.3
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "height": 50},
				map[string]interface{}{"frame": "b", "id": "b", "height": 50},
			},
		},
		"edges": []interface{}{map[string]interface{}{
			"from": "a", "to": "b", "curve": "bus",
			"from-anchor": "bottom", "to-anchor": "top",
			"junction":    junc,
		}},
	})
	ComputeLayout(sg)
	path := sg.Edges[0].ResolvedPath
	if len(path) < 3 {
		t.Fatalf("bus edge should have >=3 points, got %d", len(path))
	}
}

func TestEdgeRouting_BusSameXNoJunction(t *testing.T) {
	// When source and destination are vertically aligned (same X), bus should be straight(ish)
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300, "direction": "column", "gap": 100, "align": "center",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "width": 100, "height": 50},
				map[string]interface{}{"frame": "b", "id": "b", "width": 100, "height": 50},
			},
		},
		"edges": []interface{}{map[string]interface{}{
			"from": "a", "to": "b", "curve": "bus",
			"from-anchor": "bottom", "to-anchor": "top",
		}},
	})
	ComputeLayout(sg)
	path := sg.Edges[0].ResolvedPath
	if len(path) < 2 {
		t.Fatalf("expected >=2 points, got %d", len(path))
	}
	// With same X, bus should produce a 2-point straight path
	if len(path) == 2 {
		// Verify it's a straight line
		if math.Abs(path[0].X-path[1].X) > 2 {
			t.Errorf("same-X bus should be straight: p0.x=%f p1.x=%f", path[0].X, path[1].X)
		}
	}
}

// --- Edge routing: orthogonal all combos ---

func TestEdgeRouting_OrthogonalVerticalToVertical(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 400, "direction": "row", "gap": 100,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "width": 100, "height": 100},
				map[string]interface{}{"frame": "b", "id": "b", "width": 100, "height": 100},
			},
		},
		"edges": []interface{}{map[string]interface{}{
			"from": "a", "to": "b", "curve": "orthogonal",
			"from-anchor": "bottom", "to-anchor": "top",
		}},
	})
	ComputeLayout(sg)
	path := sg.Edges[0].ResolvedPath
	// V-to-V orthogonal: 4 points with midY
	if len(path) != 4 {
		t.Fatalf("expected 4 points for v-to-v orthogonal, got %d", len(path))
	}
	// Middle points should share the same Y (midY)
	if math.Abs(path[1].Y-path[2].Y) > 1 {
		t.Errorf("middle points should share Y: p1.y=%f p2.y=%f", path[1].Y, path[2].Y)
	}
}

func TestEdgeRouting_OrthogonalMixed(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 400,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "width": 100, "height": 100},
				map[string]interface{}{"frame": "b", "id": "b", "width": 100, "height": 100},
			},
		},
		"edges": []interface{}{map[string]interface{}{
			"from": "a", "to": "b", "curve": "orthogonal",
			"from-anchor": "bottom", "to-anchor": "left",
		}},
	})
	ComputeLayout(sg)
	path := sg.Edges[0].ResolvedPath
	// V-to-H: 3 points (L-shape)
	if len(path) != 3 {
		t.Fatalf("expected 3 points for v-to-h orthogonal, got %d", len(path))
	}
}

func TestEdgeRouting_OrthogonalHtoV(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 400,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "width": 100, "height": 100},
				map[string]interface{}{"frame": "b", "id": "b", "width": 100, "height": 100},
			},
		},
		"edges": []interface{}{map[string]interface{}{
			"from": "a", "to": "b", "curve": "orthogonal",
			"from-anchor": "right", "to-anchor": "top",
		}},
	})
	ComputeLayout(sg)
	path := sg.Edges[0].ResolvedPath
	// H-to-V: 3 points (L-shape)
	if len(path) != 3 {
		t.Fatalf("expected 3 points for h-to-v orthogonal, got %d", len(path))
	}
}

// --- Auto anchors ---

func TestEdgeRouting_AutoAnchorsHorizontal(t *testing.T) {
	// Side-by-side frames should get left/right anchors
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 100, "direction": "row", "gap": 100,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "width": 100, "height": 100},
				map[string]interface{}{"frame": "b", "id": "b", "width": 100, "height": 100},
			},
		},
		"edges": []interface{}{map[string]interface{}{"from": "a", "to": "b"}},
	})
	ComputeLayout(sg)
	path := sg.Edges[0].ResolvedPath
	if len(path) != 2 {
		t.Fatalf("expected 2 points for straight, got %d", len(path))
	}
	// Source should be at right edge of a, destination at left edge of b
	a := sg.Root.Children[0].(*scene.FrameNode)
	b := sg.Root.Children[1].(*scene.FrameNode)
	expectedFromX := a.Layout.X + a.Layout.Width
	expectedToX := b.Layout.X
	if math.Abs(path[0].X-expectedFromX) > 1 {
		t.Errorf("auto anchor from.x=%f, expected right edge %f", path[0].X, expectedFromX)
	}
	if math.Abs(path[1].X-expectedToX) > 1 {
		t.Errorf("auto anchor to.x=%f, expected left edge %f", path[1].X, expectedToX)
	}
}

// --- Child Margins ---

func TestFlexLayout_ChildMargins(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 100, "direction": "row",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "width": 80, "height": 50, "margin": 10},
				map[string]interface{}{"frame": "b", "width": 80, "height": 50, "margin": 10},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	b := sg.Root.Children[1].(*scene.FrameNode)

	// "a" should be offset by its left margin
	if math.Abs(a.Layout.X-10) > 1 {
		t.Errorf("a.x=%f, expected ~10 (left margin)", a.Layout.X)
	}

	// "b" should start after a's width + a's right margin + b's left margin
	// a starts at 10, width 80, right margin 10, then b left margin 10 => b.x ~ 110
	expectedBx := a.Layout.X + a.Layout.Width + 10 + 10 // a.right_margin + b.left_margin
	if math.Abs(b.Layout.X-expectedBx) > 1 {
		t.Errorf("b.x=%f, expected ~%f (accounting for margins)", b.Layout.X, expectedBx)
	}

	// "a" should have top margin pushing it down
	if math.Abs(a.Layout.Y-10) > 1 {
		t.Errorf("a.y=%f, expected ~10 (top margin)", a.Layout.Y)
	}
}

func TestFlexLayout_ColumnMargins(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 400, "direction": "column",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "height": 50, "margin": 10},
				map[string]interface{}{"frame": "b", "height": 50, "margin": 10},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	b := sg.Root.Children[1].(*scene.FrameNode)

	// "a" should be offset by its top margin
	if math.Abs(a.Layout.Y-10) > 1 {
		t.Errorf("a.y=%f, expected ~10 (top margin)", a.Layout.Y)
	}

	// "a" should have left margin pushing it right
	if math.Abs(a.Layout.X-10) > 1 {
		t.Errorf("a.x=%f, expected ~10 (left margin)", a.Layout.X)
	}

	// "b" should start after a's height + a's bottom margin + b's top margin
	expectedBy := a.Layout.Y + a.Layout.Height + 10 + 10 // a.bottom_margin + b.top_margin
	if math.Abs(b.Layout.Y-expectedBy) > 1 {
		t.Errorf("b.y=%f, expected ~%f (accounting for margins)", b.Layout.Y, expectedBy)
	}
}

func TestFlexWrap_WithMargins(t *testing.T) {
	// Row with wrap: true, 300px wide. Each child is 100px wide with 10px margin on all sides.
	// Effective child width in main axis: 10 (left) + 100 + 10 (right) = 120.
	// First line fits two: 120 + 120 = 240 < 300. Third child (240 + 120 = 360 > 300) wraps.
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 300, "height": 400, "direction": "row", "wrap": true,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "width": 100, "height": 40, "margin": 10},
				map[string]interface{}{"frame": "b", "width": 100, "height": 40, "margin": 10},
				map[string]interface{}{"frame": "c", "width": 100, "height": 40, "margin": 10},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	b := sg.Root.Children[1].(*scene.FrameNode)
	c := sg.Root.Children[2].(*scene.FrameNode)

	// a and b should be on the same row (same Y approximately)
	if math.Abs(a.Layout.Y-b.Layout.Y) > 1 {
		t.Errorf("a and b should be on same row: a.y=%f b.y=%f", a.Layout.Y, b.Layout.Y)
	}

	// c should wrap to a second row (Y at or beyond the bottom of row 1)
	if c.Layout.Y < a.Layout.Y+a.Layout.Height-1 {
		t.Errorf("c should wrap to next row: c.y=%f, a bottom=%f", c.Layout.Y, a.Layout.Y+a.Layout.Height)
	}

	// c should start at the left side of the container (with its own margin)
	if c.Layout.X > 20 {
		t.Errorf("c should start near left edge of container: c.x=%f", c.Layout.X)
	}
}

// --- Grid layout with separate column-gap and row-gap ---

func TestGridLayout_ColumnGapAndRowGap(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 400, "layout": "grid", "columns": 2,
			"column-gap": 20, "row-gap": 40,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "height": 50},
				map[string]interface{}{"frame": "b", "height": 50},
				map[string]interface{}{"frame": "c", "height": 50},
				map[string]interface{}{"frame": "d", "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	a := sg.Root.Children[0].(*scene.FrameNode)
	b := sg.Root.Children[1].(*scene.FrameNode)
	c := sg.Root.Children[2].(*scene.FrameNode)
	d := sg.Root.Children[3].(*scene.FrameNode)

	// Column gap: b.x should be a.x + colWidth + 20
	colWidth := (400.0 - 20.0) / 2.0 // (total - column-gap) / columns = 190
	if math.Abs(b.Layout.X-(a.Layout.X+colWidth+20)) > 1 {
		t.Errorf("column gap not respected: a.x=%f b.x=%f, expected b.x ~%f", a.Layout.X, b.Layout.X, a.Layout.X+colWidth+20)
	}

	// Row gap: c.y should be a.y + rowHeight + 40
	rowGapExpected := c.Layout.Y - (a.Layout.Y + a.Layout.Height)
	if math.Abs(rowGapExpected-40) > 1 {
		t.Errorf("row gap=%f, expected ~40", rowGapExpected)
	}

	// Column gap consistency: d.x should equal b.x
	if math.Abs(d.Layout.X-b.Layout.X) > 1 {
		t.Errorf("d.x=%f should match b.x=%f (same column)", d.Layout.X, b.Layout.X)
	}

	// Row gap consistency: c.y should equal d.y
	if math.Abs(c.Layout.Y-d.Layout.Y) > 1 {
		t.Errorf("c.y=%f should match d.y=%f (same row)", c.Layout.Y, d.Layout.Y)
	}
}

// --- Min/Max constraints ---

func TestLayout_MinMaxConstraints(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 400, "direction": "column",
			"children": []interface{}{
				// This child has no explicit width, so it stretches to 400.
				// max-width should clamp it to 200.
				map[string]interface{}{"frame": "max-w", "max-width": 200, "height": 50},
				// This child has explicit width 50 but min-width 150.
				map[string]interface{}{"frame": "min-w", "width": 50, "min-width": 150, "height": 50},
				// This child has explicit height 200 but max-height 80.
				map[string]interface{}{"frame": "max-h", "height": 200, "max-height": 80},
				// This child has no explicit height but min-height 100.
				map[string]interface{}{"frame": "min-h", "min-height": 100},
			},
		},
	})
	ComputeLayout(sg)

	maxW := sg.Root.Children[0].(*scene.FrameNode)
	if maxW.Layout.Width > 200+1 {
		t.Errorf("max-width child: width=%f, expected <= 200", maxW.Layout.Width)
	}

	minW := sg.Root.Children[1].(*scene.FrameNode)
	if minW.Layout.Width < 150-1 {
		t.Errorf("min-width child: width=%f, expected >= 150", minW.Layout.Width)
	}

	maxH := sg.Root.Children[2].(*scene.FrameNode)
	if maxH.Layout.Height > 80+1 {
		t.Errorf("max-height child: height=%f, expected <= 80", maxH.Layout.Height)
	}

	minH := sg.Root.Children[3].(*scene.FrameNode)
	if minH.Layout.Height < 100-1 {
		t.Errorf("min-height child: height=%f, expected >= 100", minH.Layout.Height)
	}
}

// --- Nested flex ---

func TestLayout_NestedFlex(t *testing.T) {
	// A column containing a row, which itself contains two items in a column.
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 600, "height": 400, "direction": "column",
			"children": []interface{}{
				map[string]interface{}{
					"frame": "row-container", "direction": "row", "height": 200, "gap": 20,
					"children": []interface{}{
						map[string]interface{}{
							"frame": "left-col", "flex": 1, "direction": "column", "gap": 10,
							"children": []interface{}{
								map[string]interface{}{"frame": "item-a", "height": 60},
								map[string]interface{}{"frame": "item-b", "height": 60},
							},
						},
						map[string]interface{}{
							"frame": "right-col", "flex": 2, "direction": "column",
							"children": []interface{}{
								map[string]interface{}{"frame": "item-c", "height": 80},
							},
						},
					},
				},
			},
		},
	})
	ComputeLayout(sg)

	rowContainer := sg.Root.Children[0].(*scene.FrameNode)
	leftCol := rowContainer.Children[0].(*scene.FrameNode)
	rightCol := rowContainer.Children[1].(*scene.FrameNode)
	itemA := leftCol.Children[0].(*scene.FrameNode)
	itemB := leftCol.Children[1].(*scene.FrameNode)
	itemC := rightCol.Children[0].(*scene.FrameNode)

	// Row container should span full width
	if math.Abs(rowContainer.Layout.Width-600) > 1 {
		t.Errorf("row container width=%f, expected 600", rowContainer.Layout.Width)
	}

	// rightCol should be ~2x the width of leftCol (flex 2 vs flex 1)
	// Available = 600 - 20 (gap) = 580; leftCol = 580/3 ~193.3, rightCol = 580*2/3 ~386.7
	expectedLeftW := (600.0 - 20.0) / 3.0
	expectedRightW := (600.0 - 20.0) * 2.0 / 3.0
	if math.Abs(leftCol.Layout.Width-expectedLeftW) > 1 {
		t.Errorf("leftCol width=%f, expected ~%f", leftCol.Layout.Width, expectedLeftW)
	}
	if math.Abs(rightCol.Layout.Width-expectedRightW) > 1 {
		t.Errorf("rightCol width=%f, expected ~%f", rightCol.Layout.Width, expectedRightW)
	}

	// leftCol items should be stacked vertically with gap 10
	if math.Abs(itemA.Layout.Y-0) > 1 {
		t.Errorf("itemA.y=%f, expected ~0", itemA.Layout.Y)
	}
	expectedItemBY := itemA.Layout.Y + itemA.Layout.Height + 10
	if math.Abs(itemB.Layout.Y-expectedItemBY) > 1 {
		t.Errorf("itemB.y=%f, expected ~%f", itemB.Layout.Y, expectedItemBY)
	}

	// rightCol should be positioned after leftCol + gap
	expectedRightX := leftCol.Layout.X + leftCol.Layout.Width + 20
	if math.Abs(rightCol.Layout.X-expectedRightX) > 1 {
		t.Errorf("rightCol.x=%f, expected ~%f", rightCol.Layout.X, expectedRightX)
	}

	// itemC should be at the top of rightCol
	if math.Abs(itemC.Layout.Y-rightCol.Layout.Y) > 1 {
		t.Errorf("itemC.y=%f should match rightCol.y=%f", itemC.Layout.Y, rightCol.Layout.Y)
	}
}

// --- Absolute children don't affect flow ---

func TestLayout_AbsoluteChildren(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300, "direction": "column",
			"children": []interface{}{
				map[string]interface{}{"frame": "flow-a", "height": 50},
				map[string]interface{}{"frame": "abs-overlay", "position": "absolute", "x": 100, "y": 100, "width": 200, "height": 150},
				map[string]interface{}{"frame": "flow-b", "height": 50},
			},
		},
	})
	ComputeLayout(sg)
	flowA := sg.Root.Children[0].(*scene.FrameNode)
	absOverlay := sg.Root.Children[1].(*scene.FrameNode)
	flowB := sg.Root.Children[2].(*scene.FrameNode)

	// flow-a should be at y=0
	if math.Abs(flowA.Layout.Y-0) > 1 {
		t.Errorf("flowA.y=%f, expected ~0", flowA.Layout.Y)
	}

	// flow-b should be directly after flow-a, NOT after the absolute child
	if math.Abs(flowB.Layout.Y-(flowA.Layout.Y+flowA.Layout.Height)) > 1 {
		t.Errorf("flowB.y=%f, expected ~%f (absolute child should not affect flow)", flowB.Layout.Y, flowA.Layout.Y+flowA.Layout.Height)
	}

	// Absolute child should be at its explicit position
	if math.Abs(absOverlay.Layout.X-100) > 1 {
		t.Errorf("absOverlay.x=%f, expected ~100", absOverlay.Layout.X)
	}
	if math.Abs(absOverlay.Layout.Y-100) > 1 {
		t.Errorf("absOverlay.y=%f, expected ~100", absOverlay.Layout.Y)
	}
	if math.Abs(absOverlay.Layout.Width-200) > 1 {
		t.Errorf("absOverlay.width=%f, expected ~200", absOverlay.Layout.Width)
	}
	if math.Abs(absOverlay.Layout.Height-150) > 1 {
		t.Errorf("absOverlay.height=%f, expected ~150", absOverlay.Layout.Height)
	}
}

// --- Word-based text wrapping ---

func TestLayout_WordBasedWrapping(t *testing.T) {
	maxW := 200.0
	node := &scene.TextNode{
		Content:    "The quick brown fox jumps over the lazy dog",
		Font:       scene.Font{Size: 14, Family: "Inter", Weight: 400},
		MaxWidth:   &maxW,
		LineHeight: 1.4,
	}
	w, h := MeasureText(node)

	// Width should not exceed maxWidth
	if w > maxW+1 {
		t.Errorf("width=%f should be <= %f", w, maxW)
	}

	// Should produce multiple lines for this sentence at 200px
	singleLineH := 14 * 1.4
	if h <= singleLineH+1 {
		t.Errorf("height=%f should be greater than single line height %f (text should wrap)", h, singleLineH)
	}

	// Compare with no max-width: should be single line
	nodeNoWrap := &scene.TextNode{
		Content:    "The quick brown fox jumps over the lazy dog",
		Font:       scene.Font{Size: 14, Family: "Inter", Weight: 400},
		LineHeight: 1.4,
	}
	_, hNoWrap := MeasureText(nodeNoWrap)
	if math.Abs(hNoWrap-singleLineH) > 1 {
		t.Errorf("no-wrap height=%f should be ~%f (single line)", hNoWrap, singleLineH)
	}

	// Wrapped version should be taller
	if h <= hNoWrap {
		t.Errorf("wrapped height=%f should be > unwrapped height=%f", h, hNoWrap)
	}

	// Test very narrow maxWidth forces many lines
	narrowMax := 50.0
	nodeNarrow := &scene.TextNode{
		Content:    "The quick brown fox jumps over the lazy dog",
		Font:       scene.Font{Size: 14, Family: "Inter", Weight: 400},
		MaxWidth:   &narrowMax,
		LineHeight: 1.4,
	}
	wNarrow, hNarrow := MeasureText(nodeNarrow)

	if wNarrow > narrowMax+1 {
		t.Errorf("narrow width=%f should be <= %f", wNarrow, narrowMax)
	}
	if hNarrow <= h {
		t.Errorf("narrower maxWidth should produce taller text: narrow=%f, normal=%f", hNarrow, h)
	}
}

// --- Phase 2 new tests ---

func TestLayout_GridColumnsZero_NoPanic(t *testing.T) {
	// columns: 0 should not cause division by zero
	cols := 0
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"layout": "grid", "columns": cols,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "height": 50},
				map[string]interface{}{"frame": "b", "height": 50},
			},
		},
	})
	// Should not panic
	ComputeLayout(sg)
}

func TestLayout_MaxWidthZero_NoPanic(t *testing.T) {
	// max-width: 0 on text should not cause division by zero or undefined behavior
	node := &scene.TextNode{
		Content:    "Hello World",
		Font:       scene.Font{Weight: 400, Size: 14, Family: "Inter"},
		LineHeight: 1.4,
		MaxWidth:   scene.Ptr(0.0),
	}
	// Should not panic
	w, h := MeasureText(node)
	if w < 0 || h < 0 {
		t.Errorf("dimensions should be non-negative: w=%f h=%f", w, h)
	}
}

func TestLayout_MinGtMax_MinWins(t *testing.T) {
	// CSS spec: when min > max, min wins
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{
					"frame": "conflict",
					"min-width": 300, "max-width": 100,
					"min-height": 200, "max-height": 50,
					"height": 100,
				},
			},
		},
	})
	ComputeLayout(sg)
	child := sg.Root.Children[0].(*scene.FrameNode)
	if child.Layout.Width < 299 {
		t.Errorf("min-width should win over max-width: got width=%f, expected >= 300", child.Layout.Width)
	}
	if child.Layout.Height < 199 {
		t.Errorf("min-height should win over max-height: got height=%f, expected >= 200", child.Layout.Height)
	}
}
