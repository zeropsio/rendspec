// Package parser converts YAML DSL input into a scene graph.
package parser

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/fxck/rendspec/internal/preprocess"
	"github.com/fxck/rendspec/internal/scene"
	"gopkg.in/yaml.v3"
)

// --- Enum validation sets ---

var validDirections = map[string]bool{"row": true, "column": true}
var validAligns = map[string]bool{"start": true, "center": true, "end": true, "stretch": true}
var validJustify = map[string]bool{"start": true, "center": true, "end": true, "between": true, "around": true}
var validPositions = map[string]bool{"relative": true, "absolute": true}
var validShapes = map[string]bool{"rect": true, "circle": true, "ellipse": true, "diamond": true}
var validLayoutModes = map[string]bool{"flex": true, "grid": true}
var validImageFit = map[string]bool{"cover": true, "contain": true, "fill": true, "none": true}
var validTextAlign = map[string]bool{"left": true, "center": true, "right": true}
var validTextDecoration = map[string]bool{"none": true, "underline": true, "strikethrough": true}
var validEdgeStyle = map[string]bool{"solid": true, "dashed": true, "dotted": true}
var validArrow = map[string]bool{"none": true, "start": true, "end": true, "both": true}
var validCurve = map[string]bool{"straight": true, "orthogonal": true, "bus": true, "vertical": true}
var validAnchor = map[string]bool{"auto": true, "top": true, "bottom": true, "left": true, "right": true}

const maxComponentDepth = 32

func warnInvalidEnum(warnings *[]string, property, value string, valid map[string]bool) {
	if value == "" {
		return
	}
	if !valid[value] {
		keys := make([]string, 0, len(valid))
		for k := range valid {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		*warnings = append(*warnings, fmt.Sprintf("invalid %s: %q (valid: %s)", property, value, strings.Join(keys, ", ")))
	}
}

// --- Public API ---

// ParseFile parses a .rds file into a SceneGraph.
func ParseFile(path string) (*scene.SceneGraph, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseString(string(raw))
}

// ParseString parses DSL source into a SceneGraph.
func ParseString(source string) (*scene.SceneGraph, error) {
	processed := preprocess.Preprocess(source)
	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(processed), &data); err != nil {
		return nil, fmt.Errorf("YAML parse error: %w", err)
	}
	if data == nil {
		data = make(map[string]interface{})
	}
	return ParseDict(data), nil
}

// ParseDocumentFile parses a multi-page document from a file.
func ParseDocumentFile(path string) (*scene.Document, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return ParseDocument(string(raw))
}

// ParseDocument parses a multi-page document from a string.
func ParseDocument(source string) (*scene.Document, error) {
	processed := preprocess.Preprocess(source)
	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(processed), &data); err != nil {
		return nil, fmt.Errorf("YAML parse error: %w", err)
	}
	if data == nil {
		data = make(map[string]interface{})
	}
	return parseDocumentDict(data), nil
}

