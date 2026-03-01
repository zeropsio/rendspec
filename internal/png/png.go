// Package png converts SVG strings to PNG bytes using resvg via WASM.
// No CGo or external binaries required.
package png

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	resvg "github.com/kanrichan/resvg-go"
)

var (
	initOnce  sync.Once
	resvgCtx  *resvg.Context
	initErr   error
	fontData  [][]byte
	fontsOnce sync.Once
)

func getContext() (*resvg.Context, error) {
	initOnce.Do(func() {
		resvgCtx, initErr = resvg.NewContext(context.Background())
	})
	return resvgCtx, initErr
}

// collectSystemFonts reads font files from system directories into memory.
// The WASM resvg module can't access the host filesystem directly,
// so we read the bytes ourselves and pass them via LoadFontData.
func collectSystemFonts() [][]byte {
	fontsOnce.Do(func() {
		var dirs []string
		switch runtime.GOOS {
		case "darwin":
			dirs = []string{
				"/System/Library/Fonts",
				"/Library/Fonts",
			}
			if home, _ := os.UserHomeDir(); home != "" {
				dirs = append(dirs, filepath.Join(home, "Library/Fonts"))
			}
		case "linux":
			dirs = []string{
				"/usr/share/fonts",
				"/usr/local/share/fonts",
			}
			if home, _ := os.UserHomeDir(); home != "" {
				dirs = append(dirs, filepath.Join(home, ".fonts"))
				dirs = append(dirs, filepath.Join(home, ".local/share/fonts"))
			}
		case "windows":
			dirs = []string{`C:\Windows\Fonts`}
			if home := os.Getenv("USERPROFILE"); home != "" {
				dirs = append(dirs, filepath.Join(home, `AppData\Local\Microsoft\Windows\Fonts`))
			}
		}

		for _, dir := range dirs {
			filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil // skip errors
				}
				if d.IsDir() {
					return nil
				}
				ext := strings.ToLower(filepath.Ext(path))
				if ext == ".ttf" || ext == ".otf" || ext == ".ttc" {
					if data, err := os.ReadFile(path); err == nil {
						fontData = append(fontData, data)
					}
				}
				return nil
			})
		}
	})
	return fontData
}

// RenderPNG converts an SVG string to PNG bytes.
// Scale multiplies the native SVG dimensions (e.g. 2 = 2x resolution).
func RenderPNG(svg string, scale float64) ([]byte, error) {
	ctx, err := getContext()
	if err != nil {
		return nil, fmt.Errorf("resvg init: %w", err)
	}

	renderer, err := ctx.NewRenderer()
	if err != nil {
		return nil, fmt.Errorf("resvg renderer: %w", err)
	}
	defer renderer.Close()

	// Load system fonts by reading files ourselves and passing data to WASM
	for _, data := range collectSystemFonts() {
		renderer.LoadFontData(data)
	}

	if scale > 1 {
		w, h := parseSVGDimensions(svg)
		if w > 0 && h > 0 {
			return renderer.RenderWithSize([]byte(svg), uint32(float64(w)*scale), uint32(float64(h)*scale))
		}
	}

	return renderer.Render([]byte(svg))
}

// parseSVGDimensions extracts width and height from an SVG string's root element.
func parseSVGDimensions(svg string) (int, int) {
	w := extractAttr(svg, "width")
	h := extractAttr(svg, "height")
	return w, h
}

func extractAttr(svg, attr string) int {
	needle := attr + "=\""
	idx := 0
	for idx < len(svg) {
		pos := strings.Index(svg[idx:], needle)
		if pos < 0 {
			return 0
		}
		idx += pos + len(needle)
		end := strings.IndexByte(svg[idx:], '"')
		if end < 0 {
			return 0
		}
		val := svg[idx : idx+end]
		var n int
		for _, c := range val {
			if c >= '0' && c <= '9' {
				n = n*10 + int(c-'0')
			} else {
				break
			}
		}
		if n > 0 {
			return n
		}
	}
	return 0
}
