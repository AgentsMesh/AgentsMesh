package runner

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/anthropics/agentsmesh/runner/internal/agentkit"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// prepareAgentHome copies the user's agent config directory to a per-pod
// isolated directory when a registered AgentHomeSpec matches an env var.
// After copying, it merges platform config using the spec's MergeConfig.
func (b *PodBuilder) prepareAgentHome(sandboxRoot, workDir string) error {
	if b.cmd == nil || b.cmd.EnvVars == nil {
		return nil
	}

	spec, agentHome := agentkit.MatchAgentHome(b.cmd.EnvVars)
	if spec == nil {
		return nil
	}

	agentHome = b.resolvePath(agentHome, sandboxRoot, workDir)

	log := logger.Pod()
	log.Info("Preparing agent home", "pod_key", b.cmd.PodKey, "env_var", spec.EnvVar, "path", agentHome)

	home := userHomeDir()
	if home != "" {
		userDir := filepath.Join(home, spec.UserDirName)
		if dirExists(userDir) {
			if err := copyDirSelective(userDir, agentHome); err != nil {
				log.Warn("Failed to copy user agent dir, creating empty",
					"source", userDir, "dest", agentHome, "error", err)
				_ = os.RemoveAll(agentHome)
				if mkErr := os.MkdirAll(agentHome, 0755); mkErr != nil {
					return fmt.Errorf("failed to create agent home: %w", mkErr)
				}
			}
		} else {
			if err := os.MkdirAll(agentHome, 0755); err != nil {
				return fmt.Errorf("failed to create agent home: %w", err)
			}
		}
	} else {
		if err := os.MkdirAll(agentHome, 0755); err != nil {
			return fmt.Errorf("failed to create agent home: %w", err)
		}
	}

	if spec.MergeConfig == nil {
		return nil
	}

	// Find matching config file in FilesToCreate and merge
	mergeIdx := -1
	for i, f := range b.cmd.FilesToCreate {
		resolvedPath := b.resolvePath(f.Path, sandboxRoot, workDir)
		parentDir := filepath.Dir(resolvedPath)
		if parentDir == agentHome && !f.IsDirectory {
			mergeIdx = i
			break
		}
	}
	if mergeIdx >= 0 {
		f := b.cmd.FilesToCreate[mergeIdx]
		configPath := b.resolvePath(f.Path, sandboxRoot, workDir)
		if err := spec.MergeConfig(configPath, f.Content); err != nil {
			log.Warn("Failed to merge agent config, writing fresh",
				"path", configPath, "error", err)
		} else {
			b.cmd.FilesToCreate = append(b.cmd.FilesToCreate[:mergeIdx], b.cmd.FilesToCreate[mergeIdx+1:]...)
			log.Info("Merged platform config into existing agent config", "path", configPath)
		}
	}

	return nil
}
