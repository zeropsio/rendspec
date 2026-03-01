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

	"github.com/fxck/rendspec/internal/handover"
	"github.com/fxck/rendspec/internal/inspect"
	"github.com/fxck/rendspec/internal/layout"
	"github.com/fxck/rendspec/internal/parser"
	pngrender "github.com/fxck/rendspec/internal/png"
	"github.com/fxck/rendspec/internal/render"
	"github.com/fxck/rendspec/internal/scene"
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
		printUsage()
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
  rendspec version
`, version)
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
		if strings.HasPrefix(a, "-") {
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
