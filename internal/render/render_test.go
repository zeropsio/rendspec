package render

import (
	"fmt"
	"strings"
	"testing"

	"github.com/zeropsio/rendspec/internal/layout"
	"github.com/zeropsio/rendspec/internal/parser"
	"github.com/zeropsio/rendspec/internal/scene"
)

func renderDict(data map[string]interface{}) string {
	sg := parser.ParseDict(data)
	layout.ComputeLayout(sg)
	return RenderSVG(sg)
}

func TestBasic_ProducesValidSVG(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{"width": 400, "height": 300},
	})
	if !strings.HasPrefix(svg, "<svg") {
		t.Error("should start with <svg")
	}
	if !strings.HasSuffix(svg, "</svg>") {
		t.Error("should end with </svg>")
	}
	if !strings.Contains(svg, `width="400"`) {
		t.Error("should contain width")
	}
	if !strings.Contains(svg, `height="300"`) {
		t.Error("should contain height")
	}
}

func TestBasic_RendersFrameRect(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "card", "fill": "#ff0000", "width": 100, "height": 50},
			},
		},
	})
	if !strings.Contains(svg, `fill="#ff0000"`) {
		t.Error("should contain fill color")
	}
	if !strings.Contains(svg, "<rect") {
		t.Error("should contain <rect")
	}
}

func TestBasic_RendersText(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"text": "Hello World"},
			},
		},
	})
	if !strings.Contains(svg, "Hello World") {
		t.Error("should contain text content")
	}
	if !strings.Contains(svg, "<text") {
		t.Error("should contain <text element")
	}
}

func TestBasic_RendersCircle(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "dot", "shape": "circle", "fill": "red", "width": 50, "height": 50},
			},
		},
	})
	if !strings.Contains(svg, "<circle") {
		t.Error("should contain <circle")
	}
}

func TestBasic_RendersEllipse(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "e", "shape": "ellipse", "fill": "blue", "width": 100, "height": 50},
			},
		},
	})
	if !strings.Contains(svg, "<ellipse") {
		t.Error("should contain <ellipse")
	}
}

func TestBasic_RendersDiamond(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "d", "shape": "diamond", "fill": "green", "width": 80, "height": 80},
			},
		},
	})
	if !strings.Contains(svg, "<polygon") {
		t.Error("should contain <polygon")
	}
}

func TestBasic_HiddenFrameNotRendered(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "hidden", "visible": false, "fill": "#999"},
			},
		},
	})
	if strings.Contains(svg, "#999") {
		t.Error("hidden frame fill should not appear")
	}
}

func TestGradient_LinearGradientDef(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "g", "fill": "linear-gradient(90deg, red, blue)", "width": 200, "height": 100},
			},
		},
	})
	if !strings.Contains(svg, "<linearGradient") {
		t.Error("should contain <linearGradient")
	}
	if !strings.Contains(svg, "url(#grad-") {
		t.Error("should reference gradient")
	}
	if !strings.Contains(svg, "<stop") {
		t.Error("should contain <stop")
	}
}

func TestGradient_RadialGradientDef(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "g", "fill": "radial-gradient(circle, #fff, #000)", "width": 100, "height": 100},
			},
		},
	})
	if !strings.Contains(svg, "<radialGradient") {
		t.Error("should contain <radialGradient")
	}
}

func TestImage_Element(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "img", "image": "photo.jpg", "width": 200, "height": 150},
			},
		},
	})
	if !strings.Contains(svg, "<image") {
		t.Error("should contain <image")
	}
	if !strings.Contains(svg, `href="photo.jpg"`) {
		t.Error("should contain href")
	}
}

func TestImage_WithRadiusClips(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "img", "image": "photo.jpg", "width": 200, "height": 150, "radius": 12},
			},
		},
	})
	if !strings.Contains(svg, "img-clip-") {
		t.Error("should contain image clip")
	}
}

