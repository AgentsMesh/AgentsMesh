// Package processmgr is the runner's single source of truth for child-process
// lifecycles. All long-lived child processes go through Manager.Start so that
// every Cmd.Start() is paired with a Cmd.Wait() under a panic-safe goroutine,
// eliminating the zombie accumulation that previously came from scattered
// exec.Command + Release patterns.
package processmgr

import (
	"io"
	"os"
	"time"

	"github.com/creack/pty"
)

// Mode selects the spawn strategy. The runner has exactly three:
//   - ModeNormal: long-lived child, reaped when the runner stops it
//   - ModePTY:    PTY-backed child, owner reads/writes via Process.PTY()
//   - ModeDaemon: double-fork detach via the __launcher__ subcommand; daemon
//                 survives runner restart because its ppid becomes init(1)
type Mode int

const (
	ModeNormal Mode = iota
	ModePTY
	ModeDaemon
)

func (m Mode) String() string {
	switch m {
	case ModeNormal:
		return "normal"
	case ModePTY:
		return "pty"
	case ModeDaemon:
		return "daemon"
	default:
		return "unknown"
	}
}

// Spec is the declarative configuration for spawning a child. Owner is required
// because every metric and debug listing is keyed on it; passing an empty
// Owner is a programmer error and Manager.Start rejects it.
//
// Stdio routing is mutually exclusive: either bind Stdin/Stdout/Stderr to
// existing io.Reader/Writer values, or set the matching PipeStdin/Stdout/
// Stderr flags to have processmgr create os.Pipe pairs that the caller can
// reach via Process.StdinWriter/StdoutReader/StderrReader. The pipe mode is
// what acp/mcp need for their JSON-RPC handshake.
type Spec struct {
	Owner       string
	Command     string
	Args        []string
	Env         []string
	Dir         string
	Mode        Mode
	StopTimeout time.Duration

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	PipeStdin  bool
	PipeStdout bool
	PipeStderr bool

	PTYSize *pty.Winsize
}

// ExitInfo captures everything a caller needs to know after a child has been
// reaped. It is populated exactly once by the reapLoop and is safe to read
// after Handle.Done() closes.
type ExitInfo struct {
	Code     int
	Signal   os.Signal
	Duration time.Duration
	Err      error
}
