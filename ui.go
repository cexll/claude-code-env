package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"

	"golang.org/x/term"
)

// terminalCapabilities holds terminal feature detection results
type terminalCapabilities struct {
	IsTerminal     bool
	SupportsRaw    bool
	SupportsANSI   bool
	SupportsCursor bool
	Width          int
	Height         int
}

// TerminalLayout contains responsive layout calculations and constraints
type TerminalLayout struct {
	Width           int
	Height          int
	SupportsANSI    bool
	ContentWidth    int // Available width for content after UI elements
	TruncationLimit int // Maximum length before content truncation
}

// DisplayFormatter manages responsive content formatting with smart truncation
type DisplayFormatter struct {
	layout     TerminalLayout
	nameWidth  int // Calculated optimal name column width
	urlWidth   int // Calculated optimal URL column width
	modelWidth int // Calculated optimal model column width
}

// EnvironmentDisplay represents an environment with responsive display formatting
type EnvironmentDisplay struct {
	Environment     Environment
	DisplayName     string   // Truncated if necessary
	DisplayURL      string   // Truncated if necessary
	DisplayModel    string   // Truncated if necessary
	TruncatedFields []string // Track what was truncated for user awareness
}

// DisplayState tracks current display content and manages updates
type DisplayState struct {
	// Content tracking
	currentLines []string // Current displayed lines
	headerLine   string   // Menu header text
	footerLine   string   // Optional footer/instructions

	// Terminal info
	terminalWidth  int // Current terminal width
	terminalHeight int // Current terminal height

	// Selection tracking
	lastSelection    int // Previous selected index
	currentSelection int // Current selected index

	// State flags
	initialized      bool // Display state initialized
	contentChanged   bool // Content needs refresh
	selectionChanged bool // Only selection indicator changed
}

// LineContext represents line rendering context for differential updates
type LineContext struct {
	lineIndex       int    // Line number (0-based)
	content         string // Line content
	isSelected      bool   // Is this the selected line
	needsUpdate     bool   // Line content changed
	previousContent string // Previous content for comparison
}

// terminalState manages terminal state restoration
type terminalState struct {
	fd       int
	oldState *term.State
	restored bool
}

// initializeDisplayState creates a new DisplayState with terminal dimensions
func initializeDisplayState() *DisplayState {
	caps := detectTerminalCapabilities()
	return &DisplayState{
		currentLines:     []string{},
		headerLine:       "",
		footerLine:       "",
		terminalWidth:    caps.Width,
		terminalHeight:   caps.Height,
		lastSelection:    -1,
		currentSelection: 0,
		initialized:      true,
		contentChanged:   true,
		selectionChanged: false,
	}
}

// UpdateContent updates the display state with new content and selection
func (ds *DisplayState) UpdateContent(lines []string, selection int) {
	if !ds.initialized {
		return
	}

	// Check if content changed
	contentChanged := len(lines) != len(ds.currentLines)
	if !contentChanged {
		for i, line := range lines {
			if i >= len(ds.currentLines) || line != ds.currentLines[i] {
				contentChanged = true
				break
			}
		}
	}

	// Check if only selection changed
	selectionChanged := selection != ds.currentSelection && !contentChanged

	// Update state
	ds.currentLines = make([]string, len(lines))
	copy(ds.currentLines, lines)
	ds.lastSelection = ds.currentSelection
	ds.currentSelection = selection
	ds.contentChanged = contentChanged
	ds.selectionChanged = selectionChanged
}

// ClearDisplay clears the display state and resets to initial state
func (ds *DisplayState) ClearDisplay() {
	if !ds.initialized {
		return
	}

	ds.currentLines = []string{}
	ds.headerLine = ""
	ds.footerLine = ""
	ds.lastSelection = -1
	ds.currentSelection = 0
	ds.contentChanged = true
	ds.selectionChanged = false
}

// RecoverFromError resets display state after errors
func (ds *DisplayState) RecoverFromError() error {
	ds.initialized = false
	ds.currentLines = nil

	// Reinitialize with simple fallback
	newState := initializeDisplayState()
	*ds = *newState

	return nil
}

// TextPositioner provides ANSI-free cursor positioning and line control
type TextPositioner struct {
	width int
}

// newTextPositioner creates a TextPositioner with terminal width
func newTextPositioner(width int) *TextPositioner {
	return &TextPositioner{
		width: width,
	}
}

// MoveToStartOfLine returns carriage return character to move cursor to line start
func (tp *TextPositioner) MoveToStartOfLine() string {
	return "\r"
}

// ClearToEndOfLine returns padding spaces to clear from cursor to end of line
func (tp *TextPositioner) ClearToEndOfLine() string {
	return strings.Repeat(" ", tp.width)
}

// ClearLine creates a string that clears an entire line using carriage return and spaces
func (tp *TextPositioner) ClearLine() string {
	return tp.MoveToStartOfLine() + tp.ClearToEndOfLine() + tp.MoveToStartOfLine()
}

