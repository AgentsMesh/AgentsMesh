//go:build windows

package envpath

// ShellCommand returns the default shell and flag for executing inline scripts.
// On Windows this is "cmd.exe" with "/C".
func ShellCommand() (shell, flag string) {
	return "cmd.exe", "/C"
}
