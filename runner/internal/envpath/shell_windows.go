//go:build windows

package envpath

import (
	"os"
	"os/exec"
)

// ShellCommand returns the default shell and flag for executing inline scripts.
// On Windows, prefer bash.exe (from Git for Windows) for compatibility with
// Unix-style preparation scripts. Falls back to cmd.exe if bash is unavailable.
func ShellCommand() (shell, flag string) {
	// Try bash in PATH first (Git for Windows adds it)
	if bash, err := exec.LookPath("bash"); err == nil {
		return bash, "-c"
	}

	// Check common Git for Windows installation paths
	for _, p := range []string{
		`C:\Program Files\Git\bin\bash.exe`,
		`C:\Program Files (x86)\Git\bin\bash.exe`,
	} {
		if _, err := os.Stat(p); err == nil {
			return p, "-c"
		}
	}

	return "cmd.exe", "/C"
}
