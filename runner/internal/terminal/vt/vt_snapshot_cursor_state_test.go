package vt

import (
	"fmt"
	"testing"
)

func TestAlternateScreenExitRestoresModeSpecificCursorAndStyle(t *testing.T) {
	tests := []struct {
		mode      int
		wantRow   int
		wantCol   int
		wantColor uint8
	}{
		{mode: 47, wantRow: 3, wantCol: 9, wantColor: 2},
		{mode: 1047, wantRow: 3, wantCol: 9, wantColor: 2},
		{mode: 1049, wantRow: 1, wantCol: 2, wantColor: 1},
	}

	for _, test := range tests {
		t.Run(fmt.Sprintf("mode_%d", test.mode), func(t *testing.T) {
			terminal := NewVirtualTerminal(20, 6, 100)
			terminal.Feed([]byte("\x1b[2;3H\x1b[31m"))
			terminal.Feed([]byte(fmt.Sprintf("\x1b[?%dh", test.mode)))
			terminal.Feed([]byte("\x1b[4;10H\x1b[32m"))

			snapshot := terminal.GetSnapshot()
			if snapshot.AltScreenMode != test.mode {
				t.Fatalf("alternate mode = %d, want %d", snapshot.AltScreenMode, test.mode)
			}

			restored := NewVirtualTerminal(snapshot.Cols, snapshot.Rows, 100)
			replayTerminalSnapshot(restored, snapshot)
			for _, candidate := range []*VirtualTerminal{terminal, restored} {
				candidate.Feed([]byte(fmt.Sprintf("\x1b[?%dlX", test.mode)))
				assertPaletteCell(t, candidate, test.wantRow, test.wantCol, 'X', test.wantColor)
			}
			if got, want := restored.GetDisplay(), terminal.GetDisplay(); got != want {
				t.Fatalf("snapshot exit display = %q, want %q", got, want)
			}
		})
	}
}

func TestSnapshotRoundTripPreservesDECAWMRightMarginState(t *testing.T) {
	t.Run("enabled wraps", func(t *testing.T) {
		source := NewVirtualTerminal(5, 3, 100)
		source.Feed([]byte("ABCDE"))
		restored := replaySnapshotToFreshTerminal(source)

		assertCursor(t, source, 0, 5)
		assertCursor(t, restored, 0, 5)
		source.Feed([]byte("Q"))
		restored.Feed([]byte("Q"))
		if got := restored.GetDisplay(); got != "ABCDE\nQ" || got != source.GetDisplay() {
			t.Fatalf("enabled DECAWM replay = %q, source %q", got, source.GetDisplay())
		}
	})

	t.Run("disabled overwrites and retains margin state", func(t *testing.T) {
		source := NewVirtualTerminal(5, 3, 100)
		source.Feed([]byte("\x1b[?7lABCDEZ"))
		restored := replaySnapshotToFreshTerminal(source)

		assertCursor(t, source, 0, 5)
		assertCursor(t, restored, 0, 5)
		source.Feed([]byte("Q"))
		restored.Feed([]byte("Q"))
		if got := restored.GetDisplay(); got != "ABCDQ" || got != source.GetDisplay() {
			t.Fatalf("disabled DECAWM replay = %q, source %q", got, source.GetDisplay())
		}

		// xterm retains the right-margin state while DECAWM is disabled. Once
		// enabled again, the next printable character performs the delayed wrap.
		source.Feed([]byte("\x1b[?7hR"))
		restored.Feed([]byte("\x1b[?7hR"))
		if got := restored.GetDisplay(); got != "ABCDQ\nR" || got != source.GetDisplay() {
			t.Fatalf("re-enabled DECAWM replay = %q, source %q", got, source.GetDisplay())
		}
	})
}

