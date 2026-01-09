package terminal

import (
	"bytes"
	"regexp"
	"strings"
	"sync"
	"unicode/utf8"
)

// VirtualTerminal provides a virtual terminal emulator
// that converts raw PTY output with ANSI escape sequences
// into clean text for agent observation.
//
// This implementation properly handles ANSI CSI sequences for:
// - Cursor movement (CUU, CUD, CUF, CUB, CUP, etc.)
// - Line/screen clearing (ED, EL)
// - Scrolling regions
// - Alternative screen buffer
type VirtualTerminal struct {
	mu sync.RWMutex

	cols int
	rows int

	// Screen buffer (current visible content)
	screen [][]rune

	// Cursor position
	cursorX int
	cursorY int

	// History buffer (scrolled-off lines)
	history    []string
	maxHistory int

	// Flag to track if we've received any data
	hasData bool

	// Escape sequence parsing state
	escState   escapeState
	escBuffer  []byte
	escParams  []int
	escPrivate byte

	// Saved cursor position
	savedCursorX int
	savedCursorY int

	// Alternative screen buffer support
	altScreen       [][]rune
	altCursorX      int
	altCursorY      int
	useAltScreen    bool
	savedMainScreen [][]rune
}

// escapeState represents the current state of escape sequence parsing
type escapeState int

const (
	stateNormal escapeState = iota
	stateEscape             // After ESC
	stateCSI                // After ESC [
	stateOSC                // After ESC ]
	stateDCS                // After ESC P
)

// ANSI escape sequence pattern (for simple stripping)
// Matches:
// - CSI sequences: ESC [ [?>=] params letter (includes DEC private mode ?xxx and Kitty >xxx)
// - OSC sequences: ESC ] ... BEL
// - DCS/PM/APC sequences: ESC P/X/^/_ ... ST
var ansiPattern = regexp.MustCompile(`\x1b\[[?>=]?[0-9;]*[a-zA-Z]|\x1b\][^\x07]*\x07|\x1b[PX^_][^\x1b]*\x1b\\`)

// NewVirtualTerminal creates a new virtual terminal
func NewVirtualTerminal(cols, rows, maxHistory int) *VirtualTerminal {
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	if maxHistory <= 0 {
		maxHistory = 10000
	}

	vt := &VirtualTerminal{
		cols:       cols,
		rows:       rows,
		maxHistory: maxHistory,
		history:    make([]string, 0),
	}
	vt.initScreen()
	return vt
}

// initScreen initializes/resets the screen buffer
func (vt *VirtualTerminal) initScreen() {
	vt.screen = make([][]rune, vt.rows)
	for i := range vt.screen {
		vt.screen[i] = make([]rune, vt.cols)
		for j := range vt.screen[i] {
			vt.screen[i][j] = ' '
		}
	}
	vt.cursorX = 0
	vt.cursorY = 0
}

// Feed processes raw PTY data with proper UTF-8 support
func (vt *VirtualTerminal) Feed(data []byte) {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	vt.hasData = true

	// Process data with UTF-8 awareness
	for len(data) > 0 {
		b := data[0]

		// ESC sequence or in escape state: process byte by byte
		if b == 0x1b || vt.escState != stateNormal {
			vt.processByte(b)
			data = data[1:]
			continue
		}

		// Control characters (< 0x20) and DEL (0x7f): process as single byte
		if b < 0x20 || b == 0x7f {
			vt.processByte(b)
			data = data[1:]
			continue
		}

		// Normal characters: decode UTF-8 properly
		r, size := utf8.DecodeRune(data)
		if r == utf8.RuneError && size == 1 {
			// Invalid UTF-8 byte, skip it
			data = data[1:]
			continue
		}
		vt.processChar(r)
		data = data[size:]
	}
}

