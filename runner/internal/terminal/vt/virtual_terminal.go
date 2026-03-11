package vt

import (
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
)

// OSCHandler is a callback for handling OSC (Operating System Command) sequences.
// oscType is the OSC code (e.g., 777 for notify, 9 for message, 0/2 for title).
// params are the semicolon-separated parameters after the OSC code.
type OSCHandler func(oscType int, params []string)

// VirtualTerminal provides a virtual terminal emulator
// that converts raw PTY output with ANSI escape sequences
// into clean text for agent observation.
//
// This implementation properly handles ANSI CSI sequences for:
// - Cursor movement (CUU, CUD, CUF, CUB, CUP, etc.)
// - Line/screen clearing (ED, EL)
// - Scrolling regions
// - Alternative screen buffer
// - SGR (Select Graphic Rendition) for colors and text attributes
type VirtualTerminal struct {
	mu sync.RWMutex

	cols int
	rows int

	// Screen buffer (current visible content) - runes only for backward compatibility
	screen [][]rune

	// Styled cell buffer - cells with color and attribute information
	cells [][]Cell

	// Cursor position
	cursorX int
	cursorY int

	// Current text style (applied to new characters)
	currentFg             Color
	currentBg             Color
	currentAttrs          CellAttrs
	currentUnderlineStyle UnderlineStyle
	currentUnderlineColor Color

	// Line wrap tracking (true if line is wrapped from previous line)
	isWrapped []bool

	// History buffer (scrolled-off lines) - plain text for backward compatibility
	history    []string
	maxHistory int

	// Styled history buffer (scrolled-off lines with full style information)
	// Each entry is a row of cells, preserving colors and attributes
	historyStyled    [][]Cell
	historyIsWrapped []bool // Wrap flags for styled history lines

	// Flag to track if we've received any data
	hasData bool

	// First data callback - triggered once when VT receives first PTY data
	onFirstData    func()
	onFirstDataMu  sync.Mutex
	firstDataFired bool

	// Escape sequence parsing state
	escState   escapeState
	escBuffer  []byte
	escParams  []int
	escPrivate byte
	escRawSeq  []byte // Raw sequence for SGR parsing with colons

	// Saved cursor position
	savedCursorX int
	savedCursorY int

	// Alternative screen buffer support
	altScreen       [][]rune
	altCells        [][]Cell
	altCursorX      int
	altCursorY      int
	useAltScreen    bool
	savedMainScreen [][]rune
	savedMainCells  [][]Cell

	// OSC sequence handler callback
	oscHandler OSCHandler
}

// NewVirtualTerminal creates a new virtual terminal
func NewVirtualTerminal(cols, rows, maxHistory int) *VirtualTerminal {
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	if maxHistory <= 0 {
		maxHistory = 100 // Small default to avoid OOM - TUI apps use alt screen anyway
	}

	vt := &VirtualTerminal{
		cols:             cols,
		rows:             rows,
		maxHistory:       maxHistory,
		history:          make([]string, 0),
		historyStyled:    make([][]Cell, 0),
		historyIsWrapped: make([]bool, 0),
	}
	vt.initScreen()
	return vt
}

// initScreen initializes/resets the screen buffer
func (vt *VirtualTerminal) initScreen() {
	vt.screen = make([][]rune, vt.rows)
	vt.cells = make([][]Cell, vt.rows)
	vt.isWrapped = make([]bool, vt.rows)
	for i := range vt.screen {
		vt.screen[i] = make([]rune, vt.cols)
		vt.cells[i] = make([]Cell, vt.cols)
		vt.isWrapped[i] = false
		for j := range vt.screen[i] {
			vt.screen[i][j] = ' '
			vt.cells[i][j] = NewCell(' ')
		}
	}
	vt.cursorX = 0
	vt.cursorY = 0
	vt.currentFg = DefaultColor()
	vt.currentBg = DefaultColor()
	vt.currentAttrs = AttrNone
	vt.currentUnderlineStyle = UnderlineNone
	vt.currentUnderlineColor = DefaultColor()
}

// Feed processes raw PTY data with proper UTF-8 support.
// Returns the current screen lines for downstream consumers (single-direction data flow).
// This avoids the need for consumers to acquire a separate lock to read screen state.
func (vt *VirtualTerminal) Feed(data []byte) []string {
	lockStart := time.Now()
	vt.mu.Lock()
	lockWait := time.Since(lockStart)
	defer vt.mu.Unlock()

	if lockWait > 10*time.Millisecond {
		logger.Terminal().Warn("VT Feed lock acquisition slow",
			"lock_wait", lockWait, "data_len", len(data))
	}

	wasHasData := vt.hasData
	vt.hasData = true
	if !wasHasData {
		// Trigger first data callback (in goroutine to avoid blocking)
		vt.onFirstDataMu.Lock()
		if !vt.firstDataFired && vt.onFirstData != nil {
			vt.firstDataFired = true
			callback := vt.onFirstData
			vt.onFirstDataMu.Unlock()
			safego.Go("vt-first-data", callback) // Execute in goroutine to avoid blocking PTY reading
		} else {
			vt.onFirstDataMu.Unlock()
		}
	}

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

	// Return current screen lines for downstream consumers (single-direction data flow)
	// This is done inside the lock to ensure consistency
	return vt.getLinesLocked()
}

// getLinesLocked returns screen lines. Caller must hold vt.mu.
func (vt *VirtualTerminal) getLinesLocked() []string {
	screen := vt.screen
	lines := make([]string, vt.rows)
	for row := 0; row < vt.rows; row++ {
		var line strings.Builder
		if row < len(screen) {
			for _, ch := range screen[row] {
				if ch == 0 {
					line.WriteRune(' ')
				} else {
					line.WriteRune(ch)
				}
			}
		}
		lines[row] = strings.TrimRight(line.String(), " ")
	}
	return lines
}

// Resize resizes the terminal
func (vt *VirtualTerminal) Resize(cols, rows int) {
	lockStart := time.Now()
	vt.mu.Lock()
	lockWait := time.Since(lockStart)
	defer vt.mu.Unlock()

	if lockWait > 10*time.Millisecond {
		logger.Terminal().Warn("VT Resize lock acquisition slow",
			"lock_wait", lockWait, "cols", cols, "rows", rows)
	}

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
