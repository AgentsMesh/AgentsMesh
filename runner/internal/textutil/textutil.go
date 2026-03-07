// Package textutil provides cross-platform text processing helpers.
package textutil

import "strings"

// NormalizeLineEndings replaces Windows-style \r\n with Unix-style \n.
// Use this before splitting text by lines when the input may come from
// platform-dependent sources (e.g., git output, config files on Windows).
func NormalizeLineEndings(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

// SplitLines splits text into lines after normalizing line endings.
// Equivalent to strings.Split(NormalizeLineEndings(s), "\n").
func SplitLines(s string) []string {
	return strings.Split(NormalizeLineEndings(s), "\n")
}
