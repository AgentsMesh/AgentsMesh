package license

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// generateTestKeyPair returns an RSA key pair and the PEM-encoded public key
// written to a temp file.
func generateTestKeyPair(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	pubBytes, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	require.NoError(t, err)

	pubPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pubBytes,
	})

	f, err := os.CreateTemp(t.TempDir(), "pubkey-*.pem")
	require.NoError(t, err)
	_, err = f.Write(pubPEM)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	return priv, f.Name()
}

// signLicense signs the license data (excluding the Signature field) using
// RSA-SHA256 and returns the base64-encoded signature.
func signLicense(t *testing.T, priv *rsa.PrivateKey, ld *LicenseData) string {
	t.Helper()
	dataToSign := struct {
		LicenseKey       string        `json:"license_key"`
		OrganizationName string        `json:"organization_name"`
		ContactEmail     string        `json:"contact_email"`
		Plan             string        `json:"plan"`
		Limits           LicenseLimits `json:"limits"`
		Features         []string      `json:"features,omitempty"`
		IssuedAt         time.Time     `json:"issued_at"`
		ExpiresAt        time.Time     `json:"expires_at"`
	}{
		LicenseKey:       ld.LicenseKey,
		OrganizationName: ld.OrganizationName,
		ContactEmail:     ld.ContactEmail,
		Plan:             ld.Plan,
		Limits:           ld.Limits,
		Features:         ld.Features,
		IssuedAt:         ld.IssuedAt,
		ExpiresAt:        ld.ExpiresAt,
	}

	jsonData, err := json.Marshal(dataToSign)
	require.NoError(t, err)

	hash := sha256.Sum256(jsonData)
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, hash[:])
	require.NoError(t, err)

	return base64.StdEncoding.EncodeToString(sig)
}

func TestLicense_ServiceConstruction(t *testing.T) {
	// Creating a service with empty config (no key, no file) should succeed.
	svc, err := NewService(nil, &config.LicenseConfig{}, slog.Default())
	require.NoError(t, err)
	assert.NotNil(t, svc)
	assert.False(t, svc.IsLicenseValid())
}

func TestLicense_ParseAndVerify_ValidSignature(t *testing.T) {
	priv, pubKeyPath := generateTestKeyPair(t)

	svc, err := NewService(nil, &config.LicenseConfig{
		PublicKeyPath: pubKeyPath,
	}, slog.Default())
	require.NoError(t, err)

	ld := &LicenseData{
		LicenseKey:       "LIC-TEST-001",
		OrganizationName: "TestCorp",
		ContactEmail:     "admin@testcorp.com",
		Plan:             "enterprise",
		Limits:           LicenseLimits{MaxUsers: 100, MaxRunners: 10, MaxRepositories: 50, MaxPodMinutes: -1},
		Features:         []string{"sso", "audit"},
		IssuedAt:         time.Now().Add(-24 * time.Hour),
		ExpiresAt:        time.Now().Add(365 * 24 * time.Hour),
	}
	ld.Signature = signLicense(t, priv, ld)

	data, err := json.Marshal(ld)
	require.NoError(t, err)

	parsed, err := svc.ParseAndVerify(data)
	require.NoError(t, err)
	assert.Equal(t, "LIC-TEST-001", parsed.LicenseKey)
	assert.Equal(t, "enterprise", parsed.Plan)
	assert.Equal(t, 100, parsed.Limits.MaxUsers)
}

