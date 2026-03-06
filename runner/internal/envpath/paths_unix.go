//go:build !windows

package envpath

import (
	"os"
	"path/filepath"
	"runtime"
)

// UserBinaryDirs returns common directories where user-installed binaries live.
//
//   - darwin: ~/.local/bin, /opt/homebrew/bin, /opt/homebrew/sbin, /usr/local/bin
//   - linux:  ~/.local/bin, /usr/local/bin, /snap/bin
func UserBinaryDirs() []string {
	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(home, ".local", "bin"),
	}

	if runtime.GOOS == "darwin" {
		dirs = append(dirs,
			"/opt/homebrew/bin",
			"/opt/homebrew/sbin",
			"/usr/local/bin",
		)
	} else {
		// linux and other unix
		dirs = append(dirs,
			"/usr/local/bin",
			"/snap/bin",
		)
	}

	return dirs
}

// exeSuffix returns the executable file extension for the current OS.
func exeSuffix() string {
	return ""
}

// DefaultSystemPath returns a minimal system PATH for the current OS.
func DefaultSystemPath() string {
	return "/usr/bin:/bin:/usr/sbin:/sbin"
}
