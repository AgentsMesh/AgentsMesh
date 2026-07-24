package vt

import (
	"strconv"
	"strings"

	"github.com/anthropics/agentsmesh/runner/internal/safego"
)

// escapeState represents the current state of escape sequence parsing
type escapeState int

const (
	stateNormal     escapeState = iota
	stateEscape                 // After ESC
	stateCSI                    // After ESC [
	stateOSC                    // After ESC ]
	stateDCS                    // After ESC P
	stateCSIDiscard             // Oversized CSI, ignore through final byte
	stateOSCDiscard             // Oversized OSC, ignore through BEL/ST
	stateDCSDiscard             // Oversized DCS, ignore through ST
)

// processByte processes a single byte through the state machine
func (vt *VirtualTerminal) processByte(b byte) {
	if (b == 0x18 || b == 0x1a) && vt.escState != stateNormal { // CAN/SUB
		vt.resetEscapeParser()
		return
	}
	if b == 0x1b && (vt.escState == stateEscape || vt.escState == stateCSI || vt.escState == stateCSIDiscard) {
		vt.beginEscape()
		return
	}

	switch vt.escState {
	case stateNormal:
		if b == 0x1b { // ESC
			vt.beginEscape()
		} else {
			vt.processChar(rune(b))
		}

	case stateEscape:
		vt.processEscapeByte(b)

	case stateCSI:
		vt.processCSI(b)

	case stateOSC:
		// OSC sequences end with BEL (0x07) or ST (ESC \)
		if b == 0x07 {
			vt.handleOSC(vt.escBuffer) // Parse and invoke OSC callback
			vt.escState = stateNormal
			vt.escBuffer = vt.escBuffer[:0]
		} else if b == 0x1b {
			// ST (ESC \) starts with ESC, need to track for next byte
			vt.appendParserByte(&vt.escBuffer, b)
		} else if len(vt.escBuffer) > 0 && vt.escBuffer[len(vt.escBuffer)-1] == 0x1b && b == '\\' {
			// ST (ESC \) completes OSC sequence
			vt.escBuffer = vt.escBuffer[:len(vt.escBuffer)-1] // Remove trailing ESC
			vt.handleOSC(vt.escBuffer)
			vt.escState = stateNormal
			vt.escBuffer = vt.escBuffer[:0]
		} else {
			vt.appendParserByte(&vt.escBuffer, b)
		}

	case stateDCS:
		// DCS sequences end with ST (ESC \)
		if b == 0x1b {
			vt.appendParserByte(&vt.escBuffer, b)
		} else if len(vt.escBuffer) > 0 && vt.escBuffer[len(vt.escBuffer)-1] == 0x1b && b == '\\' {
			vt.resetEscapeParser()
		} else {
			vt.appendParserByte(&vt.escBuffer, b)
		}

	case stateCSIDiscard, stateOSCDiscard, stateDCSDiscard:
		vt.processParserDiscardByte(b)
	}
}

// processEscapeByte handles byte after ESC
func (vt *VirtualTerminal) processEscapeByte(b byte) {
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
		vt.saveCursor()
		vt.escState = stateNormal
	case '8': // Restore cursor (DECRC)
		vt.restoreCursor()
		vt.escState = stateNormal
	case 'c': // Reset (RIS)
		vt.initScreen()
		vt.resetCursorRegisters()
		vt.resetTerminalModes()
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
}

func (vt *VirtualTerminal) beginEscape() {
	vt.canJoinGrapheme = false
	vt.escState = stateEscape
	vt.escBuffer = nil
	vt.escParams = nil
	vt.escPrivate = 0
	vt.escRawSeq = nil
	vt.discardSawESC = false
}

func (vt *VirtualTerminal) resetEscapeParser() {
	vt.escState = stateNormal
	vt.escBuffer = nil
	vt.escParams = nil
	vt.escPrivate = 0
	vt.escRawSeq = nil
	vt.discardSawESC = false
}

// handleOSC processes an OSC (Operating System Command) sequence.
// Format: OSC Ps ; Pt BEL or OSC Ps ; Pt ST
// where Ps is the OSC code and Pt is the parameter text.
func (vt *VirtualTerminal) handleOSC(data []byte) {
	if vt.oscHandler == nil {
		return
	}

	// Parse OSC content: "code;param1;param2..."
	content := string(data)
	if content == "" {
		return
	}

	// Split into code and parameters
	parts := strings.SplitN(content, ";", 2)
	if len(parts) < 1 {
		return
	}

	oscType, err := strconv.Atoi(parts[0])
	if err != nil {
		return
	}

	var params []string
	if len(parts) > 1 {
		params = strings.Split(parts[1], ";")
	}

	// Call handler in a goroutine to avoid blocking PTY processing
	handler := vt.oscHandler
	safego.Go("osc-handler", func() { handler(oscType, params) })
}
