package acme

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/go-acme/lego/v4/certificate"
)

// GetCertificate returns the current certificate
func (m *Manager) GetCertificate() *Certificate {
	m.certMu.RLock()
	defer m.certMu.RUnlock()
	return m.cert
}

// GetCertificatePEM returns certificate and key as PEM strings
func (m *Manager) GetCertificatePEM() (cert string, key string, expiry time.Time, err error) {
	m.certMu.RLock()
	defer m.certMu.RUnlock()

	if m.cert == nil {
		return "", "", time.Time{}, fmt.Errorf("no certificate available")
	}

	return string(m.cert.Certificate), string(m.cert.PrivateKey), m.cert.NotAfter, nil
}

// NeedsRenewal checks if the certificate needs renewal
func (m *Manager) NeedsRenewal() bool {
	m.certMu.RLock()
	defer m.certMu.RUnlock()

	if m.cert == nil {
		return true
	}

	renewalTime := m.cert.NotAfter.AddDate(0, 0, -m.cfg.RenewalDays)
	return time.Now().After(renewalTime)
}

// ObtainCertificate obtains a new wildcard certificate
func (m *Manager) ObtainCertificate(ctx context.Context) error {
	wildcardDomain := "*." + m.cfg.Domain

	m.logger.Info("Obtaining certificate", "domain", wildcardDomain)

	request := certificate.ObtainRequest{
		Domains: []string{wildcardDomain},
		Bundle:  true,
	}

	certificates, err := m.client.Certificate.Obtain(request)
	if err != nil {
		return fmt.Errorf("failed to obtain certificate: %w", err)
	}

	// Parse certificate to get validity dates
	block, _ := pem.Decode(certificates.Certificate)
	if block == nil {
		return fmt.Errorf("failed to decode certificate PEM")
	}

	x509Cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %w", err)
	}

	cert := &Certificate{
		Domain:      wildcardDomain,
		Certificate: certificates.Certificate,
		PrivateKey:  certificates.PrivateKey,
		NotBefore:   x509Cert.NotBefore,
		NotAfter:    x509Cert.NotAfter,
		IssuedAt:    time.Now(),
	}

	m.certMu.Lock()
	m.cert = cert
	m.certMu.Unlock()

	// Save certificate to disk
	if err := m.saveCertificate(); err != nil {
		return fmt.Errorf("failed to save certificate: %w", err)
	}

	m.logger.Info("Certificate obtained successfully",
		"domain", wildcardDomain,
		"not_before", cert.NotBefore,
		"not_after", cert.NotAfter)

	return nil
}

// RenewCertificate renews the current certificate
func (m *Manager) RenewCertificate(ctx context.Context) error {
	m.certMu.RLock()
	currentCert := m.cert
	m.certMu.RUnlock()

	if currentCert == nil {
		return m.ObtainCertificate(ctx)
	}

	m.logger.Info("Renewing certificate", "domain", currentCert.Domain)

	// For renewal, we need to obtain a new certificate
	return m.ObtainCertificate(ctx)
}

// StartAutoRenewal starts background certificate renewal
func (m *Manager) StartAutoRenewal(ctx context.Context) {
	go func() {
		// Check immediately on startup
		if m.NeedsRenewal() {
			m.logger.Info("Certificate needs renewal, obtaining new certificate...")
			if err := m.ObtainCertificate(ctx); err != nil {
				m.logger.Error("Failed to obtain certificate on startup", "error", err)
			}
		}

		// Check daily
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if m.NeedsRenewal() {
					m.logger.Info("Certificate needs renewal, renewing...")
					if err := m.RenewCertificate(ctx); err != nil {
						m.logger.Error("Failed to renew certificate", "error", err)
					}
				}
			}
		}
	}()
}
