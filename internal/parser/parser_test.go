package parser

import (
	"strings"
	"testing"

	"github.com/zeropsio/rendspec/internal/scene"
)

// --- Font ---

func TestParseFont_FullShorthand(t *testing.T) {
	f := ParseFont("700 20 Inter")
	if f.Weight != 700 || f.Size != 20 || f.Family != "Inter" {
		t.Errorf("got %+v", f)
	}
}

func TestParseFont_WeightAndSize(t *testing.T) {
	f := ParseFont("600 16")
	if f.Weight != 600 || f.Size != 16 {
		t.Errorf("got %+v", f)
	}
}

func TestParseFont_SizeAndFamily(t *testing.T) {
	f := ParseFont("14 Mono")
	if f.Size != 14 || f.Family != "Mono" {
		t.Errorf("got %+v", f)
	}
}

func TestParseFont_DictForm(t *testing.T) {
	f := ParseFont(map[string]interface{}{"weight": 500, "size": 18, "family": "Roboto"})
	if f.Weight != 500 || f.Size != 18 || f.Family != "Roboto" {
		t.Errorf("got %+v", f)
	}
}

func TestParseFont_SizeOnly(t *testing.T) {
	f := ParseFont("24")
	if f.Size != 24 {
		t.Errorf("got %+v", f)
	}
}

func TestParseFont_FamilyWithSpaces(t *testing.T) {
	f := ParseFont("400 14 Roboto Mono")
	if f.Family != "Roboto Mono" {
		t.Errorf("got family=%q", f.Family)
	}
}

// --- Spacing ---

func TestParseSpacing_SingleValue(t *testing.T) {
	s := ParseSpacing(12)
	if s.Top != 12 || s.Right != 12 || s.Bottom != 12 || s.Left != 12 {
		t.Errorf("got %+v", s)
	}
}

func TestParseSpacing_TwoValues(t *testing.T) {
	s := ParseSpacing("12 24")
	if s.Top != 12 || s.Right != 24 || s.Bottom != 12 || s.Left != 24 {
		t.Errorf("got %+v", s)
	}
}

func TestParseSpacing_FourValues(t *testing.T) {
	s := ParseSpacing("10 20 30 40")
	if s.Top != 10 || s.Right != 20 || s.Bottom != 30 || s.Left != 40 {
		t.Errorf("got %+v", s)
	}
}

func TestParseSpacing_HorizontalVertical(t *testing.T) {
	s := ParseSpacing("12 24")
	if s.Horizontal() != 48 || s.Vertical() != 24 {
		t.Errorf("H=%f V=%f", s.Horizontal(), s.Vertical())
	}
}

// --- Border ---

func TestParseBorder_FullShorthand(t *testing.T) {
	b := ParseBorder("1.5 solid #e2e8f0")
	if b.Width != 1.5 || b.Style != "solid" || b.Color != "#e2e8f0" {
		t.Errorf("got %+v", b)
	}
}

func TestParseBorder_WidthAndColor(t *testing.T) {
	b := ParseBorder("2 #333")
	if b.Width != 2 || b.Color != "#333" {
		t.Errorf("got %+v", b)
	}
}

func TestParseBorder_DictForm(t *testing.T) {
	b := ParseBorder(map[string]interface{}{"width": 1, "style": "dashed", "color": "red"})
	if b.Style != "dashed" {
		t.Errorf("got style=%q", b.Style)
	}
}

// --- Shadow ---

func TestParseShadow_Basic(t *testing.T) {
	shadows := ParseShadow("0 2 8 rgba(0,0,0,0.1)")
	if len(shadows) != 1 || shadows[0].Blur != 8 {
		t.Errorf("got %+v", shadows)
	}
}

func TestParseShadow_Multiple(t *testing.T) {
	shadows := ParseShadow("0 2 8 rgba(0,0,0,0.1) | 0 4 16 rgba(0,0,0,0.2)")
	if len(shadows) != 2 {
		t.Errorf("expected 2 shadows, got %d", len(shadows))
	}
}

func TestParseShadow_ThreeValues(t *testing.T) {
	shadows := ParseShadow("0 2 8")
	if len(shadows) != 1 || shadows[0].X != 0 || shadows[0].Y != 2 || shadows[0].Blur != 8 {
		t.Errorf("got %+v", shadows)
	}
}

// --- Gradient ---

func TestParseGradient_Linear(t *testing.T) {
	g := ParseGradient("linear-gradient(135deg, #667eea, #764ba2)")
	if g == nil || g.Type != "linear" || g.Angle != 135 || len(g.Stops) != 2 {
		t.Errorf("got %+v", g)
	}
	if g.Stops[0].Color != "#667eea" || g.Stops[1].Color != "#764ba2" {
		t.Errorf("stops: %+v", g.Stops)
	}
}

