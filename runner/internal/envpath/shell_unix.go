//go:build !windows

package envpath

// ShellCommand returns the default shell and flag for executing inline scripts.
// On Unix systems this is "/bin/sh" with "-c".
func ShellCommand() (shell, flag string) {
	return "/bin/sh", "-c"
}