func TestEdge_Path(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 200, "direction": "row",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "width": 100, "height": 50, "fill": "#eee"},
				map[string]interface{}{"frame": "b", "id": "b", "width": 100, "height": 50, "fill": "#eee"},
			},
		},
		"edges": []interface{}{map[string]interface{}{"from": "a", "to": "b"}},
	})
	if !strings.Contains(svg, "<path") {
		t.Error("should contain <path")
	}
	if !strings.Contains(svg, "marker-end") {
		t.Error("should contain marker-end")
	}
}

func TestEdge_LabelRenders(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 200, "direction": "row",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "width": 100, "height": 50, "fill": "#eee"},
				map[string]interface{}{"frame": "b", "id": "b", "width": 100, "height": 50, "fill": "#eee"},
			},
		},
		"edges": []interface{}{map[string]interface{}{"from": "a", "to": "b", "label": "HTTPS"}},
	})
	if !strings.Contains(svg, "HTTPS") {
		t.Error("should contain edge label")
	}
}

func TestEdge_DashedEdge(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 200, "direction": "row",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "width": 100, "height": 50, "fill": "#eee"},
				map[string]interface{}{"frame": "b", "id": "b", "width": 100, "height": 50, "fill": "#eee"},
			},
		},
		"edges": []interface{}{map[string]interface{}{"from": "a", "to": "b", "style": "dashed"}},
	})
	if !strings.Contains(svg, "stroke-dasharray") {
		t.Error("should contain stroke-dasharray")
	}
}

func TestZIndex_AffectsOrder(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "back", "fill": "#ff0000", "z-index": 1, "width": 100, "height": 100},
				map[string]interface{}{"frame": "front", "fill": "#0000ff", "z-index": 2, "width": 100, "height": 100},
			},
		},
	})
	redPos := strings.Index(svg, "#ff0000")
	bluePos := strings.Index(svg, "#0000ff")
	if redPos >= bluePos {
		t.Errorf("red should appear before blue: red=%d blue=%d", redPos, bluePos)
	}
}

func TestShadow_FilterCreated(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "card", "fill": "white", "shadow": "0 2 8 rgba(0,0,0,0.1)", "width": 100, "height": 50},
			},
		},
	})
	if !strings.Contains(svg, "<filter") {
		t.Error("should contain <filter")
	}
	if !strings.Contains(svg, "feDropShadow") {
		t.Error("should contain feDropShadow")
	}
}

func TestBorder_Stroke(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "b", "fill": "white", "border": "2 solid #333", "width": 100, "height": 50},
			},
		},
	})
	if !strings.Contains(svg, `stroke="#333"`) {
		t.Error("should contain border stroke")
	}
}

func TestBorder_SideBorder(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "b", "fill": "white", "border-bottom": "1 solid #ccc", "width": 100, "height": 50},
			},
		},
	})
	if !strings.Contains(svg, "<line") {
		t.Error("should contain <line for side border")
	}
}

func TestText_Opacity(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"text": "Faded", "opacity": 0.5},
			},
		},
	})
	if !strings.Contains(svg, `opacity="0.5"`) {
		t.Error("should contain opacity attribute")
	}
}

func TestText_AlignCenter(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"text": "Centered", "text-align": "center"},
			},
		},
	})
	if !strings.Contains(svg, `text-anchor="middle"`) {
		t.Error("should contain text-anchor middle")
	}
}

func TestText_AlignRight(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"text": "Right", "text-align": "right"},
			},
		},
	})
	if !strings.Contains(svg, `text-anchor="end"`) {
		t.Error("should contain text-anchor end")
	}
}

func TestText_FontWeightAndFamily(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"text": "Bold", "font": "700 24 Roboto"},
			},
		},
	})
	if !strings.Contains(svg, `font-weight="700"`) {
		t.Error("should contain font-weight 700")
	}
	if !strings.Contains(svg, `font-size="24"`) {
		t.Error("should contain font-size 24")
	}
	if !strings.Contains(svg, "Roboto") {
		t.Error("should contain Roboto family")
	}
}

func TestText_Decoration(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"text": "Underlined", "text-decoration": "underline"},
			},
		},
	})
	if !strings.Contains(svg, `text-decoration="underline"`) {
		t.Error("should contain underline decoration")
	}
}