// OverwriteLine creates a string that overwrites a line with new content
func (tp *TextPositioner) OverwriteLine(content string) string {
	// Ensure content doesn't exceed terminal width
	if len(content) > tp.width {
		content = content[:tp.width-3] + "..."
	}

	// Pad content to full width to clear any remaining characters
	paddedContent := content + strings.Repeat(" ", tp.width-len(content))

	return tp.MoveToStartOfLine() + paddedContent + tp.MoveToStartOfLine()
}

// LineRenderer manages stateful menu rendering with ANSI-free display
type LineRenderer struct {
	state      *DisplayState
	positioner *TextPositioner
	useANSI    bool // Optional enhancement only
}

// newLineRenderer creates a LineRenderer with display state
func newLineRenderer(state *DisplayState, useANSI bool) *LineRenderer {
	return &LineRenderer{
		state:      state,
		positioner: newTextPositioner(state.terminalWidth),
		useANSI:    useANSI,
	}
}

// RenderMenu renders the complete environment menu using stateful display
func (lr *LineRenderer) RenderMenu(environments []Environment, selectedIndex int, header string) {
	if !lr.state.initialized {
		return
	}

	// Detect terminal layout and create formatter
	layout := detectTerminalLayout()
	formatter := newDisplayFormatter(layout)

	// Build new content lines
	newLines := []string{}
	if header != "" {
		newLines = append(newLines, header)
	}

	for i, env := range environments {
		prefix := "  "
		if i == selectedIndex {
			if lr.useANSI {
				prefix = "► " // Use arrow for ANSI-enabled terminals
			} else {
				prefix = "* " // Use asterisk for basic terminals
			}
		}

		// Format complete line to fit within terminal width
		line := formatter.formatSingleLine(prefix, env)
		newLines = append(newLines, line)
	}

	// Update display state
	lr.state.UpdateContent(newLines, selectedIndex)

	// Render based on what changed
	if lr.state.contentChanged {
		lr.renderFullContent()
	} else if lr.state.selectionChanged {
		lr.renderSelectionChange(environments, formatter)
	}
}

// renderFullContent renders all content lines (used when content changes)
func (lr *LineRenderer) renderFullContent() {
	// Clear the screen first to prevent content stacking
	clearScreen()

	// For ANSI-free display, we simply print all lines fresh
	// This avoids complex cursor positioning issues
	for i, line := range lr.state.currentLines {
		if i == 0 {
			// First line - print without newline
			fmt.Print(lr.positioner.OverwriteLine(line))
		} else {
			// Subsequent lines - new line then content
			fmt.Print("\n")
			fmt.Print(lr.positioner.OverwriteLine(line))
		}
	}
}

// renderSelectionChange optimized rendering for selection-only changes
func (lr *LineRenderer) renderSelectionChange(environments []Environment, formatter *DisplayFormatter) {
	// For ANSI-free terminals, we'll render the full content when selection changes
	// This ensures compatibility while still being more efficient than clearScreen
	lr.renderFullContent()
}

// moveToLineAndOverwrite moves cursor to specific line and overwrites content
func (lr *LineRenderer) moveToLineAndOverwrite(lineNum int, content string) {
	// Move up to the target line using carriage returns and up sequences
	// For ANSI-free approach, we'll use multiple carriage returns with newlines
	fmt.Print(strings.Repeat("\r\n", lineNum))
	fmt.Print(lr.positioner.OverwriteLine(content))
}

// OverwriteLine overwrites a specific line with new content
func (lr *LineRenderer) OverwriteLine(lineNum int, content string) {
	if lineNum >= 0 && lineNum < len(lr.state.currentLines) {
		lr.state.currentLines[lineNum] = content
		lr.moveToLineAndOverwrite(lineNum, content)
	}
}

// restore terminal state safely
func (ts *terminalState) restore() error {
	if ts.restored || ts.oldState == nil {
		return nil
	}
	ts.restored = true
	return term.Restore(ts.fd, ts.oldState)
}

// ensureRestore guarantees terminal restoration via defer
func (ts *terminalState) ensureRestore() {
	if err := ts.restore(); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to restore terminal: %v\n", err)
	}
}

// detectTerminalCapabilities performs comprehensive terminal capability detection
func detectTerminalCapabilities() terminalCapabilities {
	fd := int(syscall.Stdin)
	caps := terminalCapabilities{
		IsTerminal: term.IsTerminal(fd),
		Width:      80, // Default fallback
		Height:     24, // Default fallback
	}

	// Determine ANSI/cursor support based on TERM even if not a TTY
	termType := os.Getenv("TERM")
	caps.SupportsANSI = termType != "" && termType != "dumb" && !strings.HasPrefix(termType, "vt5")
	caps.SupportsCursor = caps.SupportsANSI

	// Only probe raw mode and size when running in a real terminal
	if caps.IsTerminal {
		if oldState, err := term.MakeRaw(fd); err == nil {
			caps.SupportsRaw = true
			// Immediately restore to avoid corruption
			if err := term.Restore(fd, oldState); err != nil {
				caps.SupportsRaw = false
			}
		}

		if width, height, err := term.GetSize(fd); err == nil {
			caps.Width = width
			caps.Height = height
		}
	}

	return caps
}

