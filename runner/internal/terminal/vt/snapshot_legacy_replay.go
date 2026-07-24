package vt

import "strings"

// legacySerializedContent keeps snapshot v1 consumers functional while the
// v2 fields carry the independently modeled buffers, modes, cursor registers,
// and parser prefix. Old clients ignore the v2 fields and replay this string as
// a self-contained terminal state; new clients use serialized_active_content.
func (snapshot *TerminalSnapshot) legacySerializedContent() string {
	const (
		cancelSequence  = "\x18"
		resetSGR        = "\x1b[0m"
		clearScreen     = "\x1b[2J\x1b[H"
		clearScrollback = "\x1b[3J"
		altExit         = "\x1b[?1049l"
		autowrapEnable  = "\x1b[?7h"
	)

	var replay strings.Builder
	replay.WriteString(cancelSequence)
	canonicalWrap := len(snapshot.TerminalModes) > 0
	appendBuffer := func(screenMode, content string, fullHistory bool) {
		replay.WriteString(screenMode)
		if fullHistory {
			replay.WriteString(clearScrollback)
		}
		replay.WriteString(resetSGR)
		replay.WriteString(clearScreen)
		if canonicalWrap {
			replay.WriteString(autowrapEnable)
		}
		replay.WriteString(content)
	}
	appendModes := func() {
		for _, mode := range snapshot.TerminalModes {
			replay.WriteString(mode)
		}
	}

	if snapshot.IsAltScreen {
		if snapshot.SerializedNormalContent != nil {
			appendBuffer(altExit, *snapshot.SerializedNormalContent, true)
			appendModes()
			if snapshot.SavedNormalCursorReplay != nil {
				replay.WriteString(*snapshot.SavedNormalCursorReplay)
			}
		}
		appendBuffer(altScreenEnterSequence(snapshot.AltScreenMode), snapshot.SerializedContent, false)
		appendModes()
	} else {
		appendBuffer(altExit, snapshot.SerializedContent, true)
		appendModes()
	}
	replay.WriteString(snapshot.SavedCursorReplay)
	replay.WriteString(snapshot.PrecedingJoinReplay)
	for _, value := range snapshot.ParserPrefix {
		if value >= 0 && value <= 255 {
			replay.WriteByte(byte(value))
		}
	}
	return replay.String()
}

func altScreenEnterSequence(mode int) string {
	switch mode {
	case 47:
		return "\x1b[?47h"
	case 1047:
		return "\x1b[?1047h"
	default:
		return "\x1b[?1049h"
	}
}
