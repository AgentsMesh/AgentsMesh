package cache

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// extractTarGz extracts a tar.gz archive to the specified directory.
func extractTarGz(r io.Reader, targetDir string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	// Limit total entries to prevent inode exhaustion (tar bomb)
	const maxEntries = 10000

	tr := tar.NewReader(gz)
	entryCount := 0
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		entryCount++
		if entryCount > maxEntries {
			return fmt.Errorf("tar archive exceeds maximum entry count (%d)", maxEntries)
		}

		if err := extractTarEntry(tr, header, targetDir); err != nil {
			return err
		}
	}

	return nil
}

// extractTarEntry extracts a single tar entry to the target directory.
func extractTarEntry(tr *tar.Reader, header *tar.Header, targetDir string) error {
	// Sanitize path to prevent directory traversal
	name := header.Name
	if strings.Contains(name, "..") {
		return nil
	}
	targetPath := filepath.Join(targetDir, filepath.Clean(name))

	// Ensure target path is within target directory
	if !strings.HasPrefix(targetPath, filepath.Clean(targetDir)+string(os.PathSeparator)) &&
		targetPath != filepath.Clean(targetDir) {
		return nil
	}

	switch header.Typeflag {
	case tar.TypeDir:
		return extractTarDir(targetPath, header)
	case tar.TypeReg:
		return extractTarFile(tr, targetPath, header)
	default:
		// Skip symlinks, hardlinks, and other special types
		return nil
	}
}

// extractTarDir creates a directory from a tar header.
func extractTarDir(targetPath string, header *tar.Header) error {
	// Restrict directory permissions to prevent world-writable dirs
	dirMode := os.FileMode(header.Mode) & 0755
	if dirMode == 0 {
		dirMode = 0755
	}
	if err := os.MkdirAll(targetPath, dirMode); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
	}
	return nil
}

// extractTarFile extracts a regular file from a tar archive.
func extractTarFile(tr *tar.Reader, targetPath string, header *tar.Header) error {
	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent dir for %s: %w", targetPath, err)
	}

	// Restrict file permissions: strip execute bits, cap at 0644
	mode := os.FileMode(header.Mode) & 0644
	if mode == 0 {
		mode = 0644
	}

	outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", targetPath, err)
	}

	// Limit extraction size to prevent zip bombs (50MB per file)
	const maxFileSize = 50 * 1024 * 1024
	if _, err := io.Copy(outFile, io.LimitReader(tr, maxFileSize)); err != nil {
		outFile.Close()
		return fmt.Errorf("failed to extract file %s: %w", targetPath, err)
	}
	outFile.Close()
	return nil
}
