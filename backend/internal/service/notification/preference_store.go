package notification

import (
	"context"

	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
)

// PreferenceStore provides cascading preference lookup
type PreferenceStore struct {
	repo notifDomain.PreferenceRepository
}

// NewPreferenceStore creates a new PreferenceStore
func NewPreferenceStore(repo notifDomain.PreferenceRepository) *PreferenceStore {
	return &PreferenceStore{repo: repo}
}

// GetPreference returns the effective preference using cascading lookup:
// 1. Entity-specific (userID, source, entityID)
// 2. Source-level (userID, source, nil)
// 3. Default (all enabled)
func (s *PreferenceStore) GetPreference(ctx context.Context, userID int64, source, entityID string) *notifDomain.Preference {
	// 1. Entity-specific preference
	if entityID != "" {
		if rec, err := s.repo.GetPreference(ctx, userID, source, entityID); err == nil && rec != nil {
			return &notifDomain.Preference{IsMuted: rec.IsMuted, Channels: map[string]bool(rec.Channels)}
		}
	}

	// 2. Source-level preference
	if rec, err := s.repo.GetPreference(ctx, userID, source, ""); err == nil && rec != nil {
		return &notifDomain.Preference{IsMuted: rec.IsMuted, Channels: map[string]bool(rec.Channels)}
	}

	// 3. Default
	return notifDomain.DefaultPreference()
}

// ListPreferences returns all preference records for a user
func (s *PreferenceStore) ListPreferences(ctx context.Context, userID int64) ([]notifDomain.PreferenceRecord, error) {
	return s.repo.ListPreferences(ctx, userID)
}

// SetPreference sets a notification preference for a user
func (s *PreferenceStore) SetPreference(ctx context.Context, userID int64, source, entityID string, pref *notifDomain.Preference) error {
	return s.repo.SetPreference(ctx, &notifDomain.PreferenceRecord{
		UserID:   userID,
		Source:   source,
		EntityID: entityID,
		IsMuted:  pref.IsMuted,
		Channels: notifDomain.ChannelsJSON(pref.Channels),
	})
}
