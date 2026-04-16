package codex

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/tokenusage"
)

type codexParser struct{}

type codexUsageFields struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
}

type codexJSONLEntry struct {
	Type    string `json:"type"`
	Message struct {
		Model string           `json:"model"`
		Usage codexUsageFields `json:"usage"`
	} `json:"message"`
	Model string            `json:"model"`
	Usage *codexUsageFields `json:"usage"`
}

func (p *codexParser) Parse(sandboxPath string, podStartedAt time.Time) (*tokenusage.TokenUsage, error) {
	usage := tokenusage.NewTokenUsage()

	sessionsDirs := codexSessionDirs(sandboxPath)
	for _, sessionsDir := range sessionsDirs {
		if _, err := os.Stat(sessionsDir); os.IsNotExist(err) {
			continue
		}
		parseCodexSessionsDir(sessionsDir, podStartedAt, usage)
	}

	if usage.IsEmpty() {
		return nil, nil
	}
	return usage, nil
}

func codexSessionDirs(sandboxPath string) []string {
	var dirs []string

	if sandboxPath != "" {
		dirs = append(dirs, filepath.Join(sandboxPath, "codex-home", "sessions"))
	}

	if home, err := os.UserHomeDir(); err == nil && home != "" {
		dirs = append(dirs, filepath.Join(home, ".codex", "sessions"))
	}

	return dirs
}

func parseCodexSessionsDir(sessionsDir string, podStartedAt time.Time, usage *tokenusage.TokenUsage) {
	err := filepath.WalkDir(sessionsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		if !tokenusage.IsModifiedAfter(path, podStartedAt) {
			return nil
		}
		if parseErr := parseCodexJSONLFile(path, usage); parseErr != nil {
			logger.Pod().Warn("Codex parser: file parse error", "file", path, "error", parseErr)
		}
		return nil
	})
	if err != nil {
		logger.Pod().Warn("Codex parser: walk error", "dir", sessionsDir, "error", err)
	}
}

func parseCodexJSONLFile(path string, usage *tokenusage.TokenUsage) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry codexJSONLEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		if entry.Message.Model != "" && (entry.Message.Usage.InputTokens > 0 || entry.Message.Usage.OutputTokens > 0) {
			usage.Add(
				entry.Message.Model,
				entry.Message.Usage.InputTokens,
				entry.Message.Usage.OutputTokens,
				entry.Message.Usage.CacheCreationInputTokens,
				entry.Message.Usage.CacheReadInputTokens,
			)
			continue
		}

		if entry.Model != "" && entry.Usage != nil && (entry.Usage.InputTokens > 0 || entry.Usage.OutputTokens > 0) {
			usage.Add(
				entry.Model,
				entry.Usage.InputTokens,
				entry.Usage.OutputTokens,
				entry.Usage.CacheCreationInputTokens,
				entry.Usage.CacheReadInputTokens,
			)
		}
	}

	return scanner.Err()
}