func TestParseGradient_Radial(t *testing.T) {
	g := ParseGradient("radial-gradient(circle, #fff, #000)")
	if g == nil || g.Type != "radial" || len(g.Stops) != 2 {
		t.Errorf("got %+v", g)
	}
}

func TestParseGradient_WithPositions(t *testing.T) {
	g := ParseGradient("linear-gradient(90deg, red 0%, blue 100%)")
	if g.Stops[0].Position != 0.0 || g.Stops[1].Position != 1.0 {
		t.Errorf("positions: %f, %f", g.Stops[0].Position, g.Stops[1].Position)
	}
}

func TestParseGradient_DirectionKeywords(t *testing.T) {
	g := ParseGradient("linear-gradient(to right, red, blue)")
	if g.Angle != 90 {
		t.Errorf("expected angle 90, got %f", g.Angle)
	}
}

func TestParseGradient_DictForm(t *testing.T) {
	g := ParseGradient(map[string]interface{}{
		"type":  "linear",
		"angle": 45,
		"stops": []interface{}{
			map[string]interface{}{"color": "#ff0000", "position": 0},
			map[string]interface{}{"color": "#0000ff", "position": 1},
		},
	})
	if g.Type != "linear" || g.Angle != 45 || len(g.Stops) != 2 {
		t.Errorf("got %+v", g)
	}
}

func TestParseGradient_NotAGradient(t *testing.T) {
	g := ParseGradient("#ff0000")
	if g != nil {
		t.Errorf("expected nil, got %+v", g)
	}
}

// --- ParseDict ---

func TestParseDict_MinimalScene(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{"width": 400, "height": 300},
	})
	if *sg.Root.Width != 400 || *sg.Root.Height != 300 {
		t.Errorf("got w=%v h=%v", sg.Root.Width, sg.Root.Height)
	}
}

func TestParseDict_FrameWithChildren(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"text": "Hello"},
				map[string]interface{}{"frame": "card", "fill": "#fff"},
			},
		},
	})
	if len(sg.Root.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(sg.Root.Children))
	}
	if _, ok := sg.Root.Children[0].(*scene.TextNode); !ok {
		t.Error("first child should be TextNode")
	}
	if _, ok := sg.Root.Children[1].(*scene.FrameNode); !ok {
		t.Error("second child should be FrameNode")
	}
}

func TestParseDict_FrameNameBecomesID(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "my-card"},
			},
		},
	})
	fn := sg.Root.Children[0].(*scene.FrameNode)
	if fn.ID != "my-card" {
		t.Errorf("expected id 'my-card', got %q", fn.ID)
	}
}

func TestParseDict_ExplicitIDOverridesName(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "card", "id": "custom-id"},
			},
		},
	})
	fn := sg.Root.Children[0].(*scene.FrameNode)
	if fn.ID != "custom-id" {
		t.Errorf("expected id 'custom-id', got %q", fn.ID)
	}
}

func TestParseDict_EdgesParsed(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root":  map[string]interface{}{"width": 400, "height": 300},
		"edges": []interface{}{map[string]interface{}{"from": "a", "to": "b", "label": "test"}},
	})
	if len(sg.Edges) != 1 || sg.Edges[0].FromID != "a" || *sg.Edges[0].Label != "test" {
		t.Errorf("got %+v", sg.Edges)
	}
}

func TestParseDict_ThemeBuiltin(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"theme": "dark",
		"root":  map[string]interface{}{"width": 100, "height": 100},
	})
	if sg.Theme.Background != "#0f172a" {
		t.Errorf("got background=%q", sg.Theme.Background)
	}
}

func TestParseDict_ThemeCustom(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"theme": map[string]interface{}{"background": "#111", "accent": "#f00"},
		"root":  map[string]interface{}{"width": 100, "height": 100},
	})
	if sg.Theme.Background != "#111" || sg.Theme.Accent != "#f00" {
		t.Errorf("got %+v", sg.Theme)
	}
}

func TestParseDict_GradientInFill(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "grad", "fill": "linear-gradient(90deg, red, blue)"},
			},
		},
	})
	fn := sg.Root.Children[0].(*scene.FrameNode)
	if fn.Gradient == nil || fn.Gradient.Type != "linear" {
		t.Error("expected gradient")
	}
	if fn.Fill != nil {
		t.Error("fill should be nil when gradient takes over")
	}
}

func TestParseDict_ZIndexParsed(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "back", "z-index": 1},
				map[string]interface{}{"frame": "front", "z-index": 2},
			},
		},
	})
	if sg.Root.Children[0].(*scene.FrameNode).ZIndex != 1 {
		t.Error("back z-index should be 1")
	}
	if sg.Root.Children[1].(*scene.FrameNode).ZIndex != 2 {
		t.Error("front z-index should be 2")
	}
}

