package envpath

import (
	"log/slog"
	"os"
	"strings"
)

// ResolveLoginShellPATH resolves the user's effective PATH.
//
// Resolution order:
//  1. AGENTSMESH_PATH env var — if set (e.g. captured at service install time),
//     use it directly and skip the expensive login shell spawn.
//  2. Platform-specific resolution (Unix: login shell spawn; Windows: current PATH).
//  3. Fallback — current process PATH.
//
// The resolved PATH is then merged with the current process PATH to ensure
// directories from the interactive session (e.g., nvm, pyenv, rbenv) are not
// lost. This is critical for Pod Daemon mode, where the daemon subprocess
// inherits only the resolved PATH and uses it for exec.LookPath.
func ResolveLoginShellPATH() string {
	// Fast path: if the service installer captured PATH at install time,
	// use it directly — avoids the overhead of spawning a login shell.
	if envPath := os.Getenv("AGENTSMESH_PATH"); envPath != "" {
		dirs := strings.Split(envPath, string(os.PathListSeparator))
		slog.Info("envpath: using AGENTSMESH_PATH from environment", "dirs", len(dirs))
		return mergeWithCurrentPATH(envPath)
	}

	return mergeWithCurrentPATH(resolveLoginShellPATH())
}

// mergeWithCurrentPATH merges the resolved PATH with the current process PATH.
// Resolved dirs take priority (appear first); unique dirs from the current
// process PATH are appended. This ensures tools installed via version managers
// (nvm, pyenv, etc.) that initialize in .bashrc (but not login shell profiles)
// are still available when the runner was started from an interactive session.
func mergeWithCurrentPATH(resolved string) string {
	current := os.Getenv("PATH")
	if current == "" || current == resolved {
		return resolved
	}

	sep := string(os.PathListSeparator)
	resolvedDirs := strings.Split(resolved, sep)

	seen := make(map[string]bool, len(resolvedDirs))
	for _, d := range resolvedDirs {
		seen[d] = true
	}

	var extra int
	for _, d := range strings.Split(current, sep) {
		if d != "" && !seen[d] {
			resolvedDirs = append(resolvedDirs, d)
			seen[d] = true
			extra++
		}
	}

	if extra > 0 {
		slog.Info("envpath: merged additional dirs from current process PATH", "extra_dirs", extra)
	}

	return strings.Join(resolvedDirs, sep)
}
