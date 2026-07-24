package vt

import (
	"strings"
	"testing"
)

func TestZeroWidthModifiersDoNotAdvanceSnapshotCursor(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantDisplay string
		wantCol     int
	}{
		{name: "combining acute", input: "e\u0301X", wantDisplay: "e\u0301X", wantCol: 2},
		{name: "emoji variation selector", input: "❤\ufe0fX", wantDisplay: "❤\ufe0fX", wantCol: 3},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := NewVirtualTerminal(10, 3, 100)
			feedLines := source.Feed([]byte(test.input))
			assertCursor(t, source, 0, test.wantCol)
			if got := feedLines[0]; got != test.wantDisplay {
				t.Fatalf("Feed line = %q, want %q", got, test.wantDisplay)
			}
			if got := source.TryGetLines()[0]; got != test.wantDisplay {
				t.Fatalf("TryGetLines line = %q, want %q", got, test.wantDisplay)
			}
			if got := source.GetDisplay(); got != test.wantDisplay {
				t.Fatalf("display = %q, want %q", got, test.wantDisplay)
			}
			snapshot := source.GetSnapshot()
			if got := snapshot.Lines[0]; got != test.wantDisplay {
				t.Fatalf("snapshot line = %q, want %q", got, test.wantDisplay)
			}
			if !strings.Contains(snapshot.SerializedContent, test.input) {
				t.Fatalf("snapshot dropped grapheme code points: %q not in %q", test.input, snapshot.SerializedContent)
			}

			restored := replaySnapshotToFreshTerminal(source)
			assertCursor(t, restored, 0, test.wantCol)
			source.Feed([]byte("Q"))
			restored.Feed([]byte("Q"))
			if got, want := restored.GetDisplay(), source.GetDisplay(); got != want {
				t.Fatalf("snapshot continuation = %q, want %q", got, want)
			}
		})
	}
}

func TestOverwritingBaseCellClearsCombiningSuffix(t *testing.T) {
	terminal := NewVirtualTerminal(10, 3, 100)
	terminal.Feed([]byte("e\u0301\bX"))
	if got := terminal.GetDisplay(); got != "X" {
		t.Fatalf("display after overwrite = %q, want %q", got, "X")
	}
	if got := terminal.cells[0][0].Combining; got != "" {
		t.Fatalf("overwritten cell retained combining suffix %q", got)
	}
}

func TestUnicodeColumnContractMatchesXtermGraphemeAddon(t *testing.T) {
	tests := []struct {
		name    string
		parts   []string
		wantCol int
	}{
		{name: "ZWJ emoji", parts: []string{"👩", "\u200d", "💻"}, wantCol: 2},
		{name: "regional indicator flag", parts: []string{"🇺", "🇸"}, wantCol: 2},
		{name: "emoji variation selector", parts: []string{"❤", "\ufe0f"}, wantCol: 2},
		{name: "ZWJ flag", parts: []string{"🏳", "\ufe0f", "\u200d", "🌈"}, wantCol: 2},
		{name: "keycap", parts: []string{"1", "\ufe0f", "\u20e3"}, wantCol: 2},
		{name: "emoji modifier", parts: []string{"👍", "🏽"}, wantCol: 2},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			wantDisplay := strings.Join(test.parts, "")
			for _, chunked := range []bool{false, true} {
				terminal := NewVirtualTerminal(20, 3, 100)
				if chunked {
					for _, part := range test.parts {
						terminal.Feed([]byte(part))
					}
				} else {
					terminal.Feed([]byte(wantDisplay))
				}

				assertCursor(t, terminal, 0, test.wantCol)
				if got := terminal.GetDisplay(); got != wantDisplay {
					t.Fatalf("display = %q, want %q", got, wantDisplay)
				}

				restored := replaySnapshotToFreshTerminal(terminal)
				assertCursor(t, restored, 0, test.wantCol)
				if got := restored.GetDisplay(); got != wantDisplay {
					t.Fatalf("restored display = %q, want %q", got, wantDisplay)
				}
			}
		})
	}
}

