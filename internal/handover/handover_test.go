package handover

import (
	"strings"
	"testing"

	"github.com/fxck/rendspec/internal/layout"
	"github.com/fxck/rendspec/internal/parser"
	"github.com/fxck/rendspec/internal/scene"
)

func TestGenerate_MinimalScene(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 800, "height": 600,
		},
	})
	layout.ComputeLayout(sg)
	md := Generate(sg)

	if !strings.Contains(md, "# Design Handover Document") {
		t.Error("missing title")
	}
	if !strings.Contains(md, "## Overview") {
		t.Error("missing overview section")
	}
	if !strings.Contains(md, "800 x 600") {
		t.Error("missing canvas size")
	}
	if !strings.Contains(md, "## Component Tree") {
		t.Error("missing component tree section")
	}
	if !strings.Contains(md, "## Implementation Notes") {
		t.Error("missing implementation notes section")
	}
}

func TestGenerate_WithTokens(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"tokens": map[string]interface{}{
			"color": map[string]interface{}{
				"primary": "#3b82f6",
				"bg": map[string]interface{}{
					"card": "#1e293b",
				},
			},
		},
		"root": map[string]interface{}{
			"width": 400, "height": 300,
		},
	})
	layout.ComputeLayout(sg)
	md := Generate(sg)

	if !strings.Contains(md, "## Design Tokens") {
		t.Error("missing tokens section")
	}
	if !strings.Contains(md, "$color.primary") {
		t.Error("missing token path for color.primary")
	}
	if !strings.Contains(md, "$color.bg.card") {
		t.Error("missing nested token path for color.bg.card")
	}
	if !strings.Contains(md, "### CSS Variables") {
		t.Error("missing CSS variables subsection")
	}
	if !strings.Contains(md, "--color-primary:") {
		t.Error("missing CSS variable --color-primary")
	}
	if !strings.Contains(md, "--color-bg-card:") {
		t.Error("missing CSS variable --color-bg-card")
	}
}

func TestGenerate_WithComponents(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"components": map[string]interface{}{
			"card": map[string]interface{}{
				"fill":    "#1e293b",
				"radius":  12,
				"padding": 20,
				"params": map[string]interface{}{
					"title": map[string]interface{}{
						"default": "Card",
					},
				},
				"children": []interface{}{
					map[string]interface{}{"text": "{{title}}"},
				},
			},
		},
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{
					"use":   "card",
					"title": "Hello",
				},
			},
		},
	})
	layout.ComputeLayout(sg)
	md := Generate(sg)

	if !strings.Contains(md, "## Components") {
		t.Error("missing components section")
	}
	if !strings.Contains(md, "`card`") {
		t.Error("missing component name 'card'")
	}
	if !strings.Contains(md, "`title`") {
		t.Error("missing param name 'title'")
	}
	// Component instance should be annotated in tree
	if !strings.Contains(md, "[card]") {
		t.Error("missing component annotation in tree")
	}
}

func TestGenerate_TreeStructure(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"padding": 20,
			"children": []interface{}{
				map[string]interface{}{
					"frame": "header",
					"height": 50,
					"direction": "row",
					"children": []interface{}{
						map[string]interface{}{
							"text":  "Hello",
							"font":  "700 24 Inter",
							"color": "#fff",
						},
					},
				},
			},
		},
	})
	layout.ComputeLayout(sg)
	md := Generate(sg)

	// Tree should contain root
	if !strings.Contains(md, "root") || !strings.Contains(md, "400 x 300") {
		t.Error("missing root in tree")
	}
	// Tree should contain header
	if !strings.Contains(md, "frame#header") {
		t.Error("missing frame#header in tree")
	}
	// Tree should contain text
	if !strings.Contains(md, `text "Hello"`) {
		t.Error("missing text node in tree")
	}
	// Tree should have connector
	if !strings.Contains(md, "+--") {
		t.Error("missing tree connectors")
	}
}

func TestGenerate_CSSMapping(t *testing.T) {
	fillColor := "#2563eb"
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{
					"frame":     "card",
					"fill":      fillColor,
					"radius":    12,
					"padding":   "20 16",
					"direction": "row",
					"justify":   "between",
					"align":     "center",
					"gap":       8,
					"children": []interface{}{
						map[string]interface{}{"text": "A"},
					},
				},
			},
		},
	})
	layout.ComputeLayout(sg)
	md := Generate(sg)

	checks := []struct {
		name string
		want string
	}{
		{"flex-direction", "flex-direction: row;"},
		{"justify-content", "justify-content: space-between;"},
		{"align-items", "align-items: center;"},
		{"background-color", "background-color: #2563eb;"},
		{"border-radius", "border-radius: 12px;"},
		{"gap", "gap: 8px;"},
	}

	for _, c := range checks {
		if !strings.Contains(md, c.want) {
			t.Errorf("missing CSS mapping %s: want %q", c.name, c.want)
		}
	}
}

func TestGenerate_GridCSSMapping(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"layout":  "grid",
			"columns": 3,
			"gap":     16,
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "height": 50},
				map[string]interface{}{"frame": "b", "height": 50},
				map[string]interface{}{"frame": "c", "height": 50},
			},
		},
	})
	layout.ComputeLayout(sg)
	md := Generate(sg)

	if !strings.Contains(md, "display: grid;") {
		t.Error("missing display: grid")
	}
	if !strings.Contains(md, "grid-template-columns: repeat(3, 1fr);") {
		t.Error("missing grid-template-columns")
	}
}

