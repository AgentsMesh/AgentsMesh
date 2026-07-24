package mockagent

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"time"
)

const (
	terminalAltSnapshotEnterCommand = "enter-alt-buffer"
	terminalAltSnapshotExitCommand  = "exit-alt-buffer"
	terminalAltSnapshotReady        = "terminal-alt-snapshot-ready"
	terminalAltSnapshotActive       = "ALT-BUFFER-ACTIVE"
	terminalAltSnapshotSurface      = "ACTIVE-ALT-SURFACE-MUST-REPLAY"
	terminalAltSnapshotExited       = "ALT-BUFFER-EXITED"
)

// terminalAltSnapshotLoop holds the process inside DECSET 1049 until an
// explicit command exits it. A subscriber that attaches in between can only
// reconstruct both the active alternate surface and the hidden normal history
// from the Runner's snapshot.
func terminalAltSnapshotLoop(in io.Reader, out io.Writer, chunkDelay time.Duration) {
	_, _ = fmt.Fprint(out, terminalAltSnapshotReady+"\r\n")

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	active := false
	exited := false
	for scanner.Scan() {
		switch strings.TrimSpace(scanner.Text()) {
		case terminalAltSnapshotEnterCommand:
			if active || exited {
				continue
			}
			writeTerminalAltSnapshotFixture(out, chunkDelay)
			active = true
		case terminalAltSnapshotExitCommand:
			if !active || exited {
				continue
			}
			writeTerminalChunks(out, chunkDelay,
				[]byte("\x1b[?104"), []byte("9l"),
				[]byte(terminalAltSnapshotExited+"\r\n"),
			)
			active = false
			exited = true
		}
	}
}

func writeTerminalAltSnapshotFixture(out io.Writer, chunkDelay time.Duration) {
	writeTerminalChunks(out, chunkDelay,
		[]byte("\x1b[3J\x1b[2J\x1b[H"),
		[]byte(terminalNormalBufferSentinel+"\r\n"),
	)
	for i := 0; i < 48; i++ {
		_, _ = fmt.Fprintf(out, "ALT-NORMAL-SCROLL-%02d|abcdefghijklmnopqrstuvwxyz|\r\n", i)
	}

	writeTerminalChunks(out, chunkDelay,
		[]byte("\x1b[?10"), []byte("49h"),
		[]byte("\x1b[2J\x1b[H"),
		[]byte(terminalAltSnapshotActive+"\r\n"),
		[]byte(terminalAltSnapshotSurface+"\r\n"),
	)
}
