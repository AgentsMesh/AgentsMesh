package terminal

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// OnScreenUpdate should be called when the terminal screen content changes.
// Provide the current screen lines for analysis.
func (d *MultiSignalDetector) OnScreenUpdate(lines []string) {
	// Compute screen hash OUTSIDE lock to minimize lock contention
	// This is safe because computeScreenHash only reads the lines slice
	hash := d.computeScreenHash(lines)
	now := time.Now()

	// Minimal lock scope - only update state
	d.mu.Lock()
	hashChanged := hash != d.lastScreenHash
	if hashChanged {
		// Screen changed
		d.lastScreenHash = hash
		d.lastScreenTime = now
		d.screenStableTime = 0
	} else {
		// Screen stable
		d.screenStableTime = now.Sub(d.lastScreenTime)
	}
	// Store lines for prompt detection
	d.screenLines = lines
	d.mu.Unlock()

	// Debug logging OUTSIDE lock to avoid blocking PTY output
	if hashChanged && len(lines) > 0 {
		// Find non-empty lines for debugging
		var nonEmptyLines []string
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if len(trimmed) > 0 {
				// Truncate long lines
				if len(trimmed) > 60 {
					trimmed = trimmed[:60] + "..."
				}
				nonEmptyLines = append(nonEmptyLines, trimmed)
			}
		}
		// Only log last few non-empty lines
		if len(nonEmptyLines) > 5 {
			nonEmptyLines = nonEmptyLines[len(nonEmptyLines)-5:]
		}
		logger.TerminalTrace().Trace("MultiSignalDetector screen update",
			"hash_changed", hashChanged,
			"hash", hash[:8],
			"non_empty_count", len(nonEmptyLines),
			"last_non_empty_lines", nonEmptyLines)
	}
}

// OnOSCTitle should be called when an OSC title update is received.
// This is an optional signal that can boost confidence.
func (d *MultiSignalDetector) OnOSCTitle(title string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.lastOSCTitle = title
	d.lastOSCTitleTime = time.Now()
}

// computeScreenHash computes a hash of the screen content.
// It normalizes spinner/animation characters to produce stable hashes when
// only the animation frame changes (e.g., Claude Code's "Tinkering..." spinner).
func (d *MultiSignalDetector) computeScreenHash(lines []string) string {
	h := sha256.New()
	for _, line := range lines {
		// Normalize the line by replacing common spinner characters with a placeholder
		normalized := normalizeSpinnerChars(line)
		h.Write([]byte(normalized))
		h.Write([]byte{'\n'})
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// normalizeSpinnerChars replaces common spinner/animation characters with a placeholder.
// This helps maintain screen stability detection even when animations are running.
func normalizeSpinnerChars(line string) string {
	// Common spinner characters used by CLI tools
	spinnerChars := []rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏', // dots
		'⣾', '⣽', '⣻', '⢿', '⡿', '⣟', '⣯', '⣷', // braille
		'◐', '◓', '◑', '◒', // circle quarters
		'◴', '◷', '◶', '◵', // other circles
		'◰', '◳', '◲', '◱', // squares
		'▖', '▘', '▝', '▗', // corners
		'⠐', '⠂', '⠈', '⠁', '⠠', '⠄', '⠤', '⠤', // braille small
		'*', '·', '•', '●', '○', '◎', '◉', // bullets
		'✻', '✽', '✼', '✾', '✿', '❀', // flowers
		'⏸', '⏵', '⏴', '▶', '◀', '⏹', '⏺', // media controls
	}

	// Create a map for fast lookup
	spinnerSet := make(map[rune]bool)
	for _, c := range spinnerChars {
		spinnerSet[c] = true
	}

	// Replace spinner chars with a placeholder
	result := make([]rune, 0, len(line))
	for _, r := range line {
		if spinnerSet[r] {
			result = append(result, '·') // normalize to single placeholder
		} else {
			result = append(result, r)
		}
	}
	return string(result)
}
