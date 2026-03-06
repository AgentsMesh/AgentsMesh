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
func ResolveLoginShellPATH() string {
	// Fast path: if the service installer captured PATH at install time,
	// use it directly — avoids the overhead of spawning a login shell.
	if envPath := os.Getenv("AGENTSMESH_PATH"); envPath != "" {
		dirs := strings.Split(envPath, string(os.PathListSeparator))
		slog.Info("envpath: using AGENTSMESH_PATH from environment", "dirs", len(dirs))
		return envPath
	}

	return resolveLoginShellPATH()
}
