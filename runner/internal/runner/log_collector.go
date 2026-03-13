package runner

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/agentsmesh/runner/internal/config"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// maxArchiveBytes is the total file size limit for log collection (100 MB).
const maxArchiveBytes int64 = 100 * 1024 * 1024

// CollectLogs collects runner log files into a tar.gz archive.
// Returns the path to the temporary archive, its size in bytes, and any error.
// The caller is responsible for removing the temporary file.
func CollectLogs(ctx context.Context) (tarPath string, sizeBytes int64, err error) {
	log := logger.Runner()
	logDir := config.TempBaseDir()

	// Create temporary archive file
	tmpFile, err := os.CreateTemp(os.TempDir(), "agentsmesh-logs-*.tar.gz")
	if err != nil {
		return "", 0, fmt.Errorf("failed to create temp file: %w", err)
	}
	tarPath = tmpFile.Name()

	// Ensure cleanup on error
	defer func() {
		if err != nil {
			tmpFile.Close()
			os.Remove(tarPath)
			tarPath = ""
			sizeBytes = 0
		}
	}()

	gzWriter := gzip.NewWriter(tmpFile)
	tarWriter := tar.NewWriter(gzWriter)

	// Track total size to enforce limit
	var totalBytes int64

	// addFile is a helper that adds a file and enforces the size budget.
	addFile := func(fullPath, archiveName string) {
		if totalBytes >= maxArchiveBytes {
			return
		}
		written, addErr := addFileToTar(tarWriter, fullPath, archiveName, maxArchiveBytes-totalBytes)
		if addErr != nil {
			log.Warn("Failed to add file to archive", "file", archiveName, "error", addErr)
			return
		}
		totalBytes += written
	}

	// Collect runner log files (runner-YYYY-MM-DD.log*) and diagnostic files from logDir
	entries, dirErr := os.ReadDir(logDir)
	if dirErr != nil {
		log.Warn("Failed to read log directory", "dir", logDir, "error", dirErr)
	} else {
		for _, entry := range entries {
			if ctx.Err() != nil {
				return "", 0, ctx.Err()
			}
			name := entry.Name()
			if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
				continue
			}
			if logger.IsOwnLogFile(name) ||
				(strings.HasPrefix(name, "blocked-") && strings.HasSuffix(name, ".stacks")) ||
				(strings.HasPrefix(name, "diag-") && strings.HasSuffix(name, ".txt")) {
				addFile(filepath.Join(logDir, name), name)
			}
		}
	}

	// Collect pty-logs/ directory (skip symlinks)
	ptyLogDir := filepath.Join(logDir, "pty-logs")
	if info, statErr := os.Stat(ptyLogDir); statErr == nil && info.IsDir() {
		if walkErr := filepath.WalkDir(ptyLogDir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil // Skip files with errors
			}
			if ctx.Err() != nil {
				return ctx.Err()
			}
			// Skip directories and symlinks
			if d.IsDir() || d.Type()&os.ModeSymlink != 0 {
				return nil
			}
			// Stop collecting if size budget exceeded
			if totalBytes >= maxArchiveBytes {
				return filepath.SkipAll
			}
			relPath, relErr := filepath.Rel(logDir, path)
			if relErr != nil {
				return nil
			}
			archiveName := filepath.ToSlash(relPath)
			addFile(path, archiveName)
			return nil
		}); walkErr != nil {
			log.Warn("Failed to walk pty-logs directory", "error", walkErr)
		}
	}

	// Close tar and gzip writers
	if err = tarWriter.Close(); err != nil {
		return "", 0, fmt.Errorf("failed to close tar writer: %w", err)
	}
	if err = gzWriter.Close(); err != nil {
		return "", 0, fmt.Errorf("failed to close gzip writer: %w", err)
	}
	if err = tmpFile.Close(); err != nil {
		return "", 0, fmt.Errorf("failed to close temp file: %w", err)
	}

	// Get archive size
	fileInfo, err := os.Stat(tarPath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to stat archive: %w", err)
	}
	sizeBytes = fileInfo.Size()

	log.Info("Log archive created", "path", tarPath, "size_bytes", sizeBytes)
	return tarPath, sizeBytes, nil
}

// addFileToTar adds a single file to a tar archive, limiting read to maxBytes.
// Returns the number of bytes actually written to the tar.
func addFileToTar(tw *tar.Writer, filePath, archiveName string, maxBytes int64) (int64, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return 0, err
	}
	// Skip non-regular files (symlinks, devices, etc.)
	if !info.Mode().IsRegular() {
		return 0, nil
	}

	fileSize := info.Size()
	if fileSize > maxBytes {
		fileSize = maxBytes
	}

	header := &tar.Header{
		Name:    archiveName,
		Size:    fileSize,
		Mode:    int64(info.Mode()),
		ModTime: info.ModTime(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return 0, err
	}

	// Use LimitReader to read exactly the declared size, avoiding
	// both over-read (file grew) and header/data mismatch (file shrank).
	written, err := io.Copy(tw, io.LimitReader(f, fileSize))
	if err != nil {
		return written, err
	}

	// If file was truncated between Stat and Copy, pad with zeros
	// to match the declared tar header size.
	for written < fileSize {
		padding := make([]byte, min(fileSize-written, 4096))
		n, padErr := tw.Write(padding)
		written += int64(n)
		if padErr != nil {
			return written, padErr
		}
	}

	return written, nil
}