// ParseDict parses a pre-parsed dict into a SceneGraph.
func ParseDict(data map[string]interface{}) *scene.SceneGraph {
	sg := scene.NewSceneGraph()
	var warnings []string

	// Parse tokens first
	if tokens, ok := data["tokens"].(map[string]interface{}); ok {
		sg.Tokens = tokens
	}

	// Parse theme
	if themeVal, ok := data["theme"]; ok {
		sg.Theme = parseTheme(themeVal, &warnings)
	}

	// Parse components
	if comps, ok := data["components"].(map[string]interface{}); ok {
		sg.Components = comps
	}

	// Merge frames: into components
	if frames, ok := data["frames"].(map[string]interface{}); ok {
		for k, v := range frames {
			sg.Components[k] = v
		}
	}

	// Resolve tokens
	if len(sg.Tokens) > 0 {
		for name := range sg.Components {
			sg.Components[name] = resolveTokens(sg.Components[name], sg.Tokens, &warnings)
		}
		if root, ok := data["root"]; ok {
			data["root"] = resolveTokens(root, sg.Tokens, &warnings)
		}
		if edges, ok := data["edges"]; ok {
			data["edges"] = resolveTokens(edges, sg.Tokens, &warnings)
		}
	}

	// Parse root
	if rootData, ok := data["root"].(map[string]interface{}); ok {
		sg.Root = parseFrame(rootData, sg.Theme, sg.Components, &warnings, 0)
	}

	// Parse top-level edges
	if edgesData, ok := data["edges"].([]interface{}); ok {
		for _, ed := range edgesData {
			if edMap, ok := ed.(map[string]interface{}); ok {
				sg.Edges = append(sg.Edges, parseEdge(edMap, &warnings))
			}
		}
	}

	// Collect edges from children
	collectEdges(sg.Root, &sg.Edges)

	// Validate edge frame references
	frameIDs := collectFrameIDs(sg.Root, &warnings)
	for _, edge := range sg.Edges {
		if edge.FromID == "" {
			warnings = append(warnings, "edge missing 'from' property")
		}
		if edge.ToID == "" {
			warnings = append(warnings, "edge missing 'to' property")
		}
		if edge.FromID != "" && !frameIDs[edge.FromID] {
			warnings = append(warnings, fmt.Sprintf("edge references unknown frame: %s", edge.FromID))
		}
		if edge.ToID != "" && !frameIDs[edge.ToID] {
			warnings = append(warnings, fmt.Sprintf("edge references unknown frame: %s", edge.ToID))
		}
	}

	sg.Warnings = warnings
	return sg
}

// collectFrameIDs walks the frame tree and collects all frame IDs.
func collectFrameIDs(node *scene.FrameNode, warnings *[]string) map[string]bool {
	ids := make(map[string]bool)
	collectFrameIDsRecurse(node, ids, warnings)
	return ids
}

func collectFrameIDsRecurse(node *scene.FrameNode, ids map[string]bool, warnings *[]string) {
	if node.ID != "" {
		if ids[node.ID] {
			*warnings = append(*warnings, fmt.Sprintf("duplicate frame id: %q", node.ID))
		}
		ids[node.ID] = true
	}
	for _, child := range node.Children {
		if fn, ok := child.(*scene.FrameNode); ok {
			collectFrameIDsRecurse(fn, ids, warnings)
		}
	}
}

