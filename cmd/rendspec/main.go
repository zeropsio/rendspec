// rendspec CLI — render visual designs from a YAML DSL.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zeropsio/rendspec/internal/handover"
	"github.com/zeropsio/rendspec/internal/inspect"
	"github.com/zeropsio/rendspec/internal/layout"
	"github.com/zeropsio/rendspec/internal/parser"
	pngrender "github.com/zeropsio/rendspec/internal/png"
	"github.com/zeropsio/rendspec/internal/render"
	"github.com/zeropsio/rendspec/internal/scene"
)

const version = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(0)
	}

	switch os.Args[1] {
	case "render":
		os.Exit(cmdRender(os.Args[2:]))
	case "validate":
		os.Exit(cmdValidate(os.Args[2:]))
	case "inspect":
		os.Exit(cmdInspect(os.Args[2:]))
	case "handover":
		os.Exit(cmdHandover(os.Args[2:]))
	case "version", "--version", "-v":
		fmt.Printf("rendspec %s\n", version)
	case "help", "--help", "-h":
		if len(os.Args) > 2 && os.Args[2] == "syntax" {
			printSyntax()
		} else {
			printUsage()
		}
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `rendspec %s — visual design DSL renderer

Usage:
  rendspec render <file.rds|-> [-o output.svg|.png] [--scale 2]
  rendspec validate <file.rds>
  rendspec inspect <file.rds>
  rendspec handover <file.rds> [-o output.md] [--no-png] [--scale 2]
  rendspec help syntax          show full DSL reference
  rendspec version

PNG output is built-in (resvg via WASM, no external binary needed).
Use -o file.png to render as PNG, --scale to set resolution multiplier.
`, version)
}

