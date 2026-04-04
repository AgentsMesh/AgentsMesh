package runner

import (
	"fmt"
	"os"
	"path/filepath"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// prepareAgentHome copies user's agent config directory to a per-pod isolated
// directory when CODEX_HOME is set in EnvVars. This enables per-pod MCP config
// isolation without modifying the user's original ~/.codex/ directory.
//
// After copying, it merges platform MCP servers from FilesToCreate into the
// existing config.toml using TOML-aware merging (only mcp_servers section).
func (b *PodBuilder) prepareAgentHome(sandboxRoot, workDir string) error {
	if b.cmd == nil || b.cmd.EnvVars == nil {
		return nil
	}

	codexHome, ok := b.cmd.EnvVars["CODEX_HOME"]
	if !ok || codexHome == "" {
		return nil
	}

	// Resolve template variables in CODEX_HOME path
	codexHome = b.resolvePath(codexHome, sandboxRoot, workDir)

	log := logger.Pod()
	log.Info("Preparing CODEX_HOME", "pod_key", b.cmd.PodKey, "codex_home", codexHome)

	// Copy user's ~/.codex/ to per-pod codex-home (if it exists)
	home := userHomeDir()
	if home != "" {
		userCodexDir := filepath.Join(home, ".codex")
		if dirExists(userCodexDir) {
			if err := copyDirSelective(userCodexDir, codexHome); err != nil {
				log.Warn("Failed to copy user codex dir, creating empty",
					"source", userCodexDir, "dest", codexHome, "error", err)
				// Clean up partial copy before creating empty directory
				_ = os.RemoveAll(codexHome)
				if mkErr := os.MkdirAll(codexHome, 0755); mkErr != nil {
					return fmt.Errorf("failed to create codex-home: %w", mkErr)
				}
			}
		} else {
			if err := os.MkdirAll(codexHome, 0755); err != nil {
				return fmt.Errorf("failed to create codex-home: %w", err)
			}
		}
	} else {
		if err := os.MkdirAll(codexHome, 0755); err != nil {
			return fmt.Errorf("failed to create codex-home: %w", err)
		}
	}

	// Find config.toml entry in FilesToCreate and merge it with existing config
	configTomlPath := filepath.Join(codexHome, "config.toml")
	mergeIdx := -1
	for i, f := range b.cmd.FilesToCreate {
		resolvedPath := b.resolvePath(f.Path, sandboxRoot, workDir)
		if resolvedPath == configTomlPath && !f.IsDirectory {
			mergeIdx = i
			break
		}
	}
	if mergeIdx >= 0 {
		f := b.cmd.FilesToCreate[mergeIdx]
		if err := mergeTomlMcpServers(configTomlPath, f.Content); err != nil {
			log.Warn("Failed to merge TOML MCP config, writing fresh",
				"path", configTomlPath, "error", err)
			// Fall through to let createFiles write it
		} else {
			// Remove from FilesToCreate to prevent createFiles from overwriting
			b.cmd.FilesToCreate = append(b.cmd.FilesToCreate[:mergeIdx], b.cmd.FilesToCreate[mergeIdx+1:]...)
			log.Info("Merged MCP config into existing config.toml", "path", configTomlPath)
		}
	}

	return nil
}

// mergeTomlMcpServers merges platform MCP server config into an existing config.toml.
// Only the mcp_servers section is merged; all other user settings are preserved.
func mergeTomlMcpServers(configPath, platformContent string) error {
	// Parse platform MCP content
	var platformConfig map[string]interface{}
	if err := toml.Unmarshal([]byte(platformContent), &platformConfig); err != nil {
		return fmt.Errorf("failed to parse platform TOML: %w", err)
	}

	platformServers, _ := platformConfig["mcp_servers"].(map[string]interface{})
	if len(platformServers) == 0 {
		return nil // Nothing to merge
	}

	// Read existing config (if any)
	var existingConfig map[string]interface{}
	existingData, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No existing config, write platform content directly
			return os.WriteFile(configPath, []byte(platformContent), 0644)
		}
		return fmt.Errorf("failed to read existing config: %w", err)
	}

	if err := toml.Unmarshal(existingData, &existingConfig); err != nil {
		return fmt.Errorf("failed to parse existing config: %w", err)
	}

	// Merge mcp_servers: platform entries override existing ones with same key
	existingServers, _ := existingConfig["mcp_servers"].(map[string]interface{})
	if existingServers == nil {
		existingServers = make(map[string]interface{})
	}
	for k, v := range platformServers {
		existingServers[k] = v
	}
	existingConfig["mcp_servers"] = existingServers

	// Write back
	merged, err := toml.Marshal(existingConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal merged config: %w", err)
	}

	return os.WriteFile(configPath, merged, 0644)
}
