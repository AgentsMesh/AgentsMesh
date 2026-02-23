package config

import (
	"errors"
	"os"
	"path/filepath"
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
		return "/etc/agentsmesh/certs"
	}
	return filepath.Join(home, ".agentsmesh", "certs")
}

// SaveCertificates saves gRPC certificates to the default location.
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