func printSyntax() {
	fmt.Fprintf(os.Stderr, `rendspec DSL Reference
======================

Every .rds file is valid YAML with this top-level shape:

  theme: light              # optional: light | dark | blueprint | sketch | {custom}
  tokens:                   # optional: design token definitions
    color:
      primary: "#3b82f6"
  components:               # optional: reusable frame templates
    button: { ... }
  root:                     # REQUIRED — the canvas
    width: 1280
    height: 720
    ...children...
  edges:                    # optional: connections between frames
    - from: a
      to: b

PREPROCESSOR (runs before YAML parsing)
────────────────────────────────────────
Auto-quoting — Multi-word shorthand values don't need quotes:
  padding: 12 24                          font: 700 20 Inter
  border: 1 solid #e2e8f0                 shadow: 0 4 16 rgba(0,0,0,0.08)
  fill: linear-gradient(135deg, #667eea, #764ba2)

  Properties: padding, margin, font, border, border-top/right/bottom/left,
              shadow, label-font, gradient, fill

Implicit children — No need to write "children:":
  - frame: card
    fill: white
    padding: 20
    - text: "Hello"          # ← automatically becomes a child
    - frame: inner           # ← also a child

FRAME PROPERTIES (- frame: name)
────────────────────────────────
  # Sizing
  width: 200                 # explicit size in pixels
  height: 100
  min-width / max-width / min-height / max-height: N
  flex: 1                    # flex grow factor (fills available space)

  # Layout (how children are arranged)
  layout: flex|grid          # default: flex
  direction: row|column      # default: column
  align: start|center|end|stretch       # default: stretch
  justify: start|center|end|between|around   # default: start
  gap: 16                    # space between children in px
  wrap: true                 # flex wrapping (default: false)

  # Grid layout (when layout: grid)
  columns: 3                 rows: 2
  column-gap: 16             row-gap: 12

  # Spacing (CSS-like shorthand)
  padding: 20                # single value
  padding: 12 24             # vert horiz
  padding: 8 16 12 16        # top right bottom left
  padding-x: 24              padding-y: 12
  margin: 8

  # Visual
  fill: "#2563eb"            # any CSS color
  fill: linear-gradient(135deg, #667eea, #764ba2)
  fill: radial-gradient(circle, #fff, #000)
  opacity: 0.8
  radius: 12
  border: 1 solid #e2e8f0    # width style color
  border-top/right/bottom/left: 1.5 dashed #ccc
  shadow: 0 4 16 rgba(0,0,0,0.08)
  clip: true
  visible: false

  # Image
  image: "photo.jpg"
  image-fit: cover|contain|fill|none

  # Shape
  shape: rect|circle|ellipse|diamond   # default: rect

  # Z-ordering and positioning
  z-index: 2
  position: absolute
  x: 20
  y: 40

TEXT NODES (- text: "content")
──────────────────────────────
  font: 700 24 Inter         # "weight size family"
  color: "#0f172a"
  text-align: left|center|right
  line-height: 1.6
  max-width: 400             # triggers word wrapping
  letter-spacing: 1.5
  text-decoration: none|underline|strikethrough
  truncate: true
  opacity: 0.5

EDGES (connections between named frames)
────────────────────────────────────────
  edges:
    - from: client            # frame name or id
      to: server
      stroke: "#94a3b8"
      stroke-width: 2
      style: solid|dashed|dotted
      arrow: none|start|end|both
      curve: straight|orthogonal|bus|vertical
      label: "HTTPS"
      label-font: 500 11 Inter
      label-color: "#64748b"
      label-position: 0.5    # 0–1 along edge
      from-anchor: top|right|bottom|left
      to-anchor: top|right|bottom|left

COMPONENTS (reusable templates)
───────────────────────────────
  components:
    chip:
      fill: "#eff6ff"
      radius: 20
      padding: 6 16
      border: 1 solid #bfdbfe
      - text: "Default"
        font: 500 12 Inter
      variants:
        dark:
          fill: "#1e293b"
  # Usage in root:
  - use: chip
  - chip: "Custom Label"     # shorthand sets first text child
  - chip: "Dark Mode"
    variant: dark

PARAMETERIZED COMPONENTS
────────────────────────
  components:
    stat-card:
      params:
        title: { default: "Metric" }
        value: { default: "0" }
      fill: "#1e293b"
      padding: 20
      - text: "{{title}}"
      - text: "{{value}}"
        font: 700 28 Inter
  # Usage:
  - use: stat-card
    title: "Revenue"
    value: "$12,450"

DESIGN TOKENS (referenced with $ prefix)
─────────────────────────────────────────
  tokens:
    color:
      primary: "#3b82f6"
      bg:
        card: "#1e293b"
    radius:
      md: 12
  # Usage:
  fill: $color.bg.card       # resolves to "#1e293b"
  radius: $radius.md          # resolves to 12

THEMES
──────
  theme: dark                 # built-in: light | dark | blueprint | sketch
  # Or custom:
  theme:
    background: "#0f172a"
    foreground: "#f8fafc"
    muted: "#94a3b8"
    accent: "#3b82f6"
    border: "#334155"
    radius: 8
    font-family: Inter
    font-size: 14
    font-weight: 400

MULTI-PAGE DOCUMENTS
────────────────────
  pages:
    - name: Login
      root:
        width: 400
        height: 600
        ...
    - name: Dashboard
      root:
        width: 1280
        height: 800
        ...

MINIMAL EXAMPLE
───────────────
  root:
    width: 400
    height: 300
    fill: "#f8fafc"
    padding: 24
    gap: 16
    - text: "Hello World"
      font: 700 24 Inter
      color: "#0f172a"
    - frame: card
      fill: white
      radius: 12
      padding: 16
      border: 1 solid #e2e8f0
      shadow: 0 2 8 rgba(0,0,0,0.06)
      - text: "A simple card"
        font: 400 14 Inter
        color: "#475569"
`)
}

