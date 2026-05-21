package poddaemon

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/anthropics/agentsmesh/runner/internal/processmgr"
)

// startDaemon re-execs the runner binary as a detached daemon and returns the
// daemon's PID. Detachment goes through processmgr.ModeDaemon so that the
// launcher (which used to leak zombies on the runner's process table) is
// always Wait'd. The daemon itself is reparented to init(1) on Unix or made
// fully independent via DETACHED_PROCESS on Windows.
//
// _AGENTSMESH_POD_DAEMON tells the re-execed runner to enter daemon mode and
// read its config from configPath.
func startDaemon(binPath, configPath, sandboxPath string, env []string) (int, error) {
	logPath := filepath.Join(sandboxPath, "pod_daemon.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return 0, fmt.Errorf("open daemon log: %w", err)
	}
	defer logFile.Close()

	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return 0, fmt.Errorf("open devnull: %w", err)
	}
	defer devNull.Close()

	daemonEnv := append(slices.Clone(env), "_AGENTSMESH_POD_DAEMON="+configPath)

	p, err := processmgr.Global().Start(context.Background(), processmgr.Spec{
		Owner:   "poddaemon:" + filepath.Base(sandboxPath),
		Command: binPath,
		Env:     daemonEnv,
		Dir:     sandboxPath,
		Mode:    processmgr.ModeDaemon,
		Stdin:   devNull,
		Stdout:  logFile,
		Stderr:  logFile,
	})
	if err != nil {
		return 0, fmt.Errorf("start daemon: %w", err)
	}
	return p.PID(), nil
}
