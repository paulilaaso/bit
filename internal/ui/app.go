// ABOUTME: Main TUI application entry point and model initialization.
// ABOUTME: Handles the BubbleTea program lifecycle, key events, and export functionality.

package ui

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/paulilaaso/bit/internal/export"
)

func InitialModel() (model, error) {
	// No need to seed random number generator in Go 1.20+

	// Initialize text input
	ti := textinput.New()
	ti.Placeholder = "Enter text..."
	ti.Blur() // Start unfocused - user must press Enter to activate
	ti.CharLimit = TextInputCharLimit
	ti.Width = 25              // Will be adjusted in View based on panel width
	ti.ShowSuggestions = false // Disable suggestions for cleaner display

	// Configure cursor appearance
	ti.Cursor.Style = textInputCursorStyle
	ti.Cursor.SetMode(CursorBlink)

	// Configure textinput styling to match panel colors
	ti.TextStyle = textInputTextStyle
	ti.PlaceholderStyle = textInputPlaceholderStyle

	// Initialize filename input for export
	filenameInput := textinput.New()
	filenameInput.Placeholder = "Enter filename..."
	filenameInput.Blur()
	filenameInput.CharLimit = FilenameInputCharLimit
	filenameInput.Width = FilenameInputWidth
	filenameInput.ShowSuggestions = false

	// Configure filename input styling
	filenameInput.TextStyle = filenameInputTextStyle
	filenameInput.PlaceholderStyle = filenameInputPlaceholderStyle

	// Initialize export manager
	exportManager := export.NewExportManager()

	// Load available fonts (lazy loading - only metadata)
	fonts, err := loadFontList()
	if err != nil {
		return model{}, fmt.Errorf("failed to load font list: %w", err)
	}

	// Initialize text rows and cursor positions
	initialTextRows := []string{"Hello"}
	initialRowCursors := make([]int, len(initialTextRows))
	for i := range initialRowCursors {
		initialRowCursors[i] = 0
	}

	m := model{
		textInput: textInputModel{
			input:         ti,
			currentText:   "Hello",
			textRows:      initialTextRows,            // Initialize with one row
			rowCursors:    initialRowCursors,          // Initialize cursor positions for each row
			customKerning: map[int]map[int]int{0: {}}, // Initialize first row's kerning map
			currentRow:    0,                          // Start with first row
			alignment:     CenterAlignment,            // Start with center alignment
			mode:          TextEntryMode,              // Start with text input mode
		},
		font: fontModel{
			fonts:        fonts,
			selectedFont: 0,
		},
		spacing: spacingModel{
			charSpacing: DefaultCharSpacing,
			wordSpacing: DefaultWordSpacing,
			lineSpacing: DefaultLineSpacing,
			mode:        CharacterSpacingMode, // Start with character spacing
		},
		color: colorModel{
			textColor:         6,              // Start with white color (index 6 in color options)
			gradientColor:     6,              // Start with same as text color (None/disabled)
			gradientEnabled:   false,          // Start with gradient disabled
			gradientDirection: GradientUpDown, // Start with Up-Down direction
			subMode:           TextColorMode,  // Start with text color mode
		},
		scale: scaleModel{
			scale: DefaultTextScale, // Start with 1x scaling
		},
		shadow: shadowModel{
			enabled:          false,
			horizontalOffset: 0,                   // Canonical offset value
			verticalOffset:   0,                   // Canonical offset value
			horizontalIndex:  DefaultShadowPixels, // UI index for horizontal options
			verticalIndex:    5,                   // UI index for vertical options (5 = "Off")
			style:            0,                   // Start with "Light Shade"
			showWarning:      false,
			subMode:          HorizontalShadowMode, // Start with horizontal shadow
		},
		export: exportModel{
			active:           false,                            // Start with export mode disabled
			format:           exportManager.GetDefaultFormat(), // Default export format
			filenameInput:    filenameInput,
			showConfirmation: false,         // Start with export confirmation hidden
			confirmationText: "",            // No confirmation text initially
			manager:          exportManager, // Store export manager in model
		},
		uiState: uiStateModel{
			focusedPanel:  TextInputPanel, // Start with text input panel
			usesTwoRows:   false,          // Start with single row layout
			renderedLines: []string{},
		},
	}

	// Render initial text
	m.updateCurrentTextFromRows() // Sync currentText with textRows
	m.renderText()
	return m, nil
}

