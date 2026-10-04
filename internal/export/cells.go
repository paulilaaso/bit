// ABOUTME: Shared ANSI text parsing and cell geometry used by image exporters.
// ABOUTME: Both PNG and SVG backends consume forEachPaintedCell for identical output.

package export

import (
	"regexp"
	"strconv"
	"unicode"
	"unicode/utf8"
)

// CellSize defines the default pixel dimensions per terminal character cell.
// 4x scale per dimension = 16x16 pixels per cell.
const CellSize = 16

// Unicode block characters
const (
	FullBlock       = '█' // U+2588
	UpperHalfBlock  = '▀' // U+2580
	LowerHalfBlock  = '▄' // U+2584
	LightShade      = '░' // U+2591
	MediumShade     = '▒' // U+2592
	DarkShade       = '▓' // U+2593
)

// Brightness multipliers for shade characters (0.0 - 1.0)
const (
	LightShadeBrightness  = 0.25 // ~25%
	MediumShadeBrightness = 0.50 // ~50%
	DarkShadeBrightness   = 0.75 // ~75%
)

// DefaultForegroundColor is the fallback foreground color when no ANSI color is active.
var DefaultForegroundColor = [3]uint8{255, 255, 255}

// Regex patterns for parsing ANSI escape sequences
var (
	// Matches 24-bit foreground color: ESC[38;2;R;G;Bm
	colorRegex = regexp.MustCompile(`\x1b\[38;2;(\d+);(\d+);(\d+)m`)
	// Matches any ANSI escape sequence (for stripping)
	ansiStripRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)
)

// cellGeometry describes the painted span of one character cell in
// fractional cell units (0.0-1.0 relative to the cell's box).
// alpha is 1.0 for solid cells and the shade brightness for shade characters.
type cellGeometry struct {
	fracX, fracY float64 // Top-left of the span, relative to the cell
	fracW, fracH float64 // Size of the span, relative to the cell
	alpha        float64
}

// forEachPaintedCell walks one line of ANSI-colored text, invoking fn for
// every cell that paints content. col is the visible character index, r the
// character, rgb the active foreground color, and g the cell's geometry.
// Spaces, control characters, and ANSI escape sequences never invoke fn.
func forEachPaintedCell(line string, fn func(col int, r rune, rgb [3]uint8, g cellGeometry)) {
	currentColor := DefaultForegroundColor

	i := 0
	lineBytes := []byte(line)
	col := 0

	for i < len(lineBytes) {
		if lineBytes[i] == 0x1b && i+1 < len(lineBytes) && lineBytes[i+1] == '[' {
			seqStart := i
			i += 2 // Skip ESC [
			for i < len(lineBytes) && !isAnsiTerminator(lineBytes[i]) {
				i++
			}
			if i < len(lineBytes) {
				i++ // Include terminator
			}
			seq := string(lineBytes[seqStart:i])
			if matches := colorRegex.FindStringSubmatch(seq); matches != nil {
				r, _ := strconv.Atoi(matches[1])
				g, _ := strconv.Atoi(matches[2])
				b, _ := strconv.Atoi(matches[3])
				currentColor = [3]uint8{uint8(r), uint8(g), uint8(b)}
			}
			// Reset codes (\x1b[0m) are handled implicitly - color stays until changed
			continue
		}

		r, size := utf8.DecodeRune(lineBytes[i:])
		if r == utf8.RuneError && size == 1 {
			i++ // Invalid UTF-8, skip byte
			continue
		}

		if g, ok := cellGeometryFor(r); ok {
			fn(col, r, currentColor, g)
		}

		col++
		i += size
	}
}

// cellGeometryFor returns the painted span for a character, if it paints one.
func cellGeometryFor(r rune) (cellGeometry, bool) {
	switch r {
	case FullBlock:
		return cellGeometry{fracW: 1, fracH: 1, alpha: 1.0}, true
	case UpperHalfBlock:
		return cellGeometry{fracW: 1, fracH: 0.5, alpha: 1.0}, true
	case LowerHalfBlock:
		return cellGeometry{fracY: 0.5, fracW: 1, fracH: 0.5, alpha: 1.0}, true
	case LightShade:
		return cellGeometry{fracW: 1, fracH: 1, alpha: LightShadeBrightness}, true
	case MediumShade:
		return cellGeometry{fracW: 1, fracH: 1, alpha: MediumShadeBrightness}, true
	case DarkShade:
		return cellGeometry{fracW: 1, fracH: 1, alpha: DarkShadeBrightness}, true
	default:
		if unicode.IsPrint(r) && r != ' ' {
			return cellGeometry{fracW: 1, fracH: 1, alpha: 1.0}, true
		}
		return cellGeometry{}, false
	}
}

// countVisibleChars counts visible (non-ANSI) characters in a line
func countVisibleChars(line string) int {
	stripped := ansiStripRegex.ReplaceAllString(line, "")
	return utf8.RuneCountInString(stripped)
}

// isAnsiTerminator checks if a byte terminates an ANSI escape sequence
func isAnsiTerminator(b byte) bool {
	return (b >= 'A' && b <= 'Z') || (b >= 'a' && b <= 'z')
}
