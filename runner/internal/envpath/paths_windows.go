//go:build windows

package envpath

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// UserBinaryDirs returns common directories where user-installed binaries live on Windows.
//
//   - %USERPROFILE%\.local\bin
//   - %LOCALAPPDATA%\Programs
//   - %ProgramFiles%
func UserBinaryDirs() []string {
	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(home, ".local", "bin"),
	}

	if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
		dirs = append(dirs, filepath.Join(localAppData, "Programs"))
	}

	if programFiles := os.Getenv("ProgramFiles"); programFiles != "" {
		dirs = append(dirs, programFiles)
	}

	return dirs
}

// exeSuffix returns the executable file extension for Windows.
func exeSuffix() string {
	return ".exe"
}

// DefaultSystemPath returns a minimal system PATH for Windows.
func DefaultSystemPath() string {
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		systemRoot = `C:\Windows`
	}
	return filepath.Join(systemRoot, "System32") + ";" + systemRoot
}

// defaultWindowsExts is the fallback list when PATHEXT is empty.
var defaultWindowsExts = []string{".COM", ".EXE", ".BAT", ".CMD"}

// ValidateExecutable checks that path has a Windows-executable extension (from PATHEXT).
// If not (e.g. an extensionless npm Unix shell script), it probes path+ext for each
// PATHEXT extension and returns the first match. Returns "" if nothing is executable.
func ValidateExecutable(path string) string {
	exts := getPathExts()

	// Already has a valid extension — return as-is.
	ext := strings.ToUpper(filepath.Ext(path))
	for _, e := range exts {
		if ext == e {
			return path
		}
	}

	// Try appending each PATHEXT extension.
	for _, e := range exts {
		candidate := path + strings.ToLower(e)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

// SafeLookPath searches PATH for a command, respecting PATHEXT on Windows.
// Unlike exec.LookPath, it never returns extensionless Unix shell scripts.
func SafeLookPath(command string) (string, error) {
	exts := getPathExts()
	pathEnv := os.Getenv("PATH")

	for _, dir := range filepath.SplitList(pathEnv) {
		if dir == "" {
			dir = "."
		}
		for _, e := range exts {
			candidate := filepath.Join(dir, command+strings.ToLower(e))
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}
		}
	}
	return "", &exec.Error{Name: command, Err: exec.ErrNotFound}
}

// getPathExts returns executable extensions from PATHEXT, or defaults.
func getPathExts() []string {
	pathext := os.Getenv("PATHEXT")
	if pathext == "" {
		return defaultWindowsExts
	}
	parts := strings.Split(strings.ToUpper(pathext), ";")
	var exts []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			exts = append(exts, p)
		}
	}
	if len(exts) == 0 {
		return defaultWindowsExts
	}
	return exts
}
