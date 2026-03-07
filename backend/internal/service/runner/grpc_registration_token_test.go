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
		err = service.DeleteGRPCRegistrationToken(ctx, token.ID, org.ID)
		require.NoError(t, err)

		// Verify deleted
		var count int64
		db.Model(&runner.GRPCRegistrationToken{}).Where("id = ?", token.ID).Count(&count)
		assert.Zero(t, count)
	})
}
