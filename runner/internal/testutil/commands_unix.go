//go:build !windows

package testutil

import "fmt"

// EchoCommand returns a command and args that echo the given text to stdout.
func EchoCommand(text string) (cmd string, args []string) {
	return "echo", []string{text}
}

// CatCommand returns a command and args that read stdin and write to stdout.
func CatCommand() (cmd string, args []string) {
	return "cat", nil
}

// SleepCommand returns a command and args that sleep for the given number of seconds.
func SleepCommand(seconds int) (cmd string, args []string) {
	return "sleep", []string{fmt.Sprintf("%d", seconds)}
}

// TrueCommand returns a command that exits with code 0.
func TrueCommand() (cmd string, args []string) {
	return "true", nil
}

// FalseCommand returns a command that exits with a non-zero code.
func FalseCommand() (cmd string, args []string) {
	return "false", nil
}

// ShellCommand returns the default shell and its flag for inline script execution.
func ShellCommand() (cmd string, args []string) {
	return "/bin/sh", []string{"-c"}
}

// ShellScript returns the shell, flag, and script body as a ready-to-use command+args.
// Example: ShellScript("echo hello && sleep 1") → ("/bin/sh", ["-c", "echo hello && sleep 1"])
func ShellScript(script string) (cmd string, args []string) {
	return "/bin/sh", []string{"-c", script}
}
