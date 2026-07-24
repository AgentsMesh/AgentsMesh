package vt

import "testing"

func TestSnapshotPreservesLeadingGraphemeBoundaryAfterMaterializedWrap(t *testing.T) {
	tests := []struct {
		name     string
		modifier string
	}{
		{name: "combining mark", modifier: "\u0301"},
		{name: "zero width joiner", modifier: "\u200d"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			source := NewVirtualTerminal(4, 3, 100)
			source.Feed([]byte("界界\x1b[0m" + test.modifier))
			source.Resize(11, 5)

			restored := replaySnapshotToFreshTerminal(source)
			if got, want := restored.GetDisplay(), source.GetDisplay(); got != want {
				t.Fatalf("snapshot display = %q, want %q", got, want)
			}
			if got := restored.cells[0][10]; got.Text() != " " || got.Combining != "" {
				t.Fatalf("materialized wrap blank absorbed next-row modifier: %+v", got)
			}
			if got := restored.cells[1][0].Text(); got != test.modifier {
				t.Fatalf("next-row grapheme = %q, want %q", got, test.modifier)
			}
			assertCursor(t, restored, 1, 1)
		})
	}
}