func TestParseDict_GridLayoutParsed(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 800, "height": 600,
			"layout": "grid", "columns": 3,
			"column-gap": 16, "row-gap": 12,
			"children": []interface{}{
				map[string]interface{}{"frame": "a"},
				map[string]interface{}{"frame": "b"},
				map[string]interface{}{"frame": "c"},
			},
		},
	})
	if sg.Root.LayoutMode != "grid" || *sg.Root.Columns != 3 {
		t.Errorf("expected grid with 3 columns")
	}
	if *sg.Root.ColumnGap != 16 || *sg.Root.RowGap != 12 {
		t.Error("expected column-gap=16, row-gap=12")
	}
}

func TestParseDict_ImageParsed(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "img", "image": "photo.jpg", "image-fit": "contain"},
			},
		},
	})
	fn := sg.Root.Children[0].(*scene.FrameNode)
	if *fn.Image != "photo.jpg" || fn.ImageFit != "contain" {
		t.Errorf("got image=%v fit=%q", fn.Image, fn.ImageFit)
	}
}

// --- Tokens ---

func TestTokenResolution_String(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"tokens": map[string]interface{}{"color": map[string]interface{}{"primary": "#3b82f6"}},
		"root":   map[string]interface{}{"width": 400, "height": 300, "fill": "$color.primary"},
	})
	if *sg.Root.Fill != "#3b82f6" {
		t.Errorf("got fill=%v", sg.Root.Fill)
	}
}

func TestTokenResolution_Number(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"tokens": map[string]interface{}{"radius": map[string]interface{}{"md": 12}},
		"root":   map[string]interface{}{"width": 400, "height": 300, "radius": "$radius.md"},
	})
	if sg.Root.Radius != 12 {
		t.Errorf("got radius=%f", sg.Root.Radius)
	}
}

func TestTokenResolution_Nested(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"tokens": map[string]interface{}{"color": map[string]interface{}{"bg": map[string]interface{}{"card": "#1e293b"}}},
		"root":   map[string]interface{}{"width": 400, "height": 300, "fill": "$color.bg.card"},
	})
	if *sg.Root.Fill != "#1e293b" {
		t.Errorf("got fill=%v", sg.Root.Fill)
	}
}

func TestTokenResolution_UnresolvedPassesThrough(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"tokens": map[string]interface{}{},
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{map[string]interface{}{"text": "$nonexistent.token"}},
		},
	})
	tn := sg.Root.Children[0].(*scene.TextNode)
	if tn.Content != "$nonexistent.token" {
		t.Errorf("got content=%q", tn.Content)
	}
}

// --- Components ---

func TestComponent_Basic(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"components": map[string]interface{}{
			"chip": map[string]interface{}{"fill": "#eff6ff", "radius": 20, "padding": "6 16"},
		},
		"root": map[string]interface{}{
			"width": 400, "height": 200,
			"children": []interface{}{map[string]interface{}{"use": "chip"}},
		},
	})
	fn := sg.Root.Children[0].(*scene.FrameNode)
	if *fn.Fill != "#eff6ff" || fn.Radius != 20 {
		t.Errorf("got fill=%v radius=%f", fn.Fill, fn.Radius)
	}
}

func TestComponent_WithLabel(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"components": map[string]interface{}{
			"chip": map[string]interface{}{
				"fill":     "#eff6ff",
				"radius":   20,
				"children": []interface{}{map[string]interface{}{"text": "Default"}},
			},
		},
		"root": map[string]interface{}{
			"width": 400, "height": 200,
			"children": []interface{}{map[string]interface{}{"chip": "Custom Label"}},
		},
	})
	fn := sg.Root.Children[0].(*scene.FrameNode)
	tn := fn.Children[0].(*scene.TextNode)
	if tn.Content != "Custom Label" {
		t.Errorf("got content=%q", tn.Content)
	}
}

func TestComponent_Variant(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"components": map[string]interface{}{
			"chip": map[string]interface{}{
				"fill":     "#eff6ff",
				"variants": map[string]interface{}{"dark": map[string]interface{}{"fill": "#1e293b"}},
			},
		},
		"root": map[string]interface{}{
			"width": 400, "height": 200,
			"children": []interface{}{map[string]interface{}{"use": "chip", "variant": "dark"}},
		},
	})
	fn := sg.Root.Children[0].(*scene.FrameNode)
	if *fn.Fill != "#1e293b" {
		t.Errorf("got fill=%v", fn.Fill)
	}
}

