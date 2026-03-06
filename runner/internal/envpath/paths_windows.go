//go:build windows

package envpath

import (
	"os"
	"path/filepath"
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