// detectTerminalLayout creates layout configuration with responsive calculations
func detectTerminalLayout() TerminalLayout {
	caps := detectTerminalCapabilities()

	layout := TerminalLayout{
		Width:        caps.Width,
		Height:       caps.Height,
		SupportsANSI: caps.SupportsANSI,
	}

	// Calculate content width (reserve space for UI elements: prefix + brackets + spacing)
	// UI overhead: "► " (2 chars) + " (" + ") [" + "]" (4 chars) + spacing (2 chars) = 8 chars
	uiOverhead := 8
	if layout.Width < 40 {
		// Very narrow terminal - minimal overhead
		uiOverhead = 4
	}

	layout.ContentWidth = layout.Width - uiOverhead
	if layout.ContentWidth < 20 {
		layout.ContentWidth = 20 // Minimum usable width
	}

	// Set truncation limit based on content width
	layout.TruncationLimit = layout.ContentWidth - 10 // Reserve space for ellipsis and spacing
	if layout.TruncationLimit < 10 {
		layout.TruncationLimit = 10
	}

	return layout
}

// newDisplayFormatter creates a formatter with optimal column width calculations
func newDisplayFormatter(layout TerminalLayout) *DisplayFormatter {
	formatter := &DisplayFormatter{
		layout: layout,
	}

	// Allocate space proportionally: Name (40%), URL (45%), Model (15%)
	contentSpace := layout.ContentWidth
	formatter.nameWidth = int(float64(contentSpace) * 0.40)
	formatter.urlWidth = int(float64(contentSpace) * 0.45)
	formatter.modelWidth = int(float64(contentSpace) * 0.15)

	// Ensure minimum widths
	if formatter.nameWidth < 8 {
		formatter.nameWidth = 8
	}
	if formatter.urlWidth < 10 {
		formatter.urlWidth = 10
	}
	if formatter.modelWidth < 6 {
		formatter.modelWidth = 6
	}

	return formatter
}

// smartTruncateName implements intelligent name truncation
func (df *DisplayFormatter) smartTruncateName(name string) (string, bool) {
	if len(name) <= df.nameWidth {
		return name, false
	}

	// Keep beginning and end, ellipsis in middle
	if df.nameWidth < 8 {
		return name[:df.nameWidth-3] + "...", true
	}

	prefixLen := (df.nameWidth - 3) / 2
	suffixLen := df.nameWidth - 3 - prefixLen

	return name[:prefixLen] + "..." + name[len(name)-suffixLen:], true
}

// smartTruncateURL implements intelligent URL truncation
func (df *DisplayFormatter) smartTruncateURL(url string) (string, bool) {
	if len(url) <= df.urlWidth {
		return url, false
	}

	// Show protocol + domain, truncate path with ellipsis
	if strings.Contains(url, "://") {
		parts := strings.SplitN(url, "://", 2)
		if len(parts) == 2 {
			protocol := parts[0] + "://"
			remaining := parts[1]

			// Find domain part
			domainEndIdx := strings.Index(remaining, "/")
			if domainEndIdx == -1 {
				domainEndIdx = len(remaining)
			}

			domain := remaining[:domainEndIdx]
			protocolDomainLen := len(protocol) + len(domain)

			if protocolDomainLen <= df.urlWidth-3 {
				return protocol + domain + "...", true
			}
		}
	}

	// Fallback: simple truncation
	return url[:df.urlWidth-3] + "...", true
}

// smartTruncateModel implements intelligent model truncation
func (df *DisplayFormatter) smartTruncateModel(model string) (string, bool) {
	if model == "" {
		return "default", false
	}

	if len(model) <= df.modelWidth {
		return model, false
	}

	// Preserve model family identifier if possible
	if strings.HasPrefix(model, "claude-") {
		if df.modelWidth >= 10 {
			// Try to keep "claude-" prefix and truncate end
			return model[:df.modelWidth-3] + "...", true
		}
	}

	// Simple truncation
	return model[:df.modelWidth-3] + "...", true
}

// formatEnvironmentForDisplay creates responsive display formatting for an environment
func (df *DisplayFormatter) formatEnvironmentForDisplay(env Environment) EnvironmentDisplay {
	display := EnvironmentDisplay{
		Environment:     env,
		TruncatedFields: []string{},
	}

	// Format name
	var nameTruncated bool
	display.DisplayName, nameTruncated = df.smartTruncateName(env.Name)
	if nameTruncated {
		display.TruncatedFields = append(display.TruncatedFields, "name")
	}

	// Format URL
	var urlTruncated bool
	display.DisplayURL, urlTruncated = df.smartTruncateURL(env.URL)
	if urlTruncated {
		display.TruncatedFields = append(display.TruncatedFields, "url")
	}

	// Format model
	var modelTruncated bool
	display.DisplayModel, modelTruncated = df.smartTruncateModel(env.Model)
	if modelTruncated {
		display.TruncatedFields = append(display.TruncatedFields, "model")
	}

	return display
}

