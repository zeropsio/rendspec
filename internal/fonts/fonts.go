// Package fonts provides font metrics for accurate text measurement.
//
// Pre-computed average character widths (as fraction of font-size) for 50+ common fonts.
// These are derived from actual glyph metrics and provide much better layout accuracy
// than a single universal ratio.
package fonts

import "strings"

// fontMetrics maps font family name to average character width ratio.
// Values are for weight 400; bold (+5%) adjustment applied separately.
var fontMetrics = map[string]float64{
	// Sans-serif
	"Inter":              0.52,
	"Helvetica":          0.52,
	"Helvetica Neue":     0.51,
	"Arial":              0.53,
	"Roboto":             0.51,
	"Open Sans":          0.53,
	"Lato":               0.51,
	"Poppins":            0.54,
	"Montserrat":         0.55,
	"Nunito":             0.53,
	"Nunito Sans":        0.52,
	"Source Sans Pro":    0.50,
	"Source Sans 3":      0.50,
	"DM Sans":            0.52,
	"Manrope":            0.54,
	"Plus Jakarta Sans":  0.53,
	"Work Sans":          0.52,
	"Outfit":             0.52,
	"Geist":              0.51,
	"SF Pro":             0.52,
	"SF Pro Display":     0.52,
	"SF Pro Text":        0.52,
	"-apple-system":      0.52,
	"BlinkMacSystemFont": 0.52,
	"Segoe UI":           0.52,
	"system-ui":          0.52,
	"sans-serif":         0.52,
	// Serif
	"Georgia":           0.55,
	"Times New Roman":   0.50,
	"Merriweather":      0.56,
	"Playfair Display":  0.54,
	"Lora":              0.52,
	"PT Serif":          0.52,
	"Noto Serif":        0.53,
	"serif":             0.52,
	// Monospace — all ~0.60 (fixed-width)
	"Mono":            0.60,
	"Courier":         0.60,
	"Courier New":     0.60,
	"Roboto Mono":     0.60,
	"JetBrains Mono":  0.60,
	"Fira Code":       0.60,
	"Fira Mono":       0.60,
	"Source Code Pro":  0.60,
	"IBM Plex Mono":   0.60,
	"SF Mono":         0.60,
	"Geist Mono":      0.60,
	"Cascadia Code":   0.60,
	"monospace":        0.60,
	"Menlo":           0.60,
	"Consolas":        0.60,
	"Monaco":          0.60,
	// Display / decorative
	"Impact":        0.46,
	"Comic Sans MS": 0.55,
}

// lowercaseMetrics is built at init for case-insensitive lookup.
var lowercaseMetrics map[string]float64

func init() {
	lowercaseMetrics = make(map[string]float64, len(fontMetrics))
	for name, ratio := range fontMetrics {
		lowercaseMetrics[strings.ToLower(name)] = ratio
	}
}

const defaultRatio = 0.52
const uppercaseMultiplier = 1.15

var narrowChars = map[rune]bool{
	'i': true, 'I': true, 'l': true, 'j': true, '1': true, '|': true,
	'!': true, '.': true, ',': true, ';': true, ':': true, '\'': true,
	'"': true, '(': true, ')': true, '[': true, ']': true, '{': true,
	'}': true, ' ': true, '\t': true,
}

var wideChars = map[rune]bool{
	'm': true, 'M': true, 'w': true, 'W': true, 'O': true, 'Q': true,
	'D': true, '@': true, '%': true,
}

// CharWidthRatio returns the average character width ratio for a font family.
func CharWidthRatio(family string) float64 {
	// Exact match
	if ratio, ok := fontMetrics[family]; ok {
		return ratio
	}
	// Case-insensitive match
	lower := strings.ToLower(family)
	if ratio, ok := lowercaseMetrics[lower]; ok {
		return ratio
	}
	// Check if it's a monospace font by name
	if strings.Contains(lower, "mono") || strings.Contains(lower, "code") ||
		strings.Contains(lower, "courier") || strings.Contains(lower, "consolas") {
		return 0.60
	}
	return defaultRatio
}

// MeasureTextWidth measures text width with per-character adjustments for better accuracy.
func MeasureTextWidth(text string, fontSize float64, family string, weight int, letterSpacing float64) float64 {
	baseRatio := CharWidthRatio(family)
	isMono := baseRatio >= 0.59

	runes := []rune(text)
	n := len(runes)

	if isMono {
		charW := fontSize * baseRatio
		if weight >= 600 {
			charW *= 1.05
		}
		charW += letterSpacing
		return float64(n) * charW
	}

	// Proportional font: adjust per character
	total := 0.0
	avgW := fontSize * baseRatio
	for _, ch := range runes {
		if narrowChars[ch] {
			total += avgW * 0.55
		} else if wideChars[ch] {
			total += avgW * 1.35
		} else if ch >= 'A' && ch <= 'Z' {
			total += avgW * uppercaseMultiplier
		} else if ch >= '0' && ch <= '9' {
			total += avgW * 1.0
		} else {
			total += avgW
		}
	}

	if weight >= 600 {
		total *= 1.05
	}
	total += letterSpacing * float64(n)
	return total
}
