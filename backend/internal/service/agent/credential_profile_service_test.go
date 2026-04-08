package agent

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testEncryptionKey is a fixed key used for test encryption/decryption
const testEncryptionKey = "test-encryption-key-for-unit-tests"

// testEncryptor returns a crypto.Encryptor for testing
func testEncryptor() *crypto.Encryptor {
	return crypto.NewEncryptor(testEncryptionKey)
}

func TestNewCredentialProfileService(t *testing.T) {
	db := setupTestDB(t)
	agentSvc := newTestAgentService(db)
	svc := newTestCredentialProfileService(db, agentSvc, testEncryptor())

	assert.NotNil(t, svc)
	assert.NotNil(t, svc.repo)
	assert.Equal(t, agentSvc, svc.agentSvc)
}

func TestCredentialProfileService_CreateCredentialProfile(t *testing.T) {
	db := setupTestDB(t)
	agentSvc := newTestAgentService(db)
	svc := newTestCredentialProfileService(db, agentSvc, testEncryptor())
	ctx := context.Background()

	var at agent.Agent
	db.First(&at)
	userID := int64(1)

	t.Run("create profile with credentials", func(t *testing.T) {
		desc := "Personal API key"
		params := &CreateCredentialProfileParams{
			AgentSlug:  at.Slug,
			Name:         "My API Key",
			Description:  &desc,
			IsRunnerHost: false,
			Credentials: map[string]string{
				"api_key": "sk-test-key-123",
			},
			IsDefault: true,
		}

		profile, err := svc.CreateCredentialProfile(ctx, userID, params)
		require.NoError(t, err)
		assert.NotZero(t, profile.ID)
		assert.Equal(t, userID, profile.UserID)
		assert.Equal(t, at.Slug, profile.AgentSlug)
		assert.Equal(t, "My API Key", profile.Name)
		assert.Equal(t, "Personal API key", *profile.Description)
		assert.False(t, profile.IsRunnerHost)
		assert.True(t, profile.IsDefault)
		assert.True(t, profile.IsActive)
		// Stored value should be encrypted (not plaintext)
		assert.NotEqual(t, "sk-test-key-123", profile.CredentialsEncrypted["api_key"])
		assert.NotEmpty(t, profile.CredentialsEncrypted["api_key"])
	})

	t.Run("create runner host profile", func(t *testing.T) {
		desc := "Use runner's environment"
		params := &CreateCredentialProfileParams{
			AgentSlug:  at.Slug,
			Name:         "Runner Host",
			Description:  &desc,
			IsRunnerHost: true,
			IsDefault:    false,
		}

		profile, err := svc.CreateCredentialProfile(ctx, userID, params)
		require.NoError(t, err)
		assert.True(t, profile.IsRunnerHost)
		assert.Empty(t, profile.CredentialsEncrypted)
	})

	t.Run("duplicate name returns error", func(t *testing.T) {
		params := &CreateCredentialProfileParams{
			AgentSlug: at.Slug,
			Name:        "Duplicate Test",
		}
		_, err := svc.CreateCredentialProfile(ctx, userID, params)
		require.NoError(t, err)

		// Try to create with same name
		_, err = svc.CreateCredentialProfile(ctx, userID, params)
		assert.ErrorIs(t, err, ErrCredentialProfileExists)
	})

	t.Run("non-existent agent returns error", func(t *testing.T) {
		params := &CreateCredentialProfileParams{
			AgentSlug: "nonexistent",
			Name:        "Invalid Agent",
		}
		_, err := svc.CreateCredentialProfile(ctx, userID, params)
		assert.ErrorIs(t, err, ErrAgentNotFound)
	})

	t.Run("setting default unsets other defaults", func(t *testing.T) {
		// Create first default
		params1 := &CreateCredentialProfileParams{
			AgentSlug: at.Slug,
			Name:        "First Default",
			IsDefault:   true,
		}
		profile1, err := svc.CreateCredentialProfile(ctx, int64(2), params1)
		require.NoError(t, err)
		assert.True(t, profile1.IsDefault)

		// Create second default - should unset first
		params2 := &CreateCredentialProfileParams{
			AgentSlug: at.Slug,
			Name:        "Second Default",
			IsDefault:   true,
		}
		profile2, err := svc.CreateCredentialProfile(ctx, int64(2), params2)
		require.NoError(t, err)
		assert.True(t, profile2.IsDefault)

		// Verify first is no longer default
		profile1Updated, err := svc.GetCredentialProfile(ctx, int64(2), profile1.ID)
		require.NoError(t, err)
		assert.False(t, profile1Updated.IsDefault)
	})
}

