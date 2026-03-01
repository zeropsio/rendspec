package preprocess

import (
	"strings"
	"testing"
)

func TestAutoQuote_PaddingWithSpaces(t *testing.T) {
	result := autoQuote("  padding: 12 24")
	if result != `  padding: "12 24"` {
		t.Errorf("got %q", result)
	}
}

func TestAutoQuote_FontShorthand(t *testing.T) {
	result := autoQuote("  font: 700 20 Inter")
	if result != `  font: "700 20 Inter"` {
		t.Errorf("got %q", result)
	}
}

func TestAutoQuote_Border(t *testing.T) {
	result := autoQuote("  border: 1 solid #e2e8f0")
	if result != `  border: "1 solid #e2e8f0"` {
		t.Errorf("got %q", result)
	}
}

func TestAutoQuote_Shadow(t *testing.T) {
	result := autoQuote("  shadow: 0 4 16 rgba(0,0,0,0.08)")
	if result != `  shadow: "0 4 16 rgba(0,0,0,0.08)"` {
		t.Errorf("got %q", result)
	}
}

func TestAutoQuote_PreservesAlreadyQuoted(t *testing.T) {
	result := autoQuote(`  padding: "12 24"`)
	if result != `  padding: "12 24"` {
		t.Errorf("got %q", result)
	}
}

func TestAutoQuote_PreservesSingleQuoted(t *testing.T) {
	result := autoQuote("  padding: '12 24'")
	if result != "  padding: '12 24'" {
		t.Errorf("got %q", result)
	}
}

func TestAutoQuote_SingleValueUnchanged(t *testing.T) {
	result := autoQuote("  padding: 12")
	if result != "  padding: 12" {
		t.Errorf("got %q", result)
	}
}

func TestAutoQuote_NonTargetPropUnchanged(t *testing.T) {
	result := autoQuote("  width: 200 300")
	if result != "  width: 200 300" {
		t.Errorf("got %q", result)
	}
}

func TestAutoQuote_ListPrefix(t *testing.T) {
	result := autoQuote("  - font: 700 20 Inter")
	if result != `  - font: "700 20 Inter"` {
		t.Errorf("got %q", result)
	}
}

func TestAutoQuote_BorderVariants(t *testing.T) {
	result := autoQuote("  border-top: 1.5 dashed #ccc")
	if result != `  border-top: "1.5 dashed #ccc"` {
		t.Errorf("got %q", result)
	}
}

func TestAutoQuote_GradientFill(t *testing.T) {
	result := autoQuote("  fill: linear-gradient(135deg, #667eea, #764ba2)")
	if result != `  fill: "linear-gradient(135deg, #667eea, #764ba2)"` {
		t.Errorf("got %q", result)
	}
}

func TestImplicitChildren_Basic(t *testing.T) {
	text := "- frame: card\n  fill: white\n  - text: Hello\n"
	result := implicitChildren(text)
	if !strings.Contains(result, "children:") {
		t.Error("expected children: to be inserted")
	}
}

func TestImplicitChildren_NoChangeWhenAlreadyHas(t *testing.T) {
	text := "- frame: card\n  fill: white\n  children:\n    - text: Hello\n"
	result := implicitChildren(text)
	if strings.Count(result, "children:") != 1 {
		t.Error("should not add duplicate children:")
	}
}

func TestImplicitChildren_MultipleNested(t *testing.T) {
	text := "- frame: outer\n  fill: white\n  - frame: inner\n    fill: gray\n    - text: Deep\n"
	result := implicitChildren(text)
	if strings.Count(result, "children:") != 2 {
		t.Errorf("expected 2 children: keywords, got %d", strings.Count(result, "children:"))
	}
}

func TestImplicitChildren_SiblingsNotAffected(t *testing.T) {
	text := "- frame: a\n  fill: red\n- frame: b\n  fill: blue\n"
	result := implicitChildren(text)
	if strings.Contains(result, "children:") {
		t.Error("sibling sequence items should not get children:")
	}
}

func TestImplicitChildren_DeeplyNested(t *testing.T) {
	// Each level needs a property so implicitChildren can detect the child list
	text := "- frame: l1\n  fill: white\n  - frame: l2\n    fill: gray\n    - frame: l3\n      fill: red\n      - frame: l4\n        fill: blue\n        - frame: l5\n          fill: green\n          - text: Deep\n"
	result := implicitChildren(text)
	if strings.Count(result, "children:") != 5 {
		t.Errorf("expected 5 children: keywords, got %d in:\n%s", strings.Count(result, "children:"), result)
	}
}

func TestImplicitChildren_Empty(t *testing.T) {
	result := implicitChildren("")
	if result != "" {
		t.Errorf("empty input should return empty, got %q", result)
	}
}

func TestAutoQuote_Empty(t *testing.T) {
	result := autoQuote("")
	if result != "" {
		t.Errorf("empty input should return empty, got %q", result)
	}
}

func TestPreprocess_EmptyInput(t *testing.T) {
	result := Preprocess("")
	if result != "" {
		t.Errorf("empty input should return empty, got %q", result)
	}
}