func parseDocumentDict(data map[string]interface{}) *scene.Document {
	doc := scene.NewDocument()
	var warnings []string

	if tokens, ok := data["tokens"].(map[string]interface{}); ok {
		doc.Tokens = tokens
	}
	if themeVal, ok := data["theme"]; ok {
		doc.Theme = parseTheme(themeVal, &warnings)
	}
	if comps, ok := data["components"].(map[string]interface{}); ok {
		doc.Components = comps
	}
	if frames, ok := data["frames"].(map[string]interface{}); ok {
		for k, v := range frames {
			doc.Components[k] = v
		}
	}

	// Resolve tokens in components
	if len(doc.Tokens) > 0 && len(doc.Components) > 0 {
		for name := range doc.Components {
			doc.Components[name] = resolveTokens(doc.Components[name], doc.Tokens, &warnings)
		}
	}

	if pagesData, ok := data["pages"].([]interface{}); ok {
		for i, pd := range pagesData {
			pageData, ok := pd.(map[string]interface{})
			if !ok {
				continue
			}
			page := scene.Page{
				Name: getString(pageData, "name", fmt.Sprintf("Page %d", i+1)),
			}

			pageRootData, _ := pageData["root"].(map[string]interface{})
			if pageRootData == nil {
				pageRootData = make(map[string]interface{})
			}
			if len(doc.Tokens) > 0 {
				pageRootData = resolveTokens(pageRootData, doc.Tokens, &warnings).(map[string]interface{})
			}
			page.Root = parseFrame(pageRootData, doc.Theme, doc.Components, &warnings, 0)

			if edgesData, ok := pageData["edges"].([]interface{}); ok {
				if len(doc.Tokens) > 0 {
					resolved := resolveTokens(edgesData, doc.Tokens, &warnings)
					edgesData = resolved.([]interface{})
				}
				for _, ed := range edgesData {
					if edMap, ok := ed.(map[string]interface{}); ok {
						page.Edges = append(page.Edges, parseEdge(edMap, &warnings))
					}
				}
			}

			collectEdges(page.Root, &page.Edges)

			// Validate edge frame references for this page
			frameIDs := collectFrameIDs(page.Root, &warnings)
			for _, edge := range page.Edges {
				if edge.FromID != "" && !frameIDs[edge.FromID] {
					warnings = append(warnings, fmt.Sprintf("edge references unknown frame: %s", edge.FromID))
				}
				if edge.ToID != "" && !frameIDs[edge.ToID] {
					warnings = append(warnings, fmt.Sprintf("edge references unknown frame: %s", edge.ToID))
				}
			}

			doc.Pages = append(doc.Pages, page)
		}
	} else if rootData, ok := data["root"].(map[string]interface{}); ok {
		// Single-page document
		page := scene.Page{Name: "Page 1"}
		if len(doc.Tokens) > 0 {
			rootData = resolveTokens(rootData, doc.Tokens, &warnings).(map[string]interface{})
		}
		page.Root = parseFrame(rootData, doc.Theme, doc.Components, &warnings, 0)
		if edgesData, ok := data["edges"].([]interface{}); ok {
			if len(doc.Tokens) > 0 {
				resolved := resolveTokens(edgesData, doc.Tokens, &warnings)
				edgesData = resolved.([]interface{})
			}
			for _, ed := range edgesData {
				if edMap, ok := ed.(map[string]interface{}); ok {
					page.Edges = append(page.Edges, parseEdge(edMap, &warnings))
				}
			}
		}
		collectEdges(page.Root, &page.Edges)

		// Validate edge frame references
		frameIDs := collectFrameIDs(page.Root, &warnings)
		for _, edge := range page.Edges {
			if edge.FromID != "" && !frameIDs[edge.FromID] {
				warnings = append(warnings, fmt.Sprintf("edge references unknown frame: %s", edge.FromID))
			}
			if edge.ToID != "" && !frameIDs[edge.ToID] {
				warnings = append(warnings, fmt.Sprintf("edge references unknown frame: %s", edge.ToID))
			}
		}

		doc.Pages = append(doc.Pages, page)
	}

	doc.Warnings = warnings
	return doc
}

// collectEdges walks the tree and moves any EdgeNode children to the edges list.
func collectEdges(node *scene.FrameNode, edges *[]*scene.EdgeNode) {
	var newChildren []scene.Node
	for _, child := range node.Children {
		if edge, ok := child.(*scene.EdgeNode); ok {
			*edges = append(*edges, edge)
		} else {
			newChildren = append(newChildren, child)
			if frame, ok := child.(*scene.FrameNode); ok {
				collectEdges(frame, edges)
			}
		}
	}
	node.Children = newChildren
}

// --- Theme ---

func parseTheme(value interface{}, warnings *[]string) scene.Theme {
	switch v := value.(type) {
	case string:
		if theme, ok := scene.BuiltinThemes[v]; ok {
			return theme
		}
		*warnings = append(*warnings, fmt.Sprintf("unknown theme %q, using default light theme", v))
		return scene.DefaultTheme()
	case map[string]interface{}:
		theme := scene.DefaultTheme()
		if bg, ok := v["background"].(string); ok {
			theme.Background = bg
		}
		if fg, ok := v["foreground"].(string); ok {
			theme.Foreground = fg
		}
		if m, ok := v["muted"].(string); ok {
			theme.Muted = m
		}
		if a, ok := v["accent"].(string); ok {
			theme.Accent = a
		}
		if b, ok := v["border"].(string); ok {
			theme.Border = b
		}
		if r, ok := v["radius"]; ok {
			theme.Radius = toFloat(r)
		}
		if ff, ok := v["font-family"].(string); ok {
			theme.FontFamily = ff
		}
		if fs, ok := v["font-size"]; ok {
			theme.FontSize = toFloat(fs)
		}
		if fw, ok := v["font-weight"]; ok {
			theme.FontWeight = toInt(fw)
		}
		return theme
	default:
		return scene.DefaultTheme()
	}
}

