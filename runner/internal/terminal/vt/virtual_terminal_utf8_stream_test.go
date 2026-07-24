package vt

import (
	"bytes"
	"testing"
)

func TestVirtualTerminalPreservesUTF8SplitAcrossFeedCalls(t *testing.T) {
	tests := []struct {
		name string
		text string
		cut  int
	}{
		{name: "CJK", text: "中X", cut: 2},
		{name: "box drawing", text: "─X", cut: 1},
		{name: "four byte rune", text: "😀X", cut: 3},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := NewVirtualTerminal(20, 4, 100)
			data := []byte(test.text)
			source.Feed(data[:test.cut])
			before := source.GetSnapshot()
			if before.Lines[0] != "" {
				t.Fatalf("partial UTF-8 leaked into snapshot: %+v", before)
			}
			if got := intBytes(before.ParserPrefix); !bytes.Equal(got, data[:test.cut]) {
				t.Fatalf("snapshot UTF-8 prefix = %v, want %v", got, data[:test.cut])
			}

			restored := NewVirtualTerminal(20, 4, 100)
			replayTerminalSnapshot(restored, before)
			source.Feed(data[test.cut:])
			restored.Feed(data[test.cut:])
			if got := source.GetDisplay(); got != test.text {
				t.Fatalf("split UTF-8 display = %q, want %q", got, test.text)
			}
			if got := restored.GetDisplay(); got != test.text {
				t.Fatalf("snapshot split UTF-8 display = %q, want %q", got, test.text)
			}
		})
	}
}

func TestVirtualTerminalDropsInvalidUTF8WithoutPoisoningNextFeed(t *testing.T) {
	terminal := NewVirtualTerminal(20, 4, 100)
	terminal.Feed([]byte{0xff})
	terminal.Feed([]byte("valid"))
	if got := terminal.GetDisplay(); got != "valid" {
		t.Fatalf("display after invalid UTF-8 = %q", got)
	}
}
