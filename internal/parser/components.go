package parser

import (
	"fmt"
	"strings"

	"github.com/zeropsio/rendspec/internal/scene"
)

// parseComponentInstance instantiates a component with optional variant and parameters.
func parseComponentInstance(
	compName string,
	data map[string]interface{},
	compDef map[string]interface{},
	theme scene.Theme,
	components map[string]interface{},
	warnings *[]string,
	depth int,
) *scene.FrameNode {
	// Deep copy via JSON round-trip
	merged := deepCopy(compDef)

	// Extract and remove variants/params from the merged definition
	variants := extractMap(merged, "variants")
	params := extractMap(merged, "params")

	// Apply variant if specified
	if variantName, ok := data["variant"].(string); ok && variantName != "" {
		if variantDef, ok := variants[variantName].(map[string]interface{}); ok {
			for k, v := range variantDef {
				merged[k] = v
			}
		}
	}

	// Collect parameter values from the instance
	paramValues := make(map[string]string)
	for paramName, paramSpec := range params {
		if val, ok := data[paramName]; ok {
			paramValues[paramName] = fmt.Sprintf("%v", val)
		} else if specMap, ok := paramSpec.(map[string]interface{}); ok {
			if def, ok := specMap["default"]; ok {
				paramValues[paramName] = fmt.Sprintf("%v", def)
			}
		}
	}

	// Apply instance overrides (skip component-specific keys)
	skipKeys := map[string]bool{
		compName:  true,
		"use":     true,
		"variant": true,
	}
	for k := range params {
		skipKeys[k] = true
	}
	for k, v := range data {
		if skipKeys[k] {
			continue
		}
		merged[k] = v
	}

	// Interpolate {{param}} placeholders
	if len(paramValues) > 0 {
		merged = interpolateParams(merged, paramValues).(map[string]interface{})
	}

	// The component name value may be text content (label)
	if label, ok := data[compName].(string); ok && label != "" {
		children, _ := merged["children"].([]interface{})
		if children == nil {
			children = []interface{}{}
		}
		textProps := map[string]interface{}{"text": label}
		// Move font and color from component to the label text
		if f, ok := merged["font"]; ok {
			textProps["font"] = f
			delete(merged, "font")
		}
		if c, ok := merged["color"]; ok {
			textProps["color"] = c
			delete(merged, "color")
		}
		// Prepend text child
		children = append([]interface{}{textProps}, children...)
		merged["children"] = children
	}

	node := parseFrame(merged, theme, components, warnings, depth)
	node.ComponentName = compName
	return node
}

// deepCopy creates a deep copy of a map by recursively copying all values.
func deepCopy(src map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{}, len(src))
	for k, v := range src {
		result[k] = deepCopyValue(v)
	}
	return result
}

// deepCopyValue recursively copies a value. Primitives (string, int, float64,
// bool, nil) are immutable and returned as-is.
func deepCopyValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return deepCopy(val)
	case []interface{}:
		cp := make([]interface{}, len(val))
		for i, item := range val {
			cp[i] = deepCopyValue(item)
		}
		return cp
	default:
		return v
	}
}

// extractMap removes a key from a map and returns it as map[string]interface{}.
func extractMap(m map[string]interface{}, key string) map[string]interface{} {
	v, ok := m[key]
	if !ok {
		return nil
	}
	delete(m, key)
	if result, ok := v.(map[string]interface{}); ok {
		return result
	}
	return nil
}

// interpolateParams recursively replaces {{param}} placeholders in strings.
func interpolateParams(data interface{}, params map[string]string) interface{} {
	switch v := data.(type) {
	case string:
		for name, value := range params {
			v = strings.ReplaceAll(v, "{{"+name+"}}", value)
		}
		return v
	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for k, val := range v {
			result[k] = interpolateParams(val, params)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = interpolateParams(item, params)
		}
		return result
	default:
		return data
	}
}