func TestText_Strikethrough(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"text": "Struck", "text-decoration": "strikethrough"},
			},
		},
	})
	if !strings.Contains(svg, `text-decoration="line-through"`) {
		t.Error("should contain line-through decoration")
	}
}

func TestText_MultiLine_RendersTspan(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{
					"text":      "This is a very long text that should wrap across multiple lines when constrained",
					"max-width": 100,
					"font":      "400 14 Inter",
				},
			},
		},
	})
	if !strings.Contains(svg, "<tspan") {
		t.Error("multi-line text should contain <tspan elements")
	}
}

func TestText_Truncation_RendersEllipsis(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{
					"text":      "This is a very long text that should be truncated with ellipsis",
					"max-width": 80,
					"truncate":  true,
					"font":      "400 14 Inter",
				},
			},
		},
	})
	if !strings.Contains(svg, "...") {
		t.Error("truncated text should contain ellipsis")
	}
	// Should NOT contain tspan (single line)
	if strings.Contains(svg, "<tspan") {
		t.Error("truncated text should be single line, not multi-line")
	}
}

func TestFrame_BorderRadius(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "card", "fill": "white", "radius": 16, "width": 100, "height": 50},
			},
		},
	})
	if !strings.Contains(svg, `rx="16"`) {
		t.Error("should contain rx attribute for border-radius")
	}
}

func TestFrame_Opacity(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"frame": "card", "fill": "white", "opacity": 0.5, "width": 100, "height": 50},
			},
		},
	})
	if !strings.Contains(svg, `opacity="0.5"`) {
		t.Error("should contain opacity attribute on group")
	}
}

func TestEdge_BothArrowMarkers(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 200, "direction": "row",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "width": 100, "height": 50, "fill": "#eee"},
				map[string]interface{}{"frame": "b", "id": "b", "width": 100, "height": 50, "fill": "#eee"},
			},
		},
		"edges": []interface{}{map[string]interface{}{"from": "a", "to": "b", "arrow": "both"}},
	})
	if !strings.Contains(svg, "marker-end") {
		t.Error("should contain marker-end for both arrows")
	}
	if !strings.Contains(svg, "marker-start") {
		t.Error("should contain marker-start for both arrows")
	}
}

func TestEdge_DottedEdge(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 200, "direction": "row",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "width": 100, "height": 50, "fill": "#eee"},
				map[string]interface{}{"frame": "b", "id": "b", "width": 100, "height": 50, "fill": "#eee"},
			},
		},
		"edges": []interface{}{map[string]interface{}{"from": "a", "to": "b", "style": "dotted"}},
	})
	if !strings.Contains(svg, "stroke-dasharray") {
		t.Error("should contain stroke-dasharray for dotted edge")
	}
}

func TestMultipleShadows_CompositeFilter(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{
					"frame":  "card",
					"fill":   "white",
					"shadow": "0 2 8 rgba(0,0,0,0.1) | 0 4 16 rgba(0,0,0,0.2)",
					"width":  100, "height": 50,
				},
			},
		},
	})
	// Multiple shadows should be composited into a single filter
	count := strings.Count(svg, "<filter")
	if count != 1 {
		t.Errorf("expected exactly 1 composite shadow filter, got %d", count)
	}
	// Should contain feMerge for compositing
	if !strings.Contains(svg, "feMerge") {
		t.Error("composite shadow filter should contain feMerge")
	}
	// Should reference the filter
	if !strings.Contains(svg, "filter=\"url(#shadow-") {
		t.Error("should reference shadow filter")
	}
}

