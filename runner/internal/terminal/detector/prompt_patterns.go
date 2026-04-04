package detector

import "strings"

// Common prompt ending symbols
var promptEndSymbols = []string{
	">", ">>>", "»", "›", // Common input prompts
	"$", "#", "%", // Shell prompts
	"?", ":", // Question/input prompts
	"⟩", "❯", "➜", "→", // Fancy prompts
}

// hasPromptSymbol checks if the line contains a common prompt symbol.
// It checks both suffix (traditional prompts) and contains (TUI-style prompts like Claude Code).
func (d *PromptDetector) hasPromptSymbol(line string) bool {
	trimmed := strings.TrimRight(line, " \t")
	if len(trimmed) == 0 {
		return false
	}

	// Check if line ends with a prompt symbol (traditional shell prompts)
	for _, symbol := range promptEndSymbols {
		if strings.HasSuffix(trimmed, symbol) {
			return true
		}
	}

	// Check for TUI-style prompts where the symbol is in the middle
	// (e.g., Claude Code's "❯ ───" input line)
	tuiPromptSymbols := []string{"❯", "➜", "›", "»"}
	for _, symbol := range tuiPromptSymbols {
		// Check if symbol exists followed by space or box-drawing chars
		idx := strings.Index(line, symbol)
		if idx >= 0 && idx < len(line)-len(symbol) {
			// Found TUI prompt symbol with content after it
			return true
		}
	}

	// Check for box-drawing prompts (│ > │ style)
	if strings.Contains(line, "│") && strings.Contains(line, ">") {
		return true
	}

	return false
}

// isConfirmPrompt checks for y/n style confirmation prompts.
func (d *PromptDetector) isConfirmPrompt(line string) bool {
	lower := strings.ToLower(line)

	patterns := []string{
		"(y/n)",
		"[y/n]",
		"(yes/no)",
		"[yes/no]",
		"y/n:",
		"yes/no:",
		"y or n",
		"yes or no",
	}

	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}

	// Check if line ends with y/n pattern
	if strings.HasSuffix(lower, "?") {
		// Questions that likely need yes/no
		questionIndicators := []string{"continue", "proceed", "confirm", "sure", "want to", "would you"}
		for _, indicator := range questionIndicators {
			if strings.Contains(lower, indicator) {
				return true
			}
		}
	}

	return false
}

// isPermissionPrompt checks for permission/approval prompts.
func (d *PromptDetector) isPermissionPrompt(line string) bool {
	lower := strings.ToLower(line)

	keywords := []string{
		"permission",
		"approve",
		"allow",
		"authorize",
		"grant",
		"accept",
		"deny",
		"reject",
	}

	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			// Make sure it's a question or request, not just a statement
			if strings.Contains(lower, "?") ||
				strings.Contains(lower, "allow") ||
				strings.Contains(lower, "approve") {
				return true
			}
		}
	}

	return false
}

// isContinuePrompt checks for "press any key" style prompts.
func (d *PromptDetector) isContinuePrompt(line string) bool {
	lower := strings.ToLower(line)

	patterns := []string{
		"press any key",
		"press enter",
		"hit enter",
		"press return",
		"to continue",
		"to proceed",
		"when ready",
	}

	for _, p := range patterns {
		if strings.Contains(lower, p) {
			return true
		}
	}

	return false
}

// isKeyboardShortcutPrompt checks for keyboard shortcut selection prompts.
// Examples: "[Tab] Accept [Esc] Reject", "[y] Yes [n] No"
func (d *PromptDetector) isKeyboardShortcutPrompt(line string) bool {
	lower := strings.ToLower(line)

	// Check for multiple [key] patterns indicating shortcut options
	bracketCount := 0
	for i := 0; i < len(lower); i++ {
		if lower[i] == '[' {
			// Look for closing bracket
			for j := i + 1; j < len(lower) && j < i+10; j++ {
				if lower[j] == ']' {
					bracketCount++
					i = j
					break
				}
			}
		}
	}

	// If we have 2+ bracketed items, likely a shortcut prompt
	if bracketCount >= 2 {
		// Verify it contains action words
		actionWords := []string{"accept", "reject", "yes", "no", "edit", "cancel", "apply", "skip", "abort"}
		for _, word := range actionWords {
			if strings.Contains(lower, word) {
				return true
			}
		}
	}

	return false
}

// hasInputIndicator checks for generic input request indicators.
func (d *PromptDetector) hasInputIndicator(line string) bool {
	lower := strings.ToLower(line)

	// Check for question marks at the end
	trimmed := strings.TrimRight(lower, " \t")
	if strings.HasSuffix(trimmed, "?") {
		return true
	}

	// Check for "label: (default)" pattern common in CLI prompts
	// e.g., "package name: (my-project)", "version: (1.0.0)"
	if strings.Contains(lower, ":") && strings.HasSuffix(trimmed, ")") {
		// Find the colon and check if there's a parenthetical default after it
		colonIdx := strings.LastIndex(lower, ":")
		parenIdx := strings.LastIndex(lower, "(")
		if colonIdx != -1 && parenIdx != -1 && parenIdx > colonIdx {
			return true
		}
	}

	// Check for input request keywords
	keywords := []string{
		"enter",
		"input",
		"type",
		"provide",
		"specify",
		"password",
		"passphrase",
		"name",
	}

	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			// Look for request context
			if strings.Contains(lower, "please") ||
				strings.Contains(lower, "your") ||
				strings.Contains(lower, ":") {
				return true
			}
		}
	}

	return false
}

// IsPromptChar checks if a character is commonly used in prompts.
func IsPromptChar(r rune) bool {
	promptChars := []rune{'>', '<', '$', '#', '%', '?', ':', '»', '›', '⟩', '❯', '➜', '→'}
	for _, c := range promptChars {
		if r == c {
			return true
		}
	}
	return false
}
