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

func TestSnapshotNormalBufferResizePreservesContentAndStyle(t *testing.T) {
	vt := NewVirtualTerminal(8, 3, 100)
	vt.Feed([]byte("\x1b[31mhello界\x1b[0m"))

	vt.Resize(12, 5)
	if got := vt.GetDisplay(); got != "hello界" {
		t.Fatalf("growing resize lost normal-buffer content: %q", got)
	}
	cell := vt.GetCellsRow(0)[0]
	if !cell.Fg.IsPalette() || cell.Fg.Index() != 1 {
		t.Fatalf("growing resize lost cell style: %+v", cell)
	}

	snapshot := vt.GetSnapshot()
	restored := NewVirtualTerminal(snapshot.Cols, snapshot.Rows, 100)
	replayTerminalSnapshot(restored, snapshot)
	if got := restored.GetDisplay(); got != "hello界" {
		t.Fatalf("resized normal snapshot did not replay content: %q", got)
	}
}

func TestResizeGrowingRowsPullsNormalHistoryIntoViewport(t *testing.T) {
	vt := NewVirtualTerminal(8, 3, 100)
	vt.Feed([]byte("one\r\ntwo\r\nthree\r\nfour\r\nfive"))
	if vt.GetHistoryStyledLength() != 2 {
		t.Fatalf("test setup history length = %d, want 2", vt.GetHistoryStyledLength())
	}

	vt.Resize(8, 5)
	if got := vt.GetDisplay(); got != "one\ntwo\nthree\nfour\nfive" {
		t.Fatalf("row growth did not expose recent normal history: %q", got)
	}
	if vt.GetHistoryStyledLength() != 0 {
		t.Fatalf("row growth left pulled lines in history: %d", vt.GetHistoryStyledLength())
	}
	assertCursor(t, vt, 4, 4)

	restored := replaySnapshotToFreshTerminal(vt)
	if got := restored.GetDisplay(); got != vt.GetDisplay() {
		t.Fatalf("grown normal snapshot display = %q, want %q", got, vt.GetDisplay())
	}
	assertCursor(t, restored, 4, 4)
}

func TestResizeGrowingHiddenMainPullsHistory(t *testing.T) {
	vt := NewVirtualTerminal(8, 3, 100)
	vt.Feed([]byte("one\r\ntwo\r\nthree\r\nfour\r\nfive"))
	vt.Feed([]byte("\x1b[?1049h\x1b[2J\x1b[Halt"))

	vt.Resize(8, 5)
	snapshot := vt.GetSnapshot()
	restored := NewVirtualTerminal(snapshot.Cols, snapshot.Rows, 100)
	replayTerminalSnapshot(restored, snapshot)
	for _, candidate := range []*VirtualTerminal{vt, restored} {
		if got := candidate.GetDisplay(); got != "alt" {
			t.Fatalf("active alt display after hidden-main growth = %q", got)
		}
		candidate.Feed([]byte("\x1b[?1049l"))
		if got := candidate.GetDisplay(); got != "one\ntwo\nthree\nfour\nfive" {
			t.Fatalf("hidden main did not expose history after growth: %q", got)
		}
		assertCursor(t, candidate, 4, 4)
	}
}

func TestSnapshotNormalBufferResizeMovesDroppedRowsToHistory(t *testing.T) {
	vt := NewVirtualTerminal(12, 4, 100)
	vt.Feed([]byte("one\r\ntwo\r\nthree\r\nfour"))

	vt.Resize(12, 2)
	if got := vt.GetDisplay(); got != "three\nfour" {
		t.Fatalf("shrinking resize retained the wrong visible rows: %q", got)
	}
	if got := vt.GetOutput(100); got != "one\ntwo\nthree\nfour" {
		t.Fatalf("shrinking resize did not preserve dropped rows in history: %q", got)
	}
}