// formatSingleLine creates a complete line that fits within terminal width
func (df *DisplayFormatter) formatSingleLine(prefix string, env Environment) string {
	// Calculate available space for content
	// Format will be: "prefix name (url) [model]"
	prefixLen := len(prefix)

	// Static characters: " (" + ") [" + "]" = 6 characters
	staticOverhead := 6
	maxContentLen := df.layout.Width - prefixLen - staticOverhead

	// If we don't have enough space, use minimal format
	if maxContentLen < 20 {
		name := env.Name
		if len(name) > 10 {
			name = name[:7] + "..."
		}
		return fmt.Sprintf("%s%s", prefix, name)
	}

	// Distribute space: name (40%), url (45%), model (15%)
	nameSpace := int(float64(maxContentLen) * 0.40)
	urlSpace := int(float64(maxContentLen) * 0.45)
	modelSpace := maxContentLen - nameSpace - urlSpace

	// Ensure minimum sizes
	if nameSpace < 8 {
		nameSpace = 8
	}
	if urlSpace < 10 {
		urlSpace = 10
	}
	if modelSpace < 6 {
		modelSpace = 6
	}

	// Truncate fields to fit allocated space
	name := env.Name
	if len(name) > nameSpace {
		if nameSpace > 3 {
			name = name[:nameSpace-3] + "..."
		} else {
			name = name[:nameSpace]
		}
	}

	url := env.URL
	if len(url) > urlSpace {
		if urlSpace > 3 {
			url = url[:urlSpace-3] + "..."
		} else {
			url = url[:urlSpace]
		}
	}

	model := env.Model
	if model == "" {
		model = "default"
	}
	if len(model) > modelSpace {
		if modelSpace > 3 {
			model = model[:modelSpace-3] + "..."
		} else {
			model = model[:modelSpace]
		}
	}

	// Create the formatted line
	line := fmt.Sprintf("%s%s (%s) [%s]", prefix, name, url, model)

	// Final safety check - truncate if still too long
	if len(line) > df.layout.Width {
		maxLen := df.layout.Width - 3
		if maxLen > 0 {
			line = line[:maxLen] + "..."
		} else {
			line = line[:df.layout.Width]
		}
	}

	return line
}

// ArrowKey represents arrow key types for navigation
type ArrowKey int

const (
	ArrowNone ArrowKey = iota
	ArrowUp
	ArrowDown
	ArrowLeft
	ArrowRight
)

// parseKeyInput handles cross-platform key input parsing
func parseKeyInput(input []byte) (ArrowKey, rune, error) {
	if len(input) == 0 {
		return ArrowNone, 0, fmt.Errorf("empty input")
	}

	// Single character keys
	if len(input) == 1 {
		switch input[0] {
		case '\n', '\r':
			return ArrowNone, '\n', nil
		case '\x1b': // Escape
			return ArrowNone, '\x1b', nil
		case '\x03': // Ctrl+C
			return ArrowNone, '\x03', nil
		default:
			return ArrowNone, rune(input[0]), nil
		}
	}

	// Arrow key sequences (cross-platform)
	if len(input) >= 3 && input[0] == '\x1b' && input[1] == '[' {
		switch input[2] {
		case 'A':
			return ArrowUp, 0, nil
		case 'B':
			return ArrowDown, 0, nil
		case 'C':
			return ArrowRight, 0, nil
		case 'D':
			return ArrowLeft, 0, nil
		}
	}

	return ArrowNone, 0, fmt.Errorf("unrecognized key sequence")
}

// clearScreen provides ANSI-free screen clearing using line-by-line approach
func clearScreen() {
	caps := detectTerminalCapabilities()
	positioner := newTextPositioner(caps.Width)

	// Clear multiple lines by overwriting with spaces
	// Use a reasonable number of lines to clear most content
	linesToClear := 25
	if caps.Height > 0 {
		linesToClear = caps.Height
	}

	for i := 0; i < linesToClear; i++ {
		fmt.Print(positioner.ClearLine())
		if i < linesToClear-1 {
			fmt.Print("\n")
		}
	}

	// Move cursor to top by printing enough carriage returns
	fmt.Print(strings.Repeat("\r", linesToClear))
}

// Global display state for interactive menu rendering
var globalDisplayState *DisplayState
var globalLineRenderer *LineRenderer

// renderMenuStatefully provides centralized stateful rendering for both interactive modes
func renderMenuStatefully(environments []Environment, selectedIndex int, header string, useANSI bool) {
	// Initialize global state if needed
	if globalDisplayState == nil {
		globalDisplayState = initializeDisplayState()
		globalLineRenderer = newLineRenderer(globalDisplayState, useANSI)
		// Clear screen on first initialization to ensure clean start
		clearScreen()
	}

	// Update terminal dimensions in case of resize
	caps := detectTerminalCapabilities()
	globalDisplayState.terminalWidth = caps.Width
	globalDisplayState.terminalHeight = caps.Height
	globalLineRenderer.positioner = newTextPositioner(caps.Width)

	// Render using the line renderer
	globalLineRenderer.RenderMenu(environments, selectedIndex, header)
}

