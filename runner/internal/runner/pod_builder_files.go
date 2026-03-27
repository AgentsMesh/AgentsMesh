package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// createFiles creates files from the FilesToCreate list.
func (b *PodBuilder) createFiles(sandboxRoot, workDir string) error {
	absSandbox, err := filepath.Abs(sandboxRoot)
	if err != nil {
		return &client.PodError{
			Code:    client.ErrCodeFileCreate,
			Message: fmt.Sprintf("failed to resolve sandbox root: %v", err),
		}
	}
	absSandbox = filepath.Clean(absSandbox)

	for _, f := range b.cmd.FilesToCreate {
		path := b.resolvePath(f.Path, sandboxRoot, workDir)

		absPath, err := filepath.Abs(path)
		if err != nil {
			return &client.PodError{
				Code:    client.ErrCodeFileCreate,
				Message: fmt.Sprintf("failed to resolve file path: %v", err),
				Details: map[string]string{"path": f.Path},
			}
		}
		if absPath != absSandbox && !strings.HasPrefix(absPath, absSandbox+string(os.PathSeparator)) {
			return &client.PodError{
				Code:    client.ErrCodeFileCreate,
				Message: fmt.Sprintf("path %q escapes sandbox root %q (resolved: %q)", f.Path, absSandbox, absPath),
				Details: map[string]string{"path": f.Path, "sandbox_root": absSandbox, "resolved_path": absPath},
			}
		}

		if f.IsDirectory {
			if err := os.MkdirAll(path, 0755); err != nil {
				return &client.PodError{
					Code:    client.ErrCodeFileCreate,
					Message: fmt.Sprintf("failed to create directory: %v", err),
					Details: map[string]string{"path": path},
				}
			}
			continue
		}

		parentDir := filepath.Dir(path)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return &client.PodError{
				Code:    client.ErrCodeFileCreate,
				Message: fmt.Sprintf("failed to create parent directory: %v", err),
				Details: map[string]string{"path": parentDir},
			}
		}

		mode := os.FileMode(0644)
		if f.Mode != 0 {
			mode = os.FileMode(f.Mode)
		}

		if err := os.WriteFile(path, []byte(f.Content), mode); err != nil {
			return &client.PodError{
				Code:    client.ErrCodeFileCreate,
				Message: fmt.Sprintf("failed to write file: %v", err),
				Details: map[string]string{"path": path},
			}
		}

		logger.Pod().Debug("Created file", "path", path, "mode", fmt.Sprintf("%o", mode))
	}

	return nil
}

// createFilesFromProto creates files from a proto FileToCreate list.
// Used by PodFile mode where paths are already resolved.
func (b *PodBuilder) createFilesFromProto(files []*runnerv1.FileToCreate, sandboxRoot, workDir string) error {
	if len(files) == 0 {
		return nil
	}

	absSandbox, err := filepath.Abs(sandboxRoot)
	if err != nil {
		return &client.PodError{
			Code:    client.ErrCodeFileCreate,
			Message: fmt.Sprintf("failed to resolve sandbox root: %v", err),
		}
	}
	absSandbox = filepath.Clean(absSandbox)

	for _, f := range files {
		path := f.Path

		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		if absPath != absSandbox && !strings.HasPrefix(absPath, absSandbox+string(os.PathSeparator)) {
			logger.Pod().Warn("PodFile file path escapes sandbox, skipping", "path", path)
			continue
		}

		if f.IsDirectory {
			os.MkdirAll(path, 0755)
			continue
		}

		os.MkdirAll(filepath.Dir(path), 0755)
		mode := os.FileMode(0644)
		if f.Mode != 0 {
			mode = os.FileMode(f.Mode)
		}
		if err := os.WriteFile(path, []byte(f.Content), mode); err != nil {
			logger.Pod().Warn("Failed to create file (podfile)", "path", path, "error", err)
			continue
		}
		logger.Pod().Debug("Created file (podfile)", "path", path)
	}

	return nil
}
