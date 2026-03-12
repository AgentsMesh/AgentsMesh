package tokenusage

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// AiderParser parses Aider markdown chat history files.
// Aider writes chat history to:
//   - {sandboxPath}/workspace/.aider.chat.history.md
//   - {sandboxPath}/.aider.chat.history.md
//
// Token lines look like:
//
//	> Tokens: 12k sent, 3.4k received, 45k cache write, 123k cache read
//	> Tokens: 1,234 sent, 567 received
type AiderParser struct{}

// tokenLineRe matches Aider's token summary lines.
var tokenLineRe = regexp.MustCompile(`>\s*Tokens:\s*(.+)`)

// tokenValueRe extracts a numeric value with optional k/m suffix and a label.
var tokenValueRe = regexp.MustCompile(`([\d,]+(?:\.\d+)?)\s*([kmKM])?\s+(sent|received|cache\s+write|cache\s+read)`)

func (p *AiderParser) Parse(sandboxPath string, podStartedAt time.Time) (*TokenUsage, error) {
	usage := NewTokenUsage()

	candidates := []string{
		filepath.Join(sandboxPath, "workspace", ".aider.chat.history.md"),
		filepath.Join(sandboxPath, ".aider.chat.history.md"),
	}

	for _, path := range candidates {
		if !isModifiedAfter(path, podStartedAt) {
			continue
		}
		if err := parseAiderHistoryFile(path, usage); err != nil {
			logger.Pod().Warn("Aider parser: file parse error", "file", path, "error", err)
		}
	}

	if usage.IsEmpty() {
		return nil, nil
	}
	return usage, nil
}

func parseAiderHistoryFile(path string, usage *TokenUsage) error {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := scanner.Text()
		match := tokenLineRe.FindStringSubmatch(line)
		if match == nil {
			continue
		}

		tokenStr := match[1]
		var sent, received, cacheWrite, cacheRead int64

		for _, m := range tokenValueRe.FindAllStringSubmatch(tokenStr, -1) {
			value := parseTokenValue(m[1], m[2])
			label := strings.ToLower(m[3])
			switch {
			case label == "sent":
				sent = value
			case label == "received":
				received = value
			case strings.Contains(label, "cache") && strings.Contains(label, "write"):
				cacheWrite = value
			case strings.Contains(label, "cache") && strings.Contains(label, "read"):
				cacheRead = value
			}
		}

		if sent > 0 || received > 0 {
			// Aider doesn't report model name in token lines; use a generic label
			usage.Add("aider-unknown", sent, received, cacheWrite, cacheRead)
		}
	}

	return scanner.Err()
}

// parseTokenValue parses a numeric string with optional k/m suffix.
// Examples: "12" -> 12, "3.4k" -> 3400, "1,234" -> 1234
func parseTokenValue(numStr, suffix string) int64 {
	// Remove commas
	numStr = strings.ReplaceAll(numStr, ",", "")

	val, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0
	}

	switch strings.ToLower(suffix) {
	case "k":
		val *= 1000
	case "m":
		val *= 1_000_000
	}

	return int64(val)
}