// --- Property Parsers ---

// ParseFont parses '700 20 Inter' → Font.
func ParseFont(value interface{}) scene.Font {
	if value == nil {
		return scene.DefaultFont()
	}
	switch v := value.(type) {
	case map[string]interface{}:
		return scene.Font{
			Weight: getInt(v, "weight", 400),
			Size:   getFloat(v, "size", 14),
			Family: getString(v, "family", "Inter"),
		}
	default:
		s := fmt.Sprintf("%v", value)
		parts := strings.SplitN(s, " ", 3)
		switch len(parts) {
		case 3:
			w, _ := strconv.Atoi(parts[0])
			sz, _ := strconv.ParseFloat(parts[1], 64)
			return scene.Font{
				Weight: w,
				Size:   sz,
				Family: strings.Trim(parts[2], "\"'"),
			}
		case 2:
			w, wErr := strconv.Atoi(parts[0])
			sz, szErr := strconv.ParseFloat(parts[1], 64)
			if wErr == nil && szErr == nil {
				return scene.Font{Weight: w, Size: sz, Family: "Inter"}
			}
			sz2, _ := strconv.ParseFloat(parts[0], 64)
			return scene.Font{Weight: 400, Size: sz2, Family: strings.Trim(parts[1], "\"'")}
		case 1:
			sz, err := strconv.ParseFloat(parts[0], 64)
			if err == nil {
				return scene.Font{Weight: 400, Size: sz, Family: "Inter"}
			}
			return scene.Font{Weight: 400, Size: 14, Family: strings.Trim(parts[0], "\"'")}
		}
	}
	return scene.DefaultFont()
}

// ParseSpacing parses CSS shorthand: '12', '12 24', '12 24 12', '12 24 12 24'.
func ParseSpacing(value interface{}) scene.Spacing {
	switch v := value.(type) {
	case int:
		f := float64(v)
		return scene.Spacing{Top: f, Right: f, Bottom: f, Left: f}
	case float64:
		return scene.Spacing{Top: v, Right: v, Bottom: v, Left: v}
	default:
		s := fmt.Sprintf("%v", value)
		parts := strings.Fields(s)
		if len(parts) > 4 {
			parts = parts[:4] // 5+ values: use first 4
		}
		switch len(parts) {
		case 1:
			f, _ := strconv.ParseFloat(parts[0], 64)
			return scene.Spacing{Top: f, Right: f, Bottom: f, Left: f}
		case 2:
			vert, _ := strconv.ParseFloat(parts[0], 64)
			horiz, _ := strconv.ParseFloat(parts[1], 64)
			return scene.Spacing{Top: vert, Right: horiz, Bottom: vert, Left: horiz}
		case 3:
			// CSS 3-value: top horizontal bottom
			t, _ := strconv.ParseFloat(parts[0], 64)
			h, _ := strconv.ParseFloat(parts[1], 64)
			bot, _ := strconv.ParseFloat(parts[2], 64)
			return scene.Spacing{Top: t, Right: h, Bottom: bot, Left: h}
		case 4:
			t, _ := strconv.ParseFloat(parts[0], 64)
			r, _ := strconv.ParseFloat(parts[1], 64)
			b, _ := strconv.ParseFloat(parts[2], 64)
			l, _ := strconv.ParseFloat(parts[3], 64)
			return scene.Spacing{Top: t, Right: r, Bottom: b, Left: l}
		}
	}
	return scene.Spacing{}
}

