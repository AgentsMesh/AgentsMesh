package terminal

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// CleanupOldStackDumps removes output-handler diagnostics older than maxAge.
func CleanupOldStackDumps(maxAge time.Duration) {
	cleanupOldStackDumps(config.TempBaseDir(), maxAge)
}

func cleanupOldStackDumps(dir string, maxAge time.Duration) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-maxAge)
	removed := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".stacks") {
			continue
		}
		info, infoErr := entry.Info()
		if infoErr == nil && info.ModTime().Before(cutoff) {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
			removed++
		}
	}
	if removed > 0 {
		logger.Terminal().Info("Cleaned up old stack dump files", "removed", removed)
	}
}
