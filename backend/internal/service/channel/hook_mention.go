package channel

import (
	"context"
	"log/slog"

	channelDomain "github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

// UserLookup resolves usernames to user IDs within an organization
type UserLookup interface {
	GetUsersByUsernames(ctx context.Context, orgID int64, usernames []string) (map[string]int64, error)
	ValidateUserIDs(ctx context.Context, orgID int64, userIDs []int64) ([]int64, error)
}

// PodLookup validates pod keys within an organization
type PodLookup interface {
	GetPodsByKeys(ctx context.Context, orgID int64, podKeys []string) ([]string, error)
}

// NewMentionValidatorHook creates a hook that validates structured mention declarations.
// Unlike the old regex-based parser, this hook trusts the caller's structured data and
// only verifies that the mentioned entities actually exist in the organization.
// If validation removes invalid mentions, it syncs the metadata back to DB.
func NewMentionValidatorHook(userLookup UserLookup, podLookup PodLookup, repo channelDomain.ChannelRepository) PostSendHook {
	return func(ctx context.Context, mc *MessageContext) error {
		if mc.Mentions == nil {
			return nil
		}

		orgID := mc.Channel.OrganizationID
		changed := false

		// Validate user IDs belong to the organization
		if len(mc.Mentions.UserIDs) > 0 && userLookup != nil {
			validIDs, err := userLookup.ValidateUserIDs(ctx, orgID, mc.Mentions.UserIDs)
			if err != nil {
				slog.Error("failed to validate mentioned user IDs", "error", err)
			} else if len(validIDs) != len(mc.Mentions.UserIDs) {
				mc.Mentions.UserIDs = validIDs
				changed = true
			}
		}

		// Validate pod keys exist in the organization
		if len(mc.Mentions.PodKeys) > 0 && podLookup != nil {
			validKeys, err := podLookup.GetPodsByKeys(ctx, orgID, mc.Mentions.PodKeys)
			if err != nil {
				slog.Error("failed to validate mentioned pod keys", "error", err)
			} else if len(validKeys) != len(mc.Mentions.PodKeys) {
				mc.Mentions.PodKeys = validKeys
				changed = true
			}
		}

		// If validation pruned some mentions, sync the corrected metadata back to DB
		if changed && repo != nil {
			meta := make(map[string]interface{})
			if len(mc.Mentions.UserIDs) > 0 {
				meta[MetaMentionedUsers] = mc.Mentions.UserIDs
			}
			if len(mc.Mentions.PodKeys) > 0 {
				meta[MetaMentionedPods] = mc.Mentions.PodKeys
			}
			mc.Message.Metadata = meta
			if err := repo.UpdateMessageMetadata(ctx, mc.Message.ID, meta); err != nil {
				slog.Error("failed to update message metadata after validation", "error", err)
			}
		}

		return nil
	}
}