// ParseBorder parses '1.5 solid #e2e8f0' → Border.
func ParseBorder(value interface{}) scene.Border {
	switch v := value.(type) {
	case map[string]interface{}:
		return scene.Border{
			Width: getFloat(v, "width", 1),
			Style: getString(v, "style", "solid"),
			Color: getString(v, "color", "#000"),
		}
	default:
		s := fmt.Sprintf("%v", value)
		parts := strings.Fields(s)
		switch {
		case len(parts) >= 3:
			w, _ := strconv.ParseFloat(parts[0], 64)
			return scene.Border{Width: w, Style: parts[1], Color: parts[2]}
		case len(parts) == 2:
			w, _ := strconv.ParseFloat(parts[0], 64)
			return scene.Border{Width: w, Style: "solid", Color: parts[1]}
		default:
			return scene.Border{Width: 1, Style: "solid", Color: s}
		}
	}
}

// ParseShadow parses '0 2 8 rgba(0,0,0,0.1)' or '0 2 8 4 rgba(0,0,0,0.1)' or multiple separated by |.
// Supports: x y blur [spread] [color]
func ParseShadow(value interface{}) []scene.Shadow {
	s, ok := value.(string)
	if !ok {
		return nil
	}
	var shadows []scene.Shadow
	for _, part := range strings.Split(s, "|") {
		tokens := strings.SplitN(strings.TrimSpace(part), " ", 5)
		if len(tokens) >= 5 {
			// x y blur spread color
			x, _ := strconv.ParseFloat(tokens[0], 64)
			y, _ := strconv.ParseFloat(tokens[1], 64)
			blur, _ := strconv.ParseFloat(tokens[2], 64)
			spread, _ := strconv.ParseFloat(tokens[3], 64)
			shadows = append(shadows, scene.Shadow{X: x, Y: y, Blur: blur, Spread: spread, Color: tokens[4]})
		} else if len(tokens) == 4 {
			x, _ := strconv.ParseFloat(tokens[0], 64)
			y, _ := strconv.ParseFloat(tokens[1], 64)
			blur, _ := strconv.ParseFloat(tokens[2], 64)
			// Check if 4th token is numeric (spread without color) or a color
			if _, err := strconv.ParseFloat(tokens[3], 64); err == nil {
				spread, _ := strconv.ParseFloat(tokens[3], 64)
				shadows = append(shadows, scene.Shadow{X: x, Y: y, Blur: blur, Spread: spread, Color: "rgba(0,0,0,0.1)"})
			} else {
				shadows = append(shadows, scene.Shadow{X: x, Y: y, Blur: blur, Color: tokens[3]})
			}
		} else if len(tokens) == 3 {
			x, _ := strconv.ParseFloat(tokens[0], 64)
			y, _ := strconv.ParseFloat(tokens[1], 64)
			blur, _ := strconv.ParseFloat(tokens[2], 64)
			shadows = append(shadows, scene.Shadow{X: x, Y: y, Blur: blur, Color: "rgba(0,0,0,0.1)"})
		}
	}
	if len(shadows) == 0 {
		return nil
	}
	return shadows
}

// ParseRadius parses single number or '8 8 0 0'. Only supports uniform radius.
func ParseRadius(value interface{}) float64 {
	switch v := value.(type) {
	case int:
		return float64(v)
	case float64:
		return v
	default:
		s := fmt.Sprintf("%v", value)
		parts := strings.Fields(s)
		if len(parts) > 0 {
			f, _ := strconv.ParseFloat(parts[0], 64)
			return f
		}
	}
	return 0
}

// --- Node Parsers ---

func parseFrame(data map[string]interface{}, theme scene.Theme, components map[string]interface{}, warnings *[]string, depth int) *scene.FrameNode {
	node := scene.NewFrameNode()
	if components == nil {
		components = make(map[string]interface{})
	}

	// Two-pass: first set padding/margin, then let padding-x/y override
	if v, ok := data["padding"]; ok {
		node.Padding = ParseSpacing(v)
	}
	if v, ok := data["margin"]; ok {
		node.Margin = ParseSpacing(v)
	}

	for key, value := range data {
		if key == "children" || key == "padding" || key == "margin" {
			continue
		}
		applyFrameProp(node, key, value, warnings)
	}

	// Parse children
	if children, ok := data["children"].([]interface{}); ok {
		for _, childData := range children {
			childMap, ok := childData.(map[string]interface{})
			if !ok {
				continue
			}
			child := parseChild(childMap, theme, components, warnings, depth)
			if child != nil {
				node.Children = append(node.Children, child)
			}
		}
	}

	return node
}

