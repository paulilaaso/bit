package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
	"github.com/paulilaaso/bit/internal/export"
)

// ansiRegex is compiled once at package level for efficiency
var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// stripANSI removes ANSI escape sequences from text
func stripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// renderTitleView renders the title bar at the top
func (m model) renderTitleView() string {
	var title string

	if m.export.showConfirmation {
		title = titleStyle.Render(m.export.confirmationText)
	} else if m.shadow.showWarning {
		title = warningStyle.Render("⚠ Shadow not available with half-pixels. Scale up the text.")
	} else {
		titleText := "Bit"
		if m.textInput.currentText != "" {
			cleanText := strings.ReplaceAll(m.textInput.currentText, "\n", " ")
			cleanText = strings.Join(strings.Fields(cleanText), " ")
			titleText += " (" + cleanText + ")"
		}
		title = titleStyle.Render(titleText)
	}

	return lipgloss.NewStyle().
		Width(m.uiState.width).
		Align(lipgloss.Center).
		Render(title)
}

// renderControlsView renders the help text at the bottom
func (m model) renderControlsView() string {
	controls := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorGray)).
		Align(lipgloss.Center).
		Render("←→: Panels • ↑↓: Adjust • Tab: Modes • r: Random • e: Export • Esc: Quit")

	return lipgloss.NewStyle().
		Width(m.uiState.width).
		Align(lipgloss.Center).
		Render(controls)
}

// renderTextDisplayView renders the main text display area
func (m model) renderTextDisplayView(mainDisplayHeight int) string {
	renderedText := strings.Join(m.uiState.renderedLines, "\n")
	if renderedText == "" {
		renderedText = "Enter some text to see the rendered output"
	}

	// Apply text alignment within the viewport
	alignedText := m.applyTextViewport(renderedText, m.uiState.width-4)

	// Clip the text vertically to fit within the allocated height
	adjustedTextHeight := mainDisplayHeight
	maxTextLines := max(adjustedTextHeight-1, 1)
	clippedText := m.clipTextVertically(alignedText, maxTextLines)

	fixedTextDisplayStyle := createFixedTextDisplayStyle(m.uiState.width-2, adjustedTextHeight-1)
	return fixedTextDisplayStyle.Render(clippedText)
}

// renderControlPanelsView renders all control panels
func (m model) renderControlPanelsView() string {
	panelWidth, contentWidth, spacerWidth, _, _ := m.calculateLayoutParameters()

	// Create labels
	labelWidth := panelWidth + 1
	textInputLabel := m.createTextInputLabel(labelWidth)
	fontLabel := m.createFontLabel(labelWidth)
	spacingLabel := m.createSpacingLabel(labelWidth)
	colorLabel := m.createColorLabel(labelWidth)
	scaleLabel := m.createScaleLabel(labelWidth)
	shadowLabel := m.createShadowLabel(labelWidth)

	// Create panel contents
	textContent, fontContent, spacingContent, colorContent, scaleContent, shadowContent := m.createPanelContents(contentWidth)

	// Create styled panels
	textPanel, fontPanel, spacingPanel, colorPanel, scalePanel, shadowPanel := m.createStyledPanels(
		panelWidth, textContent, fontContent, spacingContent, colorContent, scaleContent, shadowContent)

	// Create labeled panels
	labeledTextPanel := lipgloss.JoinVertical(lipgloss.Left, textInputLabel, textPanel)
	labeledFontPanel := lipgloss.JoinVertical(lipgloss.Left, fontLabel, fontPanel)
	labeledSpacingPanel := lipgloss.JoinVertical(lipgloss.Left, spacingLabel, spacingPanel)
	labeledColorPanel := lipgloss.JoinVertical(lipgloss.Left, colorLabel, colorPanel)
	labeledScalePanel := lipgloss.JoinVertical(lipgloss.Left, scaleLabel, scalePanel)
	labeledShadowPanel := lipgloss.JoinVertical(lipgloss.Left, shadowLabel, shadowPanel)

	// Arrange control panels
	return m.arrangeControlPanels(spacerWidth, labeledTextPanel, labeledFontPanel,
		labeledSpacingPanel, labeledColorPanel, labeledScalePanel, labeledShadowPanel)
}

