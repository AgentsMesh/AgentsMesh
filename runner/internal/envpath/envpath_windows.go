//go:build windows

package envpath

import (
	"log/slog"
	"os"
)

// resolveLoginShellPATH on Windows returns the current process PATH directly.
// Unlike Unix, Windows services typically inherit a usable PATH that includes
// System32 and Program Files directories. There is no "login shell" concept
// equivalent to Unix's $SHELL -l -c.
func resolveLoginShellPATH() string {
	path := os.Getenv("PATH")
	slog.Info("envpath: using current process PATH (Windows)", "path_length", len(path))
	return path
}