func (m model) Init() tea.Cmd {
	return textinput.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return m, m.handleWindowResize(msg)
	case tea.KeyMsg:
		// Handle export mode first
		if m.export.active {
			return m.handleExportModeKeys(msg)
		}

		// Reset confirmations on any key press
		m.resetConfirmations()

		if !m.isInputMode() {
			return m.handleNormalModeKeys(msg)
		} else {
			return m.handleInputModeKeys(msg)
		}
	}

	return m, cmd
}

// handleInputModeKeys handles key presses while in input mode
func (m *model) handleInputModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "ctrl+c", "esc":
		return m, tea.Quit
	case "tab":
		return m, m.handleTabKey()
	case "shift+tab":
		m.uiState.focusedPanel = FocusedPanel((int(m.uiState.focusedPanel) - 1 + int(TotalPanels)) % int(TotalPanels))
		m.textInput.input.Blur()
	case "left", "right":
		if m.uiState.focusedPanel == TextInputPanel && m.textInput.mode == TextKerningMode && m.textInput.input.Focused() {
			m.textInput.input, cmd = m.textInput.input.Update(msg)
			return m, cmd
		}
		if msg.String() == "left" {
			return m.handlePanelNavigation(-1)
		}
		return m.handlePanelNavigation(1)
	case "h", "l":
		if m.uiState.focusedPanel == TextInputPanel && m.textInput.mode == TextKerningMode && m.textInput.input.Focused() {
			keyType := tea.KeyRight
			if msg.String() == "h" {
				keyType = tea.KeyLeft
			}
			m.textInput.input, cmd = m.textInput.input.Update(tea.KeyMsg{Type: keyType})
			return m, cmd
		}
		if m.textInput.mode == TextEntryMode && m.textInput.input.Focused() {
			cmd = m.updateFocusedTextInput(msg)
			return m, cmd
		}
		if msg.String() == "h" {
			return m.handlePanelNavigation(-1)
		}
		return m.handlePanelNavigation(1)
	case "up", "down":
		return m, m.handleUpDownKeys(msg)
	case "k", "j":
		if m.uiState.focusedPanel == TextInputPanel && m.textInput.mode == TextKerningMode && m.textInput.input.Focused() {
			return m, m.handleUpDownKeys(msg)
		}
		if m.textInput.mode == TextEntryMode && m.textInput.input.Focused() {
			cmd = m.updateFocusedTextInput(msg)
			return m, cmd
		}
		return m, m.handleUpDownKeys(msg)
	case "enter":
		return m, m.handleEnterKey()
	default:
		// Handle text input when focused
		if m.textInput.input.Focused() {
			m.textInput.input, cmd = m.textInput.input.Update(msg)
		} else if m.export.filenameInput.Focused() {
			// Handle filename input when focused
			m.export.filenameInput, cmd = m.export.filenameInput.Update(msg)
		}
	}

	return m, cmd
}

// handleNormalModeKeys handles key presses while in normal mode
// Normal mode is when keystrokes are used to invoke actions instead of as input
func (m *model) handleNormalModeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg.String() {
	case "ctrl+c", "esc":
		return m, tea.Quit
	case "tab":
		return m, m.handleTabKey()
	case "shift+tab":
		m.uiState.focusedPanel = FocusedPanel((int(m.uiState.focusedPanel) - 1 + int(TotalPanels)) % int(TotalPanels))
		m.textInput.input.Blur()
	case "left", "h":
		return m.handlePanelNavigation(-1)
	case "right", "l":
		return m.handlePanelNavigation(1)
	case "up", "down", "k", "j":
		return m, m.handleUpDownKeys(msg)
	case "enter":
		return m, m.handleEnterKey()
	case "q":
		return m, tea.Quit
	case "e":
		m.handleEnterExportMode()
	case "r":
		m.handleRandomize()
	default:
		// Handle text input when focused
		if m.textInput.input.Focused() {
			m.textInput.input, cmd = m.textInput.input.Update(msg)
		} else if m.export.filenameInput.Focused() {
			// Handle filename input when focused
			m.export.filenameInput, cmd = m.export.filenameInput.Update(msg)
		}
	}

	return m, cmd
}