// createTextInputLabel creates the label for the text input panel
func (m model) createTextInputLabel(labelWidth int) string {
	labelStyles := createLabelStyles()

	var labelText string
	if m.textInput.mode == TextEntryMode {
		if m.uiState.focusedPanel == TextInputPanel && m.textInput.input.Focused() {
			nonEmptyRows := countNonEmptyRows(m.textInput.textRows)
			if nonEmptyRows > 1 {
				labelText = fmt.Sprintf("Text Input (Row %d/%d)", m.textInput.currentRow+1, nonEmptyRows)
			} else {
				labelText = "Text Input"
			}
			if kerning := m.textInput.customKerning[m.textInput.currentRow][m.textInput.input.Position()]; kerning != 0 {
				labelText = fmt.Sprintf("%s %+d", labelText, kerning)
			}
		} else {
			labelText = "Text Input"
		}
	} else if m.textInput.mode == TextKerningMode {
		labelText = "Text Kerning"
		if m.uiState.focusedPanel == TextInputPanel && m.textInput.input.Focused() {
			if kerning := m.textInput.customKerning[m.textInput.currentRow][m.textInput.input.Position()]; kerning != 0 {
				labelText = fmt.Sprintf("%s %+d", labelText, kerning)
			}
		}
	} else {
		labelText = "Text Alignment"
	}

	return labelStyles.TextInput.Render(truncateText(labelText, labelWidth))
}

// createFontLabel creates the label for the font panel
func (m model) createFontLabel(labelWidth int) string {
	labelStyles := createLabelStyles()

	var labelText string
	if len(m.font.fonts) > 0 {
		labelText = fmt.Sprintf("Font %d/%d", m.font.selectedFont+1, len(m.font.fonts))
	} else {
		labelText = "Font"
	}

	return labelStyles.Font.Render(truncateText(labelText, labelWidth))
}

// createSpacingLabel creates the label for the spacing panel
func (m model) createSpacingLabel(labelWidth int) string {
	labelStyles := createLabelStyles()

	var labelText string
	var style lipgloss.Style

	switch m.spacing.mode {
	case CharacterSpacingMode:
		labelText = "Character Spacing"
		style = labelStyles.CharSpacing
	case WordSpacingMode:
		labelText = "Word Spacing"
		style = labelStyles.WordSpacing
	case LineSpacingMode:
		labelText = "Line Spacing"
		style = labelStyles.LineSpacing
	default:
		labelText = "Character Spacing"
		style = labelStyles.CharSpacing
	}

	return style.Render(truncateText(labelText, labelWidth))
}

// createColorLabel creates the label for the color panel
func (m model) createColorLabel(labelWidth int) string {
	labelStyles := createLabelStyles()

	var labelText string
	switch m.color.subMode {
	case TextColorMode:
		labelText = "Text Color 1"
	case GradientColorMode:
		labelText = "Text Color 2"
	case GradientDirectionMode:
		labelText = "Gradient ↔/↕"
	default:
		labelText = "Text Color 1"
	}

	return labelStyles.Color.Render(truncateText(labelText, labelWidth))
}

// createScaleLabel creates the label for the scale panel
func (m model) createScaleLabel(labelWidth int) string {
	labelStyles := createLabelStyles()
	return labelStyles.Scale.Render(truncateText("Text Scale", labelWidth))
}

// createShadowLabel creates the label for the shadow panel
func (m model) createShadowLabel(labelWidth int) string {
	var labelText string
	switch m.shadow.subMode {
	case HorizontalShadowMode:
		labelText = "Shadow ↔"
	case VerticalShadowMode:
		labelText = "Shadow ↕"
	case ShadowStyleMode:
		labelText = "Shadow Style"
	default:
		labelText = "Shadow ↔"
	}

	return lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPalette["Shadow"])).
		Bold(true).
		Render(truncateText(labelText, labelWidth))
}

// renderExportView renders the export UI when in export mode
func (m model) renderExportView() string {
	// Show overwrite prompt if needed
	if m.export.showOverwritePrompt {
		return m.renderOverwritePrompt()
	}

	title := titleStyle.Render(fmt.Sprintf("Export ANSI as %s", m.getFormatDescription(m.export.format)))

	formatLabel := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorExport)).
		Bold(true).
		Render("Format:")

	// Format selection
	var formatOptions []string

	selectedFormatStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(ColorExport)).
		Foreground(lipgloss.Color(ColorWhite)).
		Bold(true).
		Padding(0, 1)

	normalFormatStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorFaint)).
		Padding(0, 1)

	// Get format names from export manager
	formatNames := m.export.manager.GetFormatNames()

	for i, format := range formatNames {
		if format == m.export.format {
			formatOptions = append(formatOptions, selectedFormatStyle.Render(format))
		} else {
			formatOptions = append(formatOptions, normalFormatStyle.Render(format))
		}
		if i < len(formatNames)-1 {
			formatOptions = append(formatOptions, "  ")
		}
	}
	formatSelection := strings.Join(formatOptions, "")

	filenameLabel := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorExport)).
		Bold(true).
		Render("Filename:")

	filenameInput := m.export.filenameInput.View()

	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorFaint)).
		Render("Tab: Select format, Write filename and press ENTER to export, ESC to cancel")

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "current directory"
	}

	filename := m.export.filenameInput.Value()
	var fullPath string
	if filename != "" {
		sanitized := export.SanitizeFilename(filename)
		if sanitized != "" {
			fullPath = filepath.Join(cwd, sanitized+m.getFormatExtension(m.export.format))
		} else {
			fullPath = fmt.Sprintf("%s/", cwd)
		}
	} else {
		fullPath = fmt.Sprintf("%s/", cwd)
	}

	directoryInfo := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPalette["Shadow"])).
		Render(fmt.Sprintf("Directory: %s", fullPath))

	exportContent := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		formatLabel,
		formatSelection,
		"",
		filenameLabel,
		filenameInput,
		"",
		directoryInfo,
		"",
		instructions,
	)

	return lipgloss.NewStyle().
		Width(m.uiState.width).
		Height(m.uiState.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(exportContent)
}