func TestDECAWMWideCharacterAtRightMarginMatchesXterm(t *testing.T) {
	disabled := NewVirtualTerminal(5, 3, 100)
	disabled.Feed([]byte("\x1b[?7lABCD界"))
	if got := disabled.GetDisplay(); got != "ABCD" {
		t.Fatalf("disabled DECAWM accepted overflowing wide character: %q", got)
	}
	assertCursor(t, disabled, 0, 4)

	enabled := NewVirtualTerminal(5, 3, 100)
	enabled.Feed([]byte("ABCD界"))
	if got := enabled.GetDisplay(); got != "ABCD\n界" {
		t.Fatalf("enabled DECAWM wide-character wrap = %q", got)
	}
	if !enabled.IsLineWrapped(1) {
		t.Fatal("wide-character overflow did not mark the continuation row wrapped")
	}
}

func TestSnapshotRoundTripPreservesSavedCursorRegister(t *testing.T) {
	source := NewVirtualTerminal(20, 6, 100)
	source.Feed([]byte("\x1b[2;3H\x1b[31m\x1b[?1048h"))
	source.Feed([]byte("\x1b[4;10H\x1b[32m"))
	restored := replaySnapshotToFreshTerminal(source)

	for _, candidate := range []*VirtualTerminal{source, restored} {
		candidate.Feed([]byte("\x1b[?1048lX"))
		assertPaletteCell(t, candidate, 1, 2, 'X', 1)
	}
	if got, want := restored.GetDisplay(), source.GetDisplay(); got != want {
		t.Fatalf("saved cursor replay display = %q, want %q", got, want)
	}
}

func TestResizeClampsSavedNormalCursorLikeXterm(t *testing.T) {
	source := NewVirtualTerminal(10, 6, 100)
	source.Feed([]byte("ABCDEFGHIJ"))
	source.Feed([]byte("\x1b[1;9H\x1b[31m\x1b7"))
	source.Feed([]byte("\x1b[4;1H\x1b[32m"))
	source.Resize(5, 6)
	restored := replaySnapshotToFreshTerminal(source)

	for _, candidate := range []*VirtualTerminal{source, restored} {
		candidate.Feed([]byte("\x1b8X"))
		assertPaletteCell(t, candidate, 1, 4, 'X', 1)
	}
	if got, want := restored.GetDisplay(), source.GetDisplay(); got != want {
		t.Fatalf("resized saved cursor replay = %q, want %q", got, want)
	}
}

func TestSavedNormalCursorUsesAbsoluteBufferYAcrossScrollAndSnapshot(t *testing.T) {
	source := NewVirtualTerminal(10, 3, 100)
	source.Feed([]byte("\x1b[3;2H\x1b7\r\n"))
	restored := replaySnapshotToFreshTerminal(source)

	for _, candidate := range []*VirtualTerminal{source, restored} {
		candidate.Feed([]byte("\x1b8X"))
		cell := candidate.GetCellsRow(1)[1]
		if cell.Char != 'X' {
			t.Fatalf("saved cursor after scroll wrote %+v at row 2 column 2", cell)
		}
		assertCursor(t, candidate, 1, 2)
	}
	if got, want := restored.GetDisplay(), source.GetDisplay(); got != want {
		t.Fatalf("absolute saved cursor replay display = %q, want %q", got, want)
	}
}

func TestSavedNormalCursorMatchesXtermWhenFullScrollbackRecycles(t *testing.T) {
	source := NewVirtualTerminal(8, 3, 1)
	source.Feed([]byte("\x1b[3;1H\x1b7A\r\nB\r\n"))
	restored := replaySnapshotToFreshTerminal(source)

	for _, candidate := range []*VirtualTerminal{source, restored} {
		candidate.Feed([]byte("\x1b8X"))
		if cell := candidate.GetCellsRow(1)[0]; cell.Char != 'X' {
			t.Fatalf("recycled scrollback saved cursor wrote %+v, want row 2", cell)
		}
	}
}

func TestResizeHistoryTrimAdjustsAbsoluteSavedCursor(t *testing.T) {
	terminal := NewVirtualTerminal(8, 3, 1)
	terminal.Feed([]byte("one\r\ntwo\r\nthree\r\nfour\x1b7"))
	if got := terminal.savedCursor[0].y; got != 3 {
		t.Fatalf("saved absolute Y before resize = %d, want 3", got)
	}

	terminal.Resize(8, 2)
	if got := terminal.savedCursor[0].y; got != 2 {
		t.Fatalf("saved absolute Y after max-history resize trim = %d, want 2", got)
	}
}

