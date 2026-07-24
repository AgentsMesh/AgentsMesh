package vt

import "testing"

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