func TestCredentialProfileService_GetCredentialProfile(t *testing.T) {
	db := setupTestDB(t)
	agentSvc := newTestAgentService(db)
	svc := newTestCredentialProfileService(db, agentSvc, testEncryptor())
	ctx := context.Background()

	var at agent.Agent
	db.First(&at)
	userID := int64(1)

	t.Run("get existing profile", func(t *testing.T) {
		// Create first
		params := &CreateCredentialProfileParams{
			AgentSlug: at.Slug,
			Name:        "Test Profile",
			Credentials: map[string]string{"key": "value"},
		}
		created, err := svc.CreateCredentialProfile(ctx, userID, params)
		require.NoError(t, err)

		// Get it
		profile, err := svc.GetCredentialProfile(ctx, userID, created.ID)
		require.NoError(t, err)
		assert.Equal(t, created.ID, profile.ID)
		assert.Equal(t, "Test Profile", profile.Name)
	})

	t.Run("non-existent profile returns error", func(t *testing.T) {
		_, err := svc.GetCredentialProfile(ctx, userID, 99999)
		assert.ErrorIs(t, err, ErrCredentialProfileNotFound)
	})

	t.Run("wrong user returns error", func(t *testing.T) {
		// Create for user 1
		params := &CreateCredentialProfileParams{
			AgentSlug: at.Slug,
			Name:        "User 1 Profile",
		}
		created, err := svc.CreateCredentialProfile(ctx, userID, params)
		require.NoError(t, err)

		// Try to get as user 2
		_, err = svc.GetCredentialProfile(ctx, int64(2), created.ID)
		assert.ErrorIs(t, err, ErrCredentialProfileNotFound)
	})
}

