package pki

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"os"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPKI_IssueCertificate verifies end-to-end certificate issuance.
func TestPKI_IssueCertificate(t *testing.T) {
	svc, tmpDir := setupTestPKI(t)
	defer os.RemoveAll(tmpDir)

	info, err := svc.IssueRunnerCertificate("node-abc", "org-xyz")
	require.NoError(t, err)

	// Verify the returned PEM can be parsed into a valid x509 cert
	block, _ := pem.Decode(info.CertPEM)
	require.NotNil(t, block, "CertPEM should decode as PEM")

	cert, err := x509.ParseCertificate(block.Bytes)
	require.NoError(t, err)

	assert.Equal(t, "node-abc", cert.Subject.CommonName)
	assert.Contains(t, cert.Subject.Organization, "org-xyz")
	assert.Contains(t, cert.Subject.OrganizationalUnit, "runners")
	assert.Contains(t, cert.ExtKeyUsage, x509.ExtKeyUsageClientAuth)

	// Verify signed by our CA
	opts := x509.VerifyOptions{
		Roots:     svc.CACertPool(),
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}
	_, err = cert.Verify(opts)
	assert.NoError(t, err)
}

// TestPKI_ValidateCertificate issues then validates a certificate.
func TestPKI_ValidateCertificate(t *testing.T) {
	svc, tmpDir := setupTestPKI(t)
	defer os.RemoveAll(tmpDir)

	info, err := svc.IssueRunnerCertificate("val-node", "val-org")
	require.NoError(t, err)

	nodeID, orgSlug, serial, err := svc.ValidateCertificate(info.CertPEM)
	require.NoError(t, err)

	assert.Equal(t, "val-node", nodeID)
	assert.Equal(t, "val-org", orgSlug)
	assert.Equal(t, info.SerialNumber, serial)
}

// TestPKI_ConcurrentIssue issues 10 certs concurrently and checks uniqueness.
func TestPKI_ConcurrentIssue(t *testing.T) {
	svc, tmpDir := setupTestPKI(t)
	defer os.RemoveAll(tmpDir)

	const n = 10
	type result struct {
		info *CertificateInfo
		err  error
	}

	results := make([]result, n)
	var wg sync.WaitGroup
	wg.Add(n)

	for i := 0; i < n; i++ {
		go func(idx int) {
			defer wg.Done()
			info, err := svc.IssueRunnerCertificate(
				"concurrent-node", "concurrent-org",
			)
			results[idx] = result{info: info, err: err}
		}(i)
	}
	wg.Wait()

	serials := make(map[string]struct{}, n)
	fingerprints := make(map[string]struct{}, n)

	for i, r := range results {
		require.NoError(t, r.err, "cert %d should succeed", i)
		require.NotNil(t, r.info)

		_, dup := serials[r.info.SerialNumber]
		assert.False(t, dup, "serial %s duplicated at index %d", r.info.SerialNumber, i)
		serials[r.info.SerialNumber] = struct{}{}

		_, dup = fingerprints[r.info.Fingerprint]
		assert.False(t, dup, "fingerprint duplicated at index %d", i)
		fingerprints[r.info.Fingerprint] = struct{}{}
	}
}

// TestPKI_RevocationChecker_NilRepo verifies that a nil repo returns not-revoked.
func TestPKI_RevocationChecker_NilRepo(t *testing.T) {
	checker := NewRevocationChecker(nil)

	revoked, err := checker.IsRevoked(context.Background(), "any-serial")
	require.NoError(t, err)
	assert.False(t, revoked)

	err = checker.Revoke(context.Background(), "any-serial", "test reason")
	require.NoError(t, err)
}

// mockRevocationRepo is a simple in-memory mock.
type mockRevocationRepo struct {
	revoked map[string]bool
	mu      sync.Mutex
}

func (m *mockRevocationRepo) IsRevoked(_ context.Context, serial string) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.revoked[serial], nil
}

func (m *mockRevocationRepo) GetRevokedSerials(_ context.Context) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []string
	for s := range m.revoked {
		out = append(out, s)
	}
	return out, nil
}

func (m *mockRevocationRepo) Revoke(_ context.Context, serial, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.revoked[serial] = true
	return nil
}

// TestPKI_RevocationChecker_WithRepo verifies revocation through a mock repo.
func TestPKI_RevocationChecker_WithRepo(t *testing.T) {
	repo := &mockRevocationRepo{revoked: make(map[string]bool)}
	checker := NewRevocationChecker(repo)

	// Not revoked initially
	revoked, err := checker.IsRevoked(context.Background(), "serial-1")
	require.NoError(t, err)
	assert.False(t, revoked)

	// Revoke it
	err = checker.Revoke(context.Background(), "serial-1", "compromised")
	require.NoError(t, err)

	// Now it should be revoked
	revoked, err = checker.IsRevoked(context.Background(), "serial-1")
	require.NoError(t, err)
	assert.True(t, revoked)
}
