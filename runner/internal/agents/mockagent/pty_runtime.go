// Package mockagent implements the runtime for the e2e-mock-agent binary.
// It is intentionally split from the cmd entrypoint so that PTY and ACP
// runtimes are independently testable.
package mockagent

import (
	"bufio"
	"fmt"
	"io"
	"log/slog"
	"os"
)

// RunPTY drives the PTY-mode runtime. It writes "ready" to signal liveness
// (consumed by mcp-e2e get_pod_snapshot polling), optionally dumps whitelisted
// env vars to a file (env-bundle e2e regression), then loops echoing each
// stdin line as `got: <line>` to enable round-trip checks via send_pod_input
// + get_pod_snapshot.
//
// Returns process exit code (0 on clean EOF).
func RunPTY(scenario string, logger *slog.Logger) int {
	return runPTYWithIO(scenario, os.Stdin, os.Stdout, os.Environ(), logger)
}

func runPTYWithIO(scenario string, in io.Reader, out io.Writer, env []string, logger *slog.Logger) int {
	switch scenario {
	case "", "echo":
		_, _ = fmt.Fprintln(out, "ready")
		writeEnvDump(env)
		echoLoop(in, out)
		return 0
	default:
		_, _ = fmt.Fprintf(out, "unknown PTY scenario: %s\n", scenario)
		logger.Error("unknown PTY scenario", "scenario", scenario)
		return 2
	}
}

func echoLoop(in io.Reader, out io.Writer) {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		fmt.Fprintf(out, "got: %s\n", scanner.Text())
	}
}
