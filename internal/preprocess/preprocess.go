// Package preprocess runs text-level transformations before YAML parsing.
//
// Two passes:
// 1. Auto-quote multi-value properties (so users can write `padding: 12 24` instead of `padding: "12 24"`)
// 2. Implicit children (so users can omit `children:` keyword)
package preprocess

import (
	"regexp"
	"strings"
)

// Properties whose values need quoting when they contain spaces.
var autoquoteProps = []string{
	"padding", "margin", "font", "border",
	"border-top", "border-right", "border-bottom", "border-left",
	"shadow", "label-font", "gradient", "fill",
}

var propRE *regexp.Regexp

func init() {
	escaped := make([]string, len(autoquoteProps))
	for i, p := range autoquoteProps {
		escaped[i] = regexp.QuoteMeta(p)
	}
	pattern := `^(\s*(?:-\s+)?)(` + strings.Join(escaped, "|") + `):\s+(.+)$`
	propRE = regexp.MustCompile(pattern)
}

// autoQuote wraps unquoted multi-word values in double quotes.
func autoQuote(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		m := propRE.FindStringSubmatchIndex(line)
		if m == nil {
			continue
		}
		prefix := line[m[2]:m[3]]
		prop := line[m[4]:m[5]]
		value := strings.TrimRight(line[m[6]:m[7]], " \t")

		// Strip inline comments before quoting
		if idx := inlineCommentIdx(value); idx >= 0 {
			value = strings.TrimRight(value[:idx], " \t")
		}

		if !strings.Contains(value, " ") {
			continue
		}
		if (strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"")) ||
			(strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'")) {
			continue
		}

		// Choose quoting style to handle embedded quotes
		if strings.Contains(value, "\"") {
			if !strings.Contains(value, "'") {
				lines[i] = prefix + prop + ": '" + value + "'"
			} else {
				escaped := strings.ReplaceAll(value, "\"", "\\\"")
				lines[i] = prefix + prop + ": \"" + escaped + "\""
			}
		} else {
			lines[i] = prefix + prop + ": \"" + value + "\""
		}
	}
	return strings.Join(lines, "\n")
}

// inlineCommentIdx returns the index of ` # ` inline comment, or -1 if none.
// YAML inline comments require space before and after the hash: ` # comment`.
func inlineCommentIdx(s string) int {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case '#':
			// Must have space before AND after hash (or be at end)
			if depth == 0 && i > 0 && s[i-1] == ' ' && (i+1 >= len(s) || s[i+1] == ' ') {
				return i - 1
			}
		}
	}
	return -1
}

func getIndent(line string) int {
	count := 0
	for _, ch := range line {
		switch ch {
		case ' ':
			count++
		case '\t':
			count += 2 // treat tab as 2 spaces
		default:
			return count
		}
	}
	return count
}

func isBlankOrComment(stripped string) bool {
	return stripped == "" || strings.HasPrefix(stripped, "#")
}

// implicitChildrenOnce finds the first sequence item that needs children: insertion.
func implicitChildrenOnce(lines []string) ([]string, bool) {
	for i, line := range lines {
		stripped := strings.TrimLeft(line, " ")
		if isBlankOrComment(stripped) {
			continue
		}
		if !strings.HasPrefix(stripped, "- ") {
			continue
		}

		indent := getIndent(line)
		needsInsertion := false

		// Scan backward for the nearest sibling at the same indent
		j := i - 1
		for j >= 0 {
			prev := lines[j]
			prevStripped := strings.TrimLeft(prev, " ")
			if isBlankOrComment(prevStripped) {
				j--
				continue
			}
			prevIndent := getIndent(prev)
			if prevIndent < indent {
				break
			}
			if prevIndent == indent {
				if strings.HasPrefix(prevStripped, "- ") {
					break
				}
				needsInsertion = true
				break
			}
			j--
		}

		if !needsInsertion {
			continue
		}

		// Check for existing children: key at same indent in this block
		hasExistingChildren := false
		for k := j + 1; k < i; k++ {
			ks := strings.TrimLeft(lines[k], " \t")
			if isBlankOrComment(ks) {
				continue
			}
			ki := getIndent(lines[k])
			if ki == indent && ks == "children:" {
				hasExistingChildren = true
				break
			}
			if ki < indent {
				break
			}
		}
		if hasExistingChildren {
			continue
		}

		// Insert children: before line i, re-indent from i onward
		newLines := make([]string, 0, len(lines)+1)
		newLines = append(newLines, lines[:i]...)
		newLines = append(newLines, strings.Repeat(" ", indent)+"children:")

		k := i
		for k < len(lines) {
			childLine := lines[k]
			childStripped := strings.TrimLeft(childLine, " ")
			if isBlankOrComment(childStripped) {
				newLines = append(newLines, childLine)
				k++
				continue
			}
			childIndent := getIndent(childLine)
			if childIndent < indent {
				break
			}
			newLines = append(newLines, "  "+childLine)
			k++
		}
		newLines = append(newLines, lines[k:]...)
		return newLines, true
	}

	return lines, false
}

// implicitChildren inserts children: keywords iteratively until stable.
func implicitChildren(text string) string {
	lines := strings.Split(text, "\n")
	for range 50 {
		var changed bool
		lines, changed = implicitChildrenOnce(lines)
		if !changed {
			break
		}
	}
	return strings.Join(lines, "\n")
}

// Preprocess runs all preprocessing passes on raw DSL text.
func Preprocess(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	text = expandTabs(text)
	text = autoQuote(text)
	text = implicitChildren(text)
	return text
}

// expandTabs replaces leading tabs with 2 spaces each.
func expandTabs(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if !strings.ContainsRune(line, '\t') {
			continue
		}
		var sb strings.Builder
		inLeading := true
		for _, ch := range line {
			if inLeading && ch == '\t' {
				sb.WriteString("  ")
			} else {
				inLeading = false
				sb.WriteRune(ch)
			}
		}
		lines[i] = sb.String()
	}
	return strings.Join(lines, "\n")
}