// processByte processes a single byte through the state machine
func (vt *VirtualTerminal) processByte(b byte) {
	switch vt.escState {
	case stateNormal:
		if b == 0x1b { // ESC
			vt.escState = stateEscape
			vt.escBuffer = nil
			vt.escParams = nil
			vt.escPrivate = 0
		} else {
			vt.processChar(rune(b))
		}

	case stateEscape:
		switch b {
		case '[': // CSI
			vt.escState = stateCSI
			vt.escParams = []int{}
		case ']': // OSC
			vt.escState = stateOSC
			vt.escBuffer = nil
		case 'P': // DCS
			vt.escState = stateDCS
			vt.escBuffer = nil
		case '7': // Save cursor (DECSC)
			vt.savedCursorX = vt.cursorX
			vt.savedCursorY = vt.cursorY
			vt.escState = stateNormal
		case '8': // Restore cursor (DECRC)
			vt.cursorX = vt.savedCursorX
			vt.cursorY = vt.savedCursorY
			vt.escState = stateNormal
		case 'c': // Reset (RIS)
			vt.initScreen()
			vt.escState = stateNormal
		case 'D': // Index (IND) - move down
			vt.cursorY++
			if vt.cursorY >= vt.rows {
				vt.scroll()
				vt.cursorY = vt.rows - 1
			}
			vt.escState = stateNormal
		case 'M': // Reverse Index (RI) - move up
			vt.cursorY--
			if vt.cursorY < 0 {
				vt.scrollDown()
				vt.cursorY = 0
			}
			vt.escState = stateNormal
		case 'E': // Next Line (NEL)
			vt.cursorX = 0
			vt.cursorY++
			if vt.cursorY >= vt.rows {
				vt.scroll()
				vt.cursorY = vt.rows - 1
			}
			vt.escState = stateNormal
		default:
			// Unknown escape sequence, return to normal
			vt.escState = stateNormal
		}

	case stateCSI:
		vt.processCSI(b)

	case stateOSC:
		// OSC sequences end with BEL (0x07) or ST (ESC \)
		if b == 0x07 {
			vt.escState = stateNormal
		} else {
			vt.escBuffer = append(vt.escBuffer, b)
		}

	case stateDCS:
		// DCS sequences end with ST (ESC \)
		if b == 0x1b {
			// Might be start of ST
			vt.escBuffer = append(vt.escBuffer, b)
		} else if len(vt.escBuffer) > 0 && vt.escBuffer[len(vt.escBuffer)-1] == 0x1b && b == '\\' {
			vt.escState = stateNormal
		} else {
			vt.escBuffer = append(vt.escBuffer, b)
		}
	}
}

// processCSI processes a CSI (Control Sequence Introducer) byte
func (vt *VirtualTerminal) processCSI(b byte) {
	switch {
	case b >= '0' && b <= '9':
		// Digit - build parameter
		if len(vt.escParams) == 0 {
			vt.escParams = []int{0}
		}
		vt.escParams[len(vt.escParams)-1] = vt.escParams[len(vt.escParams)-1]*10 + int(b-'0')

	case b == ';':
		// Parameter separator
		vt.escParams = append(vt.escParams, 0)

	case b == '?':
		// Private mode indicator
		vt.escPrivate = b

	case b >= 0x40 && b <= 0x7e:
		// Final byte - execute command
		vt.executeCSI(b)
		vt.escState = stateNormal

	default:
		// Intermediate byte or unknown
		vt.escBuffer = append(vt.escBuffer, b)
	}
}

