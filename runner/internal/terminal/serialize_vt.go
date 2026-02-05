package terminal

import (
	"strings"
)

// Serialize returns the terminal state as an ANSI sequence string.
func (vt *VirtualTerminal) Serialize(opts SerializeOptions) string {
	vt.mu.RLock()
	defer vt.mu.RUnlock()

	if !vt.hasData {
		return ""
	}

	historyLen := len(vt.historyStyled)
	totalRows := historyLen + vt.rows

	var startRow, endRow int
	if opts.Range != nil {
		startRow = opts.Range.Start
		endRow = opts.Range.End
		if startRow < 0 {
			startRow = 0
		}
		if endRow >= totalRows {
			endRow = totalRows - 1
		}
	} else {
		if opts.ScrollbackLines > 0 && historyLen > opts.ScrollbackLines {
			startRow = historyLen - opts.ScrollbackLines
		} else {
			startRow = 0
		}
		endRow = totalRows - 1
	}

	if startRow > endRow {
		return ""
	}

	handler := newStringSerializeHandler(vt)
	return handler.serializeWithHistory(startRow, endRow, historyLen, false)
}

// serializeNoLock is like Serialize but assumes the lock is already held.
func (vt *VirtualTerminal) serializeNoLock(opts SerializeOptions) string {
	if !vt.hasData {
		return ""
	}

	historyLen := len(vt.historyStyled)
	totalRows := historyLen + vt.rows

	var startRow, endRow int
	if opts.Range != nil {
		startRow = opts.Range.Start
		endRow = opts.Range.End
		if startRow < 0 {
			startRow = 0
		}
		if endRow >= totalRows {
			endRow = totalRows - 1
		}
	} else {
		if opts.ScrollbackLines > 0 && historyLen > opts.ScrollbackLines {
			startRow = historyLen - opts.ScrollbackLines
		} else {
			startRow = 0
		}
		endRow = totalRows - 1
	}

	if startRow > endRow {
		return ""
	}

	handler := newStringSerializeHandler(vt)
	return handler.serializeWithHistory(startRow, endRow, historyLen, false)
}

// SerializeSimple returns a simple serialization without style information.
func (vt *VirtualTerminal) SerializeSimple(scrollbackLines int) string {
	vt.mu.RLock()
	defer vt.mu.RUnlock()

	if !vt.hasData {
		return ""
	}

	var buf strings.Builder

	historyStart := 0
	if scrollbackLines > 0 && len(vt.history) > scrollbackLines {
		historyStart = len(vt.history) - scrollbackLines
	}

	for i := historyStart; i < len(vt.history); i++ {
		buf.WriteString(vt.history[i])
		buf.WriteString("\r\n")
	}

	for row := 0; row < vt.rows; row++ {
		line := strings.TrimRight(string(vt.screen[row]), " ")
		buf.WriteString(line)
		if row < vt.rows-1 {
			buf.WriteString("\r\n")
		}
	}

	return buf.String()
}
