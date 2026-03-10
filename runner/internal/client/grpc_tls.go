// Package client provides gRPC connection management for Runner.
package client

import (
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
//   - grpcAuthority overrides SNI to a non-routable hostname to avoid proxy interception
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
		// Server cert SANs may not include the public domain; SNI is set via grpcAuthority.
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
// SNI is overridden by grpcAuthority to a non-routable hostname; still verifies
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
		// Skip default verification (which includes hostname check) since grpcAuthority
		// overrides SNI to a non-routable hostname. Chain verification is
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

// Note: Certificate renewal logic is in grpc_tls_renewal.go