// cleanupDisplayState cleans up global display state
func cleanupDisplayState() {
	if globalDisplayState != nil {
		globalDisplayState.ClearDisplay()
		globalDisplayState = nil
		globalLineRenderer = nil
	}
}

// displayEnvironmentMenu shows interactive menu with responsive layout and selection indicator
func displayEnvironmentMenu(environments []Environment, selectedIndex int) {
	// Use stateful rendering instead of clearScreen
	header := "Select environment (use ↑↓ arrows, Enter to confirm, Esc to cancel):"
	renderMenuStatefully(environments, selectedIndex, header, true)
}

// selectEnvironmentWithArrows provides 4-tier progressive fallback navigation
func selectEnvironmentWithArrows(config Config) (Environment, error) {
	if len(config.Environments) == 0 {
		return Environment{}, fmt.Errorf("no environments configured - use 'add' command to create one")
	}

	if len(config.Environments) == 1 {
		return config.Environments[0], nil
	}

	// Detect terminal capabilities
	caps := detectTerminalCapabilities()

	// Tier 4: Headless mode (no terminal or pipe detected)
	if !caps.IsTerminal {
		// Check if this is a script/pipe scenario
		if isHeadlessMode() {
			if len(config.Environments) > 0 {
				fmt.Printf("Headless mode: using first environment '%s'\n", config.Environments[0].Name)
				return config.Environments[0], nil
			}
			return Environment{}, fmt.Errorf("no environments available for headless mode")
		}
		return fallbackToNumberedSelection(config)
	}

	// Tier 1: Full interactive mode (raw + ANSI + cursor)
	if caps.SupportsRaw && caps.SupportsANSI && caps.SupportsCursor {
		return fullInteractiveSelection(config, caps)
	}

	// Tier 2: Basic interactive mode (raw mode only, no ANSI)
	if caps.SupportsRaw {
		return basicInteractiveSelection(config, caps)
	}

	// Tier 3: Numbered selection mode (no raw mode support)
	return fallbackToNumberedSelection(config)
}

// fullInteractiveSelection implements Tier 1: full featured arrow navigation with ANSI
func fullInteractiveSelection(config Config, caps terminalCapabilities) (Environment, error) {
	fd := int(syscall.Stdin)
	termState := &terminalState{fd: fd}

	// Set up raw mode with guaranteed cleanup
	var err error
	termState.oldState, err = term.MakeRaw(fd)
	if err != nil {
		return basicInteractiveSelection(config, caps)
	}
	defer termState.ensureRestore()
	defer cleanupDisplayState() // Clean up display state on exit

	selectedIndex := 0
	buffer := make([]byte, 10)

	for {
		displayEnvironmentMenu(config.Environments, selectedIndex)

		n, err := os.Stdin.Read(buffer)
		if err != nil {
			return fallbackToNumberedSelection(config)
		}

		arrow, char, err := parseKeyInput(buffer[:n])
		if err != nil {
			continue
		}

		switch arrow {
		case ArrowUp:
			selectedIndex = (selectedIndex - 1 + len(config.Environments)) % len(config.Environments)
		case ArrowDown:
			selectedIndex = (selectedIndex + 1) % len(config.Environments)
		case ArrowNone:
			switch char {
			case '\n', '\r':
				return config.Environments[selectedIndex], nil
			case '\x1b', '\x03':
				return Environment{}, fmt.Errorf("selection cancelled")
			}
		}
	}
}

// basicInteractiveSelection implements Tier 2: arrow navigation without ANSI styling
func basicInteractiveSelection(config Config, caps terminalCapabilities) (Environment, error) {
	fd := int(syscall.Stdin)
	termState := &terminalState{fd: fd}

	var err error
	termState.oldState, err = term.MakeRaw(fd)
	if err != nil {
		return fallbackToNumberedSelection(config)
	}
	defer termState.ensureRestore()
	defer cleanupDisplayState() // Clean up display state on exit

	selectedIndex := 0
	buffer := make([]byte, 10)

	for {
		displayBasicEnvironmentMenu(config.Environments, selectedIndex)

		n, err := os.Stdin.Read(buffer)
		if err != nil {
			return fallbackToNumberedSelection(config)
		}

		arrow, char, err := parseKeyInput(buffer[:n])
		if err != nil {
			continue
		}

		switch arrow {
		case ArrowUp:
			selectedIndex = (selectedIndex - 1 + len(config.Environments)) % len(config.Environments)
		case ArrowDown:
			selectedIndex = (selectedIndex + 1) % len(config.Environments)
		case ArrowNone:
			switch char {
			case '\n', '\r':
				return config.Environments[selectedIndex], nil
			case '\x1b', '\x03':
				return Environment{}, fmt.Errorf("selection cancelled")
			}
		}
	}
}

// displayBasicEnvironmentMenu shows menu without ANSI escape sequences but with responsive layout
func displayBasicEnvironmentMenu(environments []Environment, selectedIndex int) {
	// Use stateful rendering with ANSI disabled for basic mode
	header := "Select environment (use arrows, Enter to confirm, Esc to cancel):"
	renderMenuStatefully(environments, selectedIndex, header, false)
}

