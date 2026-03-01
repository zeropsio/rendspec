package parser

import (
	"fmt"
	"regexp"
	"strings"
)

var tokenRE = regexp.MustCompile(`\$([a-zA-Z_][a-zA-Z0-9_.]*)`)

// resolveTokens recursively resolves $token.path references in data.
// Use $$ to escape a literal dollar sign. Unresolved token references are appended as warnings.
func resolveTokens(data interface{}, tokens map[string]interface{}, warnings *[]string) interface{} {
	switch v := data.(type) {
	case string:
		// Handle $$ escape: temporarily replace with sentinel
		hasEscape := strings.Contains(v, "$$")
		s := v
		if hasEscape {
			s = strings.ReplaceAll(s, "$$", "\x00DLRESC\x00")
		}

		// If the entire string is a single token reference, return the raw value
		if m := tokenRE.FindStringIndex(s); m != nil && m[0] == 0 && m[1] == len(s) {
			path := s[1:] // strip $
			if val, ok := lookupToken(path, tokens); ok {
				return val
			}
			*warnings = append(*warnings, fmt.Sprintf("unresolved token: %s", v))
			result := s
			if hasEscape {
				result = strings.ReplaceAll(result, "\x00DLRESC\x00", "$")
			}
			return result
		}
		// Replace all token references in the string
		result := tokenRE.ReplaceAllStringFunc(s, func(match string) string {
			path := match[1:] // strip $
			if val, ok := lookupToken(path, tokens); ok {
				return fmt.Sprintf("%v", val)
			}
			*warnings = append(*warnings, fmt.Sprintf("unresolved token: %s", match))
			return match // leave unresolved tokens as-is
		})
		if hasEscape {
			result = strings.ReplaceAll(result, "\x00DLRESC\x00", "$")
		}
		return result

	case map[string]interface{}:
		result := make(map[string]interface{}, len(v))
		for k, val := range v {
			result[k] = resolveTokens(val, tokens, warnings)
		}
		return result

	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = resolveTokens(item, tokens, warnings)
		}
		return result

	default:
		return data
	}
}

// lookupToken looks up a dotted path in the tokens dict.
// E.g., "color.primary" → tokens["color"]["primary"]
func lookupToken(path string, tokens map[string]interface{}) (interface{}, bool) {
	parts := strings.Split(path, ".")
	var current interface{} = tokens
	for _, part := range parts {
		m, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current, ok = m[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}
