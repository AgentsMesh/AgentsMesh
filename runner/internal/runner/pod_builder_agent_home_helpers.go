package runner

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// userHomeDir returns the user's home directory, falling back gracefully.
func userHomeDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return home
}

// dirExists checks if a directory exists.
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// copyDirSelective copies a directory recursively, skipping large/transient
// subdirectories (sessions, cache) that are not needed per-pod.
// Symlinks are preserved as symlinks rather than dereferenced.
// Special files (sockets, pipes, devices) are silently skipped.
// Individual file errors are logged and skipped rather than aborting the entire copy.
func copyDirSelective(src, dst string) error {
	log := logger.Pod()

	skipDirs := map[string]bool{
		"sessions": true, // Session logs can be large
		"cache":    true, // Cache is transient
	}

	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			// Permission denied on a subdirectory — skip it, don't abort
			log.Debug("Skipping inaccessible path during copy", "path", path, "error", err)
			return nil
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dst, relPath)

		// Handle symlinks before directory checks, since WalkDir does not
		// follow symlinks and d.IsDir() returns false for symlink-to-dir.
		if d.Type()&fs.ModeSymlink != 0 {
			if symlinkErr := copySymlink(path, destPath); symlinkErr != nil {
				log.Debug("Skipping uncopiable symlink", "path", path, "error", symlinkErr)
			}
			return nil
		}

		// Skip transient directories
		if d.IsDir() {
			parts := strings.Split(relPath, string(filepath.Separator))
			if len(parts) > 0 && skipDirs[parts[0]] {
				return filepath.SkipDir
			}
			if mkErr := os.MkdirAll(destPath, 0755); mkErr != nil {
				log.Debug("Skipping uncreatable directory", "path", destPath, "error", mkErr)
			}
			return nil
		}

		// Skip special files: sockets, pipes, devices.
		// Only copy regular files (d.Type() == 0 means regular file).
		if !d.Type().IsRegular() {
			return nil
		}

		// Skip files larger than 10 MiB to avoid OOM on large binaries/databases
		info, err := d.Info()
		if err != nil {
			return nil
		}
		const maxFileSize = 10 << 20 // 10 MiB
		if info.Size() > maxFileSize {
			log.Debug("Skipping oversized file during copy", "path", path, "size", info.Size())
			return nil
		}

		// Copy regular file — skip on error to preserve partial results
		data, err := os.ReadFile(path)
		if err != nil {
			log.Debug("Skipping unreadable file during copy", "path", path, "error", err)
			return nil
		}

		if writeErr := os.WriteFile(destPath, data, info.Mode()); writeErr != nil {
			log.Debug("Skipping unwritable file during copy", "dest", destPath, "error", writeErr)
		}
		return nil
	})
}

// copySymlink recreates a symlink at dst pointing to the same target as src.
// Dangling symlinks are silently skipped.
func copySymlink(src, dst string) error {
	target, err := os.Readlink(src)
	if err != nil {
		return err
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	return os.Symlink(target, dst)
}
