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
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/tls/certprovider/pemfile"
	"google.golang.org/grpc/security/advancedtls"
)

// createAdvancedTLSCredentials creates TLS credentials using advancedtls package.
// This enables automatic certificate hot-reloading when certificate files are updated.
func (c *GRPCConnection) createAdvancedTLSCredentials() (credentials.TransportCredentials, error) {
	// Create identity certificate provider with file watching
	// This provider will automatically reload certificates when files change
	identityProvider, err := pemfile.NewProvider(pemfile.Options{
		CertFile:        c.certFile,
		KeyFile:         c.keyFile,
		RefreshDuration: 1 * time.Hour, // Check for file changes every hour
	})
	if err != nil {
		logger.GRPC().Warn("Failed to create pemfile identity provider, using fallback", "error", err)
		return c.createFallbackTLSCredentials()
	}

	// Create root certificate provider with file watching for CA
	// This allows CA certificate rotation if needed
	rootProvider, err := pemfile.NewProvider(pemfile.Options{
		RootFile:        c.caFile,
		RefreshDuration: 24 * time.Hour, // CA changes are rare, check daily
	})
	if err != nil {
		logger.GRPC().Warn("Failed to create pemfile root provider, using static CA", "error", err)
		// Fall back to static CA if file watching fails
		caCert, readErr := os.ReadFile(c.caFile)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", readErr)
		}
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		// Save identity provider for cleanup, rootProvider is nil in this path
		c.identityProvider = identityProvider
		c.rootProvider = nil

		// Use static root certificates
		options := &advancedtls.Options{
			IdentityOptions: advancedtls.IdentityCertificateOptions{
				IdentityProvider: identityProvider,
			},
			RootOptions: advancedtls.RootCertificateOptions{
				RootCertificates: caPool,
			},
			MinTLSVersion: tls.VersionTLS13,
			MaxTLSVersion: tls.VersionTLS13,
			// Only verify certificate chain, not hostname (server may use IP)
			VerificationType: advancedtls.CertVerification,
		}
		creds, err := advancedtls.NewClientCreds(options)
		if err != nil {
			return nil, err
		}
		return creds, nil
	}

	// Save providers for cleanup to prevent goroutine leaks
	c.identityProvider = identityProvider
	c.rootProvider = rootProvider

	// Create advancedtls client options with both providers
	options := &advancedtls.Options{
		IdentityOptions: advancedtls.IdentityCertificateOptions{
			IdentityProvider: identityProvider,
		},
		RootOptions: advancedtls.RootCertificateOptions{
			RootProvider: rootProvider,
		},
		MinTLSVersion: tls.VersionTLS13,
		MaxTLSVersion: tls.VersionTLS13,
		// Only verify certificate chain, not hostname (server may use IP)
		VerificationType: advancedtls.CertVerification,
	}

	creds, err := advancedtls.NewClientCreds(options)
	if err != nil {
		return nil, err
	}
	return creds, nil
}

// createFallbackTLSCredentials creates standard TLS credentials as fallback.
// Used when advancedtls is not available or fails to initialize.
func (c *GRPCConnection) createFallbackTLSCredentials() (credentials.TransportCredentials, error) {
	// Load client certificate
	cert, err := tls.LoadX509KeyPair(c.certFile, c.keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// Load CA certificate (only trust AgentMesh CA, not system CAs)
	caCert, err := os.ReadFile(c.caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	// TLS config - only trust our private CA
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		ServerName:   c.tlsServerName,
		MinVersion:   tls.VersionTLS13,
	}

	return credentials.NewTLS(tlsConfig), nil
}

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

	// Check if renewal is needed (30 days before expiry by default)
	if daysUntilExpiry <= float64(c.certRenewalDays) {
		log.Info("Certificate expires soon, triggering renewal", "days_until_expiry", daysUntilExpiry)

		// Attempt to renew certificate via REST API
		if err := c.renewCertificate(); err != nil {
			log.Error("Certificate renewal failed", "error", err)
			// Don't return here - still check for urgent reconnection
		} else {
			log.Info("Certificate renewed successfully, advancedtls will auto-reload")
		}
	}

	// Check if urgent reconnection is needed (7 days before expiry by default)
	// This ensures long-lived connections use the new certificate
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
// Returns true if the certificate is expired, along with error details if any.
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
// The new certificate is saved to files, and advancedtls will automatically reload them.
func (c *GRPCConnection) renewCertificate() error {
	if c.serverURL == "" {
		return fmt.Errorf("server URL not configured, cannot renew certificate")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Call the renewal API using mTLS
	// Note: This requires a working TLS connection with the current certificate
	result, err := RenewCertificate(ctx, RenewalRequest{
		ServerURL: c.serverURL,
		CertFile:  c.certFile,
		KeyFile:   c.keyFile,
		CAFile:    c.caFile,
	})
	if err != nil {
		return fmt.Errorf("renewal API call failed: %w", err)
	}

	// Save new certificate to files
	// advancedtls FileWatcherCertificateProvider will detect the change and reload
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
// This is used to ensure long-lived connections use the new certificate.
func (c *GRPCConnection) triggerReconnect() {
	select {
	case c.reconnectCh <- struct{}{}:
		logger.GRPC().Info("Reconnection triggered")
	default:
		// Reconnection already pending
	}
}