// renderOverwritePrompt renders the overwrite confirmation dialog
func (m model) renderOverwritePrompt() string {
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorPalette["TextInput"])).
		Bold(true).
		Render("⚠ File Already Exists")

	message := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorWhite)).
		Render(fmt.Sprintf("The file '%s' already exists.", m.export.overwriteFilename))

	question := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorWhite)).
		Render("Do you want to overwrite it?")

	// Button styles
	selectedButtonStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(ColorExport)).
		Foreground(lipgloss.Color(ColorWhite)).
		Bold(true).
		Padding(0, 3).
		MarginLeft(1).
		MarginRight(1)

	normalButtonStyle := lipgloss.NewStyle().
		Background(lipgloss.Color(ColorFaint)).
		Foreground(lipgloss.Color(ColorWhite)).
		Padding(0, 3).
		MarginLeft(1).
		MarginRight(1)

	// Render buttons
	var yesButton, noButton string
	if m.export.selectedButton == 0 {
		yesButton = selectedButtonStyle.Render("Yes")
		noButton = normalButtonStyle.Render("No")
	} else {
		yesButton = normalButtonStyle.Render("Yes")
		noButton = selectedButtonStyle.Render("No")
	}

	buttons := lipgloss.JoinHorizontal(lipgloss.Center, yesButton, noButton)

	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color(ColorFaint)).
		Render("←→: Select • Enter: Confirm • Esc: Cancel")

	promptContent := lipgloss.JoinVertical(lipgloss.Center,
		title,
		"",
		message,
		question,
		"",
		buttons,
		"",
		instructions,
	)

	return lipgloss.NewStyle().
		Width(m.uiState.width).
		Height(m.uiState.height).
		Align(lipgloss.Center, lipgloss.Center).
		Render(promptContent)
}

// applyTextViewport applies text alignment within the terminal viewport
func (m *model) applyTextViewport(text string, maxWidth int) string {
	if text == "" {
		return text
	}

	lines := strings.Split(text, "\n")
	var alignedLines []string

	// Find the maximum width of the text block
	textBlockWidth := 0
	for _, line := range lines {
		displayLine := stripANSI(line)
		displayWidth := utf8.RuneCountInString(displayLine)
		if displayWidth > textBlockWidth {
			textBlockWidth = displayWidth
		}
	}

	// If text is wider than viewport, clip it and center the clipped portion
	if textBlockWidth > maxWidth {
		for _, line := range lines {
			clippedLine := m.clipLineToMiddle(line, maxWidth)
			alignedLines = append(alignedLines, clippedLine)
		}
		return strings.Join(alignedLines, "\n")
	}

	// Text fits within viewport - apply alignment
	for _, line := range lines {
		displayLine := stripANSI(line)
		lineWidth := utf8.RuneCountInString(displayLine)

		switch m.textInput.alignment {
		case LeftAlignment:
			padding := maxWidth - lineWidth
			if padding > 0 {
				// Use styled padding to preserve ANSI codes
				alignedLines = append(alignedLines, line+m.createStyledPadding(padding))
			} else {
				alignedLines = append(alignedLines, line)
			}
		case CenterAlignment:
			if lineWidth < maxWidth {
				leftPadding := (maxWidth - lineWidth) / 2
				rightPadding := maxWidth - lineWidth - leftPadding
				// Use styled padding to preserve ANSI codes
				alignedLines = append(alignedLines, m.createStyledPadding(leftPadding)+line+m.createStyledPadding(rightPadding))
			} else {
				alignedLines = append(alignedLines, line)
			}
		case RightAlignment:
			padding := maxWidth - lineWidth
			if padding > 0 {
				// Use styled padding to preserve ANSI codes
				alignedLines = append(alignedLines, m.createStyledPadding(padding)+line)
			} else {
				alignedLines = append(alignedLines, line)
			}
		default:
			alignedLines = append(alignedLines, line)
		}
	}

	return strings.Join(alignedLines, "\n")
}

