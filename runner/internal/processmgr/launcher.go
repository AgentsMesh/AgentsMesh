package processmgr

import (
	"fmt"
	"os"
	"os/exec"
)

// RunLauncher implements the __processmgr_launcher__ subcommand. The runner's
// main() must call this when os.Args[1] == LauncherSubcommand, before any
// other initialization. It spawns the real daemon binary detached from the
// runner, reports the daemon PID back through launcherPIDFd, and exits —
// which is the move that flips the daemon's ppid to init(1).
//
// This function never returns; it always calls os.Exit.
func RunLauncher() {
	// argv[0] = self, argv[1] = subcommand marker, argv[2] = daemon binary,
	// argv[3...] = daemon args.
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "processmgr launcher: missing daemon binary")
		os.Exit(2)
	}

	binPath := os.Args[2]
	args := os.Args[3:]

	cmd := exec.Command(binPath, args...) //nolint:gosec
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	configureDaemonSysProcAttr(cmd)

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "processmgr launcher: start %s: %v\n", binPath, err)
		os.Exit(1)
	}

	// launcherPIDFd is the PID-reporting pipe wired in by startDaemon. If the
	// runner did not provide one we still want to exit cleanly — the daemon
	// is up and the runner just won't know its real PID until next discovery.
	if pipe := os.NewFile(launcherPIDFd, "processmgr-pid-pipe"); pipe != nil {
		_, _ = fmt.Fprintln(pipe, cmd.Process.Pid)
		_ = pipe.Close()
	}

	// Release detaches the *os.Process bookkeeping in the launcher's heap;
	// when the launcher exits a moment later, the kernel reparents the
	// daemon to init and the runner is fully out of the loop.
	if err := cmd.Process.Release(); err != nil {
		fmt.Fprintf(os.Stderr, "processmgr launcher: release: %v\n", err)
		os.Exit(1)
	}

	os.Exit(0)
}
