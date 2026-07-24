package vt

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestSnapshotCarriesPendingParserPrefix(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  []int
	}{
		{name: "normal", input: "text", want: nil},
		{name: "bare escape", input: "\x1b", want: []int{0x1b}},
		{name: "CSI", input: "\x1b[31", want: []int{0x1b, '[', '3', '1'}},
		{name: "OSC", input: "\x1b]0;partial", want: ints("\x1b]0;partial")},
		{name: "DCS", input: "\x1bP1;2qpartial", want: ints("\x1bP1;2qpartial")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			terminal := NewVirtualTerminal(20, 4, 100)
			terminal.Feed([]byte(test.input))
			if got := terminal.GetSnapshot().ParserPrefix; !reflect.DeepEqual(got, test.want) {
				t.Fatalf("parser prefix = %v, want %v", got, test.want)
			}
		})
	}
}

func TestSnapshotParserPrefixUsesJSONArrayAndContinuesCSI(t *testing.T) {
	source := NewVirtualTerminal(20, 4, 100)
	source.Feed([]byte("\x1b[31"))
	snapshot := source.GetSnapshot()

	encoded, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"parser_prefix":[27,91,51,49]`) {
		t.Fatalf("parser prefix was not encoded as a JSON byte array: %s", encoded)
	}

	client := NewVirtualTerminal(20, 4, 100)
	client.Feed([]byte("\x1b[999")) // Simulate a client parser interrupted at a different cut.
	client.Feed([]byte{0x18})       // Snapshot replay cancels that incomplete sequence.
	client.Feed(intBytes(snapshot.ParserPrefix))

	source.Feed([]byte("mX"))
	client.Feed([]byte("mX"))
	want := source.GetCellsRow(0)[0]
	got := client.GetCellsRow(0)[0]
	if got.Char != 'X' || got.Fg != want.Fg {
		t.Fatalf("continued CSI diverged: got %+v, want %+v", got, want)
	}
}

func TestSnapshotRestoresDECPrivateModes(t *testing.T) {
	terminal := NewVirtualTerminal(20, 4, 100)
	wantDefaults := []string{
		"\x1b[?1l", "\x1b[?7h", "\x1b[?25h", "\x1b[?1004l", "\x1b[?2004l", "\x1b[?2026l",
		"\x1b[?9l", "\x1b[?1000l", "\x1b[?1002l", "\x1b[?1003l", "\x1b[?1006l", "\x1b[?1016l",
	}
	assertModes(t, terminal.GetSnapshot(), true, wantDefaults)

	terminal.Feed([]byte("\x1b[?1h\x1b[?7l\x1b[?25l\x1b[?1004h\x1b[?2004h\x1b[?2026h"))
	wantEnabled := []string{
		"\x1b[?1h", "\x1b[?7l", "\x1b[?25l", "\x1b[?1004h", "\x1b[?2004h", "\x1b[?2026h",
		"\x1b[?9l", "\x1b[?1000l", "\x1b[?1002l", "\x1b[?1003l", "\x1b[?1006l", "\x1b[?1016l",
	}
	assertModes(t, terminal.GetSnapshot(), false, wantEnabled)

	terminal.Feed([]byte("\x1b[?1l\x1b[?7h\x1b[?25h\x1b[?1004l\x1b[?2004l\x1b[?2026l"))
	assertModes(t, terminal.GetSnapshot(), true, wantDefaults)
}

func TestSnapshotCanonicalizesExclusiveMouseModes(t *testing.T) {
	terminal := NewVirtualTerminal(20, 4, 100)
	terminal.Feed([]byte("\x1b[?1003h\x1b[?1002l\x1b[?1016h\x1b[?1006l"))
	if terminal.mouseTrackingMode != 0 || terminal.mouseEncodingMode != 0 {
		t.Fatalf("group DECRST did not clear active modes: tracking=%d encoding=%d",
			terminal.mouseTrackingMode, terminal.mouseEncodingMode)
	}
	terminal.Feed([]byte("\x1b[?1003l\x1b[?1002h\x1b[?1006l\x1b[?1016h"))
	snapshot := terminal.GetSnapshot()

	restored := NewVirtualTerminal(20, 4, 100)
	restored.Feed([]byte(strings.Join(snapshot.TerminalModes, "")))
	if restored.mouseTrackingMode != 1002 {
		t.Fatalf("mouse tracking replay = %d, want 1002", restored.mouseTrackingMode)
	}
	if restored.mouseEncodingMode != 1016 {
		t.Fatalf("mouse encoding replay = %d, want 1016", restored.mouseEncodingMode)
	}
	if !modeAppearsAfter(snapshot.TerminalModes, "\x1b[?1002h", "\x1b[?1003l") ||
		!modeAppearsAfter(snapshot.TerminalModes, "\x1b[?1016h", "\x1b[?1006l") {
		t.Fatalf("exclusive modes were not canonical reset-then-set: %q", snapshot.TerminalModes)
	}
}

func TestSnapshotTracksOnlyReplaySafePrivateModes(t *testing.T) {
	terminal := NewVirtualTerminal(20, 4, 100)
	terminal.Feed([]byte("normal\x1b[?1047halt"))
	snapshot := terminal.GetSnapshot()
	if !snapshot.IsAltScreen || snapshot.SerializedNormalContent == nil {
		t.Fatalf("1047 did not enter the alternate-screen model: %+v", snapshot)
	}
	if containsMode(snapshot.TerminalModes, "\x1b[?1047h") {
		t.Fatalf("alternate-screen transition leaked into replayable modes: %q", snapshot.TerminalModes)
	}

	terminal.Feed([]byte("\x1b[?1047l\x1b[2;4H\x1b[?1048h\x1b[4;9H\x1b[?1048l\x1b[?9999h"))
	snapshot = terminal.GetSnapshot()
	if snapshot.IsAltScreen || !strings.Contains(snapshot.SerializedContent, "normal") {
		t.Fatalf("1047 exit did not restore the normal buffer: %+v", snapshot)
	}
	if row, col := terminal.CursorPosition(); row != 1 || col != 3 {
		t.Fatalf("1048 cursor register restored (%d,%d), want (1,3)", row, col)
	}
	for _, forbidden := range []string{"\x1b[?1048h", "\x1b[?1048l", "\x1b[?9999h"} {
		if containsMode(snapshot.TerminalModes, forbidden) {
			t.Fatalf("unsafe private mode %q was made replayable: %q", forbidden, snapshot.TerminalModes)
		}
	}
}

func TestNewSnapshotExplicitlyDisablesStaleMouseTracking(t *testing.T) {
	snapshot := NewVirtualTerminal(20, 4, 100).GetSnapshot()
	for _, mode := range []int{9, 1000, 1002, 1003, 1006, 1016} {
		want := fmt.Sprintf("\x1b[?%dl", mode)
		if !containsMode(snapshot.TerminalModes, want) {
			t.Fatalf("new baseline does not clear mouse mode %d: %q", mode, snapshot.TerminalModes)
		}
	}
}

func TestPrivateCSIUDoesNotRestoreCursor(t *testing.T) {
	terminal := NewVirtualTerminal(20, 6, 100)
	terminal.Feed([]byte("\x1b[2;4H\x1b[s\x1b[4;9H"))
	wantRow, wantCol := terminal.CursorPosition()

	terminal.Feed([]byte("\x1b[>1u"))
	if row, col := terminal.CursorPosition(); row != wantRow || col != wantCol {
		t.Fatalf("private CSI u moved cursor to (%d,%d), want (%d,%d)", row, col, wantRow, wantCol)
	}
}

func TestRISSnapshotResetsPreviouslyObservedModes(t *testing.T) {
	terminal := NewVirtualTerminal(20, 4, 100)
	terminal.Feed([]byte("\x1b[?1000h\x1bc"))
	snapshot := terminal.GetSnapshot()

	if !containsMode(snapshot.TerminalModes, "\x1b[?1000l") {
		t.Fatalf("RIS snapshot forgot the disabled mouse mode: %q", snapshot.TerminalModes)
	}
}

func assertModes(t *testing.T, snapshot *TerminalSnapshot, cursorVisible bool, want []string) {
	t.Helper()
	if snapshot.CursorVisible != cursorVisible {
		t.Fatalf("cursor visibility = %v, want %v", snapshot.CursorVisible, cursorVisible)
	}
	if !reflect.DeepEqual(snapshot.TerminalModes, want) {
		t.Fatalf("terminal modes = %q, want %q", snapshot.TerminalModes, want)
	}
}

func containsMode(modes []string, want string) bool {
	for _, mode := range modes {
		if mode == want {
			return true
		}
	}
	return false
}

func modeAppearsAfter(modes []string, later, earlier string) bool {
	laterIndex, earlierIndex := -1, -1
	for index, mode := range modes {
		switch mode {
		case later:
			laterIndex = index
		case earlier:
			earlierIndex = index
		}
	}
	return earlierIndex >= 0 && laterIndex > earlierIndex
}

func ints(value string) []int {
	result := make([]int, len(value))
	for i, b := range []byte(value) {
		result[i] = int(b)
	}
	return result
}

func intBytes(value []int) []byte {
	result := make([]byte, len(value))
	for i, b := range value {
		result[i] = byte(b)
	}
	return result
}
