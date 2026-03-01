package render

import (
	"strings"
	"testing"

	"github.com/fxck/rendspec/internal/layout"
	"github.com/fxck/rendspec/internal/parser"
	"github.com/fxck/rendspec/internal/scene"
)

// renderDSL runs the full pipeline: preprocess → parse → layout → render
func renderDSL(source string) (string, error) {
	sg, err := parser.ParseString(source)
	if err != nil {
		return "", err
	}
	layout.ComputeLayout(sg)
	return RenderSVG(sg), nil
}

func TestIntegration_BasicDSL(t *testing.T) {
	svg, err := renderDSL(`
root:
  width: 800
  height: 600
  fill: "#f8fafc"
  - frame: header
    height: 60
    fill: "#1e293b"
    - text: "Hello World"
      font: 700 24 Inter
      color: "#ffffff"
`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(svg, "<svg") {
		t.Error("should start with <svg")
	}
	if !strings.Contains(svg, "Hello World") {
		t.Error("should contain text")
	}
	if !strings.Contains(svg, "#1e293b") {
		t.Error("should contain header fill")
	}
}

func TestIntegration_ComponentInstantiation(t *testing.T) {
	svg, err := renderDSL(`
components:
  button:
    fill: "#3b82f6"
    radius: 8
    padding: 8 16
    params:
      label: { default: "Click me" }
    - text: "{{label}}"
      font: 500 14 Inter
      color: "#ffffff"

root:
  width: 400
  height: 200
  direction: row
  gap: 8
  - use: button
    label: "Submit"
  - use: button
    label: "Cancel"
`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(svg, "Submit") {
		t.Error("should contain 'Submit' text")
	}
	if !strings.Contains(svg, "Cancel") {
		t.Error("should contain 'Cancel' text")
	}
	if !strings.Contains(svg, "#3b82f6") {
		t.Error("should contain button fill color")
	}
}

func TestIntegration_TokenResolution(t *testing.T) {
	svg, err := renderDSL(`
tokens:
  color:
    primary: "#3b82f6"
    bg: "#0f172a"
  radius:
    md: 12

root:
  width: 400
  height: 200
  fill: $color.bg
  - frame: card
    fill: $color.primary
    radius: $radius.md
    width: 200
    height: 100
    - text: "Hello"
      color: "#ffffff"
`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(svg, "#0f172a") {
		t.Error("should contain resolved bg token")
	}
	if !strings.Contains(svg, "#3b82f6") {
		t.Error("should contain resolved primary token")
	}
	if !strings.Contains(svg, `rx="12"`) {
		t.Error("should contain resolved radius token")
	}
}

func TestIntegration_ThemeApplication(t *testing.T) {
	svg, err := renderDSL(`
theme: dark

root:
  width: 400
  height: 200
  - text: "Dark mode"
`)
	if err != nil {
		t.Fatal(err)
	}
	// Dark theme text should use foreground color #f8fafc
	if !strings.Contains(svg, "#f8fafc") {
		t.Error("should use dark theme foreground color")
	}
}

func TestIntegration_MultiPageDocument(t *testing.T) {
	doc, err := parser.ParseDocument(`
theme: light
pages:
  - name: Login
    root:
      width: 400
      height: 600
      - text: "Login Page"
  - name: Dashboard
    root:
      width: 1280
      height: 800
      - text: "Dashboard"
`)
	if err != nil {
		t.Fatal(err)
	}

	if len(doc.Pages) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(doc.Pages))
	}

	for _, page := range doc.Pages {
		sg := &scene.SceneGraph{
			Root:  page.Root,
			Edges: page.Edges,
			Theme: doc.Theme,
		}
		layout.ComputeLayout(sg)
		svg := RenderSVG(sg)
		if !strings.HasPrefix(svg, "<svg") {
			t.Errorf("page %q should produce valid SVG", page.Name)
		}
	}
}

func TestIntegration_EdgesWithLayout(t *testing.T) {
	svg, err := renderDSL(`
root:
  width: 600
  height: 200
  direction: row
  gap: 100
  - frame: client
    width: 100
    height: 80
    fill: "#e0f2fe"
    - text: "Client"
  - frame: server
    width: 100
    height: 80
    fill: "#dcfce7"
    - text: "Server"
edges:
  - from: client
    to: server
    label: "HTTPS"
    style: dashed
    arrow: end
`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(svg, "<path") {
		t.Error("should contain edge path")
	}
	if !strings.Contains(svg, "HTTPS") {
		t.Error("should contain edge label")
	}
	if !strings.Contains(svg, "stroke-dasharray") {
		t.Error("should contain dashed style")
	}
}

func TestIntegration_GridLayout(t *testing.T) {
	svg, err := renderDSL(`
root:
  width: 600
  height: 400
  layout: grid
  columns: 3
  gap: 16
  padding: 20
  - frame: a
    fill: "#fee2e2"
    height: 100
    - text: "1"
  - frame: b
    fill: "#fef3c7"
    height: 100
    - text: "2"
  - frame: c
    fill: "#dcfce7"
    height: 100
    - text: "3"
  - frame: d
    fill: "#e0f2fe"
    height: 100
    - text: "4"
`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(svg, "#fee2e2") {
		t.Error("should contain first grid item")
	}
	if !strings.Contains(svg, "#e0f2fe") {
		t.Error("should contain fourth grid item")
	}
}

func TestIntegration_VariantComponent(t *testing.T) {
	svg, err := renderDSL(`
components:
  chip:
    fill: "#eff6ff"
    radius: 20
    padding: 6 16
    children:
      - text: "Default"
        font: 500 12 Inter
    variants:
      dark:
        fill: "#1e293b"

root:
  width: 400
  height: 200
  direction: row
  gap: 8
  - use: chip
  - chip: "Custom"
  - chip: "Dark"
    variant: dark
`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(svg, "Default") {
		t.Error("should contain default text")
	}
	if !strings.Contains(svg, "Custom") {
		t.Error("should contain custom label")
	}
	if !strings.Contains(svg, "#1e293b") {
		t.Error("should contain dark variant fill")
	}
}

func TestIntegration_EmptyRoot(t *testing.T) {
	svg, err := renderDSL(`
root:
  width: 100
  height: 100
`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(svg, "<svg") {
		t.Error("empty root should still produce valid SVG")
	}
}