func TestGenerate_Edges(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"direction": "row",
			"children": []interface{}{
				map[string]interface{}{"frame": "a", "width": 100, "height": 100},
				map[string]interface{}{"frame": "b", "width": 100, "height": 100},
			},
		},
		"edges": []interface{}{
			map[string]interface{}{
				"from":  "a",
				"to":    "b",
				"label": "connection",
				"style": "dashed",
			},
		},
	})
	layout.ComputeLayout(sg)
	md := Generate(sg)

	if !strings.Contains(md, "## Edges") {
		t.Error("missing edges section")
	}
	if !strings.Contains(md, "`a`") {
		t.Error("missing edge from id")
	}
	if !strings.Contains(md, "`b`") {
		t.Error("missing edge to id")
	}
	if !strings.Contains(md, "`connection`") {
		t.Error("missing edge label")
	}
	if !strings.Contains(md, "dashed") {
		t.Error("missing edge style")
	}
}

func TestFlattenTokens(t *testing.T) {
	tokens := map[string]interface{}{
		"color": map[string]interface{}{
			"primary":   "#3b82f6",
			"secondary": "#8b5cf6",
			"bg": map[string]interface{}{
				"card":    "#1e293b",
				"surface": "#0f172a",
			},
		},
		"radius": map[string]interface{}{
			"sm": 6,
			"md": 12,
		},
	}

	flat := FlattenTokens(tokens)

	expected := map[string]string{
		"color.primary":    "#3b82f6",
		"color.secondary":  "#8b5cf6",
		"color.bg.card":    "#1e293b",
		"color.bg.surface": "#0f172a",
		"radius.sm":        "6",
		"radius.md":        "12",
	}

	for k, want := range expected {
		if got, ok := flat[k]; !ok {
			t.Errorf("missing token %q", k)
		} else if got != want {
			t.Errorf("token %q = %q, want %q", k, got, want)
		}
	}

	if len(flat) != len(expected) {
		t.Errorf("got %d tokens, want %d", len(flat), len(expected))
	}
}

func TestGenerate_ShadowCSS(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{
					"frame":  "card",
					"shadow": "0 4 16 rgba(0,0,0,0.08)",
					"children": []interface{}{
						map[string]interface{}{"text": "X"},
					},
				},
			},
		},
	})
	layout.ComputeLayout(sg)
	md := Generate(sg)

	if !strings.Contains(md, "box-shadow:") {
		t.Error("missing box-shadow in CSS")
	}
}

func TestGenerate_ClipCSS(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{
					"frame": "card",
					"clip":  true,
					"children": []interface{}{
						map[string]interface{}{"text": "X"},
					},
				},
			},
		},
	})
	layout.ComputeLayout(sg)
	md := Generate(sg)

	if !strings.Contains(md, "overflow: hidden;") {
		t.Error("missing overflow: hidden in CSS")
	}
}

func TestGenerate_ImplementationNotes(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
		},
	})
	layout.ComputeLayout(sg)
	md := Generate(sg)

	if !strings.Contains(md, "## Implementation Notes") {
		t.Error("missing implementation notes")
	}
	if !strings.Contains(md, "DSL → CSS Property Mapping") {
		t.Error("missing CSS property mapping table")
	}
	if !strings.Contains(md, "`truncate: true`") {
		t.Error("missing truncate mapping")
	}
}

func TestGenerate_NoTokensSection_WhenEmpty(t *testing.T) {
	sg := &scene.SceneGraph{
		Root:   scene.NewFrameNode(),
		Theme:  scene.DefaultTheme(),
		Tokens: nil,
	}
	sg.Root.Width = scene.Ptr(400.0)
	sg.Root.Height = scene.Ptr(300.0)
	layout.ComputeLayout(sg)
	md := Generate(sg)

	if strings.Contains(md, "## Design Tokens") {
		t.Error("should not have tokens section when no tokens")
	}
}

func TestGenerateDocument_MultiPage(t *testing.T) {
	doc, err := parser.ParseDocument(`
theme: light
pages:
  - name: Login
    root:
      width: 400
      height: 600
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

	for i := range doc.Pages {
		sg := &scene.SceneGraph{
			Root:  doc.Pages[i].Root,
			Edges: doc.Pages[i].Edges,
			Theme: doc.Theme,
		}
		layout.ComputeLayout(sg)
	}

	md := GenerateDocument(doc, doc.Pages, nil)
	if !strings.Contains(md, "Login") {
		t.Error("should contain Login page")
	}
	if !strings.Contains(md, "Dashboard") {
		t.Error("should contain Dashboard page")
	}
}

func TestGenerate_SpecialCharactersInText(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{"text": `He said "hello" & <goodbye>`},
			},
		},
	})
	layout.ComputeLayout(sg)
	md := Generate(sg)
	// Should contain the text content without crashing
	if !strings.Contains(md, "hello") {
		t.Error("should contain text content")
	}
}

func TestGenerate_DeeplyNestedTree(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
			"children": []interface{}{
				map[string]interface{}{
					"frame": "l1",
					"children": []interface{}{
						map[string]interface{}{
							"frame": "l2",
							"children": []interface{}{
								map[string]interface{}{
									"frame": "l3",
									"children": []interface{}{
										map[string]interface{}{"text": "Deep"},
									},
								},
							},
						},
					},
				},
			},
		},
	})
	layout.ComputeLayout(sg)
	md := Generate(sg)
	if !strings.Contains(md, "Deep") {
		t.Error("should contain deeply nested text")
	}
	if !strings.Contains(md, "l3") {
		t.Error("should contain deeply nested frame")
	}
}

func TestGenerate_NoEdgesSection_WhenEmpty(t *testing.T) {
	sg := parser.ParseDict(map[string]interface{}{
		"root": map[string]interface{}{
			"width": 400, "height": 300,
		},
	})
	layout.ComputeLayout(sg)
	md := Generate(sg)

	if strings.Contains(md, "## Edges") {
		t.Error("should not have edges section when no edges")
	}
}
