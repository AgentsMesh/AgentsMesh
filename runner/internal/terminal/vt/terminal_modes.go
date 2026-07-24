package vt

import (
	"fmt"
	"sort"
)

var defaultPrivateModes = map[int]bool{
	1:    false, // DECCKM: normal cursor keys
	7:    true,  // DECAWM: auto wrap
	25:   true,  // DECTCEM: visible cursor
	1004: false, // focus reporting
	2004: false, // bracketed paste
	2026: false, // synchronized output
}

var mouseTrackingModes = []int{9, 1000, 1002, 1003}
var mouseEncodingModes = []int{1006, 1016}

func (vt *VirtualTerminal) resetTerminalModes() {
	vt.privateModes = make(map[int]bool, len(defaultPrivateModes))
	for mode, enabled := range defaultPrivateModes {
		vt.privateModes[mode] = enabled
	}
	vt.mouseTrackingMode = 0
	vt.mouseEncodingMode = 0
}

func (vt *VirtualTerminal) handlePrivateMode(set bool) {
	for _, mode := range vt.escParams {
		switch mode {
		case 47, 1047, 1049:
			if set {
				vt.enterAltScreen(mode)
			} else {
				vt.exitAltScreen(mode)
			}
		case 1048:
			if set {
				vt.saveCursor()
			} else {
				vt.restoreCursor()
			}
		case 9, 1000, 1002, 1003:
			setExclusiveMode(&vt.mouseTrackingMode, mode, set)
		case 1006, 1016:
			setExclusiveMode(&vt.mouseEncodingMode, mode, set)
		default:
			// Only replay modes with known idempotent, long-lived state semantics.
			// Unknown DEC modes and action modes such as 1048 must not run after
			// framebuffer restoration because they can switch buffers or move cursors.
			if _, tracked := vt.privateModes[mode]; tracked {
				vt.privateModes[mode] = set
			}
		}
	}
}

func setExclusiveMode(active *int, mode int, set bool) {
	if set {
		*active = mode
	} else {
		// xterm treats DECRST for any member as a reset of the whole
		// mutually-exclusive group, even when another member is active.
		*active = 0
	}
}

func (vt *VirtualTerminal) terminalModeSequences() []string {
	keys := make([]int, 0, len(vt.privateModes))
	for mode := range vt.privateModes {
		keys = append(keys, mode)
	}
	sort.Ints(keys)

	sequences := make([]string, 0, len(keys)+len(mouseTrackingModes)+len(mouseEncodingModes)+2)
	for _, mode := range keys {
		final := 'l'
		if vt.privateModes[mode] {
			final = 'h'
		}
		sequences = append(sequences, fmt.Sprintf("\x1b[?%d%c", mode, final))
	}
	sequences = appendExclusiveModeSequences(sequences, mouseTrackingModes, vt.mouseTrackingMode)
	return appendExclusiveModeSequences(sequences, mouseEncodingModes, vt.mouseEncodingMode)
}

func appendExclusiveModeSequences(sequences []string, modes []int, active int) []string {
	for _, mode := range modes {
		sequences = append(sequences, fmt.Sprintf("\x1b[?%dl", mode))
	}
	if active != 0 {
		sequences = append(sequences, fmt.Sprintf("\x1b[?%dh", active))
	}
	return sequences
}