func TestComponent_Parameterized(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"components": map[string]interface{}{
			"card": map[string]interface{}{
				"fill": "white",
				"params": map[string]interface{}{
					"title": map[string]interface{}{"type": "string", "default": "Untitled"},
				},
				"children": []interface{}{map[string]interface{}{"text": "{{title}}"}},
			},
		},
		"root": map[string]interface{}{
			"width": 400, "height": 200,
			"children": []interface{}{map[string]interface{}{"use": "card", "title": "My Card"}},
		},
	})
	fn := sg.Root.Children[0].(*scene.FrameNode)
	tn := fn.Children[0].(*scene.TextNode)
	if tn.Content != "My Card" {
		t.Errorf("got content=%q", tn.Content)
	}
}

func TestComponent_ParameterizedDefault(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"components": map[string]interface{}{
			"card": map[string]interface{}{
				"fill": "white",
				"params": map[string]interface{}{
					"title": map[string]interface{}{"type": "string", "default": "Untitled"},
				},
				"children": []interface{}{map[string]interface{}{"text": "{{title}}"}},
			},
		},
		"root": map[string]interface{}{
			"width": 400, "height": 200,
			"children": []interface{}{map[string]interface{}{"use": "card"}},
		},
	})
	fn := sg.Root.Children[0].(*scene.FrameNode)
	tn := fn.Children[0].(*scene.TextNode)
	if tn.Content != "Untitled" {
		t.Errorf("got content=%q", tn.Content)
	}
}

// --- ParseString ---

func TestParseString_Basic(t *testing.T) {
	sg, err := ParseString(`
root:
  width: 400
  height: 300
  - text: "Hello World"
    font: 700 20 Inter
`)
	if err != nil {
		t.Fatal(err)
	}
	if *sg.Root.Width != 400 || len(sg.Root.Children) != 1 {
		t.Errorf("got w=%v children=%d", sg.Root.Width, len(sg.Root.Children))
	}
	tn := sg.Root.Children[0].(*scene.TextNode)
	if tn.Content != "Hello World" {
		t.Errorf("got content=%q", tn.Content)
	}
}

// --- Multi-page ---

func TestParseDocument_MultiPage(t *testing.T) {
	doc, err := ParseDocument(`
theme: light
pages:
  - name: Login
    root:
      width: 400
      height: 300
      children:
        - text: "Login Page"
  - name: Dashboard
    root:
      width: 800
      height: 600
      children:
        - text: "Dashboard"
`)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(doc.Pages))
	}
	if doc.Pages[0].Name != "Login" || doc.Pages[1].Name != "Dashboard" {
		t.Error("page names don't match")
	}
	if *doc.Pages[0].Root.Width != 400 || *doc.Pages[1].Root.Width != 800 {
		t.Error("page widths don't match")
	}
}

func TestParseDocument_SinglePage(t *testing.T) {
	doc, err := ParseDocument(`
root:
  width: 400
  height: 300
  children:
    - text: "Solo Page"
`)
	if err != nil {
		t.Fatal(err)
	}
	if len(doc.Pages) != 1 || doc.Pages[0].Name != "Page 1" {
		t.Errorf("expected 1 page named 'Page 1', got %d pages", len(doc.Pages))
	}
}

// --- Validation Warnings ---

func TestValidation_InvalidDirection(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{"width": 400, "height": 300, "direction": "diagonal"},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "direction") && strings.Contains(w, "diagonal") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for invalid direction, got %v", sg.Warnings)
	}
}

func TestValidation_InvalidAlign(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{"width": 400, "height": 300, "align": "middle"},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "align") && strings.Contains(w, "middle") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for invalid align, got %v", sg.Warnings)
	}
}

func TestValidation_InvalidJustify(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{"width": 400, "height": 300, "justify": "space-between"},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "justify") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for invalid justify, got %v", sg.Warnings)
	}
}

func TestValidation_InvalidShape(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "s", "shape": "hexagon"},
			},
		},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "shape") && strings.Contains(w, "hexagon") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for invalid shape, got %v", sg.Warnings)
	}
}

func TestValidation_InvalidLayout(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{"width": 400, "height": 300, "layout": "masonry"},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "layout") && strings.Contains(w, "masonry") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for invalid layout, got %v", sg.Warnings)
	}
}

func TestValidation_InvalidEdgeStyle(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root":  map[string]interface{}{"width": 400, "height": 300},
		"edges": []interface{}{map[string]interface{}{"from": "a", "to": "b", "style": "wavy"}},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "style") && strings.Contains(w, "wavy") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for invalid edge style, got %v", sg.Warnings)
	}
}

func TestValidation_InvalidArrow(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root":  map[string]interface{}{"width": 400, "height": 300},
		"edges": []interface{}{map[string]interface{}{"from": "a", "to": "b", "arrow": "double"}},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "arrow") && strings.Contains(w, "double") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for invalid arrow, got %v", sg.Warnings)
	}
}

func TestValidation_InvalidCurve(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root":  map[string]interface{}{"width": 400, "height": 300},
		"edges": []interface{}{map[string]interface{}{"from": "a", "to": "b", "curve": "bezier"}},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "curve") && strings.Contains(w, "bezier") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for invalid curve, got %v", sg.Warnings)
	}
}