func applyFrameProp(node *scene.FrameNode, key string, value interface{}, warnings *[]string) {
	switch key {
	case "id":
		node.ID = fmt.Sprintf("%v", value)
	case "width":
		if fmt.Sprintf("%v", value) == "auto" {
			node.Width = nil
		} else {
			f := toFloat(value)
			node.Width = &f
		}
	case "height":
		if fmt.Sprintf("%v", value) == "auto" {
			node.Height = nil
		} else {
			f := toFloat(value)
			node.Height = &f
		}
	case "min-width":
		f := toFloat(value)
		node.MinWidth = &f
	case "max-width":
		f := toFloat(value)
		node.MaxWidth = &f
	case "min-height":
		f := toFloat(value)
		node.MinHeight = &f
	case "max-height":
		f := toFloat(value)
		node.MaxHeight = &f
	case "flex":
		f := toFloat(value)
		node.Flex = &f
	case "x":
		f := toFloat(value)
		node.X = &f
	case "y":
		f := toFloat(value)
		node.Y = &f
	case "gap":
		node.Gap = toFloat(value)
	case "opacity":
		node.Opacity = toFloat(value)
	case "z-index":
		node.ZIndex = toInt(value)
	case "position":
		s := fmt.Sprintf("%v", value)
		node.Position = s
		warnInvalidEnum(warnings, "position", s, validPositions)
	case "direction":
		s := fmt.Sprintf("%v", value)
		node.Direction = s
		warnInvalidEnum(warnings, "direction", s, validDirections)
	case "align":
		s := fmt.Sprintf("%v", value)
		node.Align = s
		warnInvalidEnum(warnings, "align", s, validAligns)
	case "justify":
		s := fmt.Sprintf("%v", value)
		node.Justify = s
		warnInvalidEnum(warnings, "justify", s, validJustify)
	case "shape":
		s := fmt.Sprintf("%v", value)
		node.Shape = s
		warnInvalidEnum(warnings, "shape", s, validShapes)
	case "wrap":
		node.Wrap = toBool(value)
	case "layout":
		s := fmt.Sprintf("%v", value)
		node.LayoutMode = s
		warnInvalidEnum(warnings, "layout", s, validLayoutModes)
	case "columns":
		i := toInt(value)
		node.Columns = &i
	case "rows":
		i := toInt(value)
		node.Rows = &i
	case "column-gap":
		f := toFloat(value)
		node.ColumnGap = &f
	case "row-gap":
		f := toFloat(value)
		node.RowGap = &f
	case "padding-x":
		f := toFloat(value)
		node.Padding.Left = f
		node.Padding.Right = f
	case "padding-y":
		f := toFloat(value)
		node.Padding.Top = f
		node.Padding.Bottom = f
	case "margin-x":
		f := toFloat(value)
		node.Margin.Left = f
		node.Margin.Right = f
	case "margin-y":
		f := toFloat(value)
		node.Margin.Top = f
		node.Margin.Bottom = f
	case "fill":
		s := fmt.Sprintf("%v", value)
		if strings.Contains(strings.ToLower(s), "gradient") {
			node.Gradient = ParseGradient(s)
		} else {
			node.Fill = &s
		}
	case "gradient":
		node.Gradient = ParseGradient(value)
	case "radius":
		node.Radius = ParseRadius(value)
	case "border":
		b := ParseBorder(value)
		warnInvalidEnum(warnings, "border style", b.Style, validEdgeStyle)
		node.Border = &b
	case "border-top":
		b := ParseBorder(value)
		warnInvalidEnum(warnings, "border-top style", b.Style, validEdgeStyle)
		node.BorderTop = &b
	case "border-right":
		b := ParseBorder(value)
		warnInvalidEnum(warnings, "border-right style", b.Style, validEdgeStyle)
		node.BorderRight = &b
	case "border-bottom":
		b := ParseBorder(value)
		warnInvalidEnum(warnings, "border-bottom style", b.Style, validEdgeStyle)
		node.BorderBottom = &b
	case "border-left":
		b := ParseBorder(value)
		warnInvalidEnum(warnings, "border-left style", b.Style, validEdgeStyle)
		node.BorderLeft = &b
	case "shadow":
		node.Shadow = ParseShadow(value)
	case "clip":
		node.Clip = toBool(value)
	case "visible":
		node.Visible = toBool(value)
	case "image":
		s := fmt.Sprintf("%v", value)
		node.Image = &s
	case "image-fit":
		s := fmt.Sprintf("%v", value)
		node.ImageFit = s
		warnInvalidEnum(warnings, "image-fit", s, validImageFit)
	default:
		// Warn about unknown properties (skip meta-keys and component-related keys)
		switch key {
		case "frame", "text", "edge", "use", "variant", "children",
			"font", "color": // font/color used by component label system
		default:
			*warnings = append(*warnings, fmt.Sprintf("unknown frame property: %q", key))
		}
	}
}

