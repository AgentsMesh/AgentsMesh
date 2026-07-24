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
	"os/exec"
	"path/filepath"
	"time"
)

// RunPTY drives the PTY-mode runtime. It writes "ready" to signal liveness
// (consumed by mcp-e2e get_pod_snapshot polling), optionally dumps whitelisted
// env vars to a file (env-bundle e2e regression), then loops echoing each
// stdin line as `got: <line>` to enable round-trip checks via send_pod_input
// + get_pod_snapshot.
//
// Returns process exit code (0 on clean EOF).
func RunPTY(scenario string, logger *slog.Logger) int {
	if scenario == "terminal_render" {
		writeEnvDump(os.Environ())
		resizeSignals, stopResizeSignals := watchTerminalResize()
		defer stopResizeSignals()
		terminalRenderLoopWithResize(
			os.Stdin,
			os.Stdout,
			terminalRenderChunkDelay,
			terminalRenderAltHold,
			resizeSignals,
			stdoutTerminalSize,
		)
		return 0
	}
	return runPTYWithIO(scenario, os.Stdin, os.Stdout, os.Environ(), logger)
}

// "❯" is in detector.promptEndSymbols → the PTY state detector scores the
// trailing screen line as a prompt and transitions the pod to "waiting".
const autopilotPrompt = "❯ "

// autopilotTurnDelay lands the executing→waiting edge past the
// AutopilotController's 5s MinTriggerGap; an instant echo gets deduped.
const autopilotTurnDelay = 6 * time.Second

func runPTYWithIO(scenario string, in io.Reader, out io.Writer, env []string, logger *slog.Logger) int {
	switch scenario {
	case "", "echo":
		_, _ = fmt.Fprintln(out, "ready")
		writeEnvDump(env)
		echoLoop(in, out)
		return 0
	case "autopilot":
		_, _ = fmt.Fprintln(out, "ready")
		writeEnvDump(env)
		_, _ = fmt.Fprint(out, autopilotPrompt)
		promptEchoLoop(in, out, autopilotTurnDelay)
		return 0
	case "autopilot_fs":
		seedSandboxGitChange()
		_, _ = fmt.Fprintln(out, "ready")
		writeEnvDump(env)
		_, _ = fmt.Fprint(out, autopilotPrompt)
		promptEchoLoop(in, out, autopilotTurnDelay)
		return 0
	case "terminal_render":
		writeEnvDump(env)
		terminalRenderLoop(in, out, terminalRenderChunkDelay, terminalRenderAltHold)
		return 0
	case "terminal_alt_snapshot":
		writeEnvDump(env)
		terminalAltSnapshotLoop(in, out, terminalRenderChunkDelay)
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

// Echo, hold for turnDelay (the non-prompt "got:" line keeps the pod
// executing), then print the prompt to drive the executing→waiting edge.
func promptEchoLoop(in io.Reader, out io.Writer, turnDelay time.Duration) {
	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		fmt.Fprintf(out, "got: %s\n", scanner.Text())
		if turnDelay > 0 {
			time.Sleep(turnDelay)
		}
		_, _ = fmt.Fprint(out, autopilotPrompt)
	}
}

// seedSandboxGitChange git-inits the pod's SandboxPath (parent of the agent's
// cwd) and drops a probe file there, so the AutopilotController's
// ProgressTracker (`git status` in SandboxPath) reports a non-empty
// files_changed on its iterations.
func seedSandboxGitChange() {
	wd, err := os.Getwd()
	if err != nil {
		return
	}
	sandbox := filepath.Dir(wd)
	_ = exec.Command("git", "-C", sandbox, "init").Run()
	_ = os.WriteFile(filepath.Join(sandbox, "autopilot-probe.txt"), []byte("probe\n"), 0o644)
}