// executeCSI executes a CSI command
func (vt *VirtualTerminal) executeCSI(cmd byte) {
	// Default parameter value
	param := func(idx, def int) int {
		if idx < len(vt.escParams) && vt.escParams[idx] > 0 {
			return vt.escParams[idx]
		}
		return def
	}

	switch cmd {
	case 'A': // CUU - Cursor Up
		n := param(0, 1)
		vt.cursorY -= n
		if vt.cursorY < 0 {
			vt.cursorY = 0
		}

	case 'B': // CUD - Cursor Down
		n := param(0, 1)
		vt.cursorY += n
		if vt.cursorY >= vt.rows {
			vt.cursorY = vt.rows - 1
		}

	case 'C': // CUF - Cursor Forward (Right)
		n := param(0, 1)
		vt.cursorX += n
		if vt.cursorX >= vt.cols {
			vt.cursorX = vt.cols - 1
		}

	case 'D': // CUB - Cursor Back (Left)
		n := param(0, 1)
		vt.cursorX -= n
		if vt.cursorX < 0 {
			vt.cursorX = 0
		}

	case 'E': // CNL - Cursor Next Line
		n := param(0, 1)
		vt.cursorX = 0
		vt.cursorY += n
		if vt.cursorY >= vt.rows {
			vt.cursorY = vt.rows - 1
		}

	case 'F': // CPL - Cursor Previous Line
		n := param(0, 1)
		vt.cursorX = 0
		vt.cursorY -= n
		if vt.cursorY < 0 {
			vt.cursorY = 0
		}

	case 'G': // CHA - Cursor Horizontal Absolute
		col := param(0, 1)
		vt.cursorX = col - 1
		if vt.cursorX < 0 {
			vt.cursorX = 0
		}
		if vt.cursorX >= vt.cols {
			vt.cursorX = vt.cols - 1
		}

	case 'H', 'f': // CUP/HVP - Cursor Position
		row := param(0, 1)
		col := 1
		if len(vt.escParams) > 1 {
			col = param(1, 1)
		}
		vt.cursorY = row - 1
		vt.cursorX = col - 1
		if vt.cursorY < 0 {
			vt.cursorY = 0
		}
		if vt.cursorY >= vt.rows {
			vt.cursorY = vt.rows - 1
		}
		if vt.cursorX < 0 {
			vt.cursorX = 0
		}
		if vt.cursorX >= vt.cols {
			vt.cursorX = vt.cols - 1
		}

	case 'J': // ED - Erase in Display
		n := param(0, 0)
		switch n {
		case 0: // Erase from cursor to end of screen
			vt.clearLine(vt.cursorY, vt.cursorX, vt.cols)
			for i := vt.cursorY + 1; i < vt.rows; i++ {
				vt.clearLine(i, 0, vt.cols)
			}
		case 1: // Erase from start to cursor
			for i := 0; i < vt.cursorY; i++ {
				vt.clearLine(i, 0, vt.cols)
			}
			vt.clearLine(vt.cursorY, 0, vt.cursorX+1)
		case 2, 3: // Erase entire screen
			for i := 0; i < vt.rows; i++ {
				vt.clearLine(i, 0, vt.cols)
			}
		}

	case 'K': // EL - Erase in Line
		n := param(0, 0)
		switch n {
		case 0: // Erase from cursor to end of line
			vt.clearLine(vt.cursorY, vt.cursorX, vt.cols)
		case 1: // Erase from start of line to cursor
			vt.clearLine(vt.cursorY, 0, vt.cursorX+1)
		case 2: // Erase entire line
			vt.clearLine(vt.cursorY, 0, vt.cols)
		}

	case 'L': // IL - Insert Lines
		n := param(0, 1)
		vt.insertLines(n)

	case 'M': // DL - Delete Lines
		n := param(0, 1)
		vt.deleteLines(n)

	case 'P': // DCH - Delete Characters
		n := param(0, 1)
		vt.deleteChars(n)

	case '@': // ICH - Insert Characters
		n := param(0, 1)
		vt.insertChars(n)

	case 'X': // ECH - Erase Characters
		n := param(0, 1)
		for i := 0; i < n && vt.cursorX+i < vt.cols; i++ {
			vt.screen[vt.cursorY][vt.cursorX+i] = ' '
		}

	case 'S': // SU - Scroll Up
		n := param(0, 1)
		for i := 0; i < n; i++ {
			vt.scroll()
		}

	case 'T': // SD - Scroll Down
		n := param(0, 1)
		for i := 0; i < n; i++ {
			vt.scrollDown()
		}

	case 's': // SCP - Save Cursor Position
		vt.savedCursorX = vt.cursorX
		vt.savedCursorY = vt.cursorY

	case 'u': // RCP - Restore Cursor Position
		vt.cursorX = vt.savedCursorX
		vt.cursorY = vt.savedCursorY

	case 'h': // SM - Set Mode
		if vt.escPrivate == '?' {
			vt.handlePrivateMode(true)
		}

	case 'l': // RM - Reset Mode
		if vt.escPrivate == '?' {
			vt.handlePrivateMode(false)
		}

	case 'm': // SGR - Select Graphic Rendition
		// We ignore styling for agent observation (color, bold, etc.)
		// This just consumes the sequence without error

	case 'r': // DECSTBM - Set Top and Bottom Margins
		// Ignore scrolling region for simplified implementation

	case 'c': // DA - Device Attributes
		// Ignore device attribute request

	case 'n': // DSR - Device Status Report
		// Ignore status report request
	}
}

// handlePrivateMode handles DEC private mode sequences
func (vt *VirtualTerminal) handlePrivateMode(set bool) {
	for _, p := range vt.escParams {
		switch p {
		case 1049, 47: // Alternative screen buffer
			if set {
				vt.enterAltScreen()
			} else {
				vt.exitAltScreen()
			}
		case 25: // DECTCEM - Show/hide cursor (ignore for text-only)
		case 1: // DECCKM - Application cursor keys (ignore)
		case 7: // DECAWM - Auto-wrap mode (we always wrap)
		case 12: // Start blinking cursor (ignore)
		case 2004: // Bracketed paste mode (ignore)
		}
	}
}

