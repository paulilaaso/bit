// ABOUTME: PNG generator that converts ANSI-colored text to PNG images.
// ABOUTME: Parses ANSI escape sequences and renders characters at 16x scale with transparency.

package export

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
)

// PNGOptions contains configuration for PNG generation
type PNGOptions struct {
	CellWidth  int // Pixels per character cell width (default: CellSize)
	CellHeight int // Pixels per character cell height (default: CellSize)
}

// DefaultPNGOptions returns default PNG generation options (16x16 per cell)
func DefaultPNGOptions() PNGOptions {
	return PNGOptions{
		CellWidth:  CellSize,
		CellHeight: CellSize,
	}
}

// TerminalAspectRatioPNGOptions returns PNG options that match terminal character
// aspect ratio. Terminal characters are typically ~2:1 (height:width), so each
// character cell is rendered twice as tall as it is wide.
func TerminalAspectRatioPNGOptions() PNGOptions {
	return PNGOptions{
		CellWidth:  16, // 16 pixels wide (4x scale)
		CellHeight: 32, // 32 pixels tall (2:1 height:width ratio)
	}
}

// GeneratePNG creates a PNG image from rendered ANSI lines.
// Returns PNG data as bytes or error.
func GeneratePNG(lines []string, options PNGOptions) ([]byte, error) {
	if len(lines) == 0 {
		return nil, fmt.Errorf("no content to export")
	}

	// Use defaults if zero values provided
	if options.CellWidth == 0 {
		options.CellWidth = CellSize
	}
	if options.CellHeight == 0 {
		options.CellHeight = CellSize
	}

	// Calculate image dimensions based on max line width
	maxWidth := 0
	for _, line := range lines {
		lineWidth := countVisibleChars(line)
		if lineWidth > maxWidth {
			maxWidth = lineWidth
		}
	}

	// Handle edge case of all-empty lines
	if maxWidth == 0 {
		maxWidth = 1
	}

	imgWidth := maxWidth * options.CellWidth
	imgHeight := len(lines) * options.CellHeight

	// Create RGBA image with transparent background (zero-initialized = transparent)
	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))

	// Render each line
	for lineIdx, line := range lines {
		renderLineToImage(img, line, lineIdx, options)
	}

	// Encode to PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode PNG: %v", err)
	}

	return buf.Bytes(), nil
}

// renderLineToImage renders a single line of ANSI text to the image.
// Shade alpha is resolved into the RGB value (opaque pixels), matching
// the shared cell semantics in cells.go.
func renderLineToImage(img *image.RGBA, line string, lineIdx int, options PNGOptions) {
	forEachPaintedCell(line, func(col int, r rune, rgb [3]uint8, g cellGeometry) {
		x := int((float64(col) + g.fracX) * float64(options.CellWidth))
		y := int(float64(lineIdx)*float64(options.CellHeight) + g.fracY*float64(options.CellHeight))
		w := int(g.fracW * float64(options.CellWidth))
		h := int(g.fracH * float64(options.CellHeight))
		c := color.RGBA{
			R: uint8(float64(rgb[0]) * g.alpha),
			G: uint8(float64(rgb[1]) * g.alpha),
			B: uint8(float64(rgb[2]) * g.alpha),
			A: 255,
		}
		fillRect(img, x, y, w, h, c)
	})
}

// fillRect fills a rectangle in the image with the given color
func fillRect(img *image.RGBA, x, y, width, height int, c color.RGBA) {
	bounds := img.Bounds()
	for dy := 0; dy < height; dy++ {
		for dx := 0; dx < width; dx++ {
			px, py := x+dx, y+dy
			if px >= bounds.Min.X && px < bounds.Max.X && py >= bounds.Min.Y && py < bounds.Max.Y {
				img.SetRGBA(px, py, c)
			}
		}
	}
}
