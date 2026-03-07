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

func TestRequestAuthURL(t *testing.T) {
	db := setupTestDB(t)
	service := newTestService(db)
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
	service := newTestService(db)
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
	service := newTestService(db)
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
		resp, err := service.AuthorizeRunner(ctx, authKey, org.ID, 1, "my-runner")
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

		_, err := service.AuthorizeRunner(ctx, "non-existent", org.ID, 1, "")
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

		_, err := service.AuthorizeRunner(ctx, authKey, org.ID, 1, "")
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

		_, err := service.AuthorizeRunner(ctx, authKey, org.ID, 1, "")
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
		r, err := service.AuthorizeRunner(ctx, authKey, org.ID, 1, "")
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
		r, err := service.AuthorizeRunner(ctx, authKey, org.ID, 1, "")
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
		_, err := service.AuthorizeRunner(ctx, authKey, org.ID, 1, "duplicate-node")
		assert.Error(t, err)
	})
}
