package claude

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

type claudeParser struct{}

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

func (p *claudeParser) Parse(sandboxPath string, podStartedAt time.Time) (*tokenusage.TokenUsage, error) {
	log := logger.Pod()
	usage := tokenusage.NewTokenUsage()

	home, err := os.UserHomeDir()
	if err != nil {
		log.Warn("Claude parser: cannot determine HOME", "error", err)
		return nil, nil
	}

	candidates := []string{sandboxPath, filepath.Join(sandboxPath, "workspace")}

	for _, candidate := range candidates {
		resolved, err := filepath.EvalSymlinks(candidate)
		if err != nil {
			continue
		}

		hash := claudePathHash(resolved)
		projectDir := filepath.Join(home, ".claude", "projects", hash)

		if _, err := os.Stat(projectDir); err != nil {
			continue
		}

		if walkErr := filepath.WalkDir(projectDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".jsonl") {
				return nil
			}
			if !tokenusage.IsModifiedAfter(path, podStartedAt) {
				return nil
			}
			if parseErr := parseClaudeJSONLFile(path, usage); parseErr != nil {
				log.Warn("Claude parser: file parse error", "file", path, "error", parseErr)
			}
			return nil
		}); walkErr != nil {
			log.Warn("Claude parser: walk error", "dir", projectDir, "error", walkErr)
		}
	}

	if usage.IsEmpty() {
		return nil, nil
	}
	return usage, nil
}

// claudePathHash reproduces the project directory naming convention used by
// Claude Code: the resolved absolute path with OS path separators replaced by "-".
//
// This is intentionally NOT using filepath helpers — it must match the external
// convention, not the local OS path semantics.
func claudePathHash(resolvedPath string) string {
	var b strings.Builder
	b.Grow(len(resolvedPath))
	for _, c := range resolvedPath {
		switch c {
		case '/', '\\':
			b.WriteByte('-')
		case ':':
			// skip (Windows drive prefix)
		default:
			b.WriteRune(c)
		}
	}
	return b.String()
}

func parseClaudeJSONLFile(path string, usage *tokenusage.TokenUsage) error {
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

		var entry claudeJSONLEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
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