func TestValidation_InvalidTextAlign(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"text": "Hi", "text-align": "justify"},
			},
		},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "text-align") && strings.Contains(w, "justify") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for invalid text-align, got %v", sg.Warnings)
	}
}

func TestValidation_InvalidTextDecoration(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"text": "Hi", "text-decoration": "overline"},
			},
		},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "text-decoration") && strings.Contains(w, "overline") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for invalid text-decoration, got %v", sg.Warnings)
	}
}

func TestValidation_InvalidImageFit(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "img", "image": "photo.jpg", "image-fit": "stretch"},
			},
		},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "image-fit") && strings.Contains(w, "stretch") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for invalid image-fit, got %v", sg.Warnings)
	}
}

func TestValidation_UnknownComponent(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"use": "nonexistent-component"},
			},
		},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "unknown component") && strings.Contains(w, "nonexistent-component") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for unknown component, got %v", sg.Warnings)
	}
}

func TestValidation_EdgeReferencesUnknownFrame(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "a"},
			},
		},
		"edges": []interface{}{map[string]interface{}{"from": "a", "to": "missing"}},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "edge references unknown frame") && strings.Contains(w, "missing") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for unknown edge target, got %v", sg.Warnings)
	}
}

func TestValidation_UnresolvedToken(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"tokens": map[string]interface{}{"color": map[string]interface{}{"primary": "#3b82f6"}},
		"root":   map[string]interface{}{"width": 400, "height": 300, "fill": "$color.nonexistent"},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "unresolved token") && strings.Contains(w, "$color.nonexistent") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for unresolved token, got %v", sg.Warnings)
	}
}

func TestValidation_ValidEnumsNoWarnings(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"direction": "row", "align": "center", "justify": "between",
			"layout": "flex", "position": "relative", "shape": "rect",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "width": 100, "height": 50},
				map[string]interface{}{"frame": "b", "id": "b", "width": 100, "height": 50},
				map[string]interface{}{"text": "Hi", "text-align": "center", "text-decoration": "underline"},
			},
		},
		"edges": []interface{}{
			map[string]interface{}{
				"from": "a", "to": "b",
				"style": "dashed", "arrow": "both", "curve": "orthogonal",
				"from-anchor": "right", "to-anchor": "left",
			},
		},
	})
	if len(sg.Warnings) != 0 {
		t.Errorf("expected no warnings for valid enums, got %v", sg.Warnings)
	}
}

func TestValidation_MalformedGradientReturnsNil(t *testing.T) {
	g := ParseGradient("not-a-gradient")
	if g != nil {
		t.Errorf("expected nil for malformed gradient, got %+v", g)
	}
}

func TestValidation_EmptyInputNoCrash(t *testing.T) {
	sg := ParseDict(map[string]interface{}{})
	if sg == nil {
		t.Error("should return non-nil SceneGraph for empty input")
	}
}

func TestValidation_NilInputNoCrash(t *testing.T) {
	sg := ParseDict(make(map[string]interface{}))
	if sg == nil {
		t.Error("should return non-nil SceneGraph for nil-like input")
	}
}

func TestParseDocument_SharedTokens(t *testing.T) {
	doc, err := ParseDocument(`
tokens:
  color:
    bg: "#1e293b"
pages:
  - name: Page 1
    root:
      width: 400
      height: 300
      fill: $color.bg
`)
	if err != nil {
		t.Fatal(err)
	}
	if *doc.Pages[0].Root.Fill != "#1e293b" {
		t.Errorf("got fill=%v", doc.Pages[0].Root.Fill)
	}
}

// --- Spacing: 3-value shorthand ---

func TestParseSpacing_ThreeValues(t *testing.T) {
	s := ParseSpacing("10 20 30")
	if s.Top != 10 {
		t.Errorf("expected Top=10, got %f", s.Top)
	}
	if s.Right != 20 {
		t.Errorf("expected Right=20, got %f", s.Right)
	}
	if s.Bottom != 30 {
		t.Errorf("expected Bottom=30, got %f", s.Bottom)
	}
	if s.Left != 20 {
		t.Errorf("expected Left=20 (same as Right), got %f", s.Left)
	}
}

// --- Shadow: with spread ---

func TestParseShadow_WithSpread(t *testing.T) {
	shadows := ParseShadow("0 2 8 4 rgba(0,0,0,0.1)")
	if len(shadows) != 1 {
		t.Fatalf("expected 1 shadow, got %d", len(shadows))
	}
	sh := shadows[0]
	if sh.X != 0 {
		t.Errorf("expected X=0, got %f", sh.X)
	}
	if sh.Y != 2 {
		t.Errorf("expected Y=2, got %f", sh.Y)
	}
	if sh.Blur != 8 {
		t.Errorf("expected Blur=8, got %f", sh.Blur)
	}
	if sh.Spread != 4 {
		t.Errorf("expected Spread=4, got %f", sh.Spread)
	}
	if sh.Color != "rgba(0,0,0,0.1)" {
		t.Errorf("expected Color=rgba(0,0,0,0.1), got %q", sh.Color)
	}
}

