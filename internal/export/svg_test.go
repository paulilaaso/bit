// ABOUTME: Tests for SVG generation from ANSI-colored text output.
// ABOUTME: Verifies correct parsing of ANSI codes, run merging, and path output.

package export

import (
	"strings"
	"testing"
)

func TestGenerateSVG_EmptyInput(t *testing.T) {
	_, err := GenerateSVG([]string{}, DefaultSVGOptions())
	if err == nil {
		t.Error("expected error for empty input, got nil")
	}
}

func TestGenerateSVG_NilInput(t *testing.T) {
	_, err := GenerateSVG(nil, DefaultSVGOptions())
	if err == nil {
		t.Error("expected error for nil input, got nil")
	}
}

func TestGenerateSVG_OnlySpaces(t *testing.T) {
	svg, err := GenerateSVG([]string{"   "}, DefaultSVGOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(svg, "<svg") || !strings.Contains(svg, "</svg>") {
		t.Error("expected SVG document structure")
	}
	if strings.Contains(svg, "<path") {
		t.Error("expected no painted paths for spaces-only input")
	}
}

func TestGenerateSVG_ViewBoxDimensions(t *testing.T) {
	// Two lines of three characters each
	lines := []string{"███", "███"}
	svg, err := GenerateSVG(lines, TerminalAspectRatioSVGOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 3 cells wide (16), 2 cells tall (32) in terminal aspect
	if !strings.Contains(svg, `viewBox="0 0 48 64"`) {
		t.Errorf("expected viewBox 0 0 48 64, got: %s", firstLine(svg))
	}
}

func TestGenerateSVG_ColorParsing(t *testing.T) {
	lines := []string{"\x1b[38;2;255;0;0m█\x1b[0m"}
	svg, err := GenerateSVG(lines, DefaultSVGOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(svg, `fill="#ff0000"`) {
		t.Errorf("expected red fill, got: %s", svg)
	}
}

func TestGenerateSVG_RunMerging(t *testing.T) {
	// Three same-color adjacent cells should merge into a single run of width 48
	lines := []string{"\x1b[38;2;255;0;0m███\x1b[0m"}
	svg, err := GenerateSVG(lines, DefaultSVGOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// "M" starts each subpath; counting them counts painted runs.
	if strings.Count(svg, "M") != 1 {
		t.Errorf("expected 1 merged run, got %d: %s", strings.Count(svg, "M"), svg)
	}
	if !strings.Contains(svg, "h48") {
		t.Errorf("expected run width 48, got: %s", svg)
	}
}

func TestGenerateSVG_NoMergingAcrossColors(t *testing.T) {
	lines := []string{
		"\x1b[38;2;255;0;0m█\x1b[38;2;0;0;255m█\x1b[0m",
	}
	svg, err := GenerateSVG(lines, DefaultSVGOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// "M" starts each subpath; counting them counts painted runs.
	if strings.Count(svg, "M") != 2 {
		t.Errorf("expected 2 separate runs for different colors, got %d: %s", strings.Count(svg, "M"), svg)
	}
	if !strings.Contains(svg, `fill="#ff0000"`) || !strings.Contains(svg, `fill="#0000ff"`) {
		t.Errorf("expected both red and blue fills, got: %s", svg)
	}
}

func TestGenerateSVG_HalfBlocks(t *testing.T) {
	lines := []string{"\x1b[38;2;0;255;0m▀▄\x1b[0m"}
	svg, err := GenerateSVG(lines, DefaultSVGOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Two segments, each half cell height (8)
	// "M" starts each subpath; counting them counts painted runs.
	if strings.Count(svg, "M") != 2 {
		t.Errorf("expected 2 segments for half blocks, got %d: %s", strings.Count(svg, "M"), svg)
	}
	if !strings.Contains(svg, "v8") {
		t.Errorf("expected half-height (8) segments, got: %s", svg)
	}
}

func TestGenerateSVG_ShadeOpacity(t *testing.T) {
	lines := []string{
		"\x1b[38;2;255;255;255m░\x1b[0m",
	}
	svg, err := GenerateSVG(lines, DefaultSVGOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(svg, `fill-opacity="0.25"`) {
		t.Errorf("expected fill-opacity 0.25 for light shade, got: %s", svg)
	}
}

func TestGenerateSVG_NonBlockCharFillsCell(t *testing.T) {
	lines := []string{"A"}
	svg, err := GenerateSVG(lines, DefaultSVGOptions())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(svg, "h16") || !strings.Contains(svg, "v16") {
		t.Errorf("expected full cell for printable char, got: %s", svg)
	}
}

func firstLine(s string) string {
	if idx := strings.Index(s, "\n"); idx >= 0 {
		return s[:idx]
	}
	return s
}