func TestZeroWidthJoinStateResetsAcrossControlSequences(t *testing.T) {
	direct := NewVirtualTerminal(10, 3, 100)
	direct.Feed([]byte("A\u0301X"))
	assertCursor(t, direct, 0, 2)
	if got := direct.cells[0][0].Text(); got != "A\u0301" {
		t.Fatalf("direct combining cell = %q, want %q", got, "A\u0301")
	}

	styled := NewVirtualTerminal(10, 3, 100)
	styled.Feed([]byte("A\x1b[31m\u0301X"))
	assertCursor(t, styled, 0, 3)
	if got := styled.cells[0][0].Text(); got != "A" {
		t.Fatalf("cell before SGR retained combining code point: %q", got)
	}
	standalone := styled.cells[0][1]
	if got := standalone.Text(); got != "\u0301" || standalone.Width != 1 || standalone.IsPlaceholder() {
		t.Fatalf("standalone zero-width cell = %+v", standalone)
	}
	if got := styled.cells[0][2].Text(); got != "X" {
		t.Fatalf("cell after standalone zero-width code point = %q, want X", got)
	}

	restored := replaySnapshotToFreshTerminal(styled)
	assertCursor(t, restored, 0, 3)
	if got := restored.cells[0][1]; got.Text() != "\u0301" || got.IsPlaceholder() {
		t.Fatalf("restored standalone zero-width cell = %+v", got)
	}
}

func TestSnapshotRoundTripPreservesZeroWidthJoinAcrossBoundary(t *testing.T) {
	source := NewVirtualTerminal(10, 3, 100)
	source.Feed([]byte("A"))
	snapshot := source.GetSnapshot()
	if snapshot.PrecedingJoinReplay == "" {
		t.Fatal("snapshot omitted preceding join replay")
	}

	restored := NewVirtualTerminal(snapshot.Cols, snapshot.Rows, 100)
	replayTerminalSnapshot(restored, snapshot)
	for _, terminal := range []*VirtualTerminal{source, restored} {
		terminal.Feed([]byte("\u0301X"))
		assertCursor(t, terminal, 0, 2)
		if got := terminal.GetDisplay(); got != "A\u0301X" {
			t.Fatalf("display after snapshot-boundary combining mark = %q", got)
		}
		if got := terminal.cells[0][0].Text(); got != "A\u0301" {
			t.Fatalf("combined cell after snapshot boundary = %q", got)
		}
	}
}

func TestSnapshotJoinReplayHandlesWideAndDelayedWrapOwners(t *testing.T) {
	tests := []struct {
		name      string
		cols      int
		content   string
		ownerCol  int
		wantCol   int
		wantOwner string
	}{
		{name: "wide owner", cols: 10, content: "界", ownerCol: 0, wantCol: 2, wantOwner: "界\u0301"},
		{name: "delayed wrap owner", cols: 5, content: "ABCDE", ownerCol: 4, wantCol: 5, wantOwner: "E\u0301"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := NewVirtualTerminal(test.cols, 3, 100)
			source.Feed([]byte(test.content))
			restored := replaySnapshotToFreshTerminal(source)

			for _, terminal := range []*VirtualTerminal{source, restored} {
				terminal.Feed([]byte("\u0301"))
				assertCursor(t, terminal, 0, test.wantCol)
				if got := terminal.cells[0][test.ownerCol].Text(); got != test.wantOwner {
					t.Fatalf("join owner = %q, want %q", got, test.wantOwner)
				}
			}

			source.Feed([]byte("X"))
			restored.Feed([]byte("X"))
			if got, want := restored.GetDisplay(), source.GetDisplay(); got != want {
				t.Fatalf("display after join continuation = %q, want %q", got, want)
			}
		})
	}
}

