package vt

import "strings"

// screenLineLocked is the single plain-text projection of one terminal row.
// The styled cell plane owns width and combining-code-point semantics; screen
// remains only as a compatibility mirror for cells without an owned rune.
func (vt *VirtualTerminal) screenLineLocked(row int) string {
	if row < 0 || row >= vt.rows || row >= len(vt.cells) {
		return ""
	}
	var line strings.Builder
	for col, cell := range vt.cells[row] {
		if cell.IsPlaceholder() {
			continue
		}
		if text := cell.Text(); text != "" {
			line.WriteString(text)
			continue
		}
		if row < len(vt.screen) && col < len(vt.screen[row]) && vt.screen[row][col] != 0 {
			line.WriteRune(vt.screen[row][col])
		} else {
			line.WriteRune(' ')
		}
	}
	return strings.TrimRight(line.String(), " ")
}

// getLinesLocked projects the current screen to plain text. Caller holds vt.mu.
func (vt *VirtualTerminal) getLinesLocked() []string {
	lines := make([]string, vt.rows)
	for row := range lines {
		lines[row] = vt.screenLineLocked(row)
	}
	return lines
}
