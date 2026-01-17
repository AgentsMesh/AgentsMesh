package updater

import (
	"fmt"
	"io"
	"os"
	"runtime"
)

// atomicReplace atomically replaces the target file with the source file.
func atomicReplace(src, dst string) error {
	// On Unix, we can use rename for atomic replacement
	// On Windows, we need a different approach
	if runtime.GOOS == "windows" {
		return atomicReplaceWindows(src, dst)
	}

	// On Unix, we can just rename
	return os.Rename(src, dst)
}

// atomicReplaceWindows handles atomic file replacement on Windows.
func atomicReplaceWindows(src, dst string) error {
	// On Windows, rename the old file first
	bakPath := dst + ".old"
	os.Remove(bakPath) // Remove any existing backup
	if err := os.Rename(dst, bakPath); err != nil {
		return fmt.Errorf("failed to backup original file: %w", err)
	}
	if err := os.Rename(src, dst); err != nil {
		// Try to restore the backup
		if rbErr := os.Rename(bakPath, dst); rbErr != nil {
			return fmt.Errorf("failed to replace file (%v) and rollback also failed (%v)", err, rbErr)
		}
		return fmt.Errorf("failed to replace file (rollback succeeded): %w", err)
	}
	// Clean up old file (ignore error as it's not critical)
	os.Remove(bakPath)
	return nil
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
