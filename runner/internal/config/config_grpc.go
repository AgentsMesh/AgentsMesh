package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/agentsmesh/runner/internal/textutil"
)

// ==================== gRPC/mTLS Configuration ====================

// UsesGRPC returns true if gRPC mode is configured (certificates present).
func (c *Config) UsesGRPC() bool {
	return c.CertFile != "" && c.KeyFile != "" && c.CAFile != "" && c.GRPCEndpoint != ""
}

// validateGRPCConfig validates gRPC-specific configuration.
func (c *Config) validateGRPCConfig() error {
	if c.GRPCEndpoint == "" {
		return errors.New("grpc_endpoint is required for gRPC mode")
	}
	if c.CertFile == "" {
		return errors.New("cert_file is required for gRPC mode")
	}
	if c.KeyFile == "" {
		return errors.New("key_file is required for gRPC mode")
	}
	if c.CAFile == "" {
		return errors.New("ca_file is required for gRPC mode")
	}

	// Verify certificate files exist
	if _, err := os.Stat(c.CertFile); os.IsNotExist(err) {
		return errors.New("certificate file not found: " + c.CertFile)
	}
	if _, err := os.Stat(c.KeyFile); os.IsNotExist(err) {
		return errors.New("private key file not found: " + c.KeyFile)
	}
	if _, err := os.Stat(c.CAFile); os.IsNotExist(err) {
		return errors.New("CA certificate file not found: " + c.CAFile)
	}

	return nil
}

// GetCertsDir returns the certificates directory path.
func (c *Config) GetCertsDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "agentsmesh", "certs")
	}
	return filepath.Join(home, ".agentsmesh", "certs")
}

// SaveCertificates saves gRPC certificates to the default location.
// Note: On Windows, Unix permission bits (0700/0600) are silently ignored.
// File access is governed by Windows ACLs which default to the creating user.
// For enhanced security on multi-user Windows systems, consider applying
// explicit ACLs via golang.org/x/sys/windows.
func (c *Config) SaveCertificates(certPEM, keyPEM, caCertPEM []byte) error {
	certsDir := c.GetCertsDir()
	if err := os.MkdirAll(certsDir, 0700); err != nil {
		return err
	}

	// Save certificate
	certPath := filepath.Join(certsDir, "runner.crt")
	if err := os.WriteFile(certPath, certPEM, 0600); err != nil {
		return err
	}

	// Save private key
	keyPath := filepath.Join(certsDir, "runner.key")
	if err := os.WriteFile(keyPath, keyPEM, 0600); err != nil {
		return err
	}

	// Save CA certificate
	caPath := filepath.Join(certsDir, "ca.crt")
	if err := os.WriteFile(caPath, caCertPEM, 0644); err != nil {
		return err
	}

	// Update config paths
	c.CertFile = certPath
	c.KeyFile = keyPath
	c.CAFile = caPath

	return nil
}

// UpdateGRPCEndpointInFile updates the grpc_endpoint field in the config file without
// requiring full re-registration. Used by auto-discovery to heal stale endpoints.
//
// It replaces only the grpc_endpoint line, preserving all comments, key ordering,
// and whitespace in the file byte-for-byte.
func UpdateGRPCEndpointInFile(configFile, newEndpoint string) error {
	data, err := os.ReadFile(configFile)
	if err != nil {
		return errors.New("failed to read config file: " + err.Error())
	}

	// Normalize line endings to handle Windows \r\n, then detect original style
	// so we can preserve it when writing back.
	raw := string(data)
	useCRLF := strings.Contains(raw, "\r\n")
	content := textutil.NormalizeLineEndings(raw)
	lines := strings.Split(content, "\n")
	found := false
	for i, line := range lines {
		if strings.HasPrefix(strings.TrimSpace(line), "grpc_endpoint:") {
			lines[i] = "grpc_endpoint: " + newEndpoint
			found = true
			break
		}
	}
	if !found {
		// Key doesn't exist yet — append it. Ensure the last existing line is
		// empty so the new key starts on its own line (handles files without
		// a trailing newline).
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "grpc_endpoint: "+newEndpoint)
	}

	// Preserve original line ending style when writing back
	lineEnding := "\n"
	if useCRLF {
		lineEnding = "\r\n"
	}
	if err := os.WriteFile(configFile, []byte(strings.Join(lines, lineEnding)), 0600); err != nil {
		return errors.New("failed to write config file: " + err.Error())
	}

	return nil
}

// LoadGRPCConfig auto-detects certificate paths if not already set in config.
func (c *Config) LoadGRPCConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil // Not an error
	}

	// Set certificate paths if files exist
	certsDir := filepath.Join(home, ".agentsmesh", "certs")
	if c.CertFile == "" {
		certPath := filepath.Join(certsDir, "runner.crt")
		if _, err := os.Stat(certPath); err == nil {
			c.CertFile = certPath
		}
	}
	if c.KeyFile == "" {
		keyPath := filepath.Join(certsDir, "runner.key")
		if _, err := os.Stat(keyPath); err == nil {
			c.KeyFile = keyPath
		}
	}
	if c.CAFile == "" {
		caPath := filepath.Join(certsDir, "ca.crt")
		if _, err := os.Stat(caPath); err == nil {
			c.CAFile = caPath
		}
	}

	return nil
}