func TestResizeShrinkingRowsKeepsCursorWindow(t *testing.T) {
	vt := NewVirtualTerminal(6, 6, 100)
	vt.Feed([]byte("R0\r\nR1\r\nR2\r\nR3\r\nR4\r\nR5"))
	vt.Feed([]byte("\x1b[3;1H"))

	vt.Resize(6, 3)
	if got := vt.GetDisplay(); got != "R0\nR1\nR2" {
		t.Fatalf("row shrink discarded the cursor window in favor of later content: %q", got)
	}
	assertCursor(t, vt, 2, 0)
}

func TestResizeGrowingColumnsCancelsDelayedWrapAtOldMargin(t *testing.T) {
	vt := NewVirtualTerminal(5, 2, 100)
	vt.Feed([]byte("ABCDE"))

	vt.Resize(8, 2)
	vt.Feed([]byte("X"))
	if got := vt.GetDisplay(); got != "ABCDEX" {
		t.Fatalf("column growth moved the delayed-wrap cursor to the new margin: %q", got)
	}
}

func TestResizeKeepsCursorLinePhysicalWrapReplayable(t *testing.T) {
	vt := NewVirtualTerminal(5, 3, 100)
	vt.Feed([]byte("ABCDEZ"))
	if !vt.IsLineWrapped(1) {
		t.Fatal("test setup did not create a soft-wrapped second row")
	}

	vt.Resize(10, 3)
	if !vt.IsLineWrapped(1) {
		t.Fatal("non-reflowed cursor line lost its soft-wrap ownership")
	}
	snapshot := vt.GetSnapshot()
	restored := NewVirtualTerminal(snapshot.Cols, snapshot.Rows, 100)
	replayTerminalSnapshot(restored, snapshot)
	if got := restored.GetDisplay(); got != "ABCDE\nZ" {
		t.Fatalf("resized snapshot joined distinct physical rows: %q", got)
	}
	if !restored.IsLineWrapped(1) {
		t.Fatal("snapshot replay lost the cursor line's soft-wrap ownership")
	}
}

func TestResizeCursorWrapCanReflowAfterSnapshotAndCursorMove(t *testing.T) {
	source := NewVirtualTerminal(5, 4, 100)
	source.Feed([]byte("\x1b[7;31mABCDE\x1b[0mZ"))
	source.Resize(10, 4)
	restored := replaySnapshotToFreshTerminal(source)

	for _, candidate := range []*VirtualTerminal{source, restored} {
		candidate.Feed([]byte("\x1b[4;1H"))
		candidate.Resize(12, 4)
		if got := candidate.GetDisplay(); got != "ABCDE     Z" {
			t.Fatalf("second resize did not reflow prior cursor line: %q", got)
		}
	}
	if got, want := restored.GetDisplay(), source.GetDisplay(); got != want {
		t.Fatalf("second resize after snapshot = %q, want %q", got, want)
	}
}

func TestResizeReflowsNonCursorNormalLineWhenShrinkingColumns(t *testing.T) {
	vt := NewVirtualTerminal(10, 6, 100)
	vt.Feed([]byte("ABCDEFGHIJ\x1b[4;1H"))

	vt.Resize(5, 6)
	if got := vt.GetDisplay(); got != "ABCDE\nFGHIJ" {
		t.Fatalf("column shrink discarded a non-cursor logical line: %q", got)
	}
	if !vt.IsLineWrapped(1) {
		t.Fatal("reflowed continuation row is not marked wrapped")
	}
	assertCursor(t, vt, 4, 0)

	restored := replaySnapshotToFreshTerminal(vt)
	if got := restored.GetDisplay(); got != vt.GetDisplay() {
		t.Fatalf("shrunk reflow snapshot display = %q, want %q", got, vt.GetDisplay())
	}
	assertCursor(t, restored, 4, 0)
}