func TestParseShadow_FourNumericValues(t *testing.T) {
	shadows := ParseShadow("0 2 8 4")
	if len(shadows) != 1 {
		t.Fatalf("expected 1 shadow, got %d", len(shadows))
	}
	sh := shadows[0]
	if sh.X != 0 {
		t.Errorf("expected X=0, got %f", sh.X)
	}
	if sh.Y != 2 {
		t.Errorf("expected Y=2, got %f", sh.Y)
	}
	if sh.Blur != 8 {
		t.Errorf("expected Blur=8, got %f", sh.Blur)
	}
	if sh.Spread != 4 {
		t.Errorf("expected Spread=4, got %f", sh.Spread)
	}
}

// --- Theme: unknown name ---

func TestParseTheme_UnknownName(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"theme": "nonexistent",
		"root":  map[string]interface{}{"width": 100, "height": 100},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "unknown theme") && strings.Contains(w, "nonexistent") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for unknown theme 'nonexistent', got %v", sg.Warnings)
	}
}

// --- Component depth guard ---

func TestComponentDepthGuard(t *testing.T) {
	// Create two components that reference each other circularly.
	// The depth guard should prevent infinite recursion.
	sg := ParseDict(map[string]interface{}{
		"components": map[string]interface{}{
			"alpha": map[string]interface{}{
				"fill": "#aaa",
				"children": []interface{}{
					map[string]interface{}{"use": "beta"},
				},
			},
			"beta": map[string]interface{}{
				"fill": "#bbb",
				"children": []interface{}{
					map[string]interface{}{"use": "alpha"},
				},
			},
		},
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"use": "alpha"},
			},
		},
	})
	// The test passes if we reach here without a stack overflow.
	if sg == nil {
		t.Fatal("expected non-nil SceneGraph")
	}
	// Verify a warning was emitted about maximum depth.
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "maximum component nesting depth") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning about maximum component nesting depth, got %v", sg.Warnings)
	}
}

// --- toBool ---

func TestToBool(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected bool
	}{
		{true, true},
		{false, false},
		{"true", true},
		{"1", true},
		{"false", false},
		{"0", false},
		{42, true},        // non-zero int returns true
		{0, false},        // zero int returns false
		{3.14, true},      // non-zero float64 returns true
		{0.0, false},      // zero float64 returns false
		{nil, false},      // nil returns false
	}
	for _, tc := range tests {
		got := toBool(tc.input)
		if got != tc.expected {
			t.Errorf("toBool(%v) = %v, want %v", tc.input, got, tc.expected)
		}
	}
}

// --- toFloat ---

func TestToFloat(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected float64
	}{
		{42, 42.0},
		{3.14, 3.14},
		{"12.5", 12.5},
		{"100", 100.0},
		{"notanumber", 0.0}, // unparseable string returns 0
		{true, 0.0},         // bool falls through to default sprintf path
	}
	for _, tc := range tests {
		got := toFloat(tc.input)
		if got != tc.expected {
			t.Errorf("toFloat(%v) = %f, want %f", tc.input, got, tc.expected)
		}
	}
}

// --- toInt ---

func TestToInt(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected int
	}{
		{42, 42},
		{3.14, 3},        // float64 truncates
		{7.9, 7},         // float64 truncates toward zero
		{"100", 100},
		{"notanumber", 0}, // unparseable string returns 0
		{true, 0},         // bool falls through to default sprintf path
	}
	for _, tc := range tests {
		got := toInt(tc.input)
		if got != tc.expected {
			t.Errorf("toInt(%v) = %d, want %d", tc.input, got, tc.expected)
		}
	}
}

// --- Shadow: multiple pipe-separated with spread ---