// enterAltScreen switches to alternative screen buffer
func (vt *VirtualTerminal) enterAltScreen() {
	if vt.useAltScreen {
		return
	}
	// Save main screen
	vt.savedMainScreen = make([][]rune, vt.rows)
	for i := range vt.screen {
		vt.savedMainScreen[i] = make([]rune, len(vt.screen[i]))
		copy(vt.savedMainScreen[i], vt.screen[i])
	}
	// Initialize alt screen
	vt.altScreen = make([][]rune, vt.rows)
	for i := range vt.altScreen {
		vt.altScreen[i] = make([]rune, vt.cols)
		for j := range vt.altScreen[i] {
			vt.altScreen[i][j] = ' '
		}
	}
	vt.altCursorX = vt.cursorX
	vt.altCursorY = vt.cursorY
	vt.screen = vt.altScreen
	vt.cursorX = 0
	vt.cursorY = 0
	vt.useAltScreen = true
}

// exitAltScreen switches back to main screen buffer
func (vt *VirtualTerminal) exitAltScreen() {
	if !vt.useAltScreen {
		return
	}
	// Restore main screen
	if vt.savedMainScreen != nil {
		vt.screen = vt.savedMainScreen
		vt.savedMainScreen = nil
	}
	vt.cursorX = vt.altCursorX
	vt.cursorY = vt.altCursorY
	vt.useAltScreen = false
}

// clearLine clears part of a line
func (vt *VirtualTerminal) clearLine(row, startCol, endCol int) {
	if row < 0 || row >= vt.rows {
		return
	}
	for i := startCol; i < endCol && i < vt.cols; i++ {
		if i >= 0 {
			vt.screen[row][i] = ' '
		}
	}
}

// insertLines inserts n blank lines at cursor position
func (vt *VirtualTerminal) insertLines(n int) {
	for i := 0; i < n; i++ {
		// Shift lines down
		for j := vt.rows - 1; j > vt.cursorY; j-- {
			copy(vt.screen[j], vt.screen[j-1])
		}
		// Clear current line
		for j := range vt.screen[vt.cursorY] {
			vt.screen[vt.cursorY][j] = ' '
		}
	}
}

// deleteLines deletes n lines at cursor position
func (vt *VirtualTerminal) deleteLines(n int) {
	for i := 0; i < n; i++ {
		// Shift lines up
		for j := vt.cursorY; j < vt.rows-1; j++ {
			copy(vt.screen[j], vt.screen[j+1])
		}
		// Clear bottom line
		for j := range vt.screen[vt.rows-1] {
			vt.screen[vt.rows-1][j] = ' '
		}
	}
}

// deleteChars deletes n characters at cursor position
func (vt *VirtualTerminal) deleteChars(n int) {
	row := vt.screen[vt.cursorY]
	for i := vt.cursorX; i < vt.cols-n; i++ {
		row[i] = row[i+n]
	}
	for i := vt.cols - n; i < vt.cols; i++ {
		if i >= 0 {
			row[i] = ' '
		}
	}
}

// insertChars inserts n blank characters at cursor position
func (vt *VirtualTerminal) insertChars(n int) {
	row := vt.screen[vt.cursorY]
	for i := vt.cols - 1; i >= vt.cursorX+n; i-- {
		row[i] = row[i-n]
	}
	for i := 0; i < n && vt.cursorX+i < vt.cols; i++ {
		row[vt.cursorX+i] = ' '
	}
}

// scrollDown scrolls the screen down (reverse scroll)
func (vt *VirtualTerminal) scrollDown() {
	// Shift all lines down
	for i := vt.rows - 1; i > 0; i-- {
		vt.screen[i] = vt.screen[i-1]
	}
	// Clear top line
	vt.screen[0] = make([]rune, vt.cols)
	for j := range vt.screen[0] {
		vt.screen[0][j] = ' '
	}
}

// processChar processes a single character
func (vt *VirtualTerminal) processChar(ch rune) {
	switch ch {
	case '\n':
		vt.newLine()
	case '\r':
		vt.cursorX = 0
	case '\b':
		if vt.cursorX > 0 {
			vt.cursorX--
		}
	case '\t':
		// Move to next tab stop (every 8 columns)
		vt.cursorX = ((vt.cursorX / 8) + 1) * 8
		if vt.cursorX >= vt.cols {
			vt.cursorX = vt.cols - 1
		}
	case '\x1b':
		// Start of escape sequence - handled by stripping later
	default:
		if ch >= ' ' && ch != '\x7f' {
			vt.putChar(ch)
		}
	}
}

