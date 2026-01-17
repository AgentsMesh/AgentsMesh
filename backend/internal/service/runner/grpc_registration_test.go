package runner

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"testing"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestAuthURL(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	t.Run("creates pending auth with all fields", func(t *testing.T) {
		req := &RequestAuthURLRequest{
			MachineKey: "test-machine-key-123",
			NodeID:     "test-node",
			Labels:     map[string]string{"env": "test"},
		}

		resp, err := service.RequestAuthURL(ctx, req, "https://example.com")
		require.NoError(t, err)
		assert.NotEmpty(t, resp.AuthURL)
		assert.NotEmpty(t, resp.AuthKey)
		assert.Equal(t, 900, resp.ExpiresIn)
		assert.Contains(t, resp.AuthURL, "https://example.com/runners/authorize?key=")

		// Verify pending auth was created in database
		var pendingAuth runner.PendingAuth
		err = db.Where("auth_key = ?", resp.AuthKey).First(&pendingAuth).Error
		require.NoError(t, err)
		assert.Equal(t, "test-machine-key-123", pendingAuth.MachineKey)
		assert.NotNil(t, pendingAuth.NodeID)
		assert.Equal(t, "test-node", *pendingAuth.NodeID)
	})

	t.Run("creates pending auth without optional fields", func(t *testing.T) {
		req := &RequestAuthURLRequest{
			MachineKey: "test-machine-key-456",
		}

		resp, err := service.RequestAuthURL(ctx, req, "https://example.com")
		require.NoError(t, err)
		assert.NotEmpty(t, resp.AuthKey)

		// Verify pending auth was created
		var pendingAuth runner.PendingAuth
		err = db.Where("auth_key = ?", resp.AuthKey).First(&pendingAuth).Error
		require.NoError(t, err)
		assert.Nil(t, pendingAuth.NodeID)
	})

	t.Run("returns error for empty machine key", func(t *testing.T) {
		req := &RequestAuthURLRequest{
			MachineKey: "",
		}

		_, err := service.RequestAuthURL(ctx, req, "https://example.com")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "machine_key is required")
	})
}

