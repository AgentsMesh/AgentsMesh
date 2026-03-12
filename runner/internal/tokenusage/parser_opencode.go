package tokenusage

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// OpenCodeParser parses OpenCode JSON message files.
// OpenCode writes message files to:
//   - {HOME}/.local/share/opencode/storage/message/*/msg_*.json
//
// Only files modified after podStartedAt are processed.
type OpenCodeParser struct{}

// openCodeMessage represents an OpenCode message JSON file.
type openCodeMessage struct {
	Model string `json:"model"`
	Usage struct {
		InputTokens              int64 `json:"input_tokens"`
		OutputTokens             int64 `json:"output_tokens"`
		CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
	} `json:"usage"`
	// Alternative field names
	TokenUsage *struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
		CachedTokens     int64 `json:"cached_tokens"`
	} `json:"token_usage"`
}

func (p *OpenCodeParser) Parse(sandboxPath string, podStartedAt time.Time) (*TokenUsage, error) {
	usage := NewTokenUsage()

	home, err := os.UserHomeDir()
	if err != nil {
		logger.Pod().Warn("OpenCode parser: cannot get home dir", "error", err)
		return nil, nil
	}

	pattern := filepath.Join(home, ".local", "share", "opencode", "storage", "message", "*", "msg_*.json")
	files, err := filepath.Glob(pattern)
	if err != nil {
		logger.Pod().Warn("OpenCode parser: glob error", "pattern", pattern, "error", err)
		return nil, nil
	}

	for _, f := range files {
		if !isModifiedAfter(f, podStartedAt) {
			continue
		}
		if err := parseOpenCodeFile(f, usage); err != nil {
			logger.Pod().Warn("OpenCode parser: file parse error", "file", f, "error", err)
		}
	}

	if usage.IsEmpty() {
		return nil, nil
	}
	return usage, nil
}

// maxOpenCodeFileSize caps the size of individual OpenCode JSON message files
// to prevent OOM when encountering unexpectedly large files.
const maxOpenCodeFileSize = 10 * 1024 * 1024 // 10 MB

func parseOpenCodeFile(path string, usage *TokenUsage) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if info.Size() > maxOpenCodeFileSize {
		logger.Pod().Warn("OpenCode parser: skipping oversized file", "file", path, "size", info.Size())
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var msg openCodeMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil // Skip malformed files
	}

	model := msg.Model
	if model == "" {
		model = "opencode-unknown"
	}

	// Try primary usage structure
	if msg.Usage.InputTokens > 0 || msg.Usage.OutputTokens > 0 {
		usage.Add(
			model,
			msg.Usage.InputTokens,
			msg.Usage.OutputTokens,
			msg.Usage.CacheCreationInputTokens,
			msg.Usage.CacheReadInputTokens,
		)
		return nil
	}

	// Try alternative token_usage structure
	if msg.TokenUsage != nil && (msg.TokenUsage.PromptTokens > 0 || msg.TokenUsage.CompletionTokens > 0) {
		usage.Add(
			model,
			msg.TokenUsage.PromptTokens,
			msg.TokenUsage.CompletionTokens,
			0,
			msg.TokenUsage.CachedTokens,
		)
	}

	return nil
}