func cmdRender(args []string) int {
	fs := flag.NewFlagSet("render", flag.ExitOnError)
	output := fs.String("o", "", "Output file path (default: stdout SVG)")
	scale := fs.Float64("scale", 2, "PNG scale factor")
	// Reorder args: Go's flag package stops at first non-flag arg,
	// so we partition into flags and positional args.
	flagArgs, posArgs := partitionArgs(args)
	fs.Parse(flagArgs)
	posArgs = append(posArgs, fs.Args()...)

	if len(posArgs) < 1 {
		fmt.Fprintln(os.Stderr, "Error: missing input file")
		return 1
	}
	inputPath := posArgs[0]

	// Support stdin
	if inputPath != "-" {
		if _, err := os.Stat(inputPath); os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "Error: file not found: %s\n", inputPath)
			return 1
		}
	}

	t0 := time.Now()

	// Read stdin once if needed
	var stdinData []byte
	if inputPath == "-" {
		var err error
		stdinData, err = io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading stdin: %v\n", err)
			return 1
		}
	}

	// Try multi-page first
	var doc *scene.Document
	var err error
	if inputPath == "-" {
		doc, err = parser.ParseDocument(string(stdinData))
	} else {
		doc, err = parser.ParseDocumentFile(inputPath)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	// Print warnings
	for _, w := range doc.Warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
	}

	if len(doc.Pages) > 1 && *output != "" {
		out := *output
		ext := filepath.Ext(out)
		stem := strings.TrimSuffix(filepath.Base(out), ext)
		dir := filepath.Dir(out)
		if ext == "" {
			ext = ".svg"
		}

		for _, page := range doc.Pages {
			sg := &scene.SceneGraph{
				Root:       page.Root,
				Edges:      page.Edges,
				Theme:      doc.Theme,
				Components: doc.Components,
				Tokens:     doc.Tokens,
			}
			layout.ComputeLayout(sg)
			svg := render.RenderSVG(sg)

			pageName := strings.ToLower(strings.ReplaceAll(page.Name, " ", "-"))
			pageOut := filepath.Join(dir, fmt.Sprintf("%s-%s%s", stem, pageName, ext))

			if ext == ".png" {
				if err := writePNG(svg, pageOut, *scale); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					return 1
				}
			} else {
				if err := os.WriteFile(pageOut, []byte(svg), 0644); err != nil {
					fmt.Fprintf(os.Stderr, "Error: %v\n", err)
					return 1
				}
			}
			fmt.Fprintf(os.Stderr, "  Page '%s' → %s\n", page.Name, pageOut)
		}

		elapsed := time.Since(t0)
		fmt.Fprintf(os.Stderr, "Rendered %d pages in %dms\n", len(doc.Pages), elapsed.Milliseconds())
		return 0
	}

	// Single page — use page from already-parsed document if available
	var sg *scene.SceneGraph
	if len(doc.Pages) == 1 {
		sg = &scene.SceneGraph{
			Root:       doc.Pages[0].Root,
			Edges:      doc.Pages[0].Edges,
			Theme:      doc.Theme,
			Components: doc.Components,
			Tokens:     doc.Tokens,
			Warnings:   doc.Warnings,
		}
	} else if inputPath == "-" {
		sg, err = parser.ParseString(string(stdinData))
	} else {
		sg, err = parser.ParseFile(inputPath)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	layout.ComputeLayout(sg)
	svg := render.RenderSVG(sg)
	elapsed := time.Since(t0)

	for _, w := range sg.Warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
	}

	if *output != "" {
		dir := filepath.Dir(*output)
		if dir != "" && dir != "." {
			os.MkdirAll(dir, 0755)
		}
		if filepath.Ext(*output) == ".png" {
			if err := writePNG(svg, *output, *scale); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return 1
			}
		} else {
			if err := os.WriteFile(*output, []byte(svg), 0644); err != nil {
				fmt.Fprintf(os.Stderr, "Error: %v\n", err)
				return 1
			}
		}
		fmt.Fprintf(os.Stderr, "Rendered %s → %s in %dms\n", inputPath, *output, elapsed.Milliseconds())
	} else {
		fmt.Print(svg)
	}

	return 0
}

func writePNG(svg, path string, scale float64) error {
	data, err := pngrender.RenderPNG(svg, scale)
	if err != nil {
		return fmt.Errorf("PNG render: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func cmdValidate(args []string) int {
	fs := flag.NewFlagSet("validate", flag.ExitOnError)
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: missing input file")
		return 1
	}
	inputPath := fs.Arg(0)

	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: file not found: %s\n", inputPath)
		return 1
	}

	sg, err := parser.ParseFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid: %s\n  Error: %v\n", inputPath, err)
		return 1
	}

	layout.ComputeLayout(sg)

	frameCount := inspect.CountFrames(sg.Root)
	textCount := inspect.CountTexts(sg.Root)
	edgeCount := len(sg.Edges)

	fmt.Printf("Valid: %s\n", inputPath)
	fmt.Printf("  Canvas: %.0f x %.0f\n", sg.Root.Layout.Width, sg.Root.Layout.Height)
	fmt.Printf("  Frames: %d\n", frameCount)
	fmt.Printf("  Text nodes: %d\n", textCount)
	fmt.Printf("  Edges: %d\n", edgeCount)
	for _, w := range sg.Warnings {
		fmt.Printf("  Warning: %s\n", w)
	}
	return 0
}

func cmdInspect(args []string) int {
	fs := flag.NewFlagSet("inspect", flag.ExitOnError)
	fs.Parse(args)

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "Error: missing input file")
		return 1
	}
	inputPath := fs.Arg(0)

	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: file not found: %s\n", inputPath)
		return 1
	}

	sg, err := parser.ParseFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	layout.ComputeLayout(sg)

	data := inspect.NodeToDict(sg.Root)
	edges := make([]map[string]interface{}, len(sg.Edges))
	for i, e := range sg.Edges {
		edges[i] = inspect.EdgeToDict(e)
	}
	data["edges"] = edges

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(data); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}
	return 0
}