// clipLineToMiddle clips a line to show the middle portion when it's too wide
func (m *model) clipLineToMiddle(line string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}

	displayLine := stripANSI(line)
	displayWidth := utf8.RuneCountInString(displayLine)

	if displayWidth <= maxWidth {
		return line
	}

	startPos := (displayWidth - maxWidth) / 2
	endPos := startPos + maxWidth

	return m.extractStyledSubstring(line, startPos, endPos)
}

// extractStyledSubstring extracts a substring from a styled line while preserving ANSI codes
// This function handles ANSI escape sequences (color codes) to ensure they are preserved
// when clipping text to fit within the viewport.
//
// Parameters:
//   - styledLine: The input string with ANSI escape sequences
//   - startPos: The starting position (in visible characters, not bytes)
//   - endPos: The ending position (in visible characters, not bytes)
//
// Returns: A substring with ANSI codes preserved for correct coloring
//
// Edge cases handled:
//   - Empty strings
//   - ANSI codes at boundaries
//   - Multi-byte Unicode characters
//   - Malformed ANSI sequences
func (m *model) extractStyledSubstring(styledLine string, startPos, endPos int) string {
	// Handle empty string
	if styledLine == "" {
		return ""
	}

	// Normalize positions
	if startPos < 0 {
		startPos = 0
	}
	if endPos < startPos {
		return ""
	}

	var result strings.Builder
	var currentPos int  // Current visible character position
	var inAnsiCode bool // Whether we're inside an ANSI escape sequence
	var ansiBuffer strings.Builder

	runes := []rune(styledLine)
	i := 0

	// Track active ANSI codes to ensure they're properly closed
	var activeAnsiCodes []string

	for i < len(runes) {
		r := runes[i]

		// Detect start of ANSI escape sequence: ESC[
		if r == '\x1b' && i+1 < len(runes) && runes[i+1] == '[' {
			inAnsiCode = true
			ansiBuffer.Reset()
			ansiBuffer.WriteRune(r)
			i++
			continue
		}

		// Process characters within ANSI escape sequence
		if inAnsiCode {
			ansiBuffer.WriteRune(r)

			// Prevent infinite loop by limiting ANSI sequence length
			// Using a more reasonable limit based on actual ANSI sequences (maximum valid SGR is ~50 chars)
			if ansiBuffer.Len() > 100 {
				// Malformed sequence, treat as regular character
				inAnsiCode = false
				// Process the ESC character as a regular character
				if currentPos >= startPos && currentPos < endPos {
					result.WriteString(ansiBuffer.String())
				}
				currentPos++
				i++
				continue
			}

			// ANSI sequences end with a letter (A-Z, a-z) or with certain special characters
			if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || r == '@' || r == ']' || r == '^' || r == '_' || r == '\\' {
				inAnsiCode = false
				ansiCode := ansiBuffer.String()

				// Track active ANSI codes
				if strings.Contains(ansiCode, "[0m") {
					// Reset code, clear active codes
					activeAnsiCodes = []string{}
				} else if r != '@' { // Don't store the reset code itself
					// Store the ANSI code for potential reapplication
					activeAnsiCodes = append(activeAnsiCodes, ansiCode)
				}

				// Include ANSI code if we're within the visible range
				if currentPos >= startPos && currentPos < endPos {
					result.WriteString(ansiCode)
				}
			}
			i++
			continue
		}

		// Process visible characters
		if currentPos >= startPos && currentPos < endPos {
			result.WriteRune(r)
		}

		// Only increment position for visible characters
		if !inAnsiCode {
			currentPos++
		}
		i++

		// Early exit if we've reached the end position
		if currentPos >= endPos {
			break
		}
	}

	// Ensure we don't exceed the end position
	// If we stopped due to reaching endPos, make sure we have proper ANSI reset
	if currentPos >= endPos && len(activeAnsiCodes) > 0 {
		// Add reset code to prevent color bleeding
		result.WriteString("\x1b[0m")
	}

	return result.String()
}

// clipTextVertically clips the text to fit within the specified number of lines
func (m *model) clipTextVertically(text string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}

	lines := strings.Split(text, "\n")
	if len(lines) <= maxLines {
		return text
	}

	clippedLines := lines[:maxLines]
	return strings.Join(clippedLines, "\n")
}

// countNonEmptyRows counts non-empty rows in text rows
func countNonEmptyRows(rows []string) int {
	count := 0
	for _, row := range rows {
		if strings.TrimSpace(row) != "" {
			count++
		}
	}
	return count
}

// createStyledPadding creates padding that preserves the last ANSI color code
func (m *model) createStyledPadding(length int) string {
	if length <= 0 {
		return ""
	}

	// For simplicity, we'll use a space character with no special styling
	// In a more complex implementation, we might track the last color used
	return strings.Repeat(" ", length)
}