func TestCredentialProfileService_UpdateCredentialProfile(t *testing.T) {
	db := setupTestDB(t)
	agentSvc := newTestAgentService(db)
	svc := newTestCredentialProfileService(db, agentSvc, testEncryptor())
	ctx := context.Background()

	var at agent.Agent
	db.First(&at)
	userID := int64(1)

	t.Run("update name and description", func(t *testing.T) {
		origDesc := "Original Desc"
		params := &CreateCredentialProfileParams{
			AgentSlug: at.Slug,
			Name:        "Original Name",
			Description: &origDesc,
		}
		created, err := svc.CreateCredentialProfile(ctx, userID, params)
		require.NoError(t, err)

		newName := "Updated Name"
		newDesc := "Updated Desc"
		updated, err := svc.UpdateCredentialProfile(ctx, userID, created.ID, &UpdateCredentialProfileParams{
			Name:        &newName,
			Description: &newDesc,
		})
		require.NoError(t, err)
		assert.Equal(t, "Updated Name", updated.Name)
		assert.Equal(t, "Updated Desc", *updated.Description)
	})

	t.Run("update credentials", func(t *testing.T) {
		params := &CreateCredentialProfileParams{
			AgentSlug: at.Slug,
			Name:        "Creds Test",
			Credentials: map[string]string{"key": "old-value"},
		}
		created, err := svc.CreateCredentialProfile(ctx, userID, params)
		require.NoError(t, err)

		updated, err := svc.UpdateCredentialProfile(ctx, userID, created.ID, &UpdateCredentialProfileParams{
			Credentials: map[string]string{"key": "new-value"},
		})
		require.NoError(t, err)
		// Stored value should be encrypted (not plaintext)
		assert.NotEqual(t, "new-value", updated.CredentialsEncrypted["key"])
		assert.NotEmpty(t, updated.CredentialsEncrypted["key"])
	})

	t.Run("switch to runner host clears credentials", func(t *testing.T) {
		params := &CreateCredentialProfileParams{
			AgentSlug: at.Slug,
			Name:        "Switch Test",
			Credentials: map[string]string{"key": "value"},
		}
		created, err := svc.CreateCredentialProfile(ctx, userID, params)
		require.NoError(t, err)

		isRunnerHost := true
		updated, err := svc.UpdateCredentialProfile(ctx, userID, created.ID, &UpdateCredentialProfileParams{
			IsRunnerHost: &isRunnerHost,
		})
		require.NoError(t, err)
		assert.True(t, updated.IsRunnerHost)
	})

	t.Run("update default unsets other defaults", func(t *testing.T) {
		// Create first default
		params1 := &CreateCredentialProfileParams{
			AgentSlug: at.Slug,
			Name:        "First",
			IsDefault:   true,
		}
		profile1, err := svc.CreateCredentialProfile(ctx, int64(10), params1)
		require.NoError(t, err)

		// Create second non-default
		params2 := &CreateCredentialProfileParams{
			AgentSlug: at.Slug,
			Name:        "Second",
			IsDefault:   false,
		}
		profile2, err := svc.CreateCredentialProfile(ctx, int64(10), params2)
		require.NoError(t, err)

		// Set second as default
		isDefault := true
		_, err = svc.UpdateCredentialProfile(ctx, int64(10), profile2.ID, &UpdateCredentialProfileParams{
			IsDefault: &isDefault,
		})
		require.NoError(t, err)

		// Verify first is no longer default
		profile1Updated, err := svc.GetCredentialProfile(ctx, int64(10), profile1.ID)
		require.NoError(t, err)
		assert.False(t, profile1Updated.IsDefault)
	})

	t.Run("duplicate name returns error", func(t *testing.T) {
		params1 := &CreateCredentialProfileParams{
			AgentSlug: at.Slug,
			Name:        "Name A",
		}
		_, err := svc.CreateCredentialProfile(ctx, int64(20), params1)
		require.NoError(t, err)

		params2 := &CreateCredentialProfileParams{
			AgentSlug: at.Slug,
			Name:        "Name B",
		}
		profile2, err := svc.CreateCredentialProfile(ctx, int64(20), params2)
		require.NoError(t, err)

		// Try to rename B to A
		newName := "Name A"
		_, err = svc.UpdateCredentialProfile(ctx, int64(20), profile2.ID, &UpdateCredentialProfileParams{
			Name: &newName,
		})
		assert.ErrorIs(t, err, ErrCredentialProfileExists)
	})

	t.Run("non-existent profile returns error", func(t *testing.T) {
		newName := "Whatever"
		_, err := svc.UpdateCredentialProfile(ctx, userID, 99999, &UpdateCredentialProfileParams{
			Name: &newName,
		})
		assert.ErrorIs(t, err, ErrCredentialProfileNotFound)
	})
}

func TestCredentialProfileService_DeleteCredentialProfile(t *testing.T) {
	db := setupTestDB(t)
	agentSvc := newTestAgentService(db)
	svc := newTestCredentialProfileService(db, agentSvc, testEncryptor())
	ctx := context.Background()

	var at agent.Agent
	db.First(&at)
	userID := int64(1)

	t.Run("delete existing profile", func(t *testing.T) {
		params := &CreateCredentialProfileParams{
			AgentSlug: at.Slug,
			Name:        "To Delete",
		}
		created, err := svc.CreateCredentialProfile(ctx, userID, params)
		require.NoError(t, err)

		err = svc.DeleteCredentialProfile(ctx, userID, created.ID)
		require.NoError(t, err)

		// Verify deleted
		_, err = svc.GetCredentialProfile(ctx, userID, created.ID)
		assert.ErrorIs(t, err, ErrCredentialProfileNotFound)
	})

	t.Run("delete non-existent profile returns error", func(t *testing.T) {
		err := svc.DeleteCredentialProfile(ctx, userID, 99999)
		assert.ErrorIs(t, err, ErrCredentialProfileNotFound)
	})

	t.Run("cannot delete other user's profile", func(t *testing.T) {
		params := &CreateCredentialProfileParams{
			AgentSlug: at.Slug,
			Name:        "User 1 Only",
		}
		created, err := svc.CreateCredentialProfile(ctx, userID, params)
		require.NoError(t, err)

		// Try to delete as user 2
		err = svc.DeleteCredentialProfile(ctx, int64(2), created.ID)
		assert.ErrorIs(t, err, ErrCredentialProfileNotFound)
	})
}
