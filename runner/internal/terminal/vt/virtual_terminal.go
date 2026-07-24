package vt

import (
	"sync"
	"time"
	"unicode/utf8"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
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
	escState        escapeState
	escBuffer       []byte
	escParams       []int
	escPrivate      byte
	escRawSeq       []byte // Raw sequence for SGR parsing with colons
	utf8Pending     []byte
	discardSawESC   bool
	canJoinGrapheme bool

	// Client-visible modes that must survive snapshot recovery.
	privateModes      map[int]bool
	mouseTrackingMode int
	mouseEncodingMode int

	// Saved cursor registers are buffer-local in xterm.
	savedCursor [2]cursorState

	// Alternative screen buffer support
	altScreen        [][]rune
	altCells         [][]Cell
	altIsWrapped     []bool
	altCursorX       int
	altCursorY       int
	altScreenMode    int
	useAltScreen     bool
	savedMainScreen  [][]rune
	savedMainCells   [][]Cell
	savedMainWrapped []bool
	savedMainFg      Color
	savedMainBg      Color
	savedMainAttrs   CellAttrs
	savedMainUlStyle UnderlineStyle
	savedMainUlColor Color

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
	vt.resetTerminalModes()
	vt.initScreen()
	return vt
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
	data = vt.prependPendingUTF8(data)
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
			if !utf8.FullRune(data) {
				vt.utf8Pending = append(vt.utf8Pending[:0], data...)
				break
			}
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
