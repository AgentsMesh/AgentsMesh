package opencode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
)

type opencodeParser struct{}

type opencodeMessage struct {
	Model string `json:"model"`
	Usage struct {
		InputTokens              int64 `json:"input_tokens"`
		OutputTokens             int64 `json:"output_tokens"`
		CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
	} `json:"usage"`
	TokenUsage *struct {
		PromptTokens     int64 `json:"prompt_tokens"`
		CompletionTokens int64 `json:"completion_tokens"`
		CachedTokens     int64 `json:"cached_tokens"`
	} `json:"token_usage"`
}

func (p *opencodeParser) Parse(sandboxPath string, podStartedAt time.Time) (*tokenusage.TokenUsage, error) {
	usage := tokenusage.NewTokenUsage()

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
		if !tokenusage.IsModifiedAfter(f, podStartedAt) {
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

const maxOpenCodeFileSize = 10 * 1024 * 1024

func parseOpenCodeFile(path string, usage *tokenusage.TokenUsage) error {
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

	var msg opencodeMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil
	}

	model := msg.Model
	if model == "" {
		model = "opencode-unknown"
	}

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
