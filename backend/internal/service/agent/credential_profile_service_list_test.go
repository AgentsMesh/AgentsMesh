package agent

import (
	"context"
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCredentialProfileService_ListCredentialProfiles(t *testing.T) {
	db := setupTestDB(t)
	agentSvc := newTestAgentService(db)
	svc := newTestCredentialProfileService(db, agentSvc, testEncryptor())
	ctx := context.Background()

	var agents []agent.Agent
	db.Find(&agents)
	require.True(t, len(agents) >= 2)

	userID := int64(100)

	// Create profiles for multiple agents
	for i, at := range agents {
		for j := 0; j < 2; j++ {
			params := &CreateCredentialProfileParams{
				AgentSlug: at.Slug,
				Name:        at.Name + " Profile " + string(rune('A'+j)),
				IsDefault:   j == 0,
			}
			_, err := svc.CreateCredentialProfile(ctx, userID, params)
			require.NoError(t, err, "failed to create profile %d-%d", i, j)
		}
	}

	// Create for another user
	params := &CreateCredentialProfileParams{
		AgentSlug: agents[0].Slug,
		Name:        "Other User Profile",
	}
	_, err := svc.CreateCredentialProfile(ctx, int64(200), params)
	require.NoError(t, err)

	t.Run("lists profiles grouped by agent", func(t *testing.T) {
		groups, err := svc.ListCredentialProfiles(ctx, userID)
		require.NoError(t, err)

		// Should have groups for each agent
		assert.Len(t, groups, len(agents))

		// Each group should have 2 profiles
		for _, group := range groups {
			assert.Len(t, group.Profiles, 2)
			assert.NotEmpty(t, group.AgentName)
			assert.NotEmpty(t, group.AgentSlug)
		}
	})

	t.Run("does not include other users profiles", func(t *testing.T) {
		groups, err := svc.ListCredentialProfiles(ctx, int64(200))
		require.NoError(t, err)

		totalProfiles := 0
		for _, group := range groups {
			totalProfiles += len(group.Profiles)
		}
		assert.Equal(t, 1, totalProfiles)
	})
}

func TestCredentialProfileService_ListCredentialProfilesForAgent(t *testing.T) {
	db := setupTestDB(t)
	agentSvc := newTestAgentService(db)
	svc := newTestCredentialProfileService(db, agentSvc, testEncryptor())
	ctx := context.Background()

	var at agent.Agent
	db.First(&at)
	userID := int64(1)

	// Create profiles
	for i := 0; i < 3; i++ {
		params := &CreateCredentialProfileParams{
			AgentSlug: at.Slug,
			Name:        "Profile " + string(rune('A'+i)),
			IsDefault:   i == 0,
		}
		_, err := svc.CreateCredentialProfile(ctx, userID, params)
		require.NoError(t, err)
	}

	t.Run("lists all profiles for agent", func(t *testing.T) {
		profiles, err := svc.ListCredentialProfilesForAgent(ctx, userID, at.Slug)
		require.NoError(t, err)
		assert.Len(t, profiles, 3)

		// First should be the default (sorted by is_default DESC, name)
		assert.True(t, profiles[0].IsDefault)
	})

	t.Run("returns empty for non-existent agent", func(t *testing.T) {
		profiles, err := svc.ListCredentialProfilesForAgent(ctx, userID, "nonexistent")
		require.NoError(t, err)
		assert.Empty(t, profiles)
	})
}

func TestCredentialProfileService_GetDefaultCredentialProfile(t *testing.T) {
	db := setupTestDB(t)
	agentSvc := newTestAgentService(db)
	svc := newTestCredentialProfileService(db, agentSvc, testEncryptor())
	ctx := context.Background()

	var at agent.Agent
	db.First(&at)
	userID := int64(1)

	t.Run("returns default profile", func(t *testing.T) {
		// Create default profile
		params := &CreateCredentialProfileParams{
			AgentSlug: at.Slug,
			Name:        "Default Profile",
			IsDefault:   true,
		}
		created, err := svc.CreateCredentialProfile(ctx, userID, params)
		require.NoError(t, err)

		// Get default
		profile, err := svc.GetDefaultCredentialProfile(ctx, userID, at.Slug)
		require.NoError(t, err)
		assert.Equal(t, created.ID, profile.ID)
		assert.True(t, profile.IsDefault)
	})

	t.Run("returns error when no default", func(t *testing.T) {
		_, err := svc.GetDefaultCredentialProfile(ctx, int64(999), at.Slug)
		assert.ErrorIs(t, err, ErrCredentialProfileNotFound)
	})
}

func TestCredentialProfileService_SetDefaultCredentialProfile(t *testing.T) {
	db := setupTestDB(t)
	agentSvc := newTestAgentService(db)
	svc := newTestCredentialProfileService(db, agentSvc, testEncryptor())
	ctx := context.Background()

	var at agent.Agent
	db.First(&at)
	userID := int64(1)

	t.Run("sets profile as default", func(t *testing.T) {
		// Create non-default profiles
		params1 := &CreateCredentialProfileParams{
			AgentSlug: at.Slug,
			Name:        "Profile 1",
			IsDefault:   false,
		}
		profile1, err := svc.CreateCredentialProfile(ctx, userID, params1)
		require.NoError(t, err)

		params2 := &CreateCredentialProfileParams{
			AgentSlug: at.Slug,
			Name:        "Profile 2",
			IsDefault:   false,
		}
		profile2, err := svc.CreateCredentialProfile(ctx, userID, params2)
		require.NoError(t, err)

		// Set profile2 as default
		updated, err := svc.SetDefaultCredentialProfile(ctx, userID, profile2.ID)
		require.NoError(t, err)
		assert.True(t, updated.IsDefault)

		// Verify profile1 is still not default
		profile1Updated, err := svc.GetCredentialProfile(ctx, userID, profile1.ID)
		require.NoError(t, err)
		assert.False(t, profile1Updated.IsDefault)
	})

	t.Run("non-existent profile returns error", func(t *testing.T) {
		_, err := svc.SetDefaultCredentialProfile(ctx, userID, 99999)
		assert.ErrorIs(t, err, ErrCredentialProfileNotFound)
	})
}

func TestCredentialProfileService_GetEffectiveCredentialsForPod(t *testing.T) {
	db := setupTestDB(t)
	agentSvc := newTestAgentService(db)
	svc := newTestCredentialProfileService(db, agentSvc, testEncryptor())
	ctx := context.Background()

	var at agent.Agent
	db.First(&at)
	userID := int64(1)

	t.Run("uses specified profile", func(t *testing.T) {
		params := &CreateCredentialProfileParams{
			AgentSlug:  at.Slug,
			Name:         "Specific Profile",
			IsRunnerHost: false,
			Credentials:  map[string]string{"api_key": "specific-key"},
			IsDefault:    false,
		}
		profile, err := svc.CreateCredentialProfile(ctx, userID, params)
		require.NoError(t, err)

		creds, isRunnerHost, err := svc.GetEffectiveCredentialsForPod(ctx, userID, at.Slug, &profile.ID)
		require.NoError(t, err)
		assert.False(t, isRunnerHost)
		assert.Equal(t, "specific-key", creds["api_key"])
	})

	t.Run("uses default profile when no profile specified", func(t *testing.T) {
		// Create default profile for user 2
		params := &CreateCredentialProfileParams{
			AgentSlug: at.Slug,
			Name:        "Default for User 2",
			IsDefault:   true,
			Credentials: map[string]string{"api_key": "default-key"},
		}
		_, err := svc.CreateCredentialProfile(ctx, int64(2), params)
		require.NoError(t, err)

		creds, isRunnerHost, err := svc.GetEffectiveCredentialsForPod(ctx, int64(2), at.Slug, nil)
		require.NoError(t, err)
		assert.False(t, isRunnerHost)
		assert.Equal(t, "default-key", creds["api_key"])
	})

	t.Run("returns runner host when no default profile", func(t *testing.T) {
		// User 999 has no profiles
		creds, isRunnerHost, err := svc.GetEffectiveCredentialsForPod(ctx, int64(999), at.Slug, nil)
		require.NoError(t, err)
		assert.True(t, isRunnerHost)
		assert.Nil(t, creds)
	})

	t.Run("returns runner host for runner host profile", func(t *testing.T) {
		params := &CreateCredentialProfileParams{
			AgentSlug:  at.Slug,
			Name:         "Runner Host Profile",
			IsRunnerHost: true,
			IsDefault:    true,
		}
		profile, err := svc.CreateCredentialProfile(ctx, int64(3), params)
		require.NoError(t, err)

		creds, isRunnerHost, err := svc.GetEffectiveCredentialsForPod(ctx, int64(3), at.Slug, &profile.ID)
		require.NoError(t, err)
		assert.True(t, isRunnerHost)
		assert.Nil(t, creds)
	})

	t.Run("returns error for non-existent profile", func(t *testing.T) {
		badID := int64(99999)
		_, _, err := svc.GetEffectiveCredentialsForPod(ctx, userID, at.Slug, &badID)
		assert.ErrorIs(t, err, ErrCredentialProfileNotFound)
	})

	t.Run("explicit RunnerHost (profileID=0) bypasses default profile", func(t *testing.T) {
		// Create a default profile with credentials for user 50
		params := &CreateCredentialProfileParams{
			AgentSlug: at.Slug,
			Name:        "Default for User 50",
			IsDefault:   true,
			Credentials: map[string]string{"api_key": "should-not-be-used"},
		}
		_, err := svc.CreateCredentialProfile(ctx, int64(50), params)
		require.NoError(t, err)

		// Explicit RunnerHost: profileID = 0 should return RunnerHost even with default present
		zero := int64(0)
		creds, isRunnerHost, err := svc.GetEffectiveCredentialsForPod(ctx, int64(50), at.Slug, &zero)
		require.NoError(t, err)
		assert.True(t, isRunnerHost, "profileID=0 should always return RunnerHost")
		assert.Nil(t, creds, "no credentials should be returned for explicit RunnerHost")
	})

	t.Run("nil profileID uses default profile when available", func(t *testing.T) {
		// User 50 already has a default profile from previous test
		creds, isRunnerHost, err := svc.GetEffectiveCredentialsForPod(ctx, int64(50), at.Slug, nil)
		require.NoError(t, err)
		assert.False(t, isRunnerHost, "nil profileID should use default profile")
		assert.Equal(t, "should-not-be-used", creds["api_key"])
	})

	t.Run("nil profileID falls back to RunnerHost when no default", func(t *testing.T) {
		// User 888 has no profiles at all
		creds, isRunnerHost, err := svc.GetEffectiveCredentialsForPod(ctx, int64(888), at.Slug, nil)
		require.NoError(t, err)
		assert.True(t, isRunnerHost, "nil profileID without default should fallback to RunnerHost")
		assert.Nil(t, creds)
	})
}
