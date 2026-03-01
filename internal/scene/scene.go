// Package scene defines the core data structures for the rendspec scene graph.
package scene

// Font represents a text font specification.
type Font struct {
	Weight int     `json:"weight"`
	Size   float64 `json:"size"`
	Family string  `json:"family"`
}

// DefaultFont returns a Font with sensible defaults.
func DefaultFont() Font {
	return Font{Weight: 400, Size: 14, Family: "Inter"}
}

// Border represents a border specification.
type Border struct {
	Width float64 `json:"width"`
	Style string  `json:"style"`
	Color string  `json:"color"`
}

// Shadow represents a drop shadow.
type Shadow struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Blur   float64 `json:"blur"`
	Spread float64 `json:"spread"`
	Color  string  `json:"color"`
}

// Spacing represents padding or margin (CSS-like shorthand).
type Spacing struct {
	Top    float64 `json:"top"`
	Right  float64 `json:"right"`
	Bottom float64 `json:"bottom"`
	Left   float64 `json:"left"`
}

// Horizontal returns left + right spacing.
func (s Spacing) Horizontal() float64 { return s.Left + s.Right }

// Vertical returns top + bottom spacing.
func (s Spacing) Vertical() float64 { return s.Top + s.Bottom }

// GradientStop represents a color stop in a gradient.
type GradientStop struct {
	Color    string  `json:"color"`
	Position float64 `json:"position"`
}

// Gradient represents a linear or radial gradient.
type Gradient struct {
	Type  string         `json:"type"`  // "linear" or "radial"
	Angle float64        `json:"angle"` // degrees, linear only
	Stops []GradientStop `json:"stops"`
	CX    float64        `json:"cx"` // radial center x (0-1)
	CY    float64        `json:"cy"` // radial center y (0-1)
}

// ComputedLayout holds the absolute position and size after layout computation.
type ComputedLayout struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// Node is the interface implemented by all scene graph nodes.
type Node interface {
	GetLayout() *ComputedLayout
}

// Point represents a 2D coordinate.
type Point struct {
	X float64
	Y float64
}

// TextNode represents a text element.
type TextNode struct {
	Content        string         `json:"content"`
	Font           Font           `json:"font"`
	Color          string         `json:"color"`
	TextAlign      string         `json:"text_align"`
	LineHeight     float64        `json:"line_height"`
	MaxWidth       *float64       `json:"max_width"`
	LetterSpacing  float64        `json:"letter_spacing"`
	TextDecoration string         `json:"text_decoration"`
	Truncate       bool           `json:"truncate"`
	Opacity        float64        `json:"opacity"`
	Layout         ComputedLayout `json:"layout"`
}

// GetLayout returns the computed layout for this node.
func (t *TextNode) GetLayout() *ComputedLayout { return &t.Layout }

// NewTextNode creates a TextNode with sensible defaults.
func NewTextNode() *TextNode {
	return &TextNode{
		Font:           DefaultFont(),
		Color:          "#0f172a",
		TextAlign:      "left",
		LineHeight:     1.4,
		TextDecoration: "none",
		Opacity:        1.0,
	}
}

// FrameNode represents a container element (the core building block).
type FrameNode struct {
	ID string `json:"id,omitempty"`

	// Sizing - nil means auto-size
	Width     *float64 `json:"width,omitempty"`
	Height    *float64 `json:"height,omitempty"`
	MinWidth  *float64 `json:"min_width,omitempty"`
	MaxWidth  *float64 `json:"max_width,omitempty"`
	MinHeight *float64 `json:"min_height,omitempty"`
	MaxHeight *float64 `json:"max_height,omitempty"`
	Flex      *float64 `json:"flex,omitempty"`

	// Positioning
	Position string   `json:"position"`
	X        *float64 `json:"x,omitempty"`
	Y        *float64 `json:"y,omitempty"`
	ZIndex   int      `json:"z_index"`

	// Layout (for children)
	Direction string  `json:"direction"`
	Align     string  `json:"align"`
	Justify   string  `json:"justify"`
	Gap       float64 `json:"gap"`
	Wrap      bool    `json:"wrap"`

	// Grid layout
	LayoutMode string   `json:"layout_mode"`
	Columns    *int     `json:"columns,omitempty"`
	Rows       *int     `json:"rows,omitempty"`
	ColumnGap  *float64 `json:"column_gap,omitempty"`
	RowGap     *float64 `json:"row_gap,omitempty"`

	// Spacing
	Padding Spacing `json:"padding"`
	Margin  Spacing `json:"margin"`

	// Visual
	Fill        *string   `json:"fill,omitempty"`
	Gradient    *Gradient `json:"gradient,omitempty"`
	Opacity     float64   `json:"opacity"`
	Radius      float64   `json:"radius"`
	Border      *Border   `json:"border,omitempty"`
	BorderTop   *Border   `json:"border_top,omitempty"`
	BorderRight *Border   `json:"border_right,omitempty"`
	BorderBottom   *Border   `json:"border_bottom,omitempty"`
	BorderLeft  *Border   `json:"border_left,omitempty"`
	Shadow      []Shadow  `json:"shadow,omitempty"`
	Clip        bool      `json:"clip"`

	// Image
	Image    *string `json:"image,omitempty"`
	ImageFit string  `json:"image_fit"`

	// Shape
	Shape string `json:"shape"`

	// Visibility
	Visible bool `json:"visible"`

	// Component tracking
	ComponentName string `json:"component_name,omitempty"`

	// Children
	Children []Node `json:"-"`

	// Computed layout
	Layout ComputedLayout `json:"layout"`
}

