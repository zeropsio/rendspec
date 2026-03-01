package scene

import (
	"testing"
)

func TestDefaultFont(t *testing.T) {
	f := DefaultFont()
	if f.Weight != 400 || f.Size != 14 || f.Family != "Inter" {
		t.Errorf("unexpected default font: %+v", f)
	}
}

func TestNewFrameNode_Defaults(t *testing.T) {
	n := NewFrameNode()
	if n.Position != "relative" {
		t.Errorf("position=%q, want relative", n.Position)
	}
	if n.Direction != "column" {
		t.Errorf("direction=%q, want column", n.Direction)
	}
	if n.Align != "stretch" {
		t.Errorf("align=%q, want stretch", n.Align)
	}
	if n.Justify != "start" {
		t.Errorf("justify=%q, want start", n.Justify)
	}
	if n.LayoutMode != "flex" {
		t.Errorf("layout=%q, want flex", n.LayoutMode)
	}
	if n.ImageFit != "cover" {
		t.Errorf("image-fit=%q, want cover", n.ImageFit)
	}
	if n.Shape != "rect" {
		t.Errorf("shape=%q, want rect", n.Shape)
	}
	if !n.Visible {
		t.Error("visible should default to true")
	}
	if n.Opacity != 1.0 {
		t.Errorf("opacity=%f, want 1.0", n.Opacity)
	}
}

func TestNewTextNode_Defaults(t *testing.T) {
	n := NewTextNode()
	if n.Color != "#0f172a" {
		t.Errorf("color=%q", n.Color)
	}
	if n.TextAlign != "left" {
		t.Errorf("text-align=%q", n.TextAlign)
	}
	if n.LineHeight != 1.4 {
		t.Errorf("line-height=%f", n.LineHeight)
	}
	if n.TextDecoration != "none" {
		t.Errorf("text-decoration=%q", n.TextDecoration)
	}
	if n.Opacity != 1.0 {
		t.Errorf("opacity=%f", n.Opacity)
	}
}

func TestNewEdgeNode_Defaults(t *testing.T) {
	e := NewEdgeNode()
	if e.FromAnchor != "auto" || e.ToAnchor != "auto" {
		t.Errorf("anchors: from=%q to=%q", e.FromAnchor, e.ToAnchor)
	}
	if e.Stroke != "#94a3b8" {
		t.Errorf("stroke=%q", e.Stroke)
	}
	if e.StrokeWidth != 1.5 {
		t.Errorf("stroke-width=%f", e.StrokeWidth)
	}
	if e.Arrow != "end" {
		t.Errorf("arrow=%q", e.Arrow)
	}
	if e.Curve != "straight" {
		t.Errorf("curve=%q", e.Curve)
	}
	if e.LabelPosition != 0.5 {
		t.Errorf("label-position=%f", e.LabelPosition)
	}
}

func TestSpacing_HorizontalVertical(t *testing.T) {
	s := Spacing{Top: 10, Right: 20, Bottom: 30, Left: 40}
	if s.Horizontal() != 60 {
		t.Errorf("horizontal=%f, want 60", s.Horizontal())
	}
	if s.Vertical() != 40 {
		t.Errorf("vertical=%f, want 40", s.Vertical())
	}
}

func TestBuiltinThemes_Exist(t *testing.T) {
	names := []string{"light", "dark", "blueprint", "sketch"}
	for _, name := range names {
		if _, ok := BuiltinThemes[name]; !ok {
			t.Errorf("missing builtin theme: %s", name)
		}
	}
}

func TestDefaultTheme_IsLight(t *testing.T) {
	dt := DefaultTheme()
	lt := BuiltinThemes["light"]
	if dt.Background != lt.Background || dt.Foreground != lt.Foreground {
		t.Error("default theme should match light theme")
	}
}

func TestPtr_Helper(t *testing.T) {
	p := Ptr(42.0)
	if *p != 42.0 {
		t.Errorf("*p=%f, want 42.0", *p)
	}
}

func TestGetLayout_Frame(t *testing.T) {
	f := NewFrameNode()
	f.Layout.X = 10
	f.Layout.Y = 20
	l := f.GetLayout()
	if l.X != 10 || l.Y != 20 {
		t.Errorf("layout x=%f y=%f", l.X, l.Y)
	}
}

func TestGetLayout_Text(t *testing.T) {
	tn := NewTextNode()
	tn.Layout.Width = 100
	l := tn.GetLayout()
	if l.Width != 100 {
		t.Errorf("layout width=%f", l.Width)
	}
}

func TestGetLayout_Edge(t *testing.T) {
	e := NewEdgeNode()
	if e.GetLayout() != nil {
		t.Error("edge GetLayout should return nil")
	}
}

func TestNewSceneGraph_Defaults(t *testing.T) {
	sg := NewSceneGraph()
	if sg.Root == nil {
		t.Error("root should not be nil")
	}
	if sg.Components == nil {
		t.Error("components should not be nil")
	}
	if sg.Tokens == nil {
		t.Error("tokens should not be nil")
	}
}

func TestNewDocument_Defaults(t *testing.T) {
	doc := NewDocument()
	if doc.Components == nil {
		t.Error("components should not be nil")
	}
	if doc.Tokens == nil {
		t.Error("tokens should not be nil")
	}
}
