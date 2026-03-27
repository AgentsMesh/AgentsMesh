package backend

import (
	"fmt"
	"os"
	"path/filepath"
)

// saveCertificateFiles saves certificate and key to files atomically.
// Uses write-to-temp-then-rename to prevent inconsistent state on crash.
func (c *Client) saveCertificateFiles(cert, key string) error {
	if c.certFile == "" || c.keyFile == "" {
		return nil // No paths configured, skip saving
	}

	// Write key first (more sensitive) — atomic via temp file + rename
	if err := atomicWriteFile(c.keyFile, []byte(key), 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	if err := atomicWriteFile(c.certFile, []byte(cert), 0644); err != nil {
		return fmt.Errorf("failed to write certificate file: %w", err)
	}

	c.logger.Info("Certificate files saved", "cert_file", c.certFile, "key_file", c.keyFile)
	return nil
}

// atomicWriteFile writes data to a temp file then renames it to the target path.
// This ensures the target file is never in a partially-written state.
func atomicWriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp.*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}

	return os.Rename(tmpName, path)
}

// loadCertificateFiles loads certificate and key from files
func (c *Client) loadCertificateFiles() error {
	if c.certFile == "" || c.keyFile == "" {
		return fmt.Errorf("certificate paths not configured")
	}

	certData, err := os.ReadFile(c.certFile)
	if err != nil {
		return fmt.Errorf("failed to read certificate file: %w", err)
	}

	keyData, err := os.ReadFile(c.keyFile)
	if err != nil {
		return fmt.Errorf("failed to read key file: %w", err)
	}

	c.mu.Lock()
	c.tlsCert = string(certData)
	c.tlsKey = string(keyData)
	c.mu.Unlock()

	return nil
}

// HasTLSCertificate returns whether a TLS certificate is available
func (c *Client) HasTLSCertificate() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tlsCert != "" && c.tlsKey != ""
}

// GetTLSCertificate returns the TLS certificate and key (PEM encoded)
func (c *Client) GetTLSCertificate() (cert, key string) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tlsCert, c.tlsKey
}

// GetTLSExpiry returns the TLS certificate expiry time string
func (c *Client) GetTLSExpiry() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.tlsExpiry
}
