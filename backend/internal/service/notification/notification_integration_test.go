package notification

import (
	"context"
	"testing"

	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
	"github.com/anthropics/agentsmesh/backend/internal/infra"
	"github.com/anthropics/agentsmesh/backend/internal/testkit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupPreferenceStore(t *testing.T) (*PreferenceStore, context.Context) {
	t.Helper()
	db := testkit.SetupTestDB(t)
	repo := infra.NewNotificationPreferenceRepository(db)
	store := NewPreferenceStore(repo)
	return store, context.Background()
}

func TestNotification_SetAndGetPreference(t *testing.T) {
	store, ctx := setupPreferenceStore(t)

	userID := int64(42)
	source := notifDomain.SourceChannelMessage
	entityID := "chan-100"

	// Set a preference
	pref := &notifDomain.Preference{
		IsMuted:  false,
		Channels: map[string]bool{"toast": true, "browser": false},
	}
	err := store.SetPreference(ctx, userID, source, entityID, pref)
	require.NoError(t, err)

	// Get it back
	got := store.GetPreference(ctx, userID, source, entityID)
	require.NotNil(t, got)
	assert.False(t, got.IsMuted)
	assert.True(t, got.Channels["toast"])
	assert.False(t, got.Channels["browser"])
}

func TestNotification_MuteAndUnmute(t *testing.T) {
	store, ctx := setupPreferenceStore(t)

	userID := int64(99)
	source := notifDomain.SourceTerminalOSC

	// Mute the source (source-level, no entity)
	mutedPref := &notifDomain.Preference{
		IsMuted:  true,
		Channels: map[string]bool{"toast": true, "browser": true},
	}
	err := store.SetPreference(ctx, userID, source, "", mutedPref)
	require.NoError(t, err)

	// Verify muted
	got := store.GetPreference(ctx, userID, source, "")
	require.NotNil(t, got)
	assert.True(t, got.IsMuted)

	// Unmute
	unmutedPref := &notifDomain.Preference{
		IsMuted:  false,
		Channels: map[string]bool{"toast": true, "browser": true},
	}
	err = store.SetPreference(ctx, userID, source, "", unmutedPref)
	require.NoError(t, err)

	// Verify unmuted
	got = store.GetPreference(ctx, userID, source, "")
	require.NotNil(t, got)
	assert.False(t, got.IsMuted)
}
