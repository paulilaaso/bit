// ABOUTME: SVG generator that converts ANSI-colored text to scalable vector graphics.
// ABOUTME: Merges adjacent same-paint cells into horizontal runs, one path per paint.

package export

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// SVGOptions contains configuration for SVG generation
type SVGOptions struct {
	CellWidth  int // ViewBox units per character cell width
	CellHeight int // ViewBox units per character cell height
}

// DefaultSVGOptions returns default SVG generation options (square cells)
func DefaultSVGOptions() SVGOptions {
	return SVGOptions{
		CellWidth:  CellSize,
		CellHeight: CellSize,
	}
}

// TerminalAspectRatioSVGOptions returns SVG options matching terminal character
// aspect ratio (~2:1 height:width), consistent with the PNG exporter.
func TerminalAspectRatioSVGOptions() SVGOptions {
	return SVGOptions{
		CellWidth:  16,
		CellHeight: 32,
	}
}

// svgSegment describes the painted rectangle of a single cell
type svgSegment struct {
	x, y     int // Top-left in viewbox units
	w, h     int // Dimensions in viewbox units
	opacity  float64
	colorHex string
}

// paintKey identifies the fill applied to a segment
type paintKey struct {
	colorHex string
	opacity  float64
}

// GenerateSVG creates an SVG document from rendered ANSI lines.
// Returns the SVG as a string or an error.
func GenerateSVG(lines []string, options SVGOptions) (string, error) {
	if len(lines) == 0 {
		return "", fmt.Errorf("no content to export")
	}

	if options.CellWidth == 0 {
		options.CellWidth = CellSize
	}
	if options.CellHeight == 0 {
		options.CellHeight = CellSize
	}

	// Calculate dimensions based on max line width
	maxWidth := 0
	for _, line := range lines {
		lineWidth := countVisibleChars(line)
		if lineWidth > maxWidth {
			maxWidth = lineWidth
		}
	}
	if maxWidth == 0 {
		maxWidth = 1
	}

	imgWidth := maxWidth * options.CellWidth
	imgHeight := len(lines) * options.CellHeight

	// Collect segments grouped by paint
	paints := make(map[paintKey][]string)

	for lineIdx, line := range lines {
		segments := svgLineSegments(line, lineIdx, options)
		runs := mergeSvgRuns(segments)
		for _, run := range runs {
			key := paintKey{colorHex: run.colorHex, opacity: run.opacity}
			paints[key] = append(paints[key], svgRunPath(run))
		}
	}

	// Sort paint keys for deterministic output (color, then opacity)
	keys := make([]paintKey, 0, len(paints))
	for k := range paints {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].colorHex != keys[j].colorHex {
			return keys[i].colorHex < keys[j].colorHex
		}
		return keys[i].opacity < keys[j].opacity
	})

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("<svg xmlns=\"http://www.w3.org/2000/svg\" viewBox=\"0 0 %d %d\" width=\"%d\" height=\"%d\">\n",
		imgWidth, imgHeight, imgWidth, imgHeight))

	for _, key := range keys {
		paths := paints[key]
		sb.WriteString(fmt.Sprintf("  <path fill=\"%s\" fill-opacity=\"%s\" d=\"%s\"/>\n",
			key.colorHex, formatOpacity(key.opacity), strings.Join(paths, "")))
	}

	sb.WriteString("</svg>\n")
	return sb.String(), nil
}

// svgLineSegments walks a line of ANSI text and returns one segment per painted cell
func svgLineSegments(line string, lineIdx int, options SVGOptions) []svgSegment {
	var segments []svgSegment

	forEachPaintedCell(line, func(col int, r rune, rgb [3]uint8, g cellGeometry) {
		x := int((float64(col) + g.fracX) * float64(options.CellWidth))
		y := int(float64(lineIdx)*float64(options.CellHeight) + g.fracY*float64(options.CellHeight))
		w := int(g.fracW * float64(options.CellWidth))
		h := int(g.fracH * float64(options.CellHeight))
		segments = append(segments, svgSegment{
			x:        x,
			y:        y,
			w:        w,
			h:        h,
			opacity:  g.alpha,
			colorHex: fmt.Sprintf("#%02x%02x%02x", rgb[0], rgb[1], rgb[2]),
		})
	})

	return segments
}

// mergeSvgRuns merges horizontally adjacent segments with identical geometry and paint
func mergeSvgRuns(segments []svgSegment) []svgSegment {
	var runs []svgSegment

	for _, seg := range segments {
		if n := len(runs); n > 0 {
			prev := &runs[n-1]
			if seg.y == prev.y && seg.h == prev.h &&
				seg.opacity == prev.opacity && seg.colorHex == prev.colorHex &&
				seg.x == prev.x+prev.w {
				prev.w += seg.w
				continue
			}
		}
		runs = append(runs, seg)
	}

	return runs
}

// svgRunPath returns the path data for a run's rectangle, relative to previous subpath
func svgRunPath(run svgSegment) string {
	return fmt.Sprintf("M%d %dh%dv%dh-%dz", run.x, run.y, run.w, run.h, run.w)
}

// formatOpacity renders an opacity value compactly (e.g. 0.25, 1)
func formatOpacity(opacity float64) string {
	return strconv.FormatFloat(opacity, 'f', -1, 64)
}