func TestSnapshotPreservesStandaloneZeroWidthParserBoundary(t *testing.T) {
	boundaries := []struct {
		name     string
		sequence string
	}{
		{name: "BEL", sequence: "\a"},
		{name: "no-op SGR", sequence: "\x1b[0m"},
		{name: "OSC title", sequence: "\x1b]0;title\a"},
	}

	for _, boundary := range boundaries {
		t.Run(boundary.name, func(t *testing.T) {
			source := NewVirtualTerminal(10, 3, 100)
			source.Feed([]byte("A" + boundary.sequence + "\u0301X"))
			restored := replaySnapshotToFreshTerminal(source)

			for _, terminal := range []*VirtualTerminal{source, restored} {
				assertCursor(t, terminal, 0, 3)
				if got := terminal.cells[0][0].Text(); got != "A" {
					t.Fatalf("cell before parser boundary = %q, want A", got)
				}
				standalone := terminal.cells[0][1]
				if standalone.Text() != "\u0301" || standalone.IsPlaceholder() {
					t.Fatalf("standalone zero-width cell = %+v", standalone)
				}
				if got := terminal.cells[0][2].Text(); got != "X" {
					t.Fatalf("cell after parser boundary = %q, want X", got)
				}
			}
		})
	}
}

func TestScreenReplacementClearsZeroWidthJoinState(t *testing.T) {
	tests := []struct {
		name          string
		replace       func(*VirtualTerminal)
		wantCursorCol int
		zeroWidthCol  int
		nextCharCol   int
	}{
		{name: "clear", replace: func(terminal *VirtualTerminal) { terminal.Clear() }, wantCursorCol: 2, zeroWidthCol: 0, nextCharCol: 1},
		{name: "resize", replace: func(terminal *VirtualTerminal) { terminal.Resize(12, 4) }, wantCursorCol: 3, zeroWidthCol: 1, nextCharCol: 2},
		{name: "RIS", replace: func(terminal *VirtualTerminal) { terminal.Feed([]byte("\x1bc")) }, wantCursorCol: 2, zeroWidthCol: 0, nextCharCol: 1},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			terminal := NewVirtualTerminal(10, 3, 100)
			terminal.Feed([]byte("A"))
			test.replace(terminal)
			terminal.Feed([]byte("\u0301X"))

			assertCursor(t, terminal, 0, test.wantCursorCol)
			standalone := terminal.cells[0][test.zeroWidthCol]
			if standalone.Text() != "\u0301" || standalone.IsPlaceholder() {
				t.Fatalf("standalone zero-width cell after screen replacement = %+v", standalone)
			}
			if got := terminal.cells[0][test.nextCharCol].Text(); got != "X" {
				t.Fatalf("cell after screen replacement = %q, want X", got)
			}
		})
	}
}

func TestDELDoesNotResetZeroWidthJoinState(t *testing.T) {
	terminal := NewVirtualTerminal(10, 3, 100)
	terminal.Feed([]byte("A\x7f\u0301X"))

	assertCursor(t, terminal, 0, 2)
	if got := terminal.cells[0][0].Text(); got != "A\u0301" {
		t.Fatalf("DEL interrupted zero-width join: %q", got)
	}
}

func TestGraphemeWidthGrowthWrapsOwnerAtomically(t *testing.T) {
	source := NewVirtualTerminal(5, 3, 100)
	source.Feed([]byte("ABCD❤"))
	assertCursor(t, source, 0, 5)

	source.Feed([]byte("\ufe0f"))
	assertCursor(t, source, 1, 2)
	if got := source.GetDisplay(); got != "ABCD\n❤️" {
		t.Fatalf("right-margin grapheme growth = %q, want %q", got, "ABCD\n❤️")
	}
	if !source.IsLineWrapped(1) {
		t.Fatal("right-margin grapheme growth did not mark the destination row wrapped")
	}
	owner := source.cells[1][0]
	if owner.Text() != "❤️" || owner.Width != 2 || !owner.HasContent ||
		!source.cells[1][1].IsPlaceholder() {
		t.Fatalf("wrapped grapheme owner is invalid: owner=%+v placeholder=%+v", owner, source.cells[1][1])
	}

	restored := replaySnapshotToFreshTerminal(source)
	assertCursor(t, restored, 1, 2)
	if got := restored.GetDisplay(); got != source.GetDisplay() {
		t.Fatalf("wrapped grapheme snapshot = %q, want %q", got, source.GetDisplay())
	}
}

