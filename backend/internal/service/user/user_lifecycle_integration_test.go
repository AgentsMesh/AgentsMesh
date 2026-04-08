package user

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestService creates a user service backed by real SQLite via testkit.
func newTestService(t *testing.T) *Service {
	t.Helper()
	db := testkit.SetupTestDB(t)
	return NewService(infra.NewUserRepository(db))
}

func TestUser_CreateAndGet(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, &CreateRequest{
		Email:    "alice@example.com",
		Username: "alice",
		Name:     "Alice Smith",
		Password: "secret123",
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Greater(t, created.ID, int64(0))

	got, err := svc.GetByID(ctx, created.ID)
	require.NoError(t, err)

	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "alice@example.com", got.Email)
	assert.Equal(t, "alice", got.Username)
	assert.Equal(t, "Alice Smith", *got.Name)
	assert.True(t, got.IsActive)
	assert.NotNil(t, got.PasswordHash, "password hash should be set")
}

func TestUser_UpdateProfile(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, &CreateRequest{
		Email:    "bob@example.com",
		Username: "bob",
		Name:     "Bob",
		Password: "pass",
	})
	require.NoError(t, err)

	newName := "Robert Smith"
	avatarURL := "https://example.com/avatar.png"
	updated, err := svc.Update(ctx, created.ID, map[string]interface{}{
		"name":       newName,
		"avatar_url": avatarURL,
	})
	require.NoError(t, err)

	assert.Equal(t, "Robert Smith", *updated.Name)
	assert.Equal(t, "https://example.com/avatar.png", *updated.AvatarURL)
	// Unchanged fields
	assert.Equal(t, "bob@example.com", updated.Email)
}

func TestUser_DuplicateEmail(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, &CreateRequest{
		Email:    "same@example.com",
		Username: "user1",
		Password: "pass",
	})
	require.NoError(t, err)

	_, err = svc.Create(ctx, &CreateRequest{
		Email:    "same@example.com",
		Username: "user2",
		Password: "pass",
	})
	assert.ErrorIs(t, err, ErrEmailAlreadyExists)
}

func TestUser_DuplicateUsername(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, &CreateRequest{
		Email:    "a@example.com",
		Username: "samename",
		Password: "pass",
	})
	require.NoError(t, err)

	_, err = svc.Create(ctx, &CreateRequest{
		Email:    "b@example.com",
		Username: "samename",
		Password: "pass",
	})
	assert.ErrorIs(t, err, ErrUsernameExists)
}

func TestUser_Authenticate(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, &CreateRequest{
		Email:    "auth@example.com",
		Username: "authuser",
		Password: "correctpass",
	})
	require.NoError(t, err)

	// Correct password
	u, err := svc.Authenticate(ctx, "auth@example.com", "correctpass")
	require.NoError(t, err)
	assert.Equal(t, "auth@example.com", u.Email)

	// Wrong password
	_, err = svc.Authenticate(ctx, "auth@example.com", "wrongpass")
	assert.ErrorIs(t, err, ErrInvalidCredentials)

	// Non-existent user
	_, err = svc.Authenticate(ctx, "nope@example.com", "x")
	assert.ErrorIs(t, err, ErrInvalidCredentials)
}

func TestUser_AuthenticateInactiveUser(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, &CreateRequest{
		Email:    "inactive@example.com",
		Username: "inactive",
		Password: "pass",
	})
	require.NoError(t, err)

	// Deactivate the user
	_, err = svc.Update(ctx, created.ID, map[string]interface{}{"is_active": false})
	require.NoError(t, err)

	_, err = svc.Authenticate(ctx, "inactive@example.com", "pass")
	assert.ErrorIs(t, err, ErrUserInactive)
}

func TestUser_UpdatePassword(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, &CreateRequest{
		Email:    "pwchange@example.com",
		Username: "pwchange",
		Password: "oldpass",
	})
	require.NoError(t, err)

	err = svc.UpdatePassword(ctx, created.ID, "newpass")
	require.NoError(t, err)

	// Old password should fail
	_, err = svc.Authenticate(ctx, "pwchange@example.com", "oldpass")
	assert.ErrorIs(t, err, ErrInvalidCredentials)

	// New password should work
	u, err := svc.Authenticate(ctx, "pwchange@example.com", "newpass")
	require.NoError(t, err)
	assert.Equal(t, created.ID, u.ID)
}

func TestUser_DeleteAndVerifyGone(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	created, err := svc.Create(ctx, &CreateRequest{
		Email:    "todelete@example.com",
		Username: "todelete",
		Password: "pass",
	})
	require.NoError(t, err)

	err = svc.Delete(ctx, created.ID)
	require.NoError(t, err)

	_, err = svc.GetByID(ctx, created.ID)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUser_GetByEmail(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, &CreateRequest{
		Email: "find@example.com", Username: "find", Password: "p",
	})
	require.NoError(t, err)

	u, err := svc.GetByEmail(ctx, "find@example.com")
	require.NoError(t, err)
	assert.Equal(t, "find@example.com", u.Email)

	_, err = svc.GetByEmail(ctx, "missing@example.com")
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestUser_GetByUsername(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.Create(ctx, &CreateRequest{
		Email: "u@example.com", Username: "findme", Password: "p",
	})
	require.NoError(t, err)

	u, err := svc.GetByUsername(ctx, "findme")
	require.NoError(t, err)
	assert.Equal(t, "findme", u.Username)

	_, err = svc.GetByUsername(ctx, "nope")
	assert.ErrorIs(t, err, ErrUserNotFound)
}