func TestResizeReflowsNonCursorNormalLineWhenGrowingColumns(t *testing.T) {
	vt := NewVirtualTerminal(5, 6, 100)
	vt.Feed([]byte("ABCDEZ\x1b[5;1H"))

	vt.Resize(10, 6)
	if got := vt.GetDisplay(); got != "ABCDEZ" {
		t.Fatalf("column growth did not join a non-cursor logical line: %q", got)
	}
	if vt.IsLineWrapped(1) {
		t.Fatal("removed continuation row left a stale wrap flag")
	}
	assertCursor(t, vt, 3, 0)
}

func TestResizeReflowAdjustsViewportCursorLikeXterm(t *testing.T) {
	t.Run("smaller reflow below cursor moves cursor down", func(t *testing.T) {
		vt := NewVirtualTerminal(10, 6, 100)
		vt.Feed([]byte("\x1b[4;1HABCDEFGHIJ\x1b[2;1H"))

		vt.Resize(5, 6)
		assertCursor(t, vt, 2, 0)
		restored := replaySnapshotToFreshTerminal(vt)
		assertCursor(t, restored, 2, 0)
		if got := restored.GetDisplay(); got != vt.GetDisplay() {
			t.Fatalf("smaller reflow snapshot display = %q, want %q", got, vt.GetDisplay())
		}
	})

	t.Run("larger reflow below cursor moves cursor up", func(t *testing.T) {
		vt := NewVirtualTerminal(5, 6, 100)
		vt.Feed([]byte("\x1b[4;1HABCDEZ\x1b[2;1H"))

		vt.Resize(10, 6)
		assertCursor(t, vt, 0, 0)
		restored := replaySnapshotToFreshTerminal(vt)
		assertCursor(t, restored, 0, 0)
		if got := restored.GetDisplay(); got != vt.GetDisplay() {
			t.Fatalf("larger reflow snapshot display = %q, want %q", got, vt.GetDisplay())
		}
	})
}

func TestResizeBottomReflowPopsOriginalRowBeforeInsertedContinuation(t *testing.T) {
	vt := NewVirtualTerminal(10, 6, 100)
	vt.Feed([]byte("\x1b[6;1HABCDEFGHIJ\x1b[1;1H"))

	vt.Resize(5, 6)
	assertCursor(t, vt, 1, 0)
	if got := resizedRowText(vt.GetCellsRow(5)); got != "FGHIJ" {
		t.Fatalf("bottom reflow row = %q, want xterm continuation %q", got, "FGHIJ")
	}
	if !vt.IsLineWrapped(5) {
		t.Fatal("bottom continuation lost its wrap flag")
	}
}

func TestFullHistorySnapshotKeepsViewportBoundaryReflowable(t *testing.T) {
	source := NewVirtualTerminal(5, 2, 100)
	source.Feed([]byte("ABCDEZ\r\n"))
	restored := replaySnapshotToFreshTerminal(source)

	if got := restored.GetOutput(100); got != source.GetOutput(100) {
		t.Fatalf("full baseline output = %q, want %q", got, source.GetOutput(100))
	}
	source.Resize(10, 2)
	restored.Resize(10, 2)
	if got, want := restored.GetDisplay(), source.GetDisplay(); got != want || got != "ABCDEZ" {
		t.Fatalf("post-baseline boundary reflow = %q, want %q", got, want)
	}
}

func TestFullHistorySnapshotReplacesExistingScrollback(t *testing.T) {
	source := NewVirtualTerminal(8, 3, 100)
	source.Feed([]byte("one\r\ntwo\r\nthree\r\nfour\r\nfive"))

	target := NewVirtualTerminal(8, 3, 100)
	target.Feed([]byte("stale-1\r\nstale-2\r\nstale-3\r\nstale-4\r\nstale-5"))
	replayTerminalSnapshot(target, source.GetSnapshot())

	if got, want := target.GetOutput(100), source.GetOutput(100); got != want {
		t.Fatalf("authoritative baseline duplicated or retained stale scrollback:\ngot:  %q\nwant: %q", got, want)
	}
	if got, want := target.GetHistoryStyledLength(), source.GetHistoryStyledLength(); got != want {
		t.Fatalf("target history length = %d, want %d", got, want)
	}
}

