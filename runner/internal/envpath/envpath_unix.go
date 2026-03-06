//go:build !windows

package envpath

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// pathSentinel is a unique prefix printed before $PATH so we can extract it
// even when the login shell profile emits other output (nvm init, banners, etc.).
const pathSentinel = "AGENTSMESH_PATH="

// resolveLoginShellPATH resolves the user's login shell PATH by spawning
// a login shell. This is critical when the runner runs as a systemd/launchd
// service, which provides only a minimal PATH (e.g. /usr/bin:/bin).
//
// Resolution order:
//  1. Login shell spawn — reads PATH from the user's interactive login shell.
//  2. Fallback — current process PATH.
func resolveLoginShellPATH() string {
	fallback := os.Getenv("PATH")

	shell := os.Getenv("SHELL")
	if shell == "" {
		slog.Warn("envpath: $SHELL not set, using current PATH")
		return fallback
	}

	if _, err := exec.LookPath(shell); err != nil {
		slog.Warn("envpath: shell binary not found, using current PATH", "shell", shell, "error", err)
		return fallback
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	shellName := filepath.Base(shell)

	// Fish shell uses space-separated PATH and different syntax — use fish-native command.
	if shellName == "fish" {
		cmd := exec.CommandContext(ctx, shell, "-l", "-c",
			fmt.Sprintf("printf '%s%%s\\n' (string join ':' $PATH)", pathSentinel))
		cmd.Env = []string{
			"HOME=" + os.Getenv("HOME"),
			"USER=" + os.Getenv("USER"),
			"LOGNAME=" + os.Getenv("LOGNAME"),
			"SHELL=" + shell,
			"TERM=dumb",
		}
		return extractPATHFromShell(cmd, shell, fallback)
	}

	// POSIX-compatible shells (bash, zsh, sh, etc.)
	cmd := exec.CommandContext(ctx, shell, "-l", "-c",
		fmt.Sprintf("printf '%s%%s\\n' \"$PATH\"", pathSentinel))
	cmd.Env = []string{
		"HOME=" + os.Getenv("HOME"),
		"USER=" + os.Getenv("USER"),
		"LOGNAME=" + os.Getenv("LOGNAME"),
		"SHELL=" + shell,
		"TERM=dumb",
	}
	return extractPATHFromShell(cmd, shell, fallback)
}

// extractPATHFromShell runs the given command and extracts PATH from its output
// by looking for the AGENTSMESH_PATH= sentinel prefix.
func extractPATHFromShell(cmd *exec.Cmd, shell, fallback string) string {
	out, err := cmd.Output()
	if err != nil {
		slog.Warn("envpath: failed to resolve login shell PATH, using current PATH", "shell", shell, "error", err)
		return fallback
	}

	var resolved string
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, pathSentinel) {
			resolved = strings.TrimPrefix(line, pathSentinel)
			break
		}
	}

	// Validate: a real PATH must be non-empty.
	if resolved == "" {
		slog.Warn("envpath: login shell returned empty PATH, using current PATH")
		return fallback
	}

	dirs := strings.Split(resolved, ":")
	slog.Info("envpath: resolved login shell PATH", "shell", filepath.Base(shell), "dirs", len(dirs))

	return resolved
}