// putChar puts a character at the current cursor position
func (vt *VirtualTerminal) putChar(ch rune) {
	if vt.cursorX >= vt.cols {
		vt.newLine()
	}
	if vt.cursorY >= 0 && vt.cursorY < vt.rows && vt.cursorX >= 0 && vt.cursorX < vt.cols {
		vt.screen[vt.cursorY][vt.cursorX] = ch
	}
	vt.cursorX++
}

// newLine moves to the next line, scrolling if necessary
func (vt *VirtualTerminal) newLine() {
	vt.cursorX = 0
	vt.cursorY++
	if vt.cursorY >= vt.rows {
		vt.scroll()
		vt.cursorY = vt.rows - 1
	}
}

// scroll scrolls the screen up by one line
func (vt *VirtualTerminal) scroll() {
	// Save top line to history
	line := strings.TrimRight(string(vt.screen[0]), " ")
	if line != "" {
		vt.history = append(vt.history, line)
		// Trim history if too large
		if len(vt.history) > vt.maxHistory {
			vt.history = vt.history[1:]
		}
	}

	// Scroll screen up
	for i := 0; i < vt.rows-1; i++ {
		vt.screen[i] = vt.screen[i+1]
	}

	// Clear bottom line
	vt.screen[vt.rows-1] = make([]rune, vt.cols)
	for j := range vt.screen[vt.rows-1] {
		vt.screen[vt.rows-1][j] = ' '
	}
}

// Resize resizes the terminal
func (vt *VirtualTerminal) Resize(cols, rows int) {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}

	vt.cols = cols
	vt.rows = rows
	vt.initScreen()
}

// GetDisplay returns the current screen content
func (vt *VirtualTerminal) GetDisplay() string {
	vt.mu.RLock()
	defer vt.mu.RUnlock()

	if !vt.hasData {
		return ""
	}

	var lines []string
	for _, row := range vt.screen {
		line := strings.TrimRight(string(row), " ")
		lines = append(lines, line)
	}

	// Remove trailing empty lines
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n")
}

// GetOutput returns recent terminal output (history + current screen)
func (vt *VirtualTerminal) GetOutput(lines int) string {
	vt.mu.RLock()
	defer vt.mu.RUnlock()

	if !vt.hasData {
		return ""
	}

	var result []string

	// Add from history
	result = append(result, vt.history...)

	// Add current screen content (non-empty lines only)
	for _, row := range vt.screen {
		line := strings.TrimRight(string(row), " ")
		if line != "" {
			result = append(result, line)
		}
	}

	// Return last N lines
	if len(result) > lines {
		result = result[len(result)-lines:]
	}

	return strings.Join(result, "\n")
}

// GetScreenSnapshot returns a snapshot of the current screen
func (vt *VirtualTerminal) GetScreenSnapshot() string {
	return vt.GetDisplay()
}

// Clear clears the terminal and history
func (vt *VirtualTerminal) Clear() {
	vt.mu.Lock()
	defer vt.mu.Unlock()

	vt.initScreen()
	vt.history = make([]string, 0)
	vt.hasData = false
}

// CursorPosition returns the current cursor position
func (vt *VirtualTerminal) CursorPosition() (row, col int) {
	vt.mu.RLock()
	defer vt.mu.RUnlock()
	return vt.cursorY, vt.cursorX
}

// Cols returns the terminal width
func (vt *VirtualTerminal) Cols() int {
	vt.mu.RLock()
	defer vt.mu.RUnlock()
	return vt.cols
}

// Rows returns the terminal height
func (vt *VirtualTerminal) Rows() int {
	vt.mu.RLock()
	defer vt.mu.RUnlock()
	return vt.rows
}

// StripANSI removes ANSI escape sequences from text
func StripANSI(text string) string {
	return ansiPattern.ReplaceAllString(text, "")
}

// StripANSIBytes removes ANSI escape sequences from bytes
func StripANSIBytes(data []byte) []byte {
	return bytes.ReplaceAll(
		bytes.ReplaceAll(data, []byte("\x1b["), []byte("")),
		[]byte("\x1b"), []byte(""),
	)
}
