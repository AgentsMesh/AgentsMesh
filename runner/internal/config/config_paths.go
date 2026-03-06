package config

import (
	"os"
	"path/filepath"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// GetWorkspace returns the workspace directory path.
// Falls back to WorkspaceRoot if Workspace is not set.
func (c *Config) GetWorkspace() string {
	if c.Workspace != "" {
		return c.Workspace
	}
	if c.WorkspaceRoot != "" {
		return c.WorkspaceRoot
	}
	// Default to user's home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "agentsmesh")
	}
	return filepath.Join(home, ".agentsmesh")
}

// GetSandboxesDir returns the sandboxes directory path.
func (c *Config) GetSandboxesDir() string {
	return filepath.Join(c.GetWorkspace(), "sandboxes")
}

// GetReposDir returns the repository cache directory path.
func (c *Config) GetReposDir() string {
	return filepath.Join(c.GetWorkspace(), "repos")
}

// GetMCPPort returns the MCP HTTP Server port.
func (c *Config) GetMCPPort() int {
	if c.MCPPort > 0 {
		return c.MCPPort
	}
	return 19000 // Default port
}

// GetPluginsDir returns the user plugins directory path.
// Returns empty string if no plugins directory is configured.
func (c *Config) GetPluginsDir() string {
	if c.PluginsDir != "" {
		return os.ExpandEnv(c.PluginsDir)
	}
	// Default to ~/.agentsmesh/plugins
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".agentsmesh", "plugins")
}

// GetLogPath returns the log file path.
func (c *Config) GetLogPath() string {
	if c.LogFile != "" {
		return os.ExpandEnv(c.LogFile)
	}
	// Default to system temp directory (can be safely deleted)
	return filepath.Join(os.TempDir(), "agentsmesh", "runner.log")
}

// GetLogConfig returns the logger configuration.
func (c *Config) GetLogConfig() logger.Config {
	return logger.Config{
		Level:       c.LogLevel,
		FilePath:    c.GetLogPath(),
		Format:      "text",              // Default to human-readable format
		MaxFileSize: 10 * 1024 * 1024,    // 10MB per file
		MaxBackups:  3,                   // Keep 3 backup files per day
		MaxDirSize:  500 * 1024 * 1024,   // 500MB total directory size
	}
}

// GetLogPTYDir returns the PTY log directory path.
// Falls back to $TMPDIR/agentsmesh/pty-logs if not set.
func (c *Config) GetLogPTYDir() string {
	if c.LogPTYDir != "" {
		return os.ExpandEnv(c.LogPTYDir)
	}
	return filepath.Join(os.TempDir(), "agentsmesh", "pty-logs")
}

