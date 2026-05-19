package notification

import (
	"context"

	notifDomain "github.com/anthropics/agentsmesh/backend/internal/domain/notification"
)

type PreferenceStore struct {
	repo notifDomain.PreferenceRepository
}

func NewPreferenceStore(repo notifDomain.PreferenceRepository) *PreferenceStore {
	return &PreferenceStore{repo: repo}
}

func (s *PreferenceStore) GetPreference(ctx context.Context, userID int64, source, entityID string) *notifDomain.Preference {
	if entityID != "" {
		if rec, err := s.repo.GetPreference(ctx, userID, source, entityID); err == nil && rec != nil {
			return &notifDomain.Preference{IsMuted: rec.IsMuted, Channels: map[string]bool(rec.Channels)}
		}
	}

	if rec, err := s.repo.GetPreference(ctx, userID, source, ""); err == nil && rec != nil {
		return &notifDomain.Preference{IsMuted: rec.IsMuted, Channels: map[string]bool(rec.Channels)}
	}

	return notifDomain.DefaultPreference()
}

func (s *PreferenceStore) ListPreferences(ctx context.Context, userID int64) ([]notifDomain.PreferenceRecord, error) {
	return s.repo.ListPreferences(ctx, userID)
}

func (s *PreferenceStore) SetPreference(ctx context.Context, userID int64, source, entityID string, pref *notifDomain.Preference) error {
	return s.repo.SetPreference(ctx, &notifDomain.PreferenceRecord{
		UserID:   userID,
		Source:   source,
		EntityID: entityID,
		IsMuted:  pref.IsMuted,
		Channels: notifDomain.ChannelsJSON(pref.Channels),
	})
}
