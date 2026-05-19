package extension

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

func packageSkillDir(dirPath string) ([]byte, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	baseDir := dirPath
	err := filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		if info.IsDir() && shouldIgnoreDir(info.Name()) {
			return filepath.SkipDir
		}

		if !info.IsDir() && shouldIgnoreFile(info.Name()) {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			f, err := os.Open(path)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(tw, f)
			f.Close()
			if copyErr != nil {
				return copyErr
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gw.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func computeDirSHA(dirPath string) (string, error) {
	h := sha256.New()

	var files []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if shouldIgnoreDir(info.Name()) && path != dirPath {
				return filepath.SkipDir
			}
			return nil
		}
		if shouldIgnoreFile(info.Name()) {
			return nil
		}
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}
		files = append(files, relPath)
		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Strings(files)

	for _, relPath := range files {
		absPath := filepath.Join(dirPath, relPath)
		content, err := os.ReadFile(absPath)
		if err != nil {
			return "", err
		}
		h.Write([]byte(relPath))
		h.Write([]byte{0}) // separator
		h.Write(content)
		h.Write([]byte{0}) // separator
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