func TestResizeMapsBothBufferLocalSavedCursorRegisters(t *testing.T) {
	source := NewVirtualTerminal(10, 4, 100)
	source.Feed([]byte("\x1b[3;8H\x1b[31m\x1b7\x1b[4;1H"))
	source.Feed([]byte("\x1b[?47h\x1b[3;9H\x1b[32m\x1b7\x1b[4;1H"))
	source.Resize(6, 3)
	restored := replaySnapshotToFreshTerminal(source)

	for _, candidate := range []*VirtualTerminal{source, restored} {
		candidate.Feed([]byte("\x1b8A"))
		assertPaletteCell(t, candidate, 1, 5, 'A', 2)
		candidate.Feed([]byte("\x1b[?47l\x1b8M"))
		assertPaletteCell(t, candidate, 1, 5, 'M', 1)
	}
	if got, want := restored.GetDisplay(), source.GetDisplay(); got != want {
		t.Fatalf("resized buffer-local cursors = %q, want %q", got, want)
	}
}

func TestSnapshotPreservesBufferLocalSavedCursorRegisters(t *testing.T) {
	for _, mode := range []int{47, 1047} {
		t.Run(fmt.Sprintf("mode_%d", mode), func(t *testing.T) {
			source := NewVirtualTerminal(20, 6, 100)
			source.Feed([]byte("\x1b[2;3H\x1b[31m\x1b7"))
			source.Feed([]byte(fmt.Sprintf("\x1b[?%dh", mode)))
			source.Feed([]byte("\x1b[4;10H\x1b[32m\x1b7\x1b[1;1H\x1b[34m"))
			restored := replaySnapshotToFreshTerminal(source)

			for _, candidate := range []*VirtualTerminal{source, restored} {
				candidate.Feed([]byte("\x1b8A"))
				assertPaletteCell(t, candidate, 3, 9, 'A', 2)
				candidate.Feed([]byte(fmt.Sprintf("\x1b[?%dl\x1b8M", mode)))
				assertPaletteCell(t, candidate, 1, 2, 'M', 1)
			}
			if got, want := restored.GetDisplay(), source.GetDisplay(); got != want {
				t.Fatalf("buffer-local register replay = %q, want %q", got, want)
			}
		})
	}
}

func TestSavedRightMarginCursorRestoresToLastColumn(t *testing.T) {
	source := NewVirtualTerminal(5, 3, 100)
	source.Feed([]byte("\x1b[?7lABCDE\x1b7\x1b[3;3H"))
	restored := replaySnapshotToFreshTerminal(source)

	for _, candidate := range []*VirtualTerminal{source, restored} {
		candidate.Feed([]byte("\x1b8Q"))
		if got := candidate.GetDisplay(); got != "ABCDQ" {
			t.Fatalf("saved right-margin cursor replay = %q, want %q", got, "ABCDQ")
		}
		assertCursor(t, candidate, 0, 5)
	}
}

func replaySnapshotToFreshTerminal(source *VirtualTerminal) *VirtualTerminal {
	snapshot := source.GetSnapshot()
	restored := NewVirtualTerminal(snapshot.Cols, snapshot.Rows, 100)
	replayTerminalSnapshot(restored, snapshot)
	return restored
}

func assertPaletteCell(
	t *testing.T,
	terminal *VirtualTerminal,
	row, col int,
	wantChar rune,
	wantColor uint8,
) {
	t.Helper()
	cell := terminal.GetCellsRow(row)[col]
	if cell.Char != wantChar || !cell.Fg.IsPalette() || cell.Fg.Index() != wantColor {
		t.Fatalf("cell (%d,%d) = %+v, want %q with palette color %d", row, col, cell, wantChar, wantColor)
	}
}

func assertCursor(t *testing.T, terminal *VirtualTerminal, wantRow, wantCol int) {
	t.Helper()
	row, col := terminal.CursorPosition()
	if row != wantRow || col != wantCol {
		t.Fatalf("cursor = (%d,%d), want (%d,%d)", row, col, wantRow, wantCol)
	}
}
