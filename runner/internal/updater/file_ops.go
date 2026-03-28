package updater

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
)

// atomicReplace atomically replaces the target file with the source file.
func atomicReplace(src, dst string) error {
	if runtime.GOOS == "windows" {
		return atomicReplaceWindows(src, dst)
	}

	// On Unix, rename is atomic.
	return os.Rename(src, dst)
}

// atomicReplaceWindows handles file replacement on Windows.
// Windows allows renaming a running executable but not deleting or overwriting it.
// Strategy: rename running exe → .old, rename new → original path, clean .old on restart.
func atomicReplaceWindows(src, dst string) error {
	bakPath := dst + ".old"

	// Try to remove a leftover .old from a previous update.
	// If it's still locked (process running), ignore the error.
	os.Remove(bakPath)

	// Rename the running executable to .old (Windows allows this).
	if err := os.Rename(dst, bakPath); err != nil {
		// If .old is locked from a previous run, try a timestamped name.
		bakPath = fmt.Sprintf("%s.old.%d", dst, os.Getpid())
		if err2 := os.Rename(dst, bakPath); err2 != nil {
			slog.Error("Failed to backup original file during Windows replace", "dst", dst, "error", err2)
			return fmt.Errorf("failed to backup original file: %w", err2)
		}
		slog.Warn("Used timestamped backup path due to locked .old file", "backup", bakPath)
	}

	// Move the new binary to the original path (now free).
	if err := os.Rename(src, dst); err != nil {
		// Rollback: restore the backup.
		if rbErr := os.Rename(bakPath, dst); rbErr != nil {
			slog.Error("Replace failed and rollback also failed", "replace_error", err, "rollback_error", rbErr)
			return fmt.Errorf("failed to replace file (%v) and rollback also failed (%v)", err, rbErr)
		}
		slog.Error("Replace failed but rollback succeeded", "error", err)
		return fmt.Errorf("failed to replace file (rollback succeeded): %w", err)
	}

	// Don't remove .old immediately — the old binary is still running.
	// CleanupOldBinaries() handles this on next startup.
	return nil
}

// CleanupOldBinaries removes leftover .old files from previous updates.
// Call this early during process startup, before any update checks.
func CleanupOldBinaries() {
	execPath, err := os.Executable()
	if err != nil {
		return
	}

	// Remove exact .old file
	os.Remove(execPath + ".old")

	// Remove any timestamped backups (e.g., runner.exe.old.12345)
	matches, _ := filepath.Glob(execPath + ".old.*")
	for _, m := range matches {
		os.Remove(m) // best-effort
	}
}

// copyFile copies a file from src to dst.
func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	dstFile, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}
