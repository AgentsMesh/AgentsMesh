package tokenusage

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// ClaudeParser parses Claude Code JSONL session files.
// Claude Code writes conversation history to JSONL files under:
//   - {sandboxPath}/.claude/projects/**/*.jsonl (pod-specific)
//
// Only files modified after podStartedAt are processed to avoid
// re-counting historical sessions from previous pod runs.
type ClaudeParser struct{}

// claudeJSONLEntry represents a single line in a Claude Code JSONL file.
type claudeJSONLEntry struct {
	Type    string `json:"type"`
	Message struct {
		Model string `json:"model"`
		Usage struct {
			InputTokens              int64 `json:"input_tokens"`
			OutputTokens             int64 `json:"output_tokens"`
			CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
		} `json:"usage"`
	} `json:"message"`
}

func (p *ClaudeParser) Parse(sandboxPath string, podStartedAt time.Time) (*TokenUsage, error) {
	usage := NewTokenUsage()

	// Only scan sandbox-local path — this is the current pod's project directory.
	// Scanning HOME would pick up sessions from other pods/projects.
	pattern := filepath.Join(sandboxPath, ".claude", "projects", "*", "*.jsonl")

	files, err := filepath.Glob(pattern)
	if err != nil {
		logger.Pod().Warn("Claude parser: glob error", "pattern", pattern, "error", err)
		return nil, nil
	}

	for _, f := range files {
		if !isModifiedAfter(f, podStartedAt) {
			continue
		}
		if err := parseClaudeJSONLFile(f, usage); err != nil {
			logger.Pod().Warn("Claude parser: file parse error", "file", f, "error", err)
		}
	}

	if usage.IsEmpty() {
		return nil, nil
	}
	return usage, nil
}

func parseClaudeJSONLFile(path string, usage *TokenUsage) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Allow large lines (up to 10MB) for JSONL entries with large content.
	// Claude Code JSONL can contain full conversation history with tool results.
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry claudeJSONLEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue // Skip malformed lines
		}

		if entry.Type != "assistant" || entry.Message.Model == "" {
			continue
		}

		u := entry.Message.Usage
		if u.InputTokens == 0 && u.OutputTokens == 0 {
			continue
		}

		usage.Add(
			entry.Message.Model,
			u.InputTokens,
			u.OutputTokens,
			u.CacheCreationInputTokens,
			u.CacheReadInputTokens,
		)
	}

	return scanner.Err()
}

// isModifiedAfter returns true if the file's modification time is at or after the given time.
func isModifiedAfter(path string, after time.Time) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.ModTime().Before(after)
}
