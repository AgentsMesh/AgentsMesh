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

// codexUsageFields supports both Anthropic-style (input_tokens / output_tokens)
// and OpenAI-style (prompt_tokens / completion_tokens) field names because
// Codex CLI emits the OpenAI naming via the Responses API.
type codexUsageFields struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
	PromptTokens             int64 `json:"prompt_tokens"`
	CompletionTokens         int64 `json:"completion_tokens"`
}

func (u *codexUsageFields) effectiveInput() int64 {
	if u.InputTokens > 0 {
		return u.InputTokens
	}
	return u.PromptTokens
}

func (u *codexUsageFields) effectiveOutput() int64 {
	if u.OutputTokens > 0 {
		return u.OutputTokens
	}
	return u.CompletionTokens
}

// codexJSONLEntry covers three observed Codex shapes. Branch precedence in
// parseCodexJSONLFile is fixed to: message → response → flat. If a single
// JSONL line populates more than one shape with positive token counts (not
// expected from current Codex CLI), only the first match contributes — we
// never sum across shapes to avoid double-counting.
//
//   - nested message (Anthropic-style): message.model + message.usage
//   - nested response (OpenAI-style):   response.model + response.usage
//   - flat:                              top-level model + usage (either naming)
type codexJSONLEntry struct {
	Type    string `json:"type"`
	Message struct {
		Model string           `json:"model"`
		Usage codexUsageFields `json:"usage"`
	} `json:"message"`
	Response struct {
		Model string           `json:"model"`
		Usage codexUsageFields `json:"usage"`
	} `json:"response"`
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

		if entry.Message.Usage.effectiveInput() > 0 || entry.Message.Usage.effectiveOutput() > 0 {
			addCodexUsage(usage, entry.Message.Model, &entry.Message.Usage, "message")
			continue
		}

		if entry.Response.Usage.effectiveInput() > 0 || entry.Response.Usage.effectiveOutput() > 0 {
			addCodexUsage(usage, entry.Response.Model, &entry.Response.Usage, "response")
			continue
		}

		if entry.Usage != nil && (entry.Usage.effectiveInput() > 0 || entry.Usage.effectiveOutput() > 0) {
			addCodexUsage(usage, entry.Model, entry.Usage, "flat")
		}
	}

	return scanner.Err()
}

// addCodexUsage attributes positive token counts to the named model, falling
// back to "codex-unknown" when the agent emitted usage without identifying
// the model — silent drop here was the original #146 failure mode and we
// must not reintroduce it for malformed entries.
func addCodexUsage(usage *tokenusage.TokenUsage, model string, u *codexUsageFields, branch string) {
	if model == "" {
		logger.Pod().Debug("Codex parser: usage with empty model; attributing to codex-unknown", "branch", branch)
		model = "codex-unknown"
	}
	usage.Add(
		model,
		u.effectiveInput(),
		u.effectiveOutput(),
		u.CacheCreationInputTokens,
		u.CacheReadInputTokens,
	)
}