func TestEndToEnd_FullCard(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300, "fill": "#f8fafc", "padding": 20,
			"children": []interface{}{
				map[string]interface{}{
					"frame": "card", "fill": "white", "radius": 12, "padding": 16, "gap": 8,
					"shadow": "0 2 8 rgba(0,0,0,0.05)", "border": "1 solid #e2e8f0",
					"children": []interface{}{
						map[string]interface{}{"text": "Card Title", "font": "600 18 Inter"},
						map[string]interface{}{"text": "Description", "font": "400 14 Inter", "color": "#64748b"},
					},
				},
			},
		},
	})
	if !strings.Contains(svg, "Card Title") {
		t.Error("should contain Card Title")
	}
	if !strings.Contains(svg, "Description") {
		t.Error("should contain Description")
	}
	if !strings.Contains(svg, "feDropShadow") {
		t.Error("should contain feDropShadow")
	}
	if !strings.HasPrefix(svg, "<svg") {
		t.Error("should start with <svg")
	}
}

func TestShadow_WithSpread(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{
					"frame": "card", "fill": "white",
					"shadow": "0 2 8 4 rgba(0,0,0,0.1)",
					"width": 100, "height": 50,
				},
			},
		},
	})
	if !strings.Contains(svg, "<filter") {
		t.Error("should contain filter for shadow with spread")
	}
	if !strings.Contains(svg, "feMorphology") {
		t.Error("shadow with spread should use feMorphology for dilate")
	}
	if !strings.Contains(svg, `operator="dilate"`) {
		t.Error("should contain dilate operator")
	}
}

func TestText_FontFallback_Monospace(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"text": "Code", "font": "400 14 Courier New"},
			},
		},
	})
	if !strings.Contains(svg, "monospace") {
		t.Error("Courier font should get monospace fallback")
	}
}

func TestText_FontFallback_Serif(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"text": "Fancy", "font": "400 14 Georgia"},
			},
		},
	})
	// The text element should have font-family='Georgia, serif'
	if !strings.Contains(svg, "Georgia, serif") {
		t.Errorf("Georgia should get serif fallback, got: %s", svg)
	}
}

func TestEdge_StartArrow(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 200, "direction": "row",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "width": 100, "height": 50, "fill": "#eee"},
				map[string]interface{}{"frame": "b", "id": "b", "width": 100, "height": 50, "fill": "#eee"},
			},
		},
		"edges": []interface{}{map[string]interface{}{"from": "a", "to": "b", "arrow": "start"}},
	})
	if !strings.Contains(svg, "marker-start") {
		t.Error("should contain marker-start for start arrow")
	}
}

func TestInterpolatePath_Midpoint(t *testing.T) {
	path := []scene.Point{{X: 0, Y: 0}, {X: 100, Y: 0}, {X: 100, Y: 100}}
	x, y := interpolatePath(path, 0.5)
	// Total length: 100 + 100 = 200. At t=0.5, target=100. That's end of first segment.
	if x != 100 || y != 0 {
		t.Errorf("expected (100, 0), got (%g, %g)", x, y)
	}
}

func TestInterpolatePath_QuarterPoint(t *testing.T) {
	path := []scene.Point{{X: 0, Y: 0}, {X: 100, Y: 0}}
	x, y := interpolatePath(path, 0.25)
	if x != 25 || y != 0 {
		t.Errorf("expected (25, 0), got (%g, %g)", x, y)
	}
}

func TestEdge_NoArrow_NoMarker(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 200, "direction": "row",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "id": "a", "width": 100, "height": 50, "fill": "#eee"},
				map[string]interface{}{"frame": "b", "id": "b", "width": 100, "height": 50, "fill": "#eee"},
			},
		},
		"edges": []interface{}{map[string]interface{}{"from": "a", "to": "b", "arrow": "none"}},
	})
	if strings.Contains(svg, "marker-end") {
		t.Error("arrow:none should not produce marker-end")
	}
	if strings.Contains(svg, "marker-start") {
		t.Error("arrow:none should not produce marker-start")
	}
	if !strings.Contains(svg, "<path") {
		t.Error("should still render edge path")
	}
}