func TestSnapshotRebuildsPartialGraphemeJoinState(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		suffix string
		want   string
	}{
		{name: "ZWJ", prefix: "👩\u200d", suffix: "💻X", want: "👩‍💻X"},
		{name: "VS16", prefix: "❤", suffix: "\ufe0fX", want: "❤️X"},
		{name: "emoji modifier", prefix: "👍", suffix: "🏽X", want: "👍🏽X"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := NewVirtualTerminal(10, 3, 100)
			source.Feed([]byte(test.prefix))
			restored := replaySnapshotToFreshTerminal(source)
			for _, terminal := range []*VirtualTerminal{source, restored} {
				terminal.Feed([]byte(test.suffix))
				if got := terminal.GetDisplay(); got != test.want {
					t.Fatalf("snapshot-boundary grapheme = %q, want %q", got, test.want)
				}
				assertCursor(t, terminal, 0, 3)
			}
		})
	}
}

func TestSnapshotPreservesControlSeparatedGraphemeCells(t *testing.T) {
	source := NewVirtualTerminal(10, 3, 100)
	source.Feed([]byte("🇺\x1b[0m🇸X"))
	restored := replaySnapshotToFreshTerminal(source)

	for _, terminal := range []*VirtualTerminal{source, restored} {
		assertCursor(t, terminal, 0, 3)
		first, second := terminal.cells[0][0], terminal.cells[0][1]
		if first.Text() != "🇺" || first.Width != 1 || second.Text() != "🇸" || second.Width != 1 {
			t.Fatalf("control-separated regional indicators rejoined: first=%+v second=%+v", first, second)
		}
	}
}

func TestPrintedSpaceContentSurvivesReflowAndSnapshot(t *testing.T) {
	source := NewVirtualTerminal(6, 5, 100)
	source.Feed([]byte("ABCD 界\x1b[4;1H"))
	if !source.cells[0][4].HasContent {
		t.Fatal("PTY-printed space was stored as a null cell")
	}

	source.Resize(10, 5)
	if got := source.GetDisplay(); got != "ABCD 界" {
		t.Fatalf("reflow discarded a printed space: %q", got)
	}
	if !source.cells[0][4].HasContent {
		t.Fatal("reflow converted a printed space into a null cell")
	}

	restored := replaySnapshotToFreshTerminal(source)
	if got := restored.GetDisplay(); got != "ABCD 界" {
		t.Fatalf("snapshot discarded a printed space: %q", got)
	}
	if !restored.cells[0][4].HasContent {
		t.Fatal("snapshot replay converted a printed space into a null cell")
	}
	if restored.cells[0][6].HasContent || !restored.cells[0][6].IsPlaceholder() {
		t.Fatalf("wide placeholder has invalid content state: %+v", restored.cells[0][6])
	}
}

func TestEraseCreatesNullCellAfterPrintedSpace(t *testing.T) {
	terminal := NewVirtualTerminal(5, 2, 100)
	terminal.Feed([]byte(" "))
	if !terminal.cells[0][0].HasContent {
		t.Fatal("printed space does not carry content")
	}
	terminal.Feed([]byte("\b\x1b[X"))
	if terminal.cells[0][0].HasContent || !terminal.cells[0][0].IsEmpty() {
		t.Fatalf("erased cell still carries printed content: %+v", terminal.cells[0][0])
	}
}
