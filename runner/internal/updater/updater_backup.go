package updater

import (
	"fmt"
	"log/slog"
	"os"
)

// Rollback restores the previous version if a backup exists.
func (u *Updater) Rollback() error {
	execPath, err := u.execPathFunc()
	if err != nil {
		slog.Error("Failed to get executable path for rollback", "error", err)
		return fmt.Errorf("failed to get executable path: %w", err)
	}

	backupPath := execPath + ".bak"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		slog.Warn("No backup found for rollback", "path", backupPath)
		return fmt.Errorf("no backup found at %s", backupPath)
	}

	if err := atomicReplace(backupPath, execPath); err != nil {
		slog.Error("Failed to restore backup", "backup", backupPath, "target", execPath, "error", err)
		return fmt.Errorf("failed to restore backup: %w", err)
	}

	slog.Info("Rollback completed successfully", "backup", backupPath)
	return nil
}

// CreateBackup creates a backup of the current executable.
func (u *Updater) CreateBackup() (string, error) {
	execPath, err := u.execPathFunc()
	if err != nil {
		slog.Error("Failed to get executable path for backup", "error", err)
		return "", fmt.Errorf("failed to get executable path: %w", err)
	}

	backupPath := execPath + ".bak"

	if err := copyFile(execPath, backupPath); err != nil {
		slog.Error("Failed to create backup", "source", execPath, "backup", backupPath, "error", err)
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	slog.Info("Backup created", "path", backupPath)
	return backupPath, nil
}