func TestInvisibleFrame_NoDefsDesync(t *testing.T) {
	// An invisible frame with shadow + gradient followed by a visible frame with gradient
	// should produce valid SVG where gradient IDs match between defs and usage.
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{
					"frame": "hidden", "visible": false,
					"fill": "linear-gradient(90deg, red, blue)", "shadow": "0 2 8 rgba(0,0,0,0.1)",
					"width": 100, "height": 50,
				},
				map[string]interface{}{
					"frame": "visible",
					"fill":  "linear-gradient(90deg, green, yellow)",
					"width": 100, "height": 50,
				},
			},
		},
	})
	// Visible frame should reference a gradient that exists in defs
	if !strings.Contains(svg, "url(#grad-") {
		t.Error("visible frame should reference gradient")
	}
	if !strings.Contains(svg, "<linearGradient") {
		t.Error("should contain linearGradient def")
	}
}

func TestClipAndShadow_CounterSync(t *testing.T) {
	// A frame with clip + shadow followed by a frame with gradient:
	// the counter must stay in sync between defs and render.
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{
					"frame": "clipped", "clip": true,
					"fill": "#fff", "shadow": "0 2 8 rgba(0,0,0,0.1)",
					"width": 100, "height": 50,
				},
				map[string]interface{}{
					"frame": "gradient",
					"fill":  "linear-gradient(90deg, red, blue)",
					"width": 100, "height": 50,
				},
			},
		},
	})
	if !strings.Contains(svg, "clip-path") {
		t.Error("should contain clip-path")
	}
	if !strings.Contains(svg, "<linearGradient") {
		t.Error("should contain gradient def")
	}
	if !strings.Contains(svg, "url(#grad-") {
		t.Error("gradient should be referenced")
	}
}

func TestSVG_NoXlinkNamespace(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{"width": 400, "height": 300},
	})
	if strings.Contains(svg, "xmlns:xlink") {
		t.Error("should not contain xmlns:xlink")
	}
}

func TestSVG_ThemeFontInStyle(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"theme": map[string]interface{}{"font-family": "Roboto"},
		"root":  map[string]interface{}{"width": 400, "height": 300},
	})
	layout.ComputeLayout(sg)
	svg := RenderSVG(sg)
	if !strings.Contains(svg, "Roboto") {
		t.Error("should use theme font family in style block")
	}
}

func TestSideBorders_SkippedForCircle(t *testing.T) {
	svg := renderDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{
					"frame": "dot", "shape": "circle", "fill": "red",
					"border-bottom": "2 solid #000",
					"width": 50, "height": 50,
				},
			},
		},
	})
	if strings.Contains(svg, "<line") {
		t.Error("side borders should not render for circle shapes")
	}
}

func BenchmarkRenderSVG(b *testing.B) {
	// Build a moderately complex scene graph
	root := scene.NewFrameNode()
	root.Width = scene.Ptr(800.0)
	root.Height = scene.Ptr(600.0)
	rootFill := "#f8fafc"
	root.Fill = &rootFill
	root.Padding = scene.Spacing{Top: 20, Right: 20, Bottom: 20, Left: 20}
	root.Gap = 16
	root.Direction = "column"

	for i := 0; i < 20; i++ {
		card := scene.NewFrameNode()
		card.ID = fmt.Sprintf("card-%d", i)
		cardFill := "#ffffff"
		card.Fill = &cardFill
		card.Radius = 12
		card.Padding = scene.Spacing{Top: 16, Right: 16, Bottom: 16, Left: 16}
		card.Gap = 8
		card.Direction = "column"
		card.Shadow = []scene.Shadow{{X: 0, Y: 2, Blur: 8, Color: "rgba(0,0,0,0.1)"}}
		card.Border = &scene.Border{Width: 1, Style: "solid", Color: "#e2e8f0"}

		title := scene.NewTextNode()
		title.Content = fmt.Sprintf("Card Title %d", i)
		title.Font = scene.Font{Weight: 700, Size: 18, Family: "Inter"}
		card.Children = append(card.Children, title)

		desc := scene.NewTextNode()
		desc.Content = "This is a card description with some text content."
		desc.Font = scene.Font{Weight: 400, Size: 14, Family: "Inter"}
		desc.Color = "#64748b"
		card.Children = append(card.Children, desc)

		root.Children = append(root.Children, card)
	}

	sg := &scene.SceneGraph{Root: root}
	layout.ComputeLayout(sg)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		RenderSVG(sg)
	}
}
