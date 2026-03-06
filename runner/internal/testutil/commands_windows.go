//go:build windows

package testutil

import "fmt"

// EchoCommand returns a command and args that echo the given text to stdout.
func EchoCommand(text string) (cmd string, args []string) {
	return "cmd.exe", []string{"/C", "echo", text}
}

// CatCommand returns a command and args that read stdin and write to stdout.
// On Windows, "more" reads stdin line by line (closest equivalent of cat).
func CatCommand() (cmd string, args []string) {
	return "cmd.exe", []string{"/C", "more"}
}

// SleepCommand returns a command and args that sleep for the given number of seconds.
// Uses PowerShell's Start-Sleep since Windows has no native sleep command.
func SleepCommand(seconds int) (cmd string, args []string) {
	return "powershell", []string{"-Command", fmt.Sprintf("Start-Sleep -Seconds %d", seconds)}
}

// TrueCommand returns a command that exits with code 0.
func TrueCommand() (cmd string, args []string) {
	return "cmd.exe", []string{"/C", "exit", "0"}
}

// FalseCommand returns a command that exits with a non-zero code.
func FalseCommand() (cmd string, args []string) {
	return "cmd.exe", []string{"/C", "exit", "1"}
}

// ShellCommand returns the default shell and its flag for inline script execution.
func ShellCommand() (cmd string, args []string) {
	return "cmd.exe", []string{"/C"}
}

// ShellScript returns the shell, flag, and script body as a ready-to-use command+args.
// Example: ShellScript("echo hello") → ("cmd.exe", ["/C", "echo hello"])
func ShellScript(script string) (cmd string, args []string) {
	return "cmd.exe", []string{"/C", script}
}