func TestPreprocess_OnlyComments(t *testing.T) {
	result := Preprocess("# This is a comment\n# Another comment\n")
	// Should not crash and should preserve comments
	if !strings.Contains(result, "# This is a comment") {
		t.Error("should preserve comments")
	}
}

func TestPreprocess_MixedPropertyTypes(t *testing.T) {
	text := "- frame: card\n  fill: white\n  padding: 12 24\n  width: 200\n  font: 700 20 Inter\n  height: 100\n  border: 1 solid #ccc\n"
	result := Preprocess(text)
	if !strings.Contains(result, `"12 24"`) {
		t.Error("padding should be quoted")
	}
	if !strings.Contains(result, `"700 20 Inter"`) {
		t.Error("font should be quoted")
	}
	if !strings.Contains(result, `"1 solid #ccc"`) {
		t.Error("border should be quoted")
	}
	// width and height should NOT be quoted
	if strings.Contains(result, `"200"`) || strings.Contains(result, `"100"`) {
		t.Error("width/height should not be quoted")
	}
}

func TestPreprocess_FullPipeline(t *testing.T) {
	text := "- frame: card\n  fill: white\n  padding: 12 24\n  - text: Hello\n    font: 700 20 Inter\n"
	result := Preprocess(text)
	if !strings.Contains(result, `"12 24"`) {
		t.Error("expected padding to be quoted")
	}
	if !strings.Contains(result, `"700 20 Inter"`) {
		t.Error("expected font to be quoted")
	}
	if !strings.Contains(result, "children:") {
		t.Error("expected children: to be inserted")
	}
}

func TestPreprocess_CRLF(t *testing.T) {
	result := Preprocess("root:\r\n  width: 100\r\n  height: 200\r\n")
	if strings.Contains(result, "\r") {
		t.Error("should normalize CRLF to LF")
	}
	if !strings.Contains(result, "width: 100") {
		t.Error("content should be preserved")
	}
}

func TestPreprocess_CROnly(t *testing.T) {
	result := Preprocess("root:\r  width: 100\r")
	if strings.Contains(result, "\r") {
		t.Error("should normalize CR to LF")
	}
}

func TestPreprocess_TabIndentation(t *testing.T) {
	result := Preprocess("- frame: card\n\tfill: white\n\t\tpadding: 12\n")
	if strings.Contains(result, "\t") {
		t.Error("tabs should be expanded to spaces")
	}
	// Tab should become 2 spaces
	if !strings.Contains(result, "  fill: white") {
		t.Errorf("tab should expand to 2 spaces, got:\n%s", result)
	}
}

func TestAutoQuote_InlineComment(t *testing.T) {
	result := autoQuote("  padding: 12 24 # some comment")
	if result != `  padding: "12 24"` {
		t.Errorf("should strip inline comment before quoting, got %q", result)
	}
}

func TestAutoQuote_ColorNotComment(t *testing.T) {
	// #e2e8f0 is a color, not a comment — no space after #
	result := autoQuote("  border: 1 solid #e2e8f0")
	if result != `  border: "1 solid #e2e8f0"` {
		t.Errorf("should preserve color value, got %q", result)
	}
}

func TestAutoQuote_EmbeddedDoubleQuotes(t *testing.T) {
	// Value contains double quotes — should use single quotes
	result := autoQuote(`  font: 700 20 "Fira Code"`)
	if !strings.HasPrefix(result, "  font: '") {
		t.Errorf("should use single quotes when value contains double quotes, got %q", result)
	}
	if !strings.Contains(result, `"Fira Code"`) {
		t.Errorf("should preserve inner double quotes, got %q", result)
	}
}

func TestPreprocess_DuplicateChildren(t *testing.T) {
	// Already has children: — should not insert a second one
	text := "- frame: card\n  fill: white\n  children:\n    - text: Hello\n"
	result := Preprocess(text)
	if strings.Count(result, "children:") != 1 {
		t.Errorf("should not add duplicate children:, got %d occurrences", strings.Count(result, "children:"))
	}
}

func TestInlineCommentIdx(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"hello # comment", 5},
		{"hello #nocomment", -1},       // no space after #
		{"#notcomment", -1},             // no space before #
		{"rgba(0,0,0,0.08) # note", 16},
		{"no comment here", -1},
		{"color: #e2e8f0", -1}, // color hex, no space after #
	}
	for _, tt := range tests {
		got := inlineCommentIdx(tt.input)
		if got != tt.want {
			t.Errorf("inlineCommentIdx(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestAutoQuote_BorderWithHexColors(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"  border: 1 solid #e0dfdb", `  border: "1 solid #e0dfdb"`},
		{"  border: 1 solid #e2e8f0", `  border: "1 solid #e2e8f0"`},
		{"  border: 1 solid #abc", `  border: "1 solid #abc"`},
		{"  border: 1 solid #000", `  border: "1 solid #000"`},
		{"border: 1 solid #e0dfdb", `border: "1 solid #e0dfdb"`},
	}
	for _, tt := range tests {
		got := autoQuote(tt.input)
		if got != tt.expected {
			t.Errorf("autoQuote(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}
