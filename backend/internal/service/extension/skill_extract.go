package extension

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

func findSkillDir(extractDir string) (string, error) {
	if fileExists(filepath.Join(extractDir, "SKILL.md")) {
		return extractDir, nil
	}

	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return "", fmt.Errorf("failed to read extracted dir: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			subDir := filepath.Join(extractDir, entry.Name())
			if fileExists(filepath.Join(subDir, "SKILL.md")) {
				return subDir, nil
			}
		}
	}

	return "", fmt.Errorf("SKILL.md not found in uploaded archive")
}

const maxTotalExtractSize = 200 * 1024 * 1024 // 200MB

func extractTarGz(reader io.Reader, targetDir string) error {
	gz, err := gzip.NewReader(reader)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gz.Close()

	var totalSize int64

	tr := tar.NewReader(gz)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar entry: %w", err)
		}

		if header.Size > 0 {
			totalSize += header.Size
			if totalSize > maxTotalExtractSize {
				return fmt.Errorf("archive exceeds maximum total decompressed size of %d bytes", maxTotalExtractSize)
			}
		}

		targetPath := filepath.Join(targetDir, filepath.Clean(header.Name))
		if !strings.HasPrefix(targetPath, filepath.Clean(targetDir)+string(os.PathSeparator)) &&
			targetPath != filepath.Clean(targetDir) {
			slog.Warn("Skipping archive entry with path traversal", "entry", header.Name)
			continue
		}

		switch header.Typeflag {
		case tar.TypeDir:
			dirMode := os.FileMode(header.Mode) & 0755
			if dirMode == 0 {
				dirMode = 0755
			}
			if err := os.MkdirAll(targetPath, dirMode); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", targetPath, err)
			}
		case tar.TypeReg:
			if err := extractTarRegularFile(tr, targetPath, header); err != nil {
				return err
			}
		case tar.TypeSymlink, tar.TypeLink:
			slog.Warn("Skipping symlink/hardlink in archive to prevent symlink attacks", "entry", header.Name, "type", header.Typeflag)
			continue
		default:
			slog.Debug("Skipping unsupported tar entry type", "entry", header.Name, "type", header.Typeflag)
			continue
		}
	}

	return nil
}

func extractTarRegularFile(tr *tar.Reader, targetPath string, header *tar.Header) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", targetPath, err)
	}
	mode := os.FileMode(header.Mode) & 0644
	if mode == 0 {
		mode = 0644
	}
	f, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, io.LimitReader(tr, 50*1024*1024))
	closeErr := f.Close()
	if copyErr != nil {
		return fmt.Errorf("failed to extract file %s: %w", targetPath, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("failed to close file %s: %w", targetPath, closeErr)
	}
	return nil
}