// GetLayout returns the computed layout for this node.
func (f *FrameNode) GetLayout() *ComputedLayout { return &f.Layout }

// NewFrameNode creates a FrameNode with sensible defaults.
func NewFrameNode() *FrameNode {
	return &FrameNode{
		Position:   "relative",
		Direction:  "column",
		Align:      "stretch",
		Justify:    "start",
		LayoutMode: "flex",
		ImageFit:   "cover",
		Shape:      "rect",
		Visible:    true,
		Opacity:    1.0,
	}
}

// EdgeNode represents a connection between two frames.
type EdgeNode struct {
	FromID        string   `json:"from_id"`
	ToID          string   `json:"to_id"`
	FromAnchor    string   `json:"from_anchor"`
	ToAnchor      string   `json:"to_anchor"`
	Stroke        string   `json:"stroke"`
	StrokeWidth   float64  `json:"stroke_width"`
	Style         string   `json:"style"`
	Arrow         string   `json:"arrow"`
	Label         *string  `json:"label,omitempty"`
	LabelFont     Font     `json:"label_font"`
	LabelColor    string   `json:"label_color"`
	LabelPosition float64  `json:"label_position"`
	Curve         string   `json:"curve"`
	CornerRadius  *float64 `json:"corner_radius,omitempty"`
	Junction      *float64 `json:"junction,omitempty"`
	ResolvedPath  []Point  `json:"-"`
}

// GetLayout returns nil for edge nodes (they don't have layout).
func (e *EdgeNode) GetLayout() *ComputedLayout { return nil }

// NewEdgeNode creates an EdgeNode with sensible defaults.
func NewEdgeNode() *EdgeNode {
	return &EdgeNode{
		FromAnchor:    "auto",
		ToAnchor:      "auto",
		Stroke:        "#94a3b8",
		StrokeWidth:   1.5,
		Style:         "solid",
		Arrow:         "end",
		LabelFont:     DefaultFont(),
		LabelColor:    "#64748b",
		LabelPosition: 0.5,
		Curve:         "straight",
	}
}

// Theme holds visual defaults.
type Theme struct {
	Background string  `json:"background"`
	Foreground string  `json:"foreground"`
	Muted      string  `json:"muted"`
	Accent     string  `json:"accent"`
	Border     string  `json:"border"`
	Radius     float64 `json:"radius"`
	FontFamily string  `json:"font_family"`
	FontSize   float64 `json:"font_size"`
	FontWeight int     `json:"font_weight"`
}

// DefaultTheme returns the light theme.
func DefaultTheme() Theme {
	return Theme{
		Background: "#ffffff",
		Foreground: "#0f172a",
		Muted:      "#64748b",
		Accent:     "#2563eb",
		Border:     "#e2e8f0",
		Radius:     8,
		FontFamily: "Inter",
		FontSize:   14,
		FontWeight: 400,
	}
}

// BuiltinThemes maps theme name to Theme.
var BuiltinThemes = map[string]Theme{
	"light": DefaultTheme(),
	"dark": {
		Background: "#0f172a",
		Foreground: "#f8fafc",
		Muted:      "#94a3b8",
		Accent:     "#3b82f6",
		Border:     "#334155",
		Radius:     8,
		FontFamily: "Inter",
		FontSize:   14,
		FontWeight: 400,
	},
	"blueprint": {
		Background: "#1e3a5f",
		Foreground: "#ffffff",
		Muted:      "#7eb8da",
		Accent:     "#4fc3f7",
		Border:     "#2e5a8a",
		Radius:     8,
		FontFamily: "Mono",
		FontSize:   14,
		FontWeight: 400,
	},
	"sketch": {
		Background: "#fffef5",
		Foreground: "#2d2d2d",
		Muted:      "#888888",
		Accent:     "#e74c3c",
		Border:     "#cccccc",
		Radius:     8,
		FontFamily: "Inter",
		FontSize:   14,
		FontWeight: 400,
	},
}

// SceneGraph is the top-level container for a single-page design.
type SceneGraph struct {
	Root       *FrameNode             `json:"root"`
	Edges      []*EdgeNode            `json:"edges"`
	Theme      Theme                  `json:"theme"`
	Components map[string]interface{} `json:"-"`
	Tokens     map[string]interface{} `json:"-"`
	Warnings   []string               `json:"warnings,omitempty"`
}

// NewSceneGraph creates a SceneGraph with defaults.
func NewSceneGraph() *SceneGraph {
	return &SceneGraph{
		Root:       NewFrameNode(),
		Theme:      DefaultTheme(),
		Components: make(map[string]interface{}),
		Tokens:     make(map[string]interface{}),
	}
}

// Page represents a single page in a multi-page document.
type Page struct {
	Name  string      `json:"name"`
	Root  *FrameNode  `json:"root"`
	Edges []*EdgeNode `json:"edges"`
}

// Document represents a multi-page document.
type Document struct {
	Pages      []Page                 `json:"pages"`
	Theme      Theme                  `json:"theme"`
	Components map[string]interface{} `json:"-"`
	Tokens     map[string]interface{} `json:"-"`
	Warnings   []string               `json:"warnings,omitempty"`
}

// NewDocument creates a Document with defaults.
func NewDocument() *Document {
	return &Document{
		Theme:      DefaultTheme(),
		Components: make(map[string]interface{}),
		Tokens:     make(map[string]interface{}),
	}
}

// Ptr returns a pointer to v. Convenience helper for optional fields.
func Ptr[T any](v T) *T { return &v }
