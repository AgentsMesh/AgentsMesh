package aggregator

import (
	"bytes"
	"testing"

	terminalvt "github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

func buildResizeFullRepaint(lines ...string) []byte {
	frame := make([]byte, 0, 256)
	frame = append(frame, syncOutputStartSeq...)
	frame = append(frame, eraseScreenSeq...)
	frame = append(frame, cursorHomeSeq...)
	frame = append(frame, []byte("\x1b[?25l")...) // Hide cursor while repainting.
	for i, line := range lines {
		if i > 0 {
			frame = append(frame, '\r', '\n')
		}
		frame = append(frame, line...)
	}
	frame = append(frame, []byte("\x1b[2;4H\x1b[?25h")...)
	frame = append(frame, syncOutputEndSeq...)
	return frame
}

func TestFrameBufferFlushCompletePreservesSynchronizedFrameByteForByte(t *testing.T) {
	want := buildResizeFullRepaint(
		"AgentsMesh terminal",
		"status: resizing",
		"press ? for help",
	)
	fb := NewFrameBuffer(4096)

	// PTY reads can split both frame boundaries and control sequences.
	for _, chunk := range [][]byte{
		want[:5],
		want[5 : len(syncOutputStartSeq)+2],
		want[len(syncOutputStartSeq)+2 : len(want)-4],
		want[len(want)-4:],
	} {
		fb.Write(chunk)
	}

	got, remaining := fb.FlushComplete()
	if remaining != 0 {
		t.Fatalf("FlushComplete left %d bytes, want 0", remaining)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("FlushComplete changed synchronized frame bytes\n got: %q\nwant: %q", got, want)
	}
}

func TestFrameBufferResizeFullRepaintPreservesEraseAndCursorHome(t *testing.T) {
	initial := buildResizeFullRepaint(
		"old-width line that must be erased completely",
		"old footer",
	)
	resized := buildResizeFullRepaint(
		"new width",
		"new footer",
	)
	fb := NewFrameBuffer(4096)
	fb.Write(initial)

	// A resize repaint commonly arrives over more than one PTY read. ED2 replaces
	// viewport cells, but it does not replace parser state established earlier.
	cut := len(syncOutputStartSeq) + len(eraseScreenSeq) + 1
	fb.Write(resized[:cut])
	fb.Write(resized[cut:])

	got, remaining := fb.FlushComplete()
	if remaining != 0 {
		t.Fatalf("FlushComplete left %d bytes, want 0", remaining)
	}
	want := append(append([]byte(nil), initial...), resized...)
	if !bytes.Equal(got, want) {
		t.Fatalf("resize repaint changed in transit\n got: %q\nwant: %q", got, want)
	}
	if !bytes.Contains(got, eraseScreenSeq) {
		t.Fatal("resize repaint lost ESC[2J erase-screen sequence")
	}
	if !bytes.Contains(got, cursorHomeSeq) {
		t.Fatal("resize repaint lost ESC[H cursor-home sequence")
	}

	// Render the flushed bytes into the same VT implementation used for runner
	// snapshots. The pre-resize cells and cursor position must not leak into the
	// repainted screen at the new geometry.
	term := terminalvt.NewVirtualTerminal(44, 4, 100)
	term.Feed([]byte("\x1b[2J\x1b[Hstale header that spans the old width\r\nstale footer\x1b[4;10H"))
	term.Resize(20, 4)
	term.Feed(got)
	if display := term.GetScreenSnapshot(); display != "new width\nnew footer" {
		t.Fatalf("resize full repaint left stale terminal cells\n got: %q\nwant: %q", display, "new width\nnew footer")
	}
}

func TestFrameBufferFlushAllPreservesIncompleteSynchronizedRepaint(t *testing.T) {
	want := make([]byte, 0, 64)
	want = append(want, syncOutputStartSeq...)
	want = append(want, eraseScreenSeq...)
	want = append(want, cursorHomeSeq...)
	want = append(want, []byte("partial resize repaint")...)

	fb := NewFrameBuffer(4096)
	fb.Write(want)

	got, remaining := fb.FlushAll()
	if remaining != 0 {
		t.Fatalf("FlushAll left %d bytes, want 0", remaining)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("FlushAll changed synchronized repaint bytes\n got: %q\nwant: %q", got, want)
	}
}
