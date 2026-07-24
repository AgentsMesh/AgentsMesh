package mockagent

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"golang.org/x/term"
)

const (
	terminalRenderCommand        = "render-probe"
	terminalRenderReady          = "terminal-render-ready"
	terminalRenderSizePrefix     = "E2E_PTY_SIZE"
	terminalRenderChunkDelay     = 24 * time.Millisecond
	terminalRenderAltHold        = 2 * time.Second
	terminalNormalBufferSentinel = "NORMAL-BUFFER-SENTINEL"
)

type terminalSizeSource func() (cols, rows int, err error)

// terminalRenderLoop waits for an explicit input command before emitting the
// fixture. This makes the E2E deterministic: the browser has subscribed to the
// live PTY stream before any of the deliberately fragmented bytes are written.
func terminalRenderLoop(in io.Reader, out io.Writer, chunkDelay, altHold time.Duration) {
	terminalRenderLoopWithResize(in, out, chunkDelay, altHold, nil, nil)
}

func terminalRenderLoopWithResize(
	in io.Reader,
	out io.Writer,
	chunkDelay time.Duration,
	altHold time.Duration,
	resizeSignals <-chan os.Signal,
	readSize terminalSizeSource,
) {
	_, _ = fmt.Fprint(out, terminalRenderReady+"\r\n")

	lines := make(chan string)
	go func() {
		defer close(lines)
		scanner := bufio.NewScanner(in)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
	}()

	// Resize output starts only after the stable fixture exists. Initial fit
	// signals are intentionally ignored so they cannot interleave with ANSI or
	// UTF-8 fragments; the explicit baseline below captures their final size.
	rendered := false
	for !rendered {
		line, ok := <-lines
		if !ok {
			return
		}
		if strings.TrimSpace(line) != terminalRenderCommand {
			continue
		}
		writeTerminalRenderFixture(out, chunkDelay, altHold)
		rendered = true
	}

	sequence := 0
	lastCols, lastRows := 0, 0
	emitSize := func() {
		if readSize == nil {
			return
		}
		cols, rows, err := readSize()
		if err != nil || cols <= 0 || rows <= 0 || (cols == lastCols && rows == lastRows) {
			return
		}
		sequence++
		lastCols, lastRows = cols, rows
		_, _ = fmt.Fprintf(out, "%s:%03d:%dx%d\r\n", terminalRenderSizePrefix, sequence, cols, rows)
	}
	emitSize()

	for {
		select {
		case _, ok := <-lines:
			if !ok {
				return
			}
			// The render fixture is one-shot; additional input is ignored.
		case _, ok := <-resizeSignals:
			if !ok {
				resizeSignals = nil
				continue
			}
			emitSize()
		}
	}
}

func stdoutTerminalSize() (cols, rows int, err error) {
	return term.GetSize(int(os.Stdout.Fd()))
}

func writeTerminalRenderFixture(out io.Writer, chunkDelay, altHold time.Duration) {
	writeTerminalChunks(out, chunkDelay,
		[]byte("\x1b["), []byte("3J"),
		[]byte("\x1b[2"), []byte("J"),
		[]byte("\x1b["), []byte("H"),
		[]byte(terminalNormalBufferSentinel+"\r\n"),
	)

	// Exercise alternate-buffer parsing in the live renderer, then return to
	// the normal buffer. The sentinel above must survive and the probe below
	// must not leak into normal-buffer scrollback.
	writeTerminalChunks(out, chunkDelay,
		[]byte("\x1b[?10"), []byte("49h"),
		[]byte("\x1b[2J\x1b[H"),
		[]byte("ALT-BUFFER-PROBE\r\n"),
	)
	if altHold > 0 {
		time.Sleep(altHold)
	}
	writeTerminalChunks(out, chunkDelay,
		[]byte("\x1b[?104"), []byte("9l"),
		[]byte("\x1b]0;Agents"), []byte("Mesh terminal E2E"), []byte("\x07"),
		[]byte("\x1bPignored-"), []byte("dcs-payload"), []byte("\x1b"), []byte("\\"),
	)

	// Produce real scrollback pressure without sleeping between lines. This is
	// intentionally bursty so the Runner aggregator and renderer scheduler see
	// the same shape as a CLI repainting a long transcript.
	for i := 0; i < 48; i++ {
		_, _ = fmt.Fprintf(out, "SCROLL-%02d|abcdefghijklmnopqrstuvwxyz|\r\n", i)
	}

	menu := []byte("☰")
	combining := []byte("\u0301")
	meltingFace := []byte("🫠")
	zwjEmoji := []byte("👩‍💻")
	variation := []byte("♥️")

	// ANSI styles force OFFSET:, the wide glyph, and SELECT-ME into separate
	// DOM-renderer spans. The E2E derives a cell width from OFFSET:, verifies the
	// glyph occupies two cells, then performs a real mouse drag over SELECT-ME.
	writeTerminalChunks(out, chunkDelay,
		[]byte("\x1b[3"), []byte("2mOFFSET:"), []byte("\x1b[0m"),
		[]byte("\x1b[35m"), menu[:1], menu[1:2], menu[2:], []byte("\x1b[0m|"),
		[]byte("\x1b[3"), []byte("6mSELECT-ME"), []byte("\x1b[0m|TAIL\r\n"),
		[]byte("\x1b[31mCOMB:A\x1b[0m\x1b[32me"), combining[:1], combining[1:], []byte("\x1b[0m"),
		[]byte("\x1b[33mB|CJK:A\x1b[0m\x1b[34m界\x1b[0m"),
		[]byte("\x1b[35mB|EMOJI:A\x1b[0m\x1b[36m"), meltingFace[:1], meltingFace[1:3], meltingFace[3:], []byte("\x1b[0m"),
		[]byte("\x1b[31mB|ZWJ:A\x1b[0m\x1b[32m"), zwjEmoji[:2], zwjEmoji[2:5], zwjEmoji[5:], []byte("\x1b[0m"),
		[]byte("\x1b[33mB|VS:A\x1b[0m\x1b[34m"), variation[:1], variation[1:3], variation[3:], []byte("\x1b[0m"),
		[]byte("\x1b[35mB|END\x1b[0m\r\n"),
		[]byte("CURSOR:xxxxx"), []byte("\x1b["), []byte("5D"),
		[]byte("OK"), []byte("\x1b[3C\r\n"),
		[]byte("E2E_RENDER_DONE\r\n"),
	)
}

func writeTerminalChunks(out io.Writer, delay time.Duration, chunks ...[]byte) {
	for _, chunk := range chunks {
		_, _ = out.Write(chunk)
		if delay > 0 {
			time.Sleep(delay)
		}
	}
}