func TestResizeReflowKeepsWideCellsAtomic(t *testing.T) {
	vt := NewVirtualTerminal(6, 6, 100)
	vt.Feed([]byte("ABC界D\x1b[5;1H"))

	vt.Resize(4, 6)
	if got := vt.GetDisplay(); got != "ABC\n界D" {
		t.Fatalf("wide-character reflow = %q, want %q", got, "ABC\n界D")
	}
	wide := vt.GetCellsRow(1)
	if wide[0].Char != '界' || wide[0].Width != 2 || !wide[1].IsPlaceholder() {
		t.Fatalf("reflow split wide-character owner: %+v %+v", wide[0], wide[1])
	}

	restored := replaySnapshotToFreshTerminal(vt)
	if got := restored.GetDisplay(); got != vt.GetDisplay() {
		t.Fatalf("wide reflow snapshot display = %q, want %q", got, vt.GetDisplay())
	}
}

func TestResizeReflowsNormalHistoryWithStyles(t *testing.T) {
	vt := NewVirtualTerminal(10, 3, 100)
	vt.Feed([]byte("\x1b[31mABCDEFGHIJ\x1b[0m\r\nsecond\r\nthird\r\ncursor"))
	if got := vt.GetHistoryStyledLength(); got != 1 {
		t.Fatalf("test setup history length = %d, want 1", got)
	}

	vt.Resize(5, 3)
	if got := vt.GetHistoryStyledLength(); got < 2 {
		t.Fatalf("reflowed history length = %d, want at least 2", got)
	}
	if first, second := resizedRowText(vt.historyStyled[0]), resizedRowText(vt.historyStyled[1]); first != "ABCDE" || second != "FGHIJ" {
		t.Fatalf("reflowed history rows = %q, %q", first, second)
	}
	if !vt.historyIsWrapped[1] {
		t.Fatal("reflowed history continuation is not marked wrapped")
	}
	for _, row := range vt.historyStyled[:2] {
		for _, cell := range row {
			if cell.Char != ' ' && (!cell.Fg.IsPalette() || cell.Fg.Index() != 1) {
				t.Fatalf("reflowed history lost red style: %+v", cell)
			}
		}
	}
}

func TestResizeAlternateBufferClipsWideCellWithoutOrphan(t *testing.T) {
	vt := NewVirtualTerminal(6, 3, 100)
	vt.Feed([]byte("main\x1b[?1049h\x1b[2J\x1b[HABC界"))

	vt.Resize(4, 3)
	if got := vt.GetDisplay(); got != "ABC" {
		t.Fatalf("alternate physical resize kept a clipped wide cell: %q", got)
	}
	last := vt.GetCellsRow(0)[3]
	if last.Char != ' ' || last.IsPlaceholder() || last.Width != 1 {
		t.Fatalf("alternate physical resize left an orphan wide cell: %+v", last)
	}

	restored := replaySnapshotToFreshTerminal(vt)
	if got := restored.GetDisplay(); got != vt.GetDisplay() {
		t.Fatalf("clipped alternate snapshot display = %q, want %q", got, vt.GetDisplay())
	}
}

func TestResizeAlternateBufferPreservesPhysicalWrapForSnapshot(t *testing.T) {
	vt := NewVirtualTerminal(5, 4, 100)
	vt.Feed([]byte("\x1b[?1049hABCDEZ"))
	vt.Resize(10, 4)
	if !vt.IsLineWrapped(1) {
		t.Fatal("alternate column resize cleared the physical wrap flag")
	}

	restored := replaySnapshotToFreshTerminal(vt)
	if !restored.IsLineWrapped(1) {
		t.Fatal("alternate snapshot replay lost the physical wrap flag")
	}
	if got := restored.GetDisplay(); got != vt.GetDisplay() {
		t.Fatalf("alternate wrapped replay display = %q, want %q", got, vt.GetDisplay())
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