// handleTabKey handles tab key presses for mode switching
func (m *model) handleTabKey() tea.Cmd {
	switch m.uiState.focusedPanel {
	case TextInputPanel:
		m.textInput.mode = TextInputMode((int(m.textInput.mode) + 1) % int(TotalTextInputModes))
		if m.textInput.mode == TextKerningMode {
			m.textInput.input.Focus()
		} else {
			m.textInput.input.Blur()
		}
	case SpacingPanel:
		m.spacing.mode = SpacingMode((int(m.spacing.mode) + 1) % int(TotalSpacingModes))
	case ColorPanel:
		m.color.subMode = ColorSubMode((int(m.color.subMode) + 1) % int(TotalColorSubModes))
	case ShadowPanel:
		m.shadow.subMode = ShadowSubMode((int(m.shadow.subMode) + 1) % int(TotalShadowSubModes))
	default:
		m.uiState.focusedPanel = FocusedPanel((int(m.uiState.focusedPanel) + 1) % int(TotalPanels))
		m.textInput.input.Blur()
	}
	return nil
}

// handleUpDownKeys handles up/down arrow key presses
func (m *model) handleUpDownKeys(msg tea.KeyMsg) tea.Cmd {
	switch m.uiState.focusedPanel {
	case TextInputPanel:
		_, cmd := m.handleTextPanelUpdate(msg)
		return cmd
	case FontPanel:
		m.handleFontPanelUpdate(msg)
	case SpacingPanel:
		m.handleSpacingPanelUpdate(msg)
	case ColorPanel:
		m.handleColorPanelUpdate(msg)
	case ScalePanel:
		m.handleScalePanelUpdate(msg)
	case ShadowPanel:
		m.handleShadowPanelUpdate(msg)
	}
	return nil
}

// handleEnterKey handles enter key presses
func (m *model) handleEnterKey() tea.Cmd {
	if m.uiState.focusedPanel == TextInputPanel && (m.textInput.mode == TextEntryMode || m.textInput.mode == TextKerningMode) {
		m.handleTextInputToggle()
	}
	return nil
}

// updateCurrentTextFromRows combines all text rows into currentText for rendering
func (m *model) updateCurrentTextFromRows() {
	m.textInput.currentText = strings.Join(m.textInput.textRows, "\n")
}

// exportText exports the rendered text to a file
func (m *model) exportText() {
	originalFilename := m.export.filenameInput.Value()
	if originalFilename == "" {
		return
	}

	// Sanitize the filename to prevent path traversal and invalid characters
	sanitizedFilename := export.SanitizeFilename(originalFilename)
	if sanitizedFilename == "" {
		m.export.showConfirmation = true
		m.export.confirmationText = "Invalid filename"
		return
	}

	formatName := m.export.format

	// Check if this is a binary format (like PNG)
	if m.export.manager.IsBinaryFormat(formatName) {
		m.exportBinaryFormat(sanitizedFilename, formatName)
		return
	}

	// Generate content based on selected format (text formats)
	var content string
	switch m.export.format {
	case "TXT":
		content = export.GenerateTXTCode(m.uiState.renderedLines)
	case "SVG":
		// renderedLines is never empty here, so generation cannot fail.
		content, _ = export.GenerateSVG(m.uiState.renderedLines, export.TerminalAspectRatioSVGOptions())
	case "GO":
		content = export.GenerateGoCode(m.uiState.renderedLines)
	case "JS":
		content = export.GenerateJSCode(m.uiState.renderedLines)
	case "PY":
		content = export.GeneratePythonCode(m.uiState.renderedLines)
	case "RS":
		content = export.GenerateRustCode(m.uiState.renderedLines)
	case "SH":
		content = export.GenerateBashCode(m.uiState.renderedLines)
	default:
		// Default to TXT if format not recognized
		content = export.GenerateTXTCode(m.uiState.renderedLines)
	}

	// Check if file exists before attempting export
	exists, finalFilename, err := m.export.manager.CheckFileExists(sanitizedFilename, formatName)
	if err != nil {
		m.export.showConfirmation = true
		m.export.confirmationText = fmt.Sprintf("Export failed: %v", err)
		return
	}

	if exists {
		// Show overwrite prompt
		m.export.showOverwritePrompt = true
		m.export.overwriteFilename = finalFilename
		m.export.overwriteContent = content
		m.export.overwriteFormat = formatName
		m.export.selectedButton = 1 // Default to "No"
		return
	}

	// File doesn't exist, proceed with export
	m.performExport(content, sanitizedFilename, formatName)
}

