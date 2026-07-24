package vt

import (
	"fmt"
	"strings"
	"testing"
)

func TestSnapshotNormalBufferIncludesRetainedHistory(t *testing.T) {
	vt := NewVirtualTerminal(20, 3, 100)
	vt.Feed([]byte("history-only\r\nvisible-one\r\nvisible-two\r\nvisible-three"))

	if vt.GetHistoryStyledLength() == 0 {
		t.Fatal("test setup did not produce scrollback history")
	}

	snapshot := vt.GetSnapshot()
	if snapshot.IsAltScreen {
		t.Fatal("normal-buffer snapshot reported alternate-screen mode")
	}
	if snapshot.SerializedNormalContent != nil {
		t.Fatal("normal-buffer snapshot unexpectedly carried a second normal buffer")
	}
	if !strings.Contains(snapshot.SerializedContent, "history-only") {
		t.Fatalf("snapshot omitted retained scrollback history: %q", snapshot.SerializedContent)
	}
	if !containsAll(snapshot.SerializedContent, "visible-one", "visible-two", "visible-three") {
		t.Fatalf("snapshot omitted visible normal-buffer content: %q", snapshot.SerializedContent)
	}

	restored := NewVirtualTerminal(snapshot.Cols, snapshot.Rows, 100)
	restored.Feed([]byte(snapshot.SerializedContent))
	if got := restored.GetDisplay(); got != vt.GetDisplay() {
		t.Fatalf("normal-buffer snapshot round-trip mismatch:\noriginal: %q\nrestored: %q\nserialized: %q",
			vt.GetDisplay(), got, snapshot.SerializedContent)
	}
	if got := restored.GetOutput(100); got != vt.GetOutput(100) {
		t.Fatalf("normal-buffer full snapshot output mismatch:\noriginal: %q\nrestored: %q",
			vt.GetOutput(100), got)
	}
}

func TestSnapshotAltBufferCarriesFullHiddenMainBuffer(t *testing.T) {
	vt := NewVirtualTerminal(20, 3, 100)
	vt.Feed([]byte("main-history\r\nmain-one\r\nmain-two\r\nmain-three"))
	wantNormalDisplay := vt.GetDisplay()
	vt.Feed([]byte("\x1b[?1049h"))
	vt.Feed([]byte("alt-one\r\nalt-two"))

	snapshot := vt.GetSnapshot()
	if snapshot.SnapshotVersion != 2 {
		t.Fatalf("snapshot version = %d, want 2", snapshot.SnapshotVersion)
	}
	if !snapshot.IsAltScreen {
		t.Fatal("alternate-buffer snapshot did not report alternate-screen mode")
	}
	if strings.Contains(snapshot.SerializedContent, "main-") {
		t.Fatalf("alternate-buffer snapshot leaked main-buffer content: %q", snapshot.SerializedContent)
	}
	if !containsAll(snapshot.SerializedContent, "alt-one", "alt-two") {
		t.Fatalf("snapshot omitted visible alternate-buffer content: %q", snapshot.SerializedContent)
	}
	if strings.Contains(snapshot.SerializedContent, "\x1b[?1049h") {
		t.Fatalf("serialized content must not duplicate the separate buffer-mode field: %q", snapshot.SerializedContent)
	}
	if snapshot.SerializedNormalContent == nil {
		t.Fatal("alternate-buffer snapshot omitted hidden normal buffer")
	}
	if !strings.Contains(*snapshot.SerializedNormalContent, "main-history") {
		t.Fatalf("hidden normal snapshot omitted retained scrollback history: %q", *snapshot.SerializedNormalContent)
	}
	if !containsAll(*snapshot.SerializedNormalContent, "main-one", "main-two", "main-three") {
		t.Fatalf("hidden normal snapshot omitted visible content: %q", *snapshot.SerializedNormalContent)
	}
	if strings.Contains(*snapshot.SerializedNormalContent, "alt-") {
		t.Fatalf("hidden normal snapshot leaked alternate-buffer content: %q", *snapshot.SerializedNormalContent)
	}
	if !containsAll(snapshot.LegacySerializedContent, "main-one", "main-two", "main-three", "alt-one", "alt-two") {
		t.Fatalf("v1 compatibility replay is not self-contained: %q", snapshot.LegacySerializedContent)
	}
	if !strings.Contains(snapshot.LegacySerializedContent, "\x1b[?1049h") {
		t.Fatalf("v1 compatibility replay omitted alternate-screen activation: %q", snapshot.LegacySerializedContent)
	}

	restored := NewVirtualTerminal(snapshot.Cols, snapshot.Rows, 100)
	replayTerminalSnapshot(restored, snapshot)
	if !restored.IsAltScreen() {
		t.Fatal("restored terminal is not in alternate-screen mode")
	}
	if got := restored.GetDisplay(); got != vt.GetDisplay() {
		t.Fatalf("alternate-buffer snapshot round-trip mismatch:\noriginal: %q\nrestored: %q\nserialized: %q",
			vt.GetDisplay(), got, snapshot.SerializedContent)
	}
	restored.Feed([]byte("\x1b[?1049l"))
	if got := restored.GetDisplay(); got != wantNormalDisplay {
		t.Fatalf("fresh-client alternate-screen exit did not restore hidden normal buffer:\nwant: %q\ngot:  %q\nserialized normal: %q",
			wantNormalDisplay, got, *snapshot.SerializedNormalContent)
	}

	legacy := NewVirtualTerminal(snapshot.Cols, snapshot.Rows, 100)
	legacy.Feed([]byte(snapshot.LegacySerializedContent))
	if !legacy.IsAltScreen() || legacy.GetDisplay() != vt.GetDisplay() {
		t.Fatalf("v1 compatibility replay did not restore active alt buffer: alt=%v display=%q",
			legacy.IsAltScreen(), legacy.GetDisplay())
	}
	legacy.Feed([]byte("\x1b[?1049l"))
	if got := legacy.GetDisplay(); got != wantNormalDisplay {
		t.Fatalf("v1 compatibility replay did not restore hidden normal buffer: got %q want %q", got, wantNormalDisplay)
	}
}

