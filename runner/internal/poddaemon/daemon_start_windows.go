//go:build windows

package poddaemon

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"syscall"
)

// startDaemon re-execs the runner binary as a detached daemon process on Windows.
func startDaemon(binPath string, configPath string, sandboxPath string, env []string) (int, error) {
	logPath := filepath.Join(sandboxPath, "pod_daemon.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return 0, fmt.Errorf("open daemon log: %w", err)
	}
	defer logFile.Close()

	daemonEnv := append(slices.Clone(env), "_AGENTSMESH_POD_DAEMON="+configPath)

	// Use NUL as stdin — daemon doesn't need it (consistent with Unix /dev/null)
	devNull, err := os.Open(os.DevNull)
	if err != nil {
		return 0, fmt.Errorf("open devnull: %w", err)
	}
	defer devNull.Close()

	attr := &os.ProcAttr{
		Dir:   sandboxPath,
		Env:   daemonEnv,
		Files: []*os.File{devNull, logFile, logFile},
		Sys: &syscall.SysProcAttr{
			CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
		},
	}

	proc, err := os.StartProcess(binPath, []string{binPath}, attr)
	if err != nil {
		return 0, fmt.Errorf("start daemon process: %w", err)
	}

	pid := proc.Pid
	if err := proc.Release(); err != nil {
		return pid, fmt.Errorf("release daemon process: %w", err)
	}

	return pid, nil
}
