package notification

import (
	"context"
	"testing"

	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
)

func TestPreferenceStore_CascadingLookup(t *testing.T) {
	repo := newMockPrefRepo()
	store := NewPreferenceStore(repo)
	ctx := context.Background()

	// No preferences set → default (all enabled)
	pref := store.GetPreference(ctx, 1, "channel:message", "42")
	if pref.IsMuted || !pref.Channels["toast"] || !pref.Channels["browser"] {
		t.Errorf("Default pref should be unmuted with toast+browser enabled, got %+v", pref)
	}

	// Set source-level preference
	store.SetPreference(ctx, 1, "channel:message", "", &notifDomain.Preference{
		IsMuted:  false,
		Channels: map[string]bool{"toast": false, "browser": true},
	})
	pref = store.GetPreference(ctx, 1, "channel:message", "42")
	if pref.Channels["toast"] {
		t.Errorf("Source-level pref should disable toast")
	}
	if !pref.Channels["browser"] {
		t.Errorf("Source-level pref should enable browser")
	}

	// Set entity-specific preference (overrides source-level)
	store.SetPreference(ctx, 1, "channel:message", "42", &notifDomain.Preference{
		IsMuted:  true,
		Channels: map[string]bool{"toast": true, "browser": true},
	})
	pref = store.GetPreference(ctx, 1, "channel:message", "42")
	if !pref.IsMuted {
		t.Errorf("Entity-specific pref should be muted")
	}

	// Different entity falls back to source-level
	pref = store.GetPreference(ctx, 1, "channel:message", "99")
	if pref.IsMuted {
		t.Errorf("Different entity should use source-level (not muted)")
	}
	if pref.Channels["toast"] {
		t.Errorf("Different entity should use source-level (toast=false)")
	}
}

func TestPreferenceStore_ListPreferences(t *testing.T) {
	repo := newMockPrefRepo()
	store := NewPreferenceStore(repo)
	ctx := context.Background()

	store.SetPreference(ctx, 1, "channel:message", "", &notifDomain.Preference{
		Channels: map[string]bool{"toast": true, "browser": false},
	})
	store.SetPreference(ctx, 1, "terminal:osc", "", &notifDomain.Preference{
		IsMuted:  true,
		Channels: map[string]bool{"toast": true, "browser": true},
	})

	prefs, err := store.ListPreferences(ctx, 1)
	if err != nil {
		t.Fatalf("ListPreferences failed: %v", err)
	}
	if len(prefs) != 2 {
		t.Errorf("Expected 2 preferences, got %d", len(prefs))
	}
}
