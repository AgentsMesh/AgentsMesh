package client

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestCert generates a self-signed certificate with the given expiry and writes it to dir.
func createTestCert(t *testing.T, dir string, notAfter time.Time) (certFile, keyFile string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     notAfter,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	require.NoError(t, err)

	certFile = filepath.Join(dir, "cert.pem")
	keyFile = filepath.Join(dir, "key.pem")

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	require.NoError(t, os.WriteFile(certFile, certPEM, 0600))

	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	require.NoError(t, os.WriteFile(keyFile, keyPEM, 0600))

	return certFile, keyFile
}

func testConn(certFile, keyFile string) *GRPCConnection {
	return &GRPCConnection{
		certFile:        certFile,
		keyFile:         keyFile,
		certRenewalDays: 30,
		certUrgentDays:  7,
	}
}

func TestGetCertDaysUntilExpiry_ValidCert(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := createTestCert(t, dir, time.Now().Add(90*24*time.Hour))
	conn := testConn(certFile, keyFile)

	days, err := conn.getCertDaysUntilExpiry()
	require.NoError(t, err)
	assert.InDelta(t, 90, days, 1.0, "expected ~90 days until expiry")
}

func TestGetCertDaysUntilExpiry_ExpiredCert(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := createTestCert(t, dir, time.Now().Add(-5*24*time.Hour))
	conn := testConn(certFile, keyFile)

	days, err := conn.getCertDaysUntilExpiry()
	require.NoError(t, err)
	assert.InDelta(t, -5, days, 1.0, "expected ~-5 days (expired)")
	assert.Less(t, days, 0.0, "days should be negative for expired cert")
}

func TestGetCertDaysUntilExpiry_InvalidPath(t *testing.T) {
	conn := testConn("/nonexistent/cert.pem", "/nonexistent/key.pem")

	_, err := conn.getCertDaysUntilExpiry()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load certificate")
}

func TestIsCertificateExpired_NotExpired(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := createTestCert(t, dir, time.Now().Add(90*24*time.Hour))
	conn := testConn(certFile, keyFile)

	expired, err := conn.IsCertificateExpired()
	require.NoError(t, err)
	assert.False(t, expired)
}

func TestIsCertificateExpired_Expired(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := createTestCert(t, dir, time.Now().Add(-1*24*time.Hour))
	conn := testConn(certFile, keyFile)

	expired, err := conn.IsCertificateExpired()
	require.NoError(t, err)
	assert.True(t, expired)
}

func TestGetCertificateExpiryInfo_NeedsRenewal(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := createTestCert(t, dir, time.Now().Add(20*24*time.Hour))
	conn := testConn(certFile, keyFile)

	info, err := conn.GetCertificateExpiryInfo()
	require.NoError(t, err)
	assert.InDelta(t, 20, info.DaysUntilExpiry, 1.0)
	assert.False(t, info.IsExpired)
	assert.True(t, info.NeedsRenewal, "20 days < 30 renewal threshold")
	assert.False(t, info.NeedsUrgent, "20 days > 7 urgent threshold")
}

func TestGetCertificateExpiryInfo_NeedsUrgent(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := createTestCert(t, dir, time.Now().Add(3*24*time.Hour))
	conn := testConn(certFile, keyFile)

	info, err := conn.GetCertificateExpiryInfo()
	require.NoError(t, err)
	assert.InDelta(t, 3, info.DaysUntilExpiry, 1.0)
	assert.False(t, info.IsExpired)
	assert.True(t, info.NeedsRenewal, "3 days < 30 renewal threshold")
	assert.True(t, info.NeedsUrgent, "3 days < 7 urgent threshold")
}

func TestGetCertificateExpiryInfo_Healthy(t *testing.T) {
	dir := t.TempDir()
	certFile, keyFile := createTestCert(t, dir, time.Now().Add(90*24*time.Hour))
	conn := testConn(certFile, keyFile)

	info, err := conn.GetCertificateExpiryInfo()
	require.NoError(t, err)
	assert.InDelta(t, 90, info.DaysUntilExpiry, 1.0)
	assert.False(t, info.IsExpired)
	assert.False(t, info.NeedsRenewal)
	assert.False(t, info.NeedsUrgent)
	assert.WithinDuration(t, time.Now().Add(90*24*time.Hour), info.ExpiresAt, 2*time.Second)
}
