package poddaemon

import "strings"

// buildWindowsCmdLine joins the executable path and arguments into a single
// command-line string suitable for Windows CreateProcess / ConPTY.
//
// Windows command-line parsing rules (from MSDN CommandLineToArgvW):
//   - Arguments containing spaces, tabs, or double-quotes must be wrapped in double-quotes.
//   - Inside a quoted argument, backslashes before a double-quote must be escaped.
//
// This is pure string manipulation with no Windows API dependency, so it lives
// in a cross-platform file for testability on all platforms.
func buildWindowsCmdLine(path string, args []string) string {
	parts := make([]string, 0, 1+len(args))
	parts = append(parts, quoteWindowsArg(path))
	for _, arg := range args {
		parts = append(parts, quoteWindowsArg(arg))
	}
	return strings.Join(parts, " ")
}

// quoteWindowsArg quotes a single argument for Windows command-line parsing.
// An argument needs quoting if it contains spaces, tabs, or double-quotes.
// Empty arguments are also quoted to preserve them as empty strings.
func quoteWindowsArg(arg string) string {
	if arg == "" {
		return `""`
	}
	if !strings.ContainsAny(arg, " \t\"") {
		return arg
	}

	// Build quoted string following MSDN escaping rules:
	// - Wrap in double quotes
	// - Escape backslashes that precede a double-quote
	// - Escape double-quotes with backslash
	var b strings.Builder
	b.WriteByte('"')

	for i := 0; i < len(arg); i++ {
		switch arg[i] {
		case '\\':
			// Count consecutive backslashes
			numBackslashes := 0
			for i < len(arg) && arg[i] == '\\' {
				numBackslashes++
				i++
			}
			if i == len(arg) {
				// Backslashes at end of arg: double them (they precede the closing quote)
				for range numBackslashes * 2 {
					b.WriteByte('\\')
				}
			} else if arg[i] == '"' {
				// Backslashes before a quote: double them + escape the quote
				for range numBackslashes * 2 {
					b.WriteByte('\\')
				}
				b.WriteString(`\"`)
			} else {
				// Backslashes not before a quote: keep as-is
				for range numBackslashes {
					b.WriteByte('\\')
				}
				b.WriteByte(arg[i])
			}
		case '"':
			b.WriteString(`\"`)
		default:
			b.WriteByte(arg[i])
		}
	}

	b.WriteByte('"')
	return b.String()
}