func TestGetAuthStatus(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	t.Run("returns pending status", func(t *testing.T) {
		// Create pending auth
		authKey := generateTestAuthKey()
		pendingAuth := &runner.PendingAuth{
			AuthKey:    authKey,
			MachineKey: "test-machine",
			ExpiresAt:  time.Now().Add(15 * time.Minute),
			Authorized: false,
		}
		require.NoError(t, db.Create(pendingAuth).Error)

		resp, err := service.GetAuthStatus(ctx, authKey, nil)
		require.NoError(t, err)
		assert.Equal(t, "pending", resp.Status)
	})

	t.Run("returns expired status", func(t *testing.T) {
		// Create expired pending auth
		authKey := generateTestAuthKey()
		pendingAuth := &runner.PendingAuth{
			AuthKey:    authKey,
			MachineKey: "test-machine",
			ExpiresAt:  time.Now().Add(-1 * time.Hour), // Already expired
			Authorized: false,
		}
		require.NoError(t, db.Create(pendingAuth).Error)

		resp, err := service.GetAuthStatus(ctx, authKey, nil)
		require.NoError(t, err)
		assert.Equal(t, "expired", resp.Status)
	})

	t.Run("returns error for non-existent auth key", func(t *testing.T) {
		_, err := service.GetAuthStatus(ctx, "non-existent-key", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("returns authorized status with certificate", func(t *testing.T) {
		// Setup PKI
		pkiService, tmpDir := setupTestPKI(t)
		defer os.RemoveAll(tmpDir)

		// Create org
		org := createTestOrg(t, db, "test-org-auth-status-1")

		// Create runner
		r := &runner.Runner{
			OrganizationID: org.ID,
			NodeID:         "test-node-auth-status",
			Status:         runner.RunnerStatusOffline,
		}
		require.NoError(t, db.Create(r).Error)

		// Create authorized pending auth
		authKey := generateTestAuthKey()
		pendingAuth := &runner.PendingAuth{
			AuthKey:        authKey,
			MachineKey:     "test-machine",
			ExpiresAt:      time.Now().Add(15 * time.Minute),
			Authorized:     true,
			RunnerID:       &r.ID,
			OrganizationID: &org.ID,
		}
		require.NoError(t, db.Create(pendingAuth).Error)

		resp, err := service.GetAuthStatus(ctx, authKey, pkiService)
		require.NoError(t, err)
		assert.Equal(t, "authorized", resp.Status)
		assert.NotEmpty(t, resp.Certificate)
		assert.NotEmpty(t, resp.PrivateKey)
		assert.NotEmpty(t, resp.CACertificate)
		assert.Equal(t, r.ID, resp.RunnerID)
	})

	t.Run("returns error when authorized but runner not created", func(t *testing.T) {
		// Create authorized pending auth without runner
		authKey := generateTestAuthKey()
		pendingAuth := &runner.PendingAuth{
			AuthKey:    authKey,
			MachineKey: "test-machine",
			ExpiresAt:  time.Now().Add(15 * time.Minute),
			Authorized: true,
			RunnerID:   nil, // No runner created yet
		}
		require.NoError(t, db.Create(pendingAuth).Error)

		_, err := service.GetAuthStatus(ctx, authKey, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "runner not created")
	})
}

func TestAuthorizeRunner(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	t.Run("authorizes pending auth", func(t *testing.T) {
		// Create org
		org := createTestOrg(t, db, "test-org-auth-1")

		// Create pending auth
		authKey := generateTestAuthKey()
		pendingAuth := &runner.PendingAuth{
			AuthKey:    authKey,
			MachineKey: "test-machine",
			ExpiresAt:  time.Now().Add(15 * time.Minute),
			Authorized: false,
		}
		require.NoError(t, db.Create(pendingAuth).Error)

		// Authorize (using function signature: authKey string, orgID int64, nodeID string)
		resp, err := service.AuthorizeRunner(ctx, authKey, org.ID, "my-runner")
		require.NoError(t, err)
		assert.NotZero(t, resp.ID)
		assert.Equal(t, "my-runner", resp.NodeID)

		// Verify pending auth was updated
		var updated runner.PendingAuth
		require.NoError(t, db.First(&updated, pendingAuth.ID).Error)
		assert.True(t, updated.Authorized)
		assert.NotNil(t, updated.RunnerID)
		assert.NotNil(t, updated.OrganizationID)
	})

	t.Run("returns error for non-existent auth key", func(t *testing.T) {
		org := createTestOrg(t, db, "test-org-auth-2")

		_, err := service.AuthorizeRunner(ctx, "non-existent", org.ID, "")
		assert.Error(t, err)
	})

	t.Run("returns error for expired auth", func(t *testing.T) {
		org := createTestOrg(t, db, "test-org-auth-3")

		// Create expired pending auth
		authKey := generateTestAuthKey()
		pendingAuth := &runner.PendingAuth{
			AuthKey:    authKey,
			MachineKey: "test-machine",
			ExpiresAt:  time.Now().Add(-1 * time.Hour),
			Authorized: false,
		}
		require.NoError(t, db.Create(pendingAuth).Error)

		_, err := service.AuthorizeRunner(ctx, authKey, org.ID, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("returns error for already authorized", func(t *testing.T) {
		org := createTestOrg(t, db, "test-org-auth-4")

		// Create already authorized pending auth
		authKey := generateTestAuthKey()
		pendingAuth := &runner.PendingAuth{
			AuthKey:    authKey,
			MachineKey: "test-machine",
			ExpiresAt:  time.Now().Add(15 * time.Minute),
			Authorized: true,
		}
		require.NoError(t, db.Create(pendingAuth).Error)

		_, err := service.AuthorizeRunner(ctx, authKey, org.ID, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already authorized")
	})

	t.Run("uses nodeID from pending auth when not provided", func(t *testing.T) {
		org := createTestOrg(t, db, "test-org-auth-5")

		// Create pending auth with nodeID pre-filled
		authKey := generateTestAuthKey()
		nodeID := "pre-filled-node-id"
		pendingAuth := &runner.PendingAuth{
			AuthKey:    authKey,
			MachineKey: "test-machine",
			NodeID:     &nodeID,
			ExpiresAt:  time.Now().Add(15 * time.Minute),
			Authorized: false,
		}
		require.NoError(t, db.Create(pendingAuth).Error)

		// Authorize with empty nodeID - should use the one from pendingAuth
		r, err := service.AuthorizeRunner(ctx, authKey, org.ID, "")
		require.NoError(t, err)
		assert.Equal(t, "pre-filled-node-id", r.NodeID)
	})

	t.Run("generates node ID when none provided", func(t *testing.T) {
		org := createTestOrg(t, db, "test-org-auth-6")

		// Create pending auth without nodeID
		authKey := generateTestAuthKey()
		pendingAuth := &runner.PendingAuth{
			AuthKey:    authKey,
			MachineKey: "test-machine",
			NodeID:     nil, // No nodeID
			ExpiresAt:  time.Now().Add(15 * time.Minute),
			Authorized: false,
		}
		require.NoError(t, db.Create(pendingAuth).Error)

		// Authorize with empty nodeID - should generate random one
		r, err := service.AuthorizeRunner(ctx, authKey, org.ID, "")
		require.NoError(t, err)
		assert.Contains(t, r.NodeID, "runner-")
	})

	t.Run("returns error for duplicate runner", func(t *testing.T) {
		org := createTestOrg(t, db, "test-org-auth-7")

		// Create existing runner
		existing := &runner.Runner{
			OrganizationID: org.ID,
			NodeID:         "duplicate-node",
			Status:         runner.RunnerStatusOffline,
		}
		require.NoError(t, db.Create(existing).Error)

		// Create pending auth
		authKey := generateTestAuthKey()
		pendingAuth := &runner.PendingAuth{
			AuthKey:    authKey,
			MachineKey: "test-machine",
			ExpiresAt:  time.Now().Add(15 * time.Minute),
			Authorized: false,
		}
		require.NoError(t, db.Create(pendingAuth).Error)

		// Try to authorize with same nodeID - should fail
		_, err := service.AuthorizeRunner(ctx, authKey, org.ID, "duplicate-node")
		assert.Error(t, err)
	})
}

func TestGenerateGRPCRegistrationToken(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	t.Run("generates token with default settings", func(t *testing.T) {
		org := createTestOrg(t, db, "test-org-grpc-1")

		req := &GenerateGRPCRegistrationTokenRequest{
			ExpiresIn: 3600,
		}
		// GenerateGRPCRegistrationToken(ctx, orgID, userID int64, req, serverURL)
		resp, err := service.GenerateGRPCRegistrationToken(ctx, org.ID, 1, req, "https://example.com")
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Token)
		assert.NotZero(t, resp.ExpiresAt)
		assert.Contains(t, resp.Command, "runner register")

		// Verify token was created in database
		var token runner.GRPCRegistrationToken
		tokenHash := hashToken(resp.Token)
		err = db.Where("token_hash = ?", tokenHash).First(&token).Error
		require.NoError(t, err)
	})

	t.Run("generates single-use token explicitly", func(t *testing.T) {
		org := createTestOrg(t, db, "test-org-grpc-2")

		req := &GenerateGRPCRegistrationTokenRequest{
			ExpiresIn: 7200,
			SingleUse: true,
		}
		resp, err := service.GenerateGRPCRegistrationToken(ctx, org.ID, 1, req, "https://example.com")
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Token)

		// Verify token is single use
		var token runner.GRPCRegistrationToken
		tokenHash := hashToken(resp.Token)
		err = db.Where("token_hash = ?", tokenHash).First(&token).Error
		require.NoError(t, err)
		assert.True(t, token.SingleUse)
	})

	t.Run("generates token with labels", func(t *testing.T) {
		org := createTestOrg(t, db, "test-org-grpc-3")

		req := &GenerateGRPCRegistrationTokenRequest{
			ExpiresIn: 3600,
			Labels:    map[string]string{"env": "production"},
		}
		resp, err := service.GenerateGRPCRegistrationToken(ctx, org.ID, 1, req, "https://example.com")
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Token)
	})
}

func TestRegisterWithToken(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	t.Run("returns error for invalid token", func(t *testing.T) {
		regReq := &RegisterWithTokenRequest{
			Token:  "invalid-token",
			NodeID: "my-runner",
		}
		_, err := service.RegisterWithToken(ctx, regReq, nil)
		assert.Error(t, err)
	})

	t.Run("returns error for expired token", func(t *testing.T) {
		org := createTestOrg(t, db, "test-org-reg-2")

		// Create token that's already expired
		token := generateTestAuthKey()
		tokenHash := hashToken(token)
		grpcToken := &runner.GRPCRegistrationToken{
			TokenHash:      tokenHash,
			OrganizationID: org.ID,
			SingleUse:      true,
			MaxUses:        1,
			ExpiresAt:      time.Now().Add(-1 * time.Hour),
		}
		require.NoError(t, db.Create(grpcToken).Error)

		regReq := &RegisterWithTokenRequest{
			Token:  token,
			NodeID: "my-runner",
		}
		_, err := service.RegisterWithToken(ctx, regReq, nil)
		assert.Error(t, err)
	})

	t.Run("successfully registers runner with token", func(t *testing.T) {
		// Setup PKI
		pkiService, tmpDir := setupTestPKI(t)
		defer os.RemoveAll(tmpDir)

		org := createTestOrg(t, db, "test-org-reg-success")

		// Create valid token
		token := generateTestAuthKey()
		tokenHash := hashToken(token)
		grpcToken := &runner.GRPCRegistrationToken{
			TokenHash:      tokenHash,
			OrganizationID: org.ID,
			SingleUse:      true,
			MaxUses:        1,
			ExpiresAt:      time.Now().Add(1 * time.Hour),
		}
		require.NoError(t, db.Create(grpcToken).Error)

		regReq := &RegisterWithTokenRequest{
			Token:  token,
			NodeID: "my-successful-runner",
		}
		resp, err := service.RegisterWithToken(ctx, regReq, pkiService)
		require.NoError(t, err)
		assert.NotZero(t, resp.RunnerID)
		assert.NotEmpty(t, resp.Certificate)
		assert.NotEmpty(t, resp.PrivateKey)
		assert.NotEmpty(t, resp.CACertificate)
		assert.Equal(t, org.Slug, resp.OrgSlug)

		// Verify runner was created
		var r runner.Runner
		require.NoError(t, db.First(&r, resp.RunnerID).Error)
		assert.Equal(t, "my-successful-runner", r.NodeID)

		// Verify token was incremented
		var updatedToken runner.GRPCRegistrationToken
		require.NoError(t, db.First(&updatedToken, grpcToken.ID).Error)
		assert.Equal(t, 1, updatedToken.UsedCount)
	})

	t.Run("generates node ID if not provided", func(t *testing.T) {
		// Setup PKI
		pkiService, tmpDir := setupTestPKI(t)
		defer os.RemoveAll(tmpDir)

		org := createTestOrg(t, db, "test-org-reg-no-nodeid")

		// Create valid token
		token := generateTestAuthKey()
		tokenHash := hashToken(token)
		grpcToken := &runner.GRPCRegistrationToken{
			TokenHash:      tokenHash,
			OrganizationID: org.ID,
			SingleUse:      true,
			MaxUses:        1,
			ExpiresAt:      time.Now().Add(1 * time.Hour),
		}
		require.NoError(t, db.Create(grpcToken).Error)

		regReq := &RegisterWithTokenRequest{
			Token:  token,
			NodeID: "", // Empty - should be auto-generated
		}
		resp, err := service.RegisterWithToken(ctx, regReq, pkiService)
		require.NoError(t, err)
		assert.NotZero(t, resp.RunnerID)

		// Verify runner has auto-generated node ID
		var r runner.Runner
		require.NoError(t, db.First(&r, resp.RunnerID).Error)
		assert.Contains(t, r.NodeID, "runner-")
	})

	t.Run("returns error for exhausted token", func(t *testing.T) {
		org := createTestOrg(t, db, "test-org-reg-exhausted")

		// Create exhausted token (used_count >= max_uses)
		token := generateTestAuthKey()
		tokenHash := hashToken(token)
		grpcToken := &runner.GRPCRegistrationToken{
			TokenHash:      tokenHash,
			OrganizationID: org.ID,
			SingleUse:      true,
			MaxUses:        1,
			UsedCount:      1, // Already used
			ExpiresAt:      time.Now().Add(1 * time.Hour),
		}
		require.NoError(t, db.Create(grpcToken).Error)

		regReq := &RegisterWithTokenRequest{
			Token:  token,
			NodeID: "my-runner",
		}
		_, err := service.RegisterWithToken(ctx, regReq, nil)
		assert.Error(t, err)
	})
}

func TestGenerateReactivationToken(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	t.Run("generates reactivation token", func(t *testing.T) {
		// Create runner
		r := &runner.Runner{
			OrganizationID: 1,
			NodeID:         "test-node-react-1",
			Status:         runner.RunnerStatusOffline,
		}
		require.NoError(t, db.Create(r).Error)

		resp, err := service.GenerateReactivationToken(ctx, r.ID, 1) // userID=1
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Token)
		assert.Equal(t, 600, resp.ExpiresIn)
		assert.Contains(t, resp.Command, "runner reactivate")

		// Verify token was saved
		var reactivation runner.ReactivationToken
		err = db.Where("runner_id = ?", r.ID).First(&reactivation).Error
		require.NoError(t, err)
		assert.NotEmpty(t, reactivation.TokenHash)
	})

	t.Run("returns error for non-existent runner", func(t *testing.T) {
		_, err := service.GenerateReactivationToken(ctx, 99999, 1)
		assert.Error(t, err)
	})
}

func TestReactivate(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	t.Run("returns error for invalid token", func(t *testing.T) {
		req := &ReactivateRequest{Token: "invalid-token"}
		_, err := service.Reactivate(ctx, req, nil)
		assert.Error(t, err)
	})

	t.Run("returns error for expired token", func(t *testing.T) {
		// Create runner
		r := &runner.Runner{
			OrganizationID: 1,
			NodeID:         "test-node-react-2",
			Status:         runner.RunnerStatusOffline,
		}
		require.NoError(t, db.Create(r).Error)

		// Create expired reactivation token
		token := generateTestAuthKey()
		tokenHash := hashToken(token)
		reactivation := &runner.ReactivationToken{
			RunnerID:  r.ID,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		require.NoError(t, db.Create(reactivation).Error)

		req := &ReactivateRequest{Token: token}
		_, err := service.Reactivate(ctx, req, nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("successfully reactivates runner", func(t *testing.T) {
		// Setup PKI
		pkiService, tmpDir := setupTestPKI(t)
		defer os.RemoveAll(tmpDir)

		org := createTestOrg(t, db, "test-org-reactivate-success")

		// Create runner
		r := &runner.Runner{
			OrganizationID: org.ID,
			NodeID:         "test-node-reactivate",
			Status:         runner.RunnerStatusOffline,
		}
		require.NoError(t, db.Create(r).Error)

		// Create valid reactivation token
		token := generateTestAuthKey()
		tokenHash := hashToken(token)
		reactivation := &runner.ReactivationToken{
			RunnerID:  r.ID,
			TokenHash: tokenHash,
			ExpiresAt: time.Now().Add(10 * time.Minute),
		}
		require.NoError(t, db.Create(reactivation).Error)

		req := &ReactivateRequest{Token: token}
		resp, err := service.Reactivate(ctx, req, pkiService)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Certificate)
		assert.NotEmpty(t, resp.PrivateKey)
		assert.NotEmpty(t, resp.CACertificate)

		// Verify token was marked as used
		var updatedToken runner.ReactivationToken
		require.NoError(t, db.First(&updatedToken, reactivation.ID).Error)
		assert.NotNil(t, updatedToken.UsedAt)
	})
}

func TestRenewCertificate(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
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
	service := NewService(db)
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
	service := NewService(db)
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

func TestGetRunnerByNodeID(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	t.Run("returns runner by node ID", func(t *testing.T) {
		// Create runner
		r := &runner.Runner{
			OrganizationID: 1,
			NodeID:         "unique-node-id",
		}
		require.NoError(t, db.Create(r).Error)

		result, err := service.GetRunnerByNodeID(ctx, "unique-node-id")
		require.NoError(t, err)
		assert.Equal(t, r.ID, result.ID)
		assert.Equal(t, "unique-node-id", result.NodeID)
	})

	t.Run("returns error for non-existent node ID", func(t *testing.T) {
		_, err := service.GetRunnerByNodeID(ctx, "non-existent-node")
		assert.Error(t, err)
	})
}

func TestListGRPCRegistrationTokens(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	t.Run("lists tokens for organization", func(t *testing.T) {
		org := createTestOrg(t, db, "test-org-list-1")

		// Create some tokens
		for i := 0; i < 3; i++ {
			genReq := &GenerateGRPCRegistrationTokenRequest{ExpiresIn: 3600}
			_, err := service.GenerateGRPCRegistrationToken(ctx, org.ID, 1, genReq, "https://example.com")
			require.NoError(t, err)
		}

		tokens, err := service.ListGRPCRegistrationTokens(ctx, org.ID)
		require.NoError(t, err)
		assert.Len(t, tokens, 3)
	})

	t.Run("returns empty list for org with no tokens", func(t *testing.T) {
		org := createTestOrg(t, db, "test-org-list-empty")

		tokens, err := service.ListGRPCRegistrationTokens(ctx, org.ID)
		require.NoError(t, err)
		assert.Empty(t, tokens)
	})
}

func TestDeleteGRPCRegistrationToken(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	t.Run("deletes token", func(t *testing.T) {
		org := createTestOrg(t, db, "test-org-del-1")

		// Create token
		genReq := &GenerateGRPCRegistrationTokenRequest{ExpiresIn: 3600}
		_, err := service.GenerateGRPCRegistrationToken(ctx, org.ID, 1, genReq, "https://example.com")
		require.NoError(t, err)

		// Get token ID
		var token runner.GRPCRegistrationToken
		require.NoError(t, db.Where("organization_id = ?", org.ID).First(&token).Error)

		// Delete
		err = service.DeleteGRPCRegistrationToken(ctx, token.ID)
		require.NoError(t, err)

		// Verify deleted
		var count int64
		db.Model(&runner.GRPCRegistrationToken{}).Where("id = ?", token.ID).Count(&count)
		assert.Zero(t, count)
	})
}

func TestCleanupExpiredPendingAuths(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	t.Run("cleans up expired pending auths", func(t *testing.T) {
		// Create expired pending auth
		expiredAuth := &runner.PendingAuth{
			AuthKey:    generateTestAuthKey(),
			MachineKey: "expired-machine",
			ExpiresAt:  time.Now().Add(-1 * time.Hour),
		}
		require.NoError(t, db.Create(expiredAuth).Error)

		// Create valid pending auth
		validAuth := &runner.PendingAuth{
			AuthKey:    generateTestAuthKey(),
			MachineKey: "valid-machine",
			ExpiresAt:  time.Now().Add(1 * time.Hour),
		}
		require.NoError(t, db.Create(validAuth).Error)

		// Cleanup
		err := service.CleanupExpiredPendingAuths(ctx)
		require.NoError(t, err)

		// Verify expired was deleted
		var count int64
		db.Model(&runner.PendingAuth{}).Where("id = ?", expiredAuth.ID).Count(&count)
		assert.Zero(t, count)

		// Verify valid was kept
		db.Model(&runner.PendingAuth{}).Where("id = ?", validAuth.ID).Count(&count)
		assert.Equal(t, int64(1), count)
	})
}

func TestCleanupExpiredReactivationTokens(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	ctx := context.Background()

	t.Run("cleans up expired reactivation tokens", func(t *testing.T) {
		// Create runner
		r := &runner.Runner{
			OrganizationID: 1,
			NodeID:         "test-node-cleanup",
		}
		require.NoError(t, db.Create(r).Error)

		// Create expired reactivation token
		expiredToken := &runner.ReactivationToken{
			RunnerID:  r.ID,
			TokenHash: "expired-hash",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		require.NoError(t, db.Create(expiredToken).Error)

		// Create valid reactivation token
		validToken := &runner.ReactivationToken{
			RunnerID:  r.ID,
			TokenHash: "valid-hash",
			ExpiresAt: time.Now().Add(1 * time.Hour),
		}
		require.NoError(t, db.Create(validToken).Error)

		// Cleanup
		err := service.CleanupExpiredReactivationTokens(ctx)
		require.NoError(t, err)

		// Verify expired was deleted
		var count int64
		db.Model(&runner.ReactivationToken{}).Where("id = ?", expiredToken.ID).Count(&count)
		assert.Zero(t, count)

		// Verify valid was kept
		db.Model(&runner.ReactivationToken{}).Where("id = ?", validToken.ID).Count(&count)
		assert.Equal(t, int64(1), count)
	})
}

// Helper functions

func generateTestAuthKey() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
