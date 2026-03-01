package parser

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/fxck/rendspec/internal/scene"
)

var gradientRE = regexp.MustCompile(`(?i)(linear|radial)-gradient\(\s*(.+)\s*\)`)

// ParseGradient parses CSS-like gradient syntax or dict form.
func ParseGradient(value interface{}) *scene.Gradient {
	switch v := value.(type) {
	case map[string]interface{}:
		return parseGradientDict(v)
	case string:
		return parseGradientString(v)
	default:
		return nil
	}
}

func parseGradientDict(v map[string]interface{}) *scene.Gradient {
	g := &scene.Gradient{
		Type:  getString(v, "type", "linear"),
		Angle: getFloat(v, "angle", 0),
		CX:    getFloat(v, "cx", 0.5),
		CY:    getFloat(v, "cy", 0.5),
	}
	if stops, ok := v["stops"].([]interface{}); ok {
		for _, s := range stops {
			if sd, ok := s.(map[string]interface{}); ok {
				g.Stops = append(g.Stops, scene.GradientStop{
					Color:    getString(sd, "color", "#000"),
					Position: getFloat(sd, "position", 0),
				})
			}
		}
	}
	return g
}

func parseGradientString(s string) *scene.Gradient {
	m := gradientRE.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return nil
	}

	gradType := strings.ToLower(m[1])
	inner := strings.TrimSpace(m[2])

	g := &scene.Gradient{Type: gradType, CX: 0.5, CY: 0.5}

	parts := splitGradientArgs(inner)
	if len(parts) == 0 {
		return nil
	}

	first := strings.TrimSpace(parts[0])
	stopStart := 0

	if gradType == "linear" {
		if angle, ok := parseAngle(first); ok {
			g.Angle = angle
			stopStart = 1
		}
	} else if gradType == "radial" {
		if first == "circle" || first == "ellipse" || first == "closest-side" || first == "farthest-side" {
			stopStart = 1
		}
		if strings.Contains(first, "at") {
			atParts := strings.SplitN(first, "at", 2)
			if len(atParts) > 1 {
				coords := strings.Fields(strings.TrimSpace(atParts[1]))
				if len(coords) >= 2 {
					g.CX = parsePercentage(coords[0])
					g.CY = parsePercentage(coords[1])
				}
			}
			stopStart = 1
		}
	}

	// Parse color stops
	stops := parts[stopStart:]
	n := len(stops)
	for i, stopStr := range stops {
		color, pos := parseColorStop(strings.TrimSpace(stopStr), i, n)
		g.Stops = append(g.Stops, scene.GradientStop{Color: color, Position: pos})
	}

	return g
}

// splitGradientArgs splits gradient arguments by commas, respecting parentheses.
func splitGradientArgs(s string) []string {
	var parts []string
	depth := 0
	var current strings.Builder
	for _, ch := range s {
		switch ch {
		case '(':
			depth++
			current.WriteRune(ch)
		case ')':
			depth--
			current.WriteRune(ch)
		case ',':
			if depth == 0 {
				parts = append(parts, current.String())
				current.Reset()
			} else {
				current.WriteRune(ch)
			}
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// parseAngle parses gradient angle. '135deg', 'to right', etc.
func parseAngle(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "deg") {
		v, err := strconv.ParseFloat(s[:len(s)-3], 64)
		if err != nil {
			return 0, false
		}
		return v, true
	}
	directions := map[string]float64{
		"to top":          0,
		"to right":        90,
		"to bottom":       180,
		"to left":         270,
		"to top right":    45,
		"to top left":     315,
		"to bottom right": 135,
		"to bottom left":  225,
	}
	if v, ok := directions[s]; ok {
		return v, true
	}
	return 0, false
}

// parsePercentage parses a percentage string to 0-1 float.
func parsePercentage(s string) float64 {
	s = strings.TrimSpace(s)
	if strings.HasSuffix(s, "%") {
		v, err := strconv.ParseFloat(s[:len(s)-1], 64)
		if err != nil {
			return 0.5
		}
		return v / 100
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0.5
	}
	if v <= 1 {
		return v
	}
	return v / 100
}

// parseColorStop parses a color stop like '#ff0000 50%' → (color, position).
func parseColorStop(s string, index, total int) (string, float64) {
	s = strings.TrimSpace(s)
	// Check for trailing percentage
	parts := strings.Fields(s)
	if len(parts) == 2 {
		trailing := parts[len(parts)-1]
		if strings.HasSuffix(trailing, "%") || isNumeric(trailing) {
			return parts[0], parsePercentage(trailing)
		}
	}
	// No explicit position — distribute evenly
	if total > 1 {
		return s, float64(index) / float64(total-1)
	}
	return s, 0
}

func isNumeric(s string) bool {
	_, err := strconv.ParseFloat(s, 64)
	return err == nil
}
