// Package client provides gRPC connection management for Runner.
package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"os"
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// certRenewalChecker periodically checks certificate expiry and triggers renewal.
func (c *GRPCConnection) certRenewalChecker(ctx context.Context, done <-chan struct{}) {
	ticker := time.NewTicker(c.certRenewalCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.checkCertificateExpiry()
		}
	}
}

// checkCertificateExpiry checks if the certificate needs renewal.
func (c *GRPCConnection) checkCertificateExpiry() {
	log := logger.GRPC()
	daysUntilExpiry, err := c.getCertDaysUntilExpiry()
	if err != nil {
		log.Error("Failed to check certificate expiry", "error", err)
		return
	}

	logger.GRPCTrace().Trace("Certificate expiry check", "days_until_expiry", daysUntilExpiry)

	if daysUntilExpiry <= float64(c.certRenewalDays) {
		log.Info("Certificate expires soon, triggering renewal", "days_until_expiry", daysUntilExpiry)
		if err := c.renewCertificate(); err != nil {
			log.Error("Certificate renewal failed", "error", err)
		} else {
			log.Info("Certificate renewed successfully, advancedtls will auto-reload")
		}
	}

	if daysUntilExpiry <= float64(c.certUrgentDays) {
		log.Warn("Certificate expiring urgently, triggering reconnection", "days_until_expiry", daysUntilExpiry)
		c.triggerReconnect()
	}
}

// getCertDaysUntilExpiry returns the number of days until the certificate expires.
func (c *GRPCConnection) getCertDaysUntilExpiry() (float64, error) {
	cert, err := tls.LoadX509KeyPair(c.certFile, c.keyFile)
	if err != nil {
		return 0, fmt.Errorf("failed to load certificate: %w", err)
	}

	if len(cert.Certificate) == 0 {
		return 0, fmt.Errorf("no certificate found")
	}

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return 0, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return time.Until(x509Cert.NotAfter).Hours() / 24, nil
}

// IsCertificateExpired checks if the certificate has expired.
func (c *GRPCConnection) IsCertificateExpired() (bool, error) {
	daysUntilExpiry, err := c.getCertDaysUntilExpiry()
	if err != nil {
		return false, err
	}
	return daysUntilExpiry <= 0, nil
}

// CertificateExpiryInfo returns detailed information about certificate expiry.
type CertificateExpiryInfo struct {
	DaysUntilExpiry float64
	ExpiresAt       time.Time
	IsExpired       bool
	NeedsRenewal    bool
	NeedsUrgent     bool
}

// GetCertificateExpiryInfo returns detailed certificate expiry information.
func (c *GRPCConnection) GetCertificateExpiryInfo() (*CertificateExpiryInfo, error) {
	cert, err := tls.LoadX509KeyPair(c.certFile, c.keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	if len(cert.Certificate) == 0 {
		return nil, fmt.Errorf("no certificate found")
	}

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	daysUntilExpiry := time.Until(x509Cert.NotAfter).Hours() / 24

	return &CertificateExpiryInfo{
		DaysUntilExpiry: daysUntilExpiry,
		ExpiresAt:       x509Cert.NotAfter,
		IsExpired:       daysUntilExpiry <= 0,
		NeedsRenewal:    daysUntilExpiry <= float64(c.certRenewalDays),
		NeedsUrgent:     daysUntilExpiry <= float64(c.certUrgentDays),
	}, nil
}

// renewCertificate calls the Backend REST API to renew the certificate.
func (c *GRPCConnection) renewCertificate() error {
	if c.serverURL == "" {
		return fmt.Errorf("server URL not configured, cannot renew certificate")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := RenewCertificate(ctx, RenewalRequest{
		ServerURL: c.serverURL,
		CertFile:  c.certFile,
		KeyFile:   c.keyFile,
		CAFile:    c.caFile,
	})
	if err != nil {
		return fmt.Errorf("renewal API call failed: %w", err)
	}

	if err := os.WriteFile(c.certFile, []byte(result.Certificate), 0600); err != nil {
		return fmt.Errorf("failed to save new certificate: %w", err)
	}

	if err := os.WriteFile(c.keyFile, []byte(result.PrivateKey), 0600); err != nil {
		return fmt.Errorf("failed to save new private key: %w", err)
	}

	logger.GRPC().Info("New certificate saved",
		"expires_at", time.Unix(result.ExpiresAt, 0).Format(time.RFC3339))

	return nil
}

// triggerReconnect signals the connection loop to reconnect.
func (c *GRPCConnection) triggerReconnect() {
	select {
	case c.reconnectCh <- struct{}{}:
		logger.GRPC().Info("Reconnection triggered")
	default:
		// Reconnection already pending
	}
}