func TestSnapshotBlankScreenPreservesCursor(t *testing.T) {
	vt := NewVirtualTerminal(20, 3, 100)
	vt.Feed([]byte("\x1b[2J\x1b[2;7H"))

	snapshot := vt.GetSnapshot()
	if snapshot.SerializedContent == "" {
		t.Fatal("blank snapshot omitted cursor state")
	}

	restored := NewVirtualTerminal(snapshot.Cols, snapshot.Rows, 100)
	restored.Feed([]byte(snapshot.SerializedContent))
	wantRow, wantCol := vt.CursorPosition()
	gotRow, gotCol := restored.CursorPosition()
	if gotRow != wantRow || gotCol != wantCol {
		t.Fatalf("cursor mismatch: got (%d, %d), want (%d, %d); serialized: %q",
			gotRow, gotCol, wantRow, wantCol, snapshot.SerializedContent)
	}
}

func TestSnapshotAltBufferResizePreservesBuffersConsistently(t *testing.T) {
	vt := NewVirtualTerminal(8, 4, 100)
	vt.Feed([]byte("main界界"))
	vt.Feed([]byte("\x1b[?1049h"))
	vt.Feed([]byte("alt界界"))

	vt.Resize(5, 3)
	if vt.Cols() != 5 || vt.Rows() != 3 {
		t.Fatalf("resize dimensions: got %dx%d, want 5x3", vt.Cols(), vt.Rows())
	}
	if !vt.IsAltScreen() {
		t.Fatal("resize left alternate-screen mode")
	}
	if got := vt.GetDisplay(); !strings.Contains(got, "alt界") {
		t.Fatalf("resize did not safely preserve the visible alternate buffer: %q", got)
	}
	wantAltDisplay := vt.GetDisplay()
	if len(vt.screen) != 3 || len(vt.altScreen) != 3 || len(vt.screen[0]) != 5 || len(vt.altScreen[0]) != 5 {
		t.Fatalf("alternate buffer dimensions are inconsistent after resize: screen=%dx%d alt=%dx%d",
			len(vt.screen[0]), len(vt.screen), len(vt.altScreen[0]), len(vt.altScreen))
	}
	if &vt.screen[0][0] != &vt.altScreen[0][0] || &vt.cells[0][0] != &vt.altCells[0][0] {
		t.Fatal("active screen no longer aliases the alternate buffer after resize")
	}
	if len(vt.isWrapped) != 3 || len(vt.altIsWrapped) != 3 || len(vt.savedMainWrapped) != 3 {
		t.Fatalf("wrap flags have stale dimensions after resize: active=%d alt=%d main=%d",
			len(vt.isWrapped), len(vt.altIsWrapped), len(vt.savedMainWrapped))
	}
	if &vt.isWrapped[0] != &vt.altIsWrapped[0] {
		t.Fatal("active wrap flags no longer alias the alternate buffer after resize")
	}

	snapshot := vt.GetSnapshot()
	if !snapshot.IsAltScreen || !strings.Contains(snapshot.SerializedContent, "alt界") {
		t.Fatalf("resized alternate snapshot is invalid: %+v", snapshot)
	}
	if snapshot.SerializedNormalContent == nil || !strings.Contains(*snapshot.SerializedNormalContent, "main") {
		t.Fatalf("resized hidden normal snapshot lost preserved content: %+v", snapshot.SerializedNormalContent)
	}

	restored := NewVirtualTerminal(snapshot.Cols, snapshot.Rows, 100)
	replayTerminalSnapshot(restored, snapshot)
	if got := restored.GetDisplay(); got != wantAltDisplay {
		t.Fatalf("resized alternate snapshot did not restore preserved content: %q", got)
	}
	restored.Feed([]byte("\x1b[?1049l"))
	if got := restored.GetDisplay(); got != "main" {
		t.Fatalf("resized alternate snapshot did not restore hidden normal content: %q", got)
	}

	vt.Feed([]byte("\x1b[?1049l"))
	if vt.IsAltScreen() {
		t.Fatal("failed to leave alternate-screen mode after resize")
	}
	if vt.Cols() != 5 || vt.Rows() != 3 || len(vt.screen) != 3 || len(vt.screen[0]) != 5 {
		t.Fatalf("main buffer restored with stale dimensions: cols=%d rows=%d buffer=%dx%d",
			vt.Cols(), vt.Rows(), len(vt.screen[0]), len(vt.screen))
	}
	if got := vt.GetDisplay(); got != "main" {
		t.Fatalf("resize did not restore preserved hidden main-buffer content: got %q", got)
	}
}

