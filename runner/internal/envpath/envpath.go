package envpath

import (
	"context"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ResolveLoginShellPATH resolves the user's login shell PATH by spawning
// a login shell. This is critical when the runner runs as a systemd/launchd
// service, which provides only a minimal PATH (e.g. /usr/bin:/bin).
//
// On any failure, it falls back to the current process PATH.
func ResolveLoginShellPATH() string {
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

	cmd := exec.CommandContext(ctx, shell, "-l", "-c", "echo $PATH")
	cmd.Env = []string{
		"HOME=" + os.Getenv("HOME"),
		"USER=" + os.Getenv("USER"),
		"LOGNAME=" + os.Getenv("LOGNAME"),
		"SHELL=" + shell,
		"TERM=dumb",
	}

	out, err := cmd.Output()
	if err != nil {
		slog.Warn("envpath: failed to resolve login shell PATH, using current PATH", "shell", shell, "error", err)
		return fallback
	}

	resolved := strings.TrimSpace(string(out))
	if resolved == "" {
		slog.Warn("envpath: login shell returned empty PATH, using current PATH")
		return fallback
	}

	dirs := strings.Split(resolved, ":")
	slog.Info("envpath: resolved login shell PATH", "dirs", len(dirs))

	return resolved
}
