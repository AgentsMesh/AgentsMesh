package acme

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// loadCertificate loads certificate from disk
func (m *Manager) loadCertificate() error {
	certPath := filepath.Join(m.cfg.StorageDir, "certificate.json")

	data, err := os.ReadFile(certPath)
	if err != nil {
		return err
	}

	var cert Certificate
	if err := json.Unmarshal(data, &cert); err != nil {
		return fmt.Errorf("failed to unmarshal certificate: %w", err)
	}

	m.certMu.Lock()
	m.cert = &cert
	m.certMu.Unlock()

	m.logger.Info("Loaded existing certificate",
		"domain", cert.Domain,
		"not_after", cert.NotAfter)

	return nil
}

// saveCertificate saves certificate to disk
func (m *Manager) saveCertificate() error {
	m.certMu.RLock()
	cert := m.cert
	m.certMu.RUnlock()

	if cert == nil {
		return fmt.Errorf("no certificate to save")
	}

	certPath := filepath.Join(m.cfg.StorageDir, "certificate.json")

	data, err := json.MarshalIndent(cert, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal certificate: %w", err)
	}

	if err := os.WriteFile(certPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write certificate file: %w", err)
	}

	// Also save PEM files for debugging/manual use
	certPEMPath := filepath.Join(m.cfg.StorageDir, "cert.pem")
	keyPEMPath := filepath.Join(m.cfg.StorageDir, "key.pem")

	if err := os.WriteFile(certPEMPath, cert.Certificate, 0644); err != nil {
		m.logger.Error("Failed to write cert.pem", "error", err)
	}
	if err := os.WriteFile(keyPEMPath, cert.PrivateKey, 0600); err != nil {
		m.logger.Error("Failed to write key.pem", "error", err)
	}

	return nil
}