// isHeadlessMode detects if running in a script/pipe environment
func isHeadlessMode() bool {
	// Check if stdout is being redirected/piped
	if fi, err := os.Stdout.Stat(); err == nil {
		return (fi.Mode() & os.ModeCharDevice) == 0
	}

	// Check common CI/automation environment variables
	ciVars := []string{"CI", "CONTINUOUS_INTEGRATION", "GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL"}
	for _, envVar := range ciVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	return false
}

// fallbackToNumberedSelection uses existing numbered selection menu
func fallbackToNumberedSelection(config Config) (Environment, error) {
	fmt.Println("Arrow key navigation not supported, using numbered selection:")
	return selectEnvironmentOriginal(config)
}

// secureInput prompts for input without echoing characters to terminal
func secureInput(prompt string) (string, error) {
	if _, err := fmt.Print(prompt); err != nil {
		return "", fmt.Errorf("failed to display prompt: %w", err)
	}

	// Get file descriptor for stdin
	fd := int(syscall.Stdin)

	// Check if stdin is a terminal
	if !term.IsTerminal(fd) {
		return "", fmt.Errorf("secure input requires a terminal")
	}

	// Save original terminal state
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return "", fmt.Errorf("failed to set terminal raw mode: %w", err)
	}

	// Ensure terminal state is restored on exit
	defer func() {
		if err := term.Restore(fd, oldState); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to restore terminal state: %v\n", err)
		}
	}()

	var input []byte
	buffer := make([]byte, 1)

	for {
		// Read one character at a time
		n, err := os.Stdin.Read(buffer)
		if err != nil {
			return "", fmt.Errorf("failed to read input: %w", err)
		}
		if n == 0 {
			continue
		}

		char := buffer[0]

		// Handle special characters
		switch char {
		case '\n', '\r': // Enter key
			// Print newline after hidden input
			if _, err := fmt.Println(); err != nil {
				return "", fmt.Errorf("failed to print newline: %w", err)
			}
			// Clear sensitive data from buffer
			for i := range buffer {
				buffer[i] = 0
			}
			return string(input), nil

		case 127, 8: // Backspace/Delete
			if len(input) > 0 {
				input = input[:len(input)-1]
			}

		case 3: // Ctrl+C
			return "", fmt.Errorf("input cancelled by user")

		case 4: // Ctrl+D (EOF)
			if len(input) == 0 {
				return "", fmt.Errorf("EOF received")
			}

		default:
			// Only accept printable characters
			if char >= 32 && char <= 126 {
				input = append(input, char)
			}
		}
	}
}

// regularInput prompts for regular (non-sensitive) input with validation
func regularInput(prompt string) (string, error) {
	if _, err := fmt.Print(prompt); err != nil {
		return "", fmt.Errorf("failed to display prompt: %w", err)
	}

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}

	return strings.TrimSpace(input), nil
}

// selectEnvironment provides an interactive menu to select from available environments
func selectEnvironment(config Config) (Environment, error) {
	// Try arrow key navigation first, fallback to numbered selection
	return selectEnvironmentWithArrows(config)
}

// selectEnvironmentOriginal is the numbered selection implementation with responsive layout
func selectEnvironmentOriginal(config Config) (Environment, error) {
	if len(config.Environments) == 0 {
		return Environment{}, fmt.Errorf("no environments configured - use 'add' command to create one")
	}

	if len(config.Environments) == 1 {
		return config.Environments[0], nil
	}

	// Display environments with responsive formatting
	if _, err := fmt.Println("Select environment:"); err != nil {
		return Environment{}, fmt.Errorf("failed to display menu: %w", err)
	}

	// Detect terminal layout and create formatter
	layout := detectTerminalLayout()
	formatter := newDisplayFormatter(layout)

	for i, env := range config.Environments {
		// Format complete line to fit within terminal width
		prefix := fmt.Sprintf("%d. ", i+1)
		line := formatter.formatSingleLine(prefix, env)

		if _, err := fmt.Println(line); err != nil {
			return Environment{}, fmt.Errorf("failed to display environment option: %w", err)
		}
	}

	// Get user selection
	input, err := regularInput(fmt.Sprintf("Enter number (1-%d): ", len(config.Environments)))
	if err != nil {
		return Environment{}, fmt.Errorf("environment selection failed: %w", err)
	}

	// Validate selection
	choice, err := strconv.Atoi(input)
	if err != nil {
		return Environment{}, fmt.Errorf("invalid selection - must be a number: %w", err)
	}

	if choice < 1 || choice > len(config.Environments) {
		return Environment{}, fmt.Errorf("invalid selection - must be between 1 and %d", len(config.Environments))
	}

	return config.Environments[choice-1], nil
}

