package tokenusage

import (
	"bufio"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// CodexParser parses Codex CLI JSONL session files.
// Codex CLI writes session data to JSONL files under:
//   - {CODEX_HOME}/sessions/YYYY/MM/DD/rollout-*.jsonl
//
// The parser checks multiple locations in priority order:
//  1. {sandboxPath}/codex-home/sessions/ (per-pod CODEX_HOME set by platform)
//  2. {HOME}/.codex/sessions/ (default user-level location)
//
// Only files modified after podStartedAt are processed.
type CodexParser struct{}

// codexUsageFields holds token count fields shared by nested and flat structures.
type codexUsageFields struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
}

// codexJSONLEntry represents a Codex CLI JSONL entry with usage info.
// Codex emits two formats: nested (message.model + message.usage) and flat (model + usage).
type codexJSONLEntry struct {
	Type    string `json:"type"`
	Message struct {
		Model string           `json:"model"`
		Usage codexUsageFields `json:"usage"`
	} `json:"message"`
	// Flat structure (alternative format)
	Model string            `json:"model"`
	Usage *codexUsageFields `json:"usage"`
}

func (p *CodexParser) Parse(sandboxPath string, podStartedAt time.Time) (*TokenUsage, error) {
	usage := NewTokenUsage()

	// Check multiple session directories in priority order
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

// codexSessionDirs returns candidate session directories in priority order.
// Per-pod CODEX_HOME (sandbox) is checked first, then user-level ~/.codex/.
func codexSessionDirs(sandboxPath string) []string {
	var dirs []string

	// 1. Per-pod CODEX_HOME inside sandbox (set by platform via CODEX_HOME env var)
	if sandboxPath != "" {
		dirs = append(dirs, filepath.Join(sandboxPath, "codex-home", "sessions"))
	}

	// 2. Default user-level location
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		dirs = append(dirs, filepath.Join(home, ".codex", "sessions"))
	}

	return dirs
}

// parseCodexSessionsDir walks a sessions directory for JSONL files.
func parseCodexSessionsDir(sessionsDir string, podStartedAt time.Time, usage *TokenUsage) {
	err := filepath.WalkDir(sessionsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible entries
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".jsonl") {
			return nil
		}
		if !isModifiedAfter(path, podStartedAt) {
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

func parseCodexJSONLFile(path string, usage *TokenUsage) error {
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

		// Try nested message.usage structure first (like Claude)
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

		// Try flat structure
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