func TestLicense_ParseAndVerify_InvalidSignature(t *testing.T) {
	_, pubKeyPath := generateTestKeyPair(t)

	svc, err := NewService(nil, &config.LicenseConfig{
		PublicKeyPath: pubKeyPath,
	}, slog.Default())
	require.NoError(t, err)

	ld := &LicenseData{
		LicenseKey:       "LIC-BAD-SIG",
		OrganizationName: "BadCorp",
		ContactEmail:     "bad@corp.com",
		Plan:             "pro",
		Limits:           LicenseLimits{MaxUsers: 5},
		IssuedAt:         time.Now(),
		ExpiresAt:        time.Now().Add(24 * time.Hour),
		Signature:        base64.StdEncoding.EncodeToString([]byte("invalid-signature")),
	}

	data, err := json.Marshal(ld)
	require.NoError(t, err)

	_, err = svc.ParseAndVerify(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signature")
}

func TestLicense_ParseAndVerify_Expired(t *testing.T) {
	priv, pubKeyPath := generateTestKeyPair(t)

	svc, err := NewService(nil, &config.LicenseConfig{
		PublicKeyPath: pubKeyPath,
	}, slog.Default())
	require.NoError(t, err)

	ld := &LicenseData{
		LicenseKey:       "LIC-EXPIRED",
		OrganizationName: "OldCorp",
		ContactEmail:     "old@corp.com",
		Plan:             "pro",
		Limits:           LicenseLimits{MaxUsers: 5},
		IssuedAt:         time.Now().Add(-48 * time.Hour),
		ExpiresAt:        time.Now().Add(-24 * time.Hour), // already expired
	}
	ld.Signature = signLicense(t, priv, ld)

	data, err := json.Marshal(ld)
	require.NoError(t, err)

	_, err = svc.ParseAndVerify(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestLicense_CheckLimits(t *testing.T) {
	svc, err := NewService(nil, &config.LicenseConfig{}, slog.Default())
	require.NoError(t, err)

	// No license loaded — should error
	err = svc.CheckLimits(1, 1, 1, 1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no active license")

	// Manually set a license
	svc.mu.Lock()
	svc.currentLicense = &LicenseData{
		LicenseKey: "LIC-LIMITS",
		Limits:     LicenseLimits{MaxUsers: 10, MaxRunners: 5, MaxRepositories: 20, MaxPodMinutes: 1000},
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	svc.mu.Unlock()

	// Within limits
	assert.NoError(t, svc.CheckLimits(5, 3, 10, 500))

	// Exceeded users
	err = svc.CheckLimits(11, 3, 10, 500)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "user limit exceeded")

	// Exceeded runners
	err = svc.CheckLimits(5, 6, 10, 500)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "runner limit exceeded")

	// Unlimited (-1) should pass any amount
	svc.mu.Lock()
	svc.currentLicense.Limits.MaxUsers = -1
	svc.mu.Unlock()
	assert.NoError(t, svc.CheckLimits(99999, 3, 10, 500))
}

func TestLicense_GetLicenseStatus(t *testing.T) {
	svc, err := NewService(nil, &config.LicenseConfig{}, slog.Default())
	require.NoError(t, err)

	// No license → inactive
	status := svc.GetLicenseStatus()
	assert.False(t, status.IsActive)
	assert.Contains(t, status.Message, "No license installed")

	// Active license
	svc.mu.Lock()
	svc.currentLicense = &LicenseData{
		LicenseKey:       "LIC-STATUS",
		OrganizationName: "StatusCorp",
		Plan:             "enterprise",
		Limits:           LicenseLimits{MaxUsers: 100},
		Features:         []string{"sso"},
		ExpiresAt:        time.Now().Add(365 * 24 * time.Hour),
	}
	svc.mu.Unlock()

	status = svc.GetLicenseStatus()
	assert.True(t, status.IsActive)
	assert.Equal(t, "LIC-STATUS", status.LicenseKey)
	assert.Contains(t, status.Message, "active")
}

func TestLicense_HasFeature(t *testing.T) {
	svc, err := NewService(nil, &config.LicenseConfig{}, slog.Default())
	require.NoError(t, err)

	// No license
	assert.False(t, svc.HasFeature("sso"))

	svc.mu.Lock()
	svc.currentLicense = &LicenseData{
		LicenseKey: "LIC-FEAT",
		Features:   []string{"sso", "audit"},
		ExpiresAt:  time.Now().Add(24 * time.Hour),
	}
	svc.mu.Unlock()

	assert.True(t, svc.HasFeature("sso"))
	assert.True(t, svc.HasFeature("audit"))
	assert.False(t, svc.HasFeature("unknown"))
}