// promptForEnvironment collects new environment details with validation
func promptForEnvironment(config Config) (Environment, error) {
	var env Environment
	var err error

	// Get environment name
	for {
		env.Name, err = regularInput("Environment name: ")
		if err != nil {
			return Environment{}, fmt.Errorf("failed to get environment name: %w", err)
		}

		// Validate name
		if err := validateName(env.Name); err != nil {
			if _, printErr := fmt.Printf("Invalid name: %v\n", err); printErr != nil {
				return Environment{}, fmt.Errorf("failed to display error: %w", printErr)
			}
			continue
		}

		// Check for duplicate
		if _, exists := findEnvironmentByName(config, env.Name); exists {
			if _, printErr := fmt.Printf("Environment '%s' already exists\n", env.Name); printErr != nil {
				return Environment{}, fmt.Errorf("failed to display error: %w", printErr)
			}
			continue
		}

		break
	}

	// Get base URL
	for {
		env.URL, err = regularInput("Base URL: ")
		if err != nil {
			return Environment{}, fmt.Errorf("failed to get base URL: %w", err)
		}

		// Validate URL
		if err := validateURL(env.URL); err != nil {
			if _, printErr := fmt.Printf("Invalid URL: %v\n", err); printErr != nil {
				return Environment{}, fmt.Errorf("failed to display error: %w", printErr)
			}
			continue
		}

		break
	}

	// Get API key (secure input)
	for {
		env.APIKey, err = secureInput("API Key (hidden): ")
		if err != nil {
			return Environment{}, fmt.Errorf("failed to get API key: %w", err)
		}

		// Validate API key
		if err := validateAPIKey(env.APIKey); err != nil {
			if _, printErr := fmt.Printf("Invalid API key: %v\n", err); printErr != nil {
				return Environment{}, fmt.Errorf("failed to display error: %w", printErr)
			}
			continue
		}

		break
	}

	// Choose API key environment variable name
	for {
		if _, printErr := fmt.Println("Select API key environment variable:"); printErr != nil {
			return Environment{}, fmt.Errorf("failed to display prompt: %w", printErr)
		}
		if _, printErr := fmt.Println("  1) ANTHROPIC_API_KEY (default)"); printErr != nil {
			return Environment{}, fmt.Errorf("failed to display option: %w", printErr)
		}
		if _, printErr := fmt.Println("  2) ANTHROPIC_AUTH_TOKEN"); printErr != nil {
			return Environment{}, fmt.Errorf("failed to display option: %w", printErr)
		}
		choice, err := regularInput("Enter choice [1/2] (default 1): ")
		if err != nil {
			return Environment{}, fmt.Errorf("failed to get selection: %w", err)
		}
		choice = strings.TrimSpace(choice)
		if choice == "" || choice == "1" {
			env.APIKeyEnv = "ANTHROPIC_API_KEY"
		} else if choice == "2" {
			env.APIKeyEnv = "ANTHROPIC_AUTH_TOKEN"
		} else {
			if _, printErr := fmt.Println("Invalid choice. Please enter 1 or 2."); printErr != nil {
				return Environment{}, fmt.Errorf("failed to display error: %w", printErr)
			}
			continue
		}
		break
	}

	// Get model (optional)
	for {
		env.Model, err = regularInput("Model (optional, press Enter for default): ")
		if err != nil {
			return Environment{}, fmt.Errorf("failed to get model: %w", err)
		}

		// Validate model
		if err := validateModel(env.Model); err != nil {
			if _, printErr := fmt.Printf("Invalid model: %v\n", err); printErr != nil {
				return Environment{}, fmt.Errorf("failed to display error: %w", printErr)
			}
			continue
		}

		break
	}

	// Get additional environment variables (optional)
	env.EnvVars = make(map[string]string)
	if _, printErr := fmt.Println("Additional environment variables (optional):"); printErr != nil {
		return Environment{}, fmt.Errorf("failed to display prompt: %w", printErr)
	}
	if _, printErr := fmt.Println("Examples: ANTHROPIC_SMALL_FAST_MODEL, ANTHROPIC_TIMEOUT, etc."); printErr != nil {
		return Environment{}, fmt.Errorf("failed to display examples: %w", printErr)
	}
	if _, printErr := fmt.Println("Enter variable name (press Enter when done):"); printErr != nil {
		return Environment{}, fmt.Errorf("failed to display prompt: %w", printErr)
	}

	for {
		var varName string
		varName, err = regularInput("Variable name: ")
		if err != nil {
			return Environment{}, fmt.Errorf("failed to get variable name: %w", err)
		}

		// If empty, we're done
		if varName == "" {
			break
		}

		// Validate variable name using proper environment variable naming conventions
		if !isValidEnvVarName(varName) {
			if _, printErr := fmt.Printf("Invalid variable name '%s'. Must start with letter/underscore and contain only letters, numbers, and underscores.\n", varName); printErr != nil {
				return Environment{}, fmt.Errorf("failed to display error: %w", printErr)
			}
			continue
		}

		// Warn about potential conflicts with common system variables
		if isCommonSystemVar(varName) {
			if _, printErr := fmt.Printf("Warning: '%s' is a common system variable. This may override existing system settings.\n", varName); printErr != nil {
				return Environment{}, fmt.Errorf("failed to display warning: %w", printErr)
			}
		}

		// Get variable value
		var varValue string
		varValue, err = regularInput(fmt.Sprintf("Value for %s: ", varName))
		if err != nil {
			return Environment{}, fmt.Errorf("failed to get variable value: %w", err)
		}

		// Store the variable
		env.EnvVars[varName] = varValue
		if _, printErr := fmt.Printf("Added %s=%s\n", varName, varValue); printErr != nil {
			return Environment{}, fmt.Errorf("failed to display confirmation: %w", printErr)
		}
	}

	return env, nil
}