func cmdHandover(args []string) int {
	fs := flag.NewFlagSet("handover", flag.ExitOnError)
	output := fs.String("o", "", "Output file path (default: stdout)")
	scale := fs.Float64("scale", 2, "PNG scale factor")
	noPng := fs.Bool("no-png", false, "Skip PNG rendering")
	flagArgs, posArgs := partitionArgs(args)
	fs.Parse(flagArgs)
	posArgs = append(posArgs, fs.Args()...)

	if len(posArgs) < 1 {
		fmt.Fprintln(os.Stderr, "Error: missing input file")
		return 1
	}
	inputPath := posArgs[0]

	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: file not found: %s\n", inputPath)
		return 1
	}

	t0 := time.Now()

	// Try multi-page first
	doc, err := parser.ParseDocumentFile(inputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		return 1
	}

	// Determine output directory for PNGs (next to the .md file)
	renderPng := *output != "" && !*noPng
	outDir := "."
	mdStem := ""
	if *output != "" {
		outDir = filepath.Dir(*output)
		mdStem = strings.TrimSuffix(filepath.Base(*output), filepath.Ext(*output))
		if outDir != "" && outDir != "." {
			os.MkdirAll(outDir, 0755)
		}
	}

	// Print warnings
	for _, w := range doc.Warnings {
		fmt.Fprintf(os.Stderr, "Warning: %s\n", w)
	}

	var md string
	if len(doc.Pages) > 1 {
		pageImages := make(map[int]string)
		for i := range doc.Pages {
			sg := &scene.SceneGraph{
				Root:       doc.Pages[i].Root,
				Edges:      doc.Pages[i].Edges,
				Theme:      doc.Theme,
				Components: doc.Components,
				Tokens:     doc.Tokens,
			}
			layout.ComputeLayout(sg)

			if renderPng {
				svg := render.RenderSVG(sg)
				pageName := strings.ToLower(strings.ReplaceAll(doc.Pages[i].Name, " ", "-"))
				pngName := fmt.Sprintf("%s-%s.png", mdStem, pageName)
				pngPath := filepath.Join(outDir, pngName)
				if err := writePNG(svg, pngPath, *scale); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: PNG render failed: %v\n", err)
				} else {
					pageImages[i] = pngName
					fmt.Fprintf(os.Stderr, "  PNG: %s\n", pngPath)
				}
			}
		}
		md = handover.GenerateDocument(doc, doc.Pages, pageImages)
	} else {
		sg, err := parser.ParseFile(inputPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		layout.ComputeLayout(sg)

		var opts handover.Options
		if renderPng {
			svg := render.RenderSVG(sg)
			pngName := mdStem + ".png"
			pngPath := filepath.Join(outDir, pngName)
			if err := writePNG(svg, pngPath, *scale); err != nil {
				fmt.Fprintf(os.Stderr, "Warning: PNG render failed: %v\n", err)
			} else {
				opts.ImagePaths = []string{pngName}
				fmt.Fprintf(os.Stderr, "  PNG: %s\n", pngPath)
			}
		}
		md = handover.Generate(sg, opts)
	}

	elapsed := time.Since(t0)

	if *output != "" {
		if err := os.WriteFile(*output, []byte(md), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return 1
		}
		fmt.Fprintf(os.Stderr, "Handover %s → %s in %dms\n", inputPath, *output, elapsed.Milliseconds())
	} else {
		fmt.Print(md)
	}

	return 0
}


// partitionArgs splits args into flag args (starting with -) and positional args,
// keeping flag values together with their flags.
func partitionArgs(args []string) (flagArgs, posArgs []string) {
	knownFlags := map[string]bool{"o": true, "scale": true, "no-png": false}
	i := 0
	for i < len(args) {
		a := args[i]
		if a == "-" {
			// Bare "-" means stdin, treat as positional
			posArgs = append(posArgs, a)
			i++
		} else if strings.HasPrefix(a, "-") {
			flagArgs = append(flagArgs, a)
			// Check if this flag takes a value (next arg)
			name := strings.TrimLeft(a, "-")
			if eqIdx := strings.Index(name, "="); eqIdx >= 0 {
				// -o=file or --scale=2 form, value is inline
				i++
				continue
			}
			if knownFlags[name] && i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
			i++
		} else {
			posArgs = append(posArgs, a)
			i++
		}
	}
	return
}
