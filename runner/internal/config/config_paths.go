package config

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// TempBaseDir returns the base temporary directory for agentsmesh.
// On Unix (macOS/Linux): /tmp/agentsmesh — a predictable, easy-to-find path.
// On Windows: os.TempDir()/agentsmesh — since /tmp doesn't exist.
//
// macOS's os.TempDir() returns /var/folders/xx/.../T/ which is hard to locate,
// so we use /tmp directly for better developer experience.
func TempBaseDir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.TempDir(), "agentsmesh")
	}
	return "/tmp/agentsmesh"
}

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
		return TempBaseDir()
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

// DefaultLogFileName returns the base log file name (e.g., "runner.log").
// Used by both the logger (for rotation naming) and the log collector (for pattern matching).
func DefaultLogFileName() string {
	return "runner.log"
}

// GetLogPath returns the log file path.
// Always uses TempBaseDir for predictable, easy-to-find log location.
func (c *Config) GetLogPath() string {
	return filepath.Join(TempBaseDir(), DefaultLogFileName())
}

// GetLogConfig returns the logger configuration.
func (c *Config) GetLogConfig() logger.Config {
	return logger.Config{
		Level:       c.LogLevel,
		FilePath:    c.GetLogPath(),
		Format:      "text",            // Default to human-readable format
		MaxFileSize: 10 * 1024 * 1024,  // 10MB per file
		MaxBackups:  3,                 // Keep 3 backup files per day
		MaxDirSize:  500 * 1024 * 1024, // 500MB total directory size
	}
}

// GetLogPTYDir returns the PTY log directory path.
// Always uses TempBaseDir for predictable, easy-to-find log location.
func (c *Config) GetLogPTYDir() string {
	return filepath.Join(TempBaseDir(), "pty-logs")
}