func TestParseShadow_MultiplePipeSeparated(t *testing.T) {
	shadows := ParseShadow("0 2 8 rgba(0,0,0,0.1) | 0 4 16 4 rgba(0,0,0,0.2)")
	if len(shadows) != 2 {
		t.Fatalf("expected 2 shadows, got %d", len(shadows))
	}
	// First shadow: no spread
	if shadows[0].X != 0 || shadows[0].Y != 2 || shadows[0].Blur != 8 {
		t.Errorf("shadow[0] x/y/blur: got %f/%f/%f", shadows[0].X, shadows[0].Y, shadows[0].Blur)
	}
	if shadows[0].Spread != 0 {
		t.Errorf("shadow[0] expected Spread=0, got %f", shadows[0].Spread)
	}
	if shadows[0].Color != "rgba(0,0,0,0.1)" {
		t.Errorf("shadow[0] expected Color=rgba(0,0,0,0.1), got %q", shadows[0].Color)
	}
	// Second shadow: with spread
	if shadows[1].X != 0 || shadows[1].Y != 4 || shadows[1].Blur != 16 {
		t.Errorf("shadow[1] x/y/blur: got %f/%f/%f", shadows[1].X, shadows[1].Y, shadows[1].Blur)
	}
	if shadows[1].Spread != 4 {
		t.Errorf("shadow[1] expected Spread=4, got %f", shadows[1].Spread)
	}
	if shadows[1].Color != "rgba(0,0,0,0.2)" {
		t.Errorf("shadow[1] expected Color=rgba(0,0,0,0.2), got %q", shadows[1].Color)
	}
}

// --- Enum validation ---

func TestEnumValidation(t *testing.T) {
	tests := []struct {
		name     string
		data     map[string]interface{}
		property string
		value    string
	}{
		{
			name: "invalid direction",
			data: map[string]interface{}{
				"root": map[string]interface{}{"width": 100, "height": 100, "direction": "diagonal"},
			},
			property: "direction",
			value:    "diagonal",
		},
		{
			name: "invalid align",
			data: map[string]interface{}{
				"root": map[string]interface{}{"width": 100, "height": 100, "align": "middle"},
			},
			property: "align",
			value:    "middle",
		},
		{
			name: "invalid justify",
			data: map[string]interface{}{
				"root": map[string]interface{}{"width": 100, "height": 100, "justify": "space-evenly"},
			},
			property: "justify",
			value:    "space-evenly",
		},
		{
			name: "invalid layout",
			data: map[string]interface{}{
				"root": map[string]interface{}{"width": 100, "height": 100, "layout": "masonry"},
			},
			property: "layout",
			value:    "masonry",
		},
		{
			name: "invalid position",
			data: map[string]interface{}{
				"root": map[string]interface{}{"width": 100, "height": 100, "position": "fixed"},
			},
			property: "position",
			value:    "fixed",
		},
		{
			name: "invalid shape in child",
			data: map[string]interface{}{
				"root": map[string]interface{}{
					"width": 100, "height": 100,
					"children": []interface{}{
						map[string]interface{}{"frame": "s", "shape": "triangle"},
					},
				},
			},
			property: "shape",
			value:    "triangle",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sg := ParseDict(tc.data)
			found := false
			for _, w := range sg.Warnings {
				if strings.Contains(w, tc.property) && strings.Contains(w, tc.value) {
					found = true
				}
			}
			if !found {
				t.Errorf("expected warning containing %q and %q, got %v", tc.property, tc.value, sg.Warnings)
			}
		})
	}
}

// --- Duplicate frame IDs ---

func TestValidation_DuplicateFrameID(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "card", "id": "dup"},
				map[string]interface{}{"frame": "other", "id": "dup"},
			},
		},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "duplicate frame id") && strings.Contains(w, "dup") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected warning for duplicate frame id, got %v", sg.Warnings)
	}
}

func TestValidation_NoDuplicateFrameID(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a"},
				map[string]interface{}{"frame": "b", "id": "b"},
			},
		},
	})
	for _, w := range sg.Warnings {
		if strings.Contains(w, "duplicate frame id") {
			t.Errorf("unexpected duplicate warning: %v", sg.Warnings)
		}
	}
}

// --- Token $$ escape ---

func TestTokenResolution_DollarEscape(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"tokens": map[string]interface{}{"color": map[string]interface{}{"primary": "#3b82f6"}},
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"text": "Price: $$10.00"},
			},
		},
	})
	tn := sg.Root.Children[0].(*scene.TextNode)
	if tn.Content != "Price: $10.00" {
		t.Errorf("expected 'Price: $10.00', got %q", tn.Content)
	}
	// Should not produce any token warnings
	for _, w := range sg.Warnings {
		if strings.Contains(w, "unresolved token") {
			t.Errorf("unexpected token warning: %s", w)
		}
	}
}

func TestTokenResolution_DollarEscapeWithTokens(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"tokens": map[string]interface{}{"color": map[string]interface{}{"primary": "#3b82f6"}},
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				// Mix of $$ escape and real token
				map[string]interface{}{"text": "$$USD $color.primary"},
			},
		},
	})
	tn := sg.Root.Children[0].(*scene.TextNode)
	if tn.Content != "$USD #3b82f6" {
		t.Errorf("expected '$USD #3b82f6', got %q", tn.Content)
	}
}

// --- Edge corner-radius ---