func parseChild(data map[string]interface{}, theme scene.Theme, components map[string]interface{}, warnings *[]string, depth int) scene.Node {
	// Text node
	if _, ok := data["text"]; ok {
		return parseTextNode(data, theme, warnings)
	}

	// Edge node
	if _, ok := data["edge"]; ok {
		return parseEdge(data, warnings)
	}

	// Guard against infinite component recursion
	if depth >= maxComponentDepth {
		*warnings = append(*warnings, "maximum component nesting depth reached")
		return nil
	}

	// Component instance via `use:` key
	if useName, ok := data["use"].(string); ok {
		if compDef, ok := components[useName]; ok {
			if defMap, ok := compDef.(map[string]interface{}); ok {
				return parseComponentInstance(useName, data, defMap, theme, components, warnings, depth+1)
			}
		}
		*warnings = append(*warnings, fmt.Sprintf("unknown component: %s", useName))
	}

	// Component instance via component name as key
	for compName, compDef := range components {
		if _, ok := data[compName]; ok {
			if defMap, ok := compDef.(map[string]interface{}); ok {
				return parseComponentInstance(compName, data, defMap, theme, components, warnings, depth+1)
			}
		}
	}

	// Frame node (named or anonymous)
	if frameName, ok := data["frame"]; ok {
		props := make(map[string]interface{})
		for k, v := range data {
			if k != "frame" {
				props[k] = v
			}
		}
		// Only use frame name as id if no explicit id is set
		if name, ok := frameName.(string); ok && name != "" {
			if _, hasID := props["id"]; !hasID {
				props["id"] = name
			}
		}
		return parseFrame(props, theme, components, warnings, depth)
	}

	// Treat as anonymous frame
	return parseFrame(data, theme, components, warnings, depth)
}

func parseTextNode(data map[string]interface{}, theme scene.Theme, warnings *[]string) *scene.TextNode {
	node := scene.NewTextNode()
	node.Content = fmt.Sprintf("%v", data["text"])
	node.Color = theme.Foreground
	node.Font = scene.Font{
		Weight: theme.FontWeight,
		Size:   theme.FontSize,
		Family: theme.FontFamily,
	}

	for key, value := range data {
		switch key {
		case "text":
			continue
		case "font":
			node.Font = ParseFont(value)
		case "color":
			node.Color = fmt.Sprintf("%v", value)
		case "text-align":
			s := fmt.Sprintf("%v", value)
			node.TextAlign = s
			warnInvalidEnum(warnings, "text-align", s, validTextAlign)
		case "line-height":
			node.LineHeight = toFloat(value)
		case "max-width":
			f := toFloat(value)
			node.MaxWidth = &f
		case "letter-spacing":
			node.LetterSpacing = toFloat(value)
		case "text-decoration":
			s := fmt.Sprintf("%v", value)
			node.TextDecoration = s
			warnInvalidEnum(warnings, "text-decoration", s, validTextDecoration)
		case "truncate":
			node.Truncate = toBool(value)
		case "opacity":
			node.Opacity = toFloat(value)
		default:
			*warnings = append(*warnings, fmt.Sprintf("unknown text property: %q", key))
		}
	}
	return node
}

