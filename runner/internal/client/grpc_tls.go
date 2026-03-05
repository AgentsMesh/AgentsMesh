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
	"google.golang.org/grpc/credentials/tls/certprovider"
	"google.golang.org/grpc/credentials/tls/certprovider/pemfile"
	"google.golang.org/grpc/security/advancedtls"
)

// createAdvancedTLSCredentials creates TLS credentials using advancedtls package.
// This enables automatic certificate hot-reloading when certificate files are updated.
//
// Uses CertVerification (chain-only, no hostname check) because:
//   - Private PKI: both server and client certs are signed by our own CA
//   - Server cert SANs may not include the public hostname (e.g., api.agentsmesh.cn)
//   - SNI must remain as the dial target hostname for correct Traefik TCP routing
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
	rootProvider, err := pemfile.NewProvider(pemfile.Options{
		RootFile:        c.caFile,
		RefreshDuration: 24 * time.Hour, // CA changes are rare, check daily
	})
	if err != nil {
		logger.GRPC().Warn("Failed to create pemfile root provider, using static CA", "error", err)
		return c.createStaticCACredentials(identityProvider)
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
		// Only verify certificate chain, not hostname.
		// Server cert SANs may not include the public domain; SNI must stay as dial target for routing.
		VerificationType: advancedtls.CertVerification,
	}

	creds, err := advancedtls.NewClientCreds(options)
	if err != nil {
		return nil, err
	}
	return creds, nil
}

// createStaticCACredentials creates advancedtls credentials with a static CA pool.
// Used when the root certificate file watcher fails to initialize.
func (c *GRPCConnection) createStaticCACredentials(identityProvider certprovider.Provider) (credentials.TransportCredentials, error) {
	caCert, err := os.ReadFile(c.caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	c.identityProvider = identityProvider
	c.rootProvider = nil

	options := &advancedtls.Options{
		IdentityOptions: advancedtls.IdentityCertificateOptions{
			IdentityProvider: identityProvider,
		},
		RootOptions: advancedtls.RootCertificateOptions{
			RootCertificates: caPool,
		},
		MinTLSVersion:    tls.VersionTLS13,
		MaxTLSVersion:    tls.VersionTLS13,
		VerificationType: advancedtls.CertVerification,
	}
	creds, err := advancedtls.NewClientCreds(options)
	if err != nil {
		return nil, err
	}
	return creds, nil
}

// createFallbackTLSCredentials creates standard TLS credentials as fallback.
// Used when advancedtls pemfile providers fail to initialize.
//
// Uses InsecureSkipVerify + VerifyConnection to verify chain without hostname check.
// This keeps SNI as the dial target for correct proxy routing while still verifying
// the server cert is signed by our private CA.
func (c *GRPCConnection) createFallbackTLSCredentials() (credentials.TransportCredentials, error) {
	cert, err := tls.LoadX509KeyPair(c.certFile, c.keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	caCert, err := os.ReadFile(c.caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS13,
		// Skip default verification (which includes hostname check) so SNI stays as
		// the dial target hostname for correct Traefik routing. Chain verification is
		// performed in VerifyConnection below.
		InsecureSkipVerify: true,
		VerifyConnection: func(cs tls.ConnectionState) error {
			if len(cs.PeerCertificates) == 0 {
				return fmt.Errorf("server presented no certificates")
			}
			// Verify the certificate chain against our private CA (no hostname check)
			opts := x509.VerifyOptions{
				Roots:         caPool,
				Intermediates: x509.NewCertPool(),
			}
			for _, cert := range cs.PeerCertificates[1:] {
				opts.Intermediates.AddCert(cert)
			}
			_, err := cs.PeerCertificates[0].Verify(opts)
			return err
		},
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