func TestParseEdge_CornerRadius(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root":  map[string]interface{}{"width": 400, "height": 300},
		"edges": []interface{}{map[string]interface{}{"from": "a", "to": "b", "corner-radius": 8}},
	})
	if sg.Edges[0].CornerRadius == nil || *sg.Edges[0].CornerRadius != 8 {
		t.Errorf("expected corner-radius=8, got %v", sg.Edges[0].CornerRadius)
	}
}

// --- toBool with numeric types for clip/visible ---

func TestParseDict_ClipWithNumericTrue(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "clip": 1},
			},
		},
	})
	fn := sg.Root.Children[0].(*scene.FrameNode)
	if !fn.Clip {
		t.Error("clip should be true when set to 1")
	}
}

func TestParseDict_VisibleWithBoolFalse(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "visible": false},
			},
		},
	})
	fn := sg.Root.Children[0].(*scene.FrameNode)
	if fn.Visible {
		t.Error("visible should be false")
	}
}

// --- Deep copy correctness ---

func TestComponent_DeepCopyIsolation(t *testing.T) {
	// Verify that modifying an instance doesn't affect the component definition
	sg := ParseDict(map[string]interface{}{
		"components": map[string]interface{}{
			"card": map[string]interface{}{
				"fill":   "#aaa",
				"radius": 8,
				"children": []interface{}{
					map[string]interface{}{"text": "Default"},
				},
			},
		},
		"root": map[string]interface{}{
			"width": 400, "height": 200,
			"children": []interface{}{
				map[string]interface{}{"use": "card", "fill": "#bbb"},
				map[string]interface{}{"use": "card"},
			},
		},
	})
	first := sg.Root.Children[0].(*scene.FrameNode)
	second := sg.Root.Children[1].(*scene.FrameNode)
	if *first.Fill != "#bbb" {
		t.Errorf("first instance fill should be #bbb, got %v", *first.Fill)
	}
	if *second.Fill != "#aaa" {
		t.Errorf("second instance fill should be #aaa (unmodified), got %v", *second.Fill)
	}
}

// --- New tests for Phase 3 fixes ---

func TestParseFont_Nil(t *testing.T) {
	f := ParseFont(nil)
	if f != scene.DefaultFont() {
		t.Errorf("ParseFont(nil) should return DefaultFont, got %+v", f)
	}
}

func TestWarnInvalidEnum_DeterministicOrder(t *testing.T) {
	var warnings []string
	valid := map[string]bool{"a": true, "b": true, "c": true}
	warnInvalidEnum(&warnings, "test", "z", valid)
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	// Should contain sorted keys
	if !strings.Contains(warnings[0], "a, b, c") {
		t.Errorf("expected sorted keys, got: %s", warnings[0])
	}
}

func TestUnknownFrameProperty_Warning(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{
					"frame": "card", "fill": "#fff",
					"unknown-prop": "value",
				},
			},
		},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "unknown frame property") && strings.Contains(w, "unknown-prop") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about unknown-prop, got: %v", sg.Warnings)
	}
}

func TestUnknownTextProperty_Warning(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{
					"text": "Hello", "bad-prop": "x",
				},
			},
		},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "unknown text property") && strings.Contains(w, "bad-prop") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about bad-prop, got: %v", sg.Warnings)
	}
}

func TestUnknownEdgeProperty_Warning(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "width": 50, "height": 50},
				map[string]interface{}{"frame": "b", "id": "b", "width": 50, "height": 50},
			},
		},
		"edges": []interface{}{
			map[string]interface{}{"from": "a", "to": "b", "bad-edge-prop": "x"},
		},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "unknown edge property") && strings.Contains(w, "bad-edge-prop") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about bad-edge-prop, got: %v", sg.Warnings)
	}
}

func TestEdgeMissingFrom_Warning(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "width": 50, "height": 50},
			},
		},
		"edges": []interface{}{
			map[string]interface{}{"to": "a"},
		},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "edge missing 'from'") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about missing from, got: %v", sg.Warnings)
	}
}

func TestEdgeMissingTo_Warning(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "width": 50, "height": 50},
			},
		},
		"edges": []interface{}{
			map[string]interface{}{"from": "a"},
		},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "edge missing 'to'") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about missing to, got: %v", sg.Warnings)
	}
}

func TestBorderStyleValidation_Warning(t *testing.T) {
	sg := ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{
					"frame": "b", "fill": "white",
					"border": "2 invalid #333",
				},
			},
		},
	})
	found := false
	for _, w := range sg.Warnings {
		if strings.Contains(w, "border style") && strings.Contains(w, "invalid") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning about invalid border style, got: %v", sg.Warnings)
	}
}

func TestParseSpacing_FiveValues(t *testing.T) {
	s := ParseSpacing("10 20 30 40 50")
	if s.Top != 10 || s.Right != 20 || s.Bottom != 30 || s.Left != 40 {
		t.Errorf("5-value spacing should use first 4: got %+v", s)
	}
}
