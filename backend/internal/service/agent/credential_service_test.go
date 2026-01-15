package agent

import (
	"context"
	"testing"

	"github.com/anthropics/agentmesh/backend/internal/domain/agent"
	"github.com/anthropics/agentmesh/backend/pkg/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupCredentialTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	require.NoError(t, err)

	// Create tables
	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS user_agent_credentials (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			agent_type_id INTEGER NOT NULL,
			credentials_encrypted TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, agent_type_id)
		)
	`).Error
	require.NoError(t, err)


	err = db.Exec(`
		CREATE TABLE IF NOT EXISTS agent_types (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			slug TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL,
			description TEXT,
			launch_command TEXT,
			executable TEXT,
			default_args TEXT,
			config_schema TEXT,
			command_template TEXT,
			files_template TEXT,
			is_builtin INTEGER NOT NULL DEFAULT 0,
			is_active INTEGER NOT NULL DEFAULT 1,
			credential_schema TEXT,
			status_detection TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
	require.NoError(t, err)

	return db
}

func TestNewCredentialService(t *testing.T) {
	db := setupCredentialTestDB(t)
	encryptor := crypto.NewEncryptor("test-key-32-bytes-long-1234567!")

	svc := NewCredentialService(db, encryptor)
	assert.NotNil(t, svc)
	assert.Equal(t, db, svc.db)
	assert.Equal(t, encryptor, svc.encryptor)
}

func TestSetAndGetUserCredentials(t *testing.T) {
	ctx := context.Background()
	db := setupCredentialTestDB(t)
	encryptor := crypto.NewEncryptor("test-key-32-bytes-long-1234567!")
	svc := NewCredentialService(db, encryptor)

	t.Run("set and get user credentials", func(t *testing.T) {
		creds := map[string]string{
			"api_key":   "sk-test-key-123",
			"auth_token": "auth-token-456",
		}

		err := svc.SetUserCredentials(ctx, 1, 1, creds)
		require.NoError(t, err)

		retrieved, err := svc.GetUserCredentials(ctx, 1, 1)
		require.NoError(t, err)
		assert.Equal(t, creds["api_key"], retrieved["api_key"])
		assert.Equal(t, creds["auth_token"], retrieved["auth_token"])
	})

	t.Run("update existing credentials", func(t *testing.T) {
		creds := map[string]string{
			"api_key": "new-key-789",
		}

		err := svc.SetUserCredentials(ctx, 1, 1, creds)
		require.NoError(t, err)

		retrieved, err := svc.GetUserCredentials(ctx, 1, 1)
		require.NoError(t, err)
		assert.Equal(t, "new-key-789", retrieved["api_key"])
	})

	t.Run("get non-existent credentials", func(t *testing.T) {
		_, err := svc.GetUserCredentials(ctx, 999, 1)
		assert.Error(t, err)
		assert.Equal(t, ErrCredentialsNotFound, err)
	})
}


func TestDeleteUserCredentials(t *testing.T) {
	ctx := context.Background()
	db := setupCredentialTestDB(t)
	encryptor := crypto.NewEncryptor("test-key-32-bytes-long-1234567!")
	svc := NewCredentialService(db, encryptor)

	// Create credentials first
	err := svc.SetUserCredentials(ctx, 1, 1, map[string]string{"api_key": "test"})
	require.NoError(t, err)

	// Delete
	err = svc.DeleteUserCredentials(ctx, 1, 1)
	require.NoError(t, err)

	// Verify deleted
	_, err = svc.GetUserCredentials(ctx, 1, 1)
	assert.Error(t, err)
	assert.Equal(t, ErrCredentialsNotFound, err)
}


func TestEncryptionWithoutEncryptor(t *testing.T) {
	ctx := context.Background()
	db := setupCredentialTestDB(t)
	// No encryptor
	svc := NewCredentialService(db, nil)

	t.Run("stores and retrieves plain credentials", func(t *testing.T) {
		creds := map[string]string{
			"api_key": "plain-key",
		}

		err := svc.SetUserCredentials(ctx, 1, 1, creds)
		require.NoError(t, err)

		retrieved, err := svc.GetUserCredentials(ctx, 1, 1)
		require.NoError(t, err)
		assert.Equal(t, "plain-key", retrieved["api_key"])
	})
}

func TestDecryptCredentialsMap(t *testing.T) {
	db := setupCredentialTestDB(t)
	encryptor := crypto.NewEncryptor("test-key-32-bytes-long-1234567!")
	svc := NewCredentialService(db, encryptor)

	t.Run("returns error for nil credentials", func(t *testing.T) {
		_, err := svc.decryptCredentialsMap(nil)
		assert.Error(t, err)
		assert.Equal(t, ErrCredentialsNotFound, err)
	})

	t.Run("handles unencrypted values gracefully", func(t *testing.T) {
		// Values that aren't encrypted should still work
		creds := agent.EncryptedCredentials{
			"plain_key": "plain_value",
		}

		result, err := svc.decryptCredentialsMap(creds)
		require.NoError(t, err)
		assert.Equal(t, "plain_value", result["plain_key"])
	})
}

func TestErrorConstants(t *testing.T) {
	assert.Equal(t, "credentials not found", ErrCredentialsNotFound.Error())
	assert.Equal(t, "failed to decrypt credentials", ErrDecryptionFailed.Error())
}
