package vt

import "testing"

func TestEraseWidePlaceholderClearsOwner(t *testing.T) {
	terminal := NewVirtualTerminal(4, 2, 100)
	terminal.Feed([]byte("界\b\x1b[K"))

	if got := terminal.GetDisplay(); got != "" {
		t.Fatalf("display after erasing wide placeholder = %q, want empty", got)
	}
	for col := 0; col < 2; col++ {
		if cell := terminal.cells[0][col]; cell.HasContent || cell.IsPlaceholder() || cell.Width != 1 {
			t.Fatalf("cell %d retained a wide fragment: %+v", col, cell)
		}
	}
}

func TestResizeCursorOnWidePlaceholderClearsOwnerBeforeWrap(t *testing.T) {
	source := NewVirtualTerminal(6, 6, 100)
	source.Feed([]byte("A界"))
	source.Resize(3, 2)
	assertCursor(t, source, 0, 2)

	source.Feed([]byte("界"))
	if got := source.GetDisplay(); got != "A\n界" {
		t.Fatalf("display after writing from resized placeholder = %q, want %q", got, "A\n界")
	}
	if owner := source.cells[0][1]; owner.HasContent || owner.Width != 1 {
		t.Fatalf("overwritten wide owner survived without its placeholder: %+v", owner)
	}

	restored := replaySnapshotToFreshTerminal(source)
	if got, want := restored.GetDisplay(), source.GetDisplay(); got != want {
		t.Fatalf("snapshot display = %q, want %q", got, want)
	}
}