// displayEnvironments formats and shows the environment list with responsive layout and API key masking
func displayEnvironments(config Config) error {
	if len(config.Environments) == 0 {
		if _, err := fmt.Println("No environments configured."); err != nil {
			return fmt.Errorf("failed to display message: %w", err)
		}
		if _, err := fmt.Println("Use 'add' command to create your first environment."); err != nil {
			return fmt.Errorf("failed to display message: %w", err)
		}
		return nil
	}

	if _, err := fmt.Printf("Configured environments (%d):\n", len(config.Environments)); err != nil {
		return fmt.Errorf("failed to display header: %w", err)
	}

	// Detect terminal layout for responsive formatting
	layout := detectTerminalLayout()
	formatter := newDisplayFormatter(layout)

	for _, env := range config.Environments {
		// Mask API key (show only first 4 and last 4 characters)
		maskedKey := maskAPIKey(env.APIKey)

		// Format environment with responsive layout
		display := formatter.formatEnvironmentForDisplay(env)

		if _, err := fmt.Printf("\n  Name:  %s\n", display.DisplayName); err != nil {
			return fmt.Errorf("failed to display environment name: %w", err)
		}
		if _, err := fmt.Printf("  URL:   %s\n", display.DisplayURL); err != nil {
			return fmt.Errorf("failed to display environment URL: %w", err)
		}
		if _, err := fmt.Printf("  Model: %s\n", display.DisplayModel); err != nil {
			return fmt.Errorf("failed to display model: %w", err)
		}
		if _, err := fmt.Printf("  Key:   %s\n", maskedKey); err != nil {
			return fmt.Errorf("failed to display masked API key: %w", err)
		}

		// Show selected API key env var name
		keyVar := env.APIKeyEnv
		if keyVar == "" {
			keyVar = "ANTHROPIC_API_KEY"
		}
		if _, err := fmt.Printf("  Key Var: %s\n", keyVar); err != nil {
			return fmt.Errorf("failed to display api key env var: %w", err)
		}

		// Display additional environment variables if any
		if len(env.EnvVars) > 0 {
			if _, err := fmt.Printf("  Env Variables:\n"); err != nil {
				return fmt.Errorf("failed to display env vars header: %w", err)
			}
			for key, value := range env.EnvVars {
				if _, err := fmt.Printf("    %s=%s\n", key, value); err != nil {
					return fmt.Errorf("failed to display env var: %w", err)
				}
			}
		}

		// Show truncation warning if any fields were truncated
		if len(display.TruncatedFields) > 0 {
			if _, err := fmt.Printf("  (Truncated: %s)\n", strings.Join(display.TruncatedFields, ", ")); err != nil {
				return fmt.Errorf("failed to display truncation warning: %w", err)
			}
		}
	}

	return nil
}

// isValidEnvVarName validates environment variable names using proper naming conventions
func isValidEnvVarName(name string) bool {
	// Environment variable names should:
	// - Start with a letter (A-Z, a-z) or underscore (_)
	// - Contain only letters, numbers, and underscores
	// - Not be empty
	if len(name) == 0 {
		return false
	}

	// Check first character
	first := name[0]
	if !((first >= 'A' && first <= 'Z') || (first >= 'a' && first <= 'z') || first == '_') {
		return false
	}

	// Check remaining characters
	for i := 1; i < len(name); i++ {
		char := name[i]
		if !((char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9') || char == '_') {
			return false
		}
	}

	return true
}

// isCommonSystemVar checks if the variable name might conflict with common system variables
func isCommonSystemVar(name string) bool {
	commonVars := []string{
		"PATH", "HOME", "USER", "SHELL", "TERM", "LANG", "LC_ALL", "PWD", "OLDPWD",
		"TMPDIR", "TMP", "TEMP", "EDITOR", "PAGER", "BROWSER", "DISPLAY", "XDG_CONFIG_HOME",
		"GOPATH", "GOROOT", "JAVA_HOME", "NODE_ENV", "PYTHONPATH", "CLASSPATH",
		"LD_LIBRARY_PATH", "DYLD_LIBRARY_PATH", "PKG_CONFIG_PATH",
	}

	upperName := strings.ToUpper(name)
	for _, commonVar := range commonVars {
		if upperName == commonVar {
			return true
		}
	}

	return false
}

// maskAPIKey masks an API key showing only first and last few characters
func maskAPIKey(apiKey string) string {
	if len(apiKey) <= 8 {
		return strings.Repeat("*", len(apiKey))
	}

	return apiKey[:4] + strings.Repeat("*", len(apiKey)-8) + apiKey[len(apiKey)-4:]
}
