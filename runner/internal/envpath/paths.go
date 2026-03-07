package envpath

import (
	"os"
	"os/exec"
	"path/filepath"
)

// PathListSeparator returns the OS-specific PATH list separator (":" on Unix, ";" on Windows).
func PathListSeparator() string {
	return string(os.PathListSeparator)
}

// PrependToPath prepends dirs to current, using the OS-specific separator.
// Directories that already appear in current are skipped to avoid duplication.
// Uses filepath.SplitList for exact element matching (avoids substring false positives).
func PrependToPath(current string, dirs ...string) string {
	sep := PathListSeparator()

	// Build element set for exact matching
	existing := make(map[string]struct{})
	for _, elem := range filepath.SplitList(current) {
		existing[elem] = struct{}{}
	}

	for i := len(dirs) - 1; i >= 0; i-- {
		dir := dirs[i]
		if dir == "" {
			continue
		}
		if _, found := existing[dir]; !found {
			current = dir + sep + current
			existing[dir] = struct{}{}
		}
	}
	return current
}

// LookPathFallback searches common user binary directories for a command
// when exec.LookPath fails (e.g. when running under a minimal service PATH).
// Returns the full path if found, empty string otherwise.
func LookPathFallback(command string) string {
	// Try standard LookPath first — it respects the current PATH.
	if path, err := exec.LookPath(command); err == nil {
		return path
	}

	// Fallback: search platform-specific user binary directories.
	for _, dir := range UserBinaryDirs() {
		candidate := filepath.Join(dir, command+exeSuffix())
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
		// Also try without suffix (e.g. shell scripts without .exe on Windows).
		if exeSuffix() != "" {
			candidate = filepath.Join(dir, command)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate
			}
		}
	}

	return ""
}