// exportBinaryFormat handles export of binary formats like PNG
func (m *model) exportBinaryFormat(filename, formatName string) {
	var content []byte
	var err error

	switch formatName {
	case "PNG":
		options := export.TerminalAspectRatioPNGOptions()
		content, err = export.GeneratePNG(m.uiState.renderedLines, options)
		if err != nil {
			m.export.showConfirmation = true
			m.export.confirmationText = fmt.Sprintf("PNG generation failed: %v", err)
			return
		}
	default:
		m.export.showConfirmation = true
		m.export.confirmationText = fmt.Sprintf("Unknown binary format: %s", formatName)
		return
	}

	// Check if file exists before attempting export
	exists, finalFilename, err := m.export.manager.CheckFileExists(filename, formatName)
	if err != nil {
		m.export.showConfirmation = true
		m.export.confirmationText = fmt.Sprintf("Export failed: %v", err)
		return
	}

	if exists {
		// Show overwrite prompt with binary content
		m.export.showOverwritePrompt = true
		m.export.overwriteFilename = finalFilename
		m.export.overwriteBinaryContent = content
		m.export.overwriteContent = "" // Clear text content
		m.export.overwriteFormat = formatName
		m.export.selectedButton = 1 // Default to "No"
		return
	}

	// File doesn't exist, proceed with export
	m.performBinaryExport(content, filename, formatName)
}

// performBinaryExport writes binary content to file
func (m *model) performBinaryExport(content []byte, filename, formatName string) {
	err := m.export.manager.ExportBinary(content, filename, formatName)
	if err != nil {
		m.export.showConfirmation = true
		m.export.confirmationText = fmt.Sprintf("Export failed: %v", err)
		return
	}

	// Set export confirmation message using the actual filename that was saved
	cwd, _ := os.Getwd()
	sanitizedFilename := export.SanitizeFilename(filename)
	format := m.export.manager.GetFormatByName(formatName)
	if format != nil && !strings.HasSuffix(sanitizedFilename, format.Extension) {
		sanitizedFilename += format.Extension
	}
	m.export.showConfirmation = true
	m.export.confirmationText = fmt.Sprintf("Exported to %s/%s", cwd, sanitizedFilename)
}

// performExport actually writes the file
func (m *model) performExport(content, filename, formatName string) {
	err := m.export.manager.Export(content, filename, formatName)
	if err != nil {
		m.export.showConfirmation = true
		m.export.confirmationText = fmt.Sprintf("Export failed: %v", err)
		return
	}

	// Set export confirmation message using the actual filename that was saved
	cwd, _ := os.Getwd()
	// Sanitize the filename and add extension if needed to match what was actually saved
	sanitizedFilename := export.SanitizeFilename(filename)
	format := m.export.manager.GetFormatByName(formatName)
	if format != nil && !strings.HasSuffix(sanitizedFilename, format.Extension) {
		sanitizedFilename += format.Extension
	}
	m.export.showConfirmation = true
	m.export.confirmationText = fmt.Sprintf("Exported to %s/%s", cwd, sanitizedFilename)
}

// getFormatDescription returns the description for a given export format
func (m model) getFormatDescription(format string) string {
	return m.export.manager.GetFormatDescription(format)
}

// getFormatExtension returns the file extension for a given export format
func (m model) getFormatExtension(format string) string {
	return m.export.manager.GetFormatExtension(format)
}

func (m model) View() string {
	// If in export mode, show export UI instead of normal UI
	if m.export.active {
		return m.renderExportView()
	}

	// Calculate heights for different sections
	controlPanelsHeight := 3
	if m.uiState.usesTwoRows {
		controlPanelsHeight = 8
	}

	controlsHeight := 1
	titleHeight := 1
	minRequiredHeight := titleHeight + controlPanelsHeight + controlsHeight + 2

	// Calculate available space for text display
	availableForText := m.uiState.height - minRequiredHeight
	minTextHeight := 3
	mainDisplayHeight := max(availableForText, minTextHeight)

	// Render each section
	centeredTitle := m.renderTitleView()
	textDisplay := m.renderTextDisplayView(mainDisplayHeight)
	controlPanels := m.renderControlPanelsView()
	centeredControls := m.renderControlsView()

	// Combine everything
	content := lipgloss.JoinVertical(lipgloss.Left,
		centeredTitle,
		textDisplay,
		controlPanels,
		centeredControls,
	)

	return lipgloss.NewStyle().
		MaxWidth(m.uiState.width).
		MaxHeight(m.uiState.height).
		Render(content)
}

func isUpKey(txt string) bool {
	return txt == "up" || txt == "k"
}

func isDownKey(txt string) bool {
	return txt == "down" || txt == "j"
}

func (m *model) isInputMode() bool {
	return m.uiState.focusedPanel == TextInputPanel && (m.textInput.mode == TextEntryMode || m.textInput.mode == TextKerningMode) && m.textInput.input.Focused()
}
