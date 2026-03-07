package runner

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenewCertificate(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	t.Run("returns error for non-existent runner", func(t *testing.T) {
		_, err := service.RenewCertificate(ctx, "non-existent-node", "serial", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "runner not found")
	})

	t.Run("returns error for certificate mismatch", func(t *testing.T) {
		org := createTestOrg(t, db, "test-org-renew-mismatch")

		// Create runner with specific cert serial
		oldSerial := "old-serial-123"
		r := &runner.Runner{
			OrganizationID:   org.ID,
			NodeID:           "test-node-renew-mismatch",
			Status:           runner.RunnerStatusOffline,
			CertSerialNumber: &oldSerial,
		}
		require.NoError(t, db.Create(r).Error)

		// Try to renew with wrong serial
		_, err := service.RenewCertificate(ctx, "test-node-renew-mismatch", "wrong-serial", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "certificate mismatch")
	})

	t.Run("successfully renews certificate", func(t *testing.T) {
		// Setup PKI
		pkiService, tmpDir := setupTestPKI(t)
		defer os.RemoveAll(tmpDir)

		org := createTestOrg(t, db, "test-org-renew-success")

		// Create runner with certificate
		oldSerial := "old-serial-for-renewal"
		r := &runner.Runner{
			OrganizationID:   org.ID,
			NodeID:           "test-node-renew-success",
			Status:           runner.RunnerStatusOnline,
			CertSerialNumber: &oldSerial,
		}
		require.NoError(t, db.Create(r).Error)

		// Create old certificate record
		oldCert := &runner.Certificate{
			RunnerID:     r.ID,
			SerialNumber: oldSerial,
			Fingerprint:  "old-fingerprint",
			IssuedAt:     time.Now().Add(-30 * 24 * time.Hour),
			ExpiresAt:    time.Now().Add(30 * 24 * time.Hour),
		}
		require.NoError(t, db.Create(oldCert).Error)

		// Renew certificate
		resp, err := service.RenewCertificate(ctx, "test-node-renew-success", oldSerial, pkiService)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Certificate)
		assert.NotEmpty(t, resp.PrivateKey)
		assert.True(t, resp.ExpiresAt.After(time.Now()))

		// Verify old certificate was revoked
		var updatedOldCert runner.Certificate
		require.NoError(t, db.First(&updatedOldCert, oldCert.ID).Error)
		assert.NotNil(t, updatedOldCert.RevokedAt)
		assert.NotNil(t, updatedOldCert.RevocationReason)
		assert.Equal(t, "renewed", *updatedOldCert.RevocationReason)

		// Verify runner was updated with new cert serial
		var updatedRunner runner.Runner
		require.NoError(t, db.First(&updatedRunner, r.ID).Error)
		assert.NotEqual(t, oldSerial, *updatedRunner.CertSerialNumber)
	})
}

func TestRevokeCertificate(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	t.Run("revokes certificate", func(t *testing.T) {
		// Create runner with certificate
		r := &runner.Runner{
			OrganizationID: 1,
			NodeID:         "test-node-cert-1",
		}
		require.NoError(t, db.Create(r).Error)

		cert := &runner.Certificate{
			RunnerID:     r.ID,
			SerialNumber: "test-serial-123",
			Fingerprint:  "test-fingerprint",
			IssuedAt:     time.Now(),
			ExpiresAt:    time.Now().Add(365 * 24 * time.Hour),
		}
		require.NoError(t, db.Create(cert).Error)

		// Revoke
		err := service.RevokeCertificate(ctx, "test-serial-123", "testing")
		require.NoError(t, err)

		// Verify revoked
		var updated runner.Certificate
		require.NoError(t, db.First(&updated, cert.ID).Error)
		assert.NotNil(t, updated.RevokedAt)
		assert.NotNil(t, updated.RevocationReason)
		assert.Equal(t, "testing", *updated.RevocationReason)
	})
}

func TestIsCertificateRevoked(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
	ctx := context.Background()

	t.Run("returns false for non-revoked certificate", func(t *testing.T) {
		// Create runner with non-revoked certificate
		r := &runner.Runner{
			OrganizationID: 1,
			NodeID:         "test-node-cert-2",
		}
		require.NoError(t, db.Create(r).Error)

		cert := &runner.Certificate{
			RunnerID:     r.ID,
			SerialNumber: "non-revoked-serial",
			Fingerprint:  "test-fingerprint",
			IssuedAt:     time.Now(),
			ExpiresAt:    time.Now().Add(365 * 24 * time.Hour),
		}
		require.NoError(t, db.Create(cert).Error)

		revoked, err := service.IsCertificateRevoked(ctx, "non-revoked-serial")
		require.NoError(t, err)
		assert.False(t, revoked)
	})

	t.Run("returns true for revoked certificate", func(t *testing.T) {
		// Create runner with revoked certificate
		r := &runner.Runner{
			OrganizationID: 1,
			NodeID:         "test-node-cert-3",
		}
		require.NoError(t, db.Create(r).Error)

		now := time.Now()
		reason := "test revocation"
		cert := &runner.Certificate{
			RunnerID:         r.ID,
			SerialNumber:     "revoked-serial",
			Fingerprint:      "test-fingerprint",
			IssuedAt:         now,
			ExpiresAt:        now.Add(365 * 24 * time.Hour),
			RevokedAt:        &now,
			RevocationReason: &reason,
		}
		require.NoError(t, db.Create(cert).Error)

		revoked, err := service.IsCertificateRevoked(ctx, "revoked-serial")
		require.NoError(t, err)
		assert.True(t, revoked)
	})

	t.Run("returns false for non-existent certificate", func(t *testing.T) {
		revoked, err := service.IsCertificateRevoked(ctx, "non-existent-serial")
		require.NoError(t, err)
		assert.False(t, revoked)
	})
}
