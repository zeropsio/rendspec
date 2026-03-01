package fonts

import (
	"math"
	"testing"
)

func TestCharWidthRatio_KnownFont(t *testing.T) {
	if r := CharWidthRatio("Inter"); r != 0.52 {
		t.Errorf("expected 0.52, got %f", r)
	}
}

func TestCharWidthRatio_MonospaceFont(t *testing.T) {
	if r := CharWidthRatio("JetBrains Mono"); r != 0.60 {
		t.Errorf("expected 0.60, got %f", r)
	}
}

func TestCharWidthRatio_UnknownFallback(t *testing.T) {
	if r := CharWidthRatio("CustomFont"); r != 0.52 {
		t.Errorf("expected 0.52 (default), got %f", r)
	}
}

func TestCharWidthRatio_MonoDetectionByName(t *testing.T) {
	if r := CharWidthRatio("MyCustomMono"); r != 0.60 {
		t.Errorf("expected 0.60 (mono), got %f", r)
	}
}

func TestCharWidthRatio_CaseInsensitive(t *testing.T) {
	tests := []string{"inter", "INTER", "Inter"}
	for _, name := range tests {
		if r := CharWidthRatio(name); r != 0.52 {
			t.Errorf("CharWidthRatio(%q) = %f, want 0.52", name, r)
		}
	}
}

func TestMeasureTextWidth_Basic(t *testing.T) {
	w := MeasureTextWidth("Hello", 14, "Inter", 400, 0)
	if w <= 0 {
		t.Errorf("expected positive width, got %f", w)
	}
}

func TestMeasureTextWidth_Empty(t *testing.T) {
	w := MeasureTextWidth("", 14, "Inter", 400, 0)
	if w != 0 {
		t.Errorf("expected 0 for empty string, got %f", w)
	}
}

func TestMeasureTextWidth_LongerTextWider(t *testing.T) {
	w1 := MeasureTextWidth("Hi", 14, "Inter", 400, 0)
	w2 := MeasureTextWidth("Hello World", 14, "Inter", 400, 0)
	if w2 <= w1 {
		t.Errorf("longer text should be wider: %f <= %f", w2, w1)
	}
}

func TestMeasureTextWidth_LargerFontWider(t *testing.T) {
	w1 := MeasureTextWidth("Hello", 12, "Inter", 400, 0)
	w2 := MeasureTextWidth("Hello", 24, "Inter", 400, 0)
	if w2 <= w1 {
		t.Errorf("larger font should be wider: %f <= %f", w2, w1)
	}
}

func TestMeasureTextWidth_BoldWider(t *testing.T) {
	w1 := MeasureTextWidth("Hello", 14, "Inter", 400, 0)
	w2 := MeasureTextWidth("Hello", 14, "Inter", 700, 0)
	if w2 <= w1 {
		t.Errorf("bold should be wider: %f <= %f", w2, w1)
	}
}

func TestMeasureTextWidth_LetterSpacing(t *testing.T) {
	w1 := MeasureTextWidth("Hello", 14, "Inter", 400, 0)
	w2 := MeasureTextWidth("Hello", 14, "Inter", 400, 2)
	if w2 <= w1 {
		t.Errorf("letter-spacing should increase width: %f <= %f", w2, w1)
	}
}

func TestMeasureTextWidth_MonospaceUniform(t *testing.T) {
	wi := MeasureTextWidth("iiiii", 14, "Mono", 400, 0)
	wm := MeasureTextWidth("MMMMM", 14, "Mono", 400, 0)
	if math.Abs(wi-wm) > 0.01 {
		t.Errorf("monospace should be uniform: i=%f, M=%f", wi, wm)
	}
}

func TestMeasureTextWidth_ProportionalNarrowChars(t *testing.T) {
	wn := MeasureTextWidth("illll", 14, "Inter", 400, 0)
	ww := MeasureTextWidth("MWMWM", 14, "Inter", 400, 0)
	if wn >= ww {
		t.Errorf("narrow chars should be narrower: %f >= %f", wn, ww)
	}
}