func TestAlternateScreenScrollDoesNotPolluteNormalHistory(t *testing.T) {
	vt := NewVirtualTerminal(8, 2, 100)
	vt.Feed([]byte("main"))
	vt.Feed([]byte("\x1b[?1049h\x1b[2J\x1b[Halt-one\r\nalt-two\r\nalt-three"))
	if got := vt.GetHistoryStyledLength(); got != 0 {
		t.Fatalf("alternate-screen scroll added %d normal history rows", got)
	}

	vt.Feed([]byte("\x1b[?1049l"))
	if got := vt.GetOutput(100); got != "main" {
		t.Fatalf("alternate-screen scroll polluted normal output history: %q", got)
	}
}

func TestSnapshotAltBufferHasIndependentWrapState(t *testing.T) {
	vt := NewVirtualTerminal(5, 4, 100)
	vt.Feed([]byte("12345A"))
	if !vt.IsLineWrapped(1) {
		t.Fatal("test setup did not wrap the main buffer")
	}

	vt.Feed([]byte("\x1b[?1049h"))
	if vt.IsLineWrapped(1) {
		t.Fatal("alternate buffer inherited main-buffer wrap flags")
	}
	vt.Feed([]byte("one\r\ntwo"))

	snapshot := vt.GetSnapshot()
	restored := NewVirtualTerminal(snapshot.Cols, snapshot.Rows, 100)
	restored.Feed([]byte("\x1b[?1049h"))
	restored.Feed([]byte(snapshot.SerializedContent))
	if got := restored.GetDisplay(); got != vt.GetDisplay() {
		t.Fatalf("alternate snapshot used polluted wrap state:\noriginal: %q\nrestored: %q\nserialized: %q",
			vt.GetDisplay(), got, snapshot.SerializedContent)
	}

	vt.Feed([]byte("\x1b[?1049l"))
	if !vt.IsLineWrapped(1) {
		t.Fatal("exiting alternate screen did not restore main-buffer wrap flags")
	}
}

func TestSnapshotAltBufferRestoresHiddenNormalStyle(t *testing.T) {
	vt := NewVirtualTerminal(20, 3, 100)
	vt.Feed([]byte("\x1b[31mnormal"))
	vt.Feed([]byte("\x1b[?1049h\x1b[32malt"))

	snapshot := vt.GetSnapshot()
	restored := NewVirtualTerminal(snapshot.Cols, snapshot.Rows, 100)
	replayTerminalSnapshot(restored, snapshot)
	restored.Feed([]byte("\x1b[?1049lX"))

	cell := restored.GetCellsRow(0)[6]
	if cell.Char != 'X' || !cell.Fg.IsPalette() || cell.Fg.Index() != 1 {
		t.Fatalf("hidden normal SGR was not restored after leaving alt: %+v", cell)
	}
}

func replayTerminalSnapshot(target *VirtualTerminal, snapshot *TerminalSnapshot) {
	const resetAndClear = "\x1b[?7h\x1b[0m\x1b[2J\x1b[H"
	const resetNormal = "\x1b[?1049l\x1b[3J" + resetAndClear
	replayModes := func() {
		target.Feed([]byte(strings.Join(snapshot.TerminalModes, "")))
	}
	replayCursor := func(value string) {
		target.Feed([]byte(value))
	}
	target.Feed([]byte{0x18})
	if snapshot.IsAltScreen {
		if snapshot.SerializedNormalContent != nil {
			target.Feed([]byte(resetNormal + *snapshot.SerializedNormalContent))
			replayModes()
			if snapshot.SavedNormalCursorReplay != nil {
				replayCursor(*snapshot.SavedNormalCursorReplay)
			}
		}
		mode := snapshot.AltScreenMode
		if mode != 47 && mode != 1047 && mode != 1049 {
			mode = 1049
		}
		target.Feed([]byte(fmt.Sprintf("\x1b[?%dh%s%s", mode, resetAndClear, snapshot.SerializedContent)))
		replayModes()
		replayCursor(snapshot.SavedCursorReplay)
		target.Feed([]byte(snapshot.PrecedingJoinReplay))
		target.Feed(intBytes(snapshot.ParserPrefix))
		return
	}
	target.Feed([]byte(resetNormal + snapshot.SerializedContent))
	replayModes()
	replayCursor(snapshot.SavedCursorReplay)
	target.Feed([]byte(snapshot.PrecedingJoinReplay))
	target.Feed(intBytes(snapshot.ParserPrefix))
}