func parseEdge(data map[string]interface{}, warnings *[]string) *scene.EdgeNode {
	edge := scene.NewEdgeNode()
	// Handle nested edge properties
	if props, ok := data["edge"].(map[string]interface{}); ok {
		for k, v := range props {
			applyEdgeProp(edge, k, v, warnings)
		}
	}
	// Also handle flat properties
	for key, value := range data {
		if key == "edge" {
			continue
		}
		applyEdgeProp(edge, key, value, warnings)
	}
	return edge
}

func applyEdgeProp(edge *scene.EdgeNode, key string, value interface{}, warnings *[]string) {
	switch key {
	case "from":
		edge.FromID = fmt.Sprintf("%v", value)
	case "to":
		edge.ToID = fmt.Sprintf("%v", value)
	case "from-anchor":
		s := fmt.Sprintf("%v", value)
		edge.FromAnchor = s
		warnInvalidEnum(warnings, "from-anchor", s, validAnchor)
	case "to-anchor":
		s := fmt.Sprintf("%v", value)
		edge.ToAnchor = s
		warnInvalidEnum(warnings, "to-anchor", s, validAnchor)
	case "stroke":
		edge.Stroke = fmt.Sprintf("%v", value)
	case "stroke-width":
		edge.StrokeWidth = toFloat(value)
	case "style":
		s := fmt.Sprintf("%v", value)
		edge.Style = s
		warnInvalidEnum(warnings, "style", s, validEdgeStyle)
	case "arrow":
		s := fmt.Sprintf("%v", value)
		edge.Arrow = s
		warnInvalidEnum(warnings, "arrow", s, validArrow)
	case "label":
		s := fmt.Sprintf("%v", value)
		edge.Label = &s
	case "label-font":
		edge.LabelFont = ParseFont(value)
	case "label-color":
		edge.LabelColor = fmt.Sprintf("%v", value)
	case "label-position":
		edge.LabelPosition = toFloat(value)
	case "curve":
		s := fmt.Sprintf("%v", value)
		edge.Curve = s
		warnInvalidEnum(warnings, "curve", s, validCurve)
	case "corner-radius":
		f := toFloat(value)
		edge.CornerRadius = &f
	case "junction":
		f := toFloat(value)
		edge.Junction = &f
	default:
		if key != "edge" {
			*warnings = append(*warnings, fmt.Sprintf("unknown edge property: %q", key))
		}
	}
}

// --- Helpers ---

func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case int:
		return float64(val)
	case float64:
		return val
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		f, _ := strconv.ParseFloat(fmt.Sprintf("%v", v), 64)
		return f
	}
}

func toInt(v interface{}) int {
	switch val := v.(type) {
	case int:
		return val
	case float64:
		return int(val)
	case string:
		i, _ := strconv.Atoi(val)
		return i
	default:
		i, _ := strconv.Atoi(fmt.Sprintf("%v", v))
		return i
	}
}

func toBool(v interface{}) bool {
	switch val := v.(type) {
	case bool:
		return val
	case int:
		return val != 0
	case float64:
		return val != 0
	case string:
		return val == "true" || val == "1"
	default:
		return false
	}
}

func getString(m map[string]interface{}, key, def string) string {
	if v, ok := m[key]; ok {
		return fmt.Sprintf("%v", v)
	}
	return def
}

func getFloat(m map[string]interface{}, key string, def float64) float64 {
	if v, ok := m[key]; ok {
		return toFloat(v)
	}
	return def
}

func getInt(m map[string]interface{}, key string, def int) int {
	if v, ok := m[key]; ok {
		return toInt(v)
	}
	return def
}
