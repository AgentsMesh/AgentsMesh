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

// NewMentionValidatorHook validates that mentioned entities exist in the org.
// Invalid mentions are pruned from the MentionResult (affecting notification routing)
// and synced back to the persisted mentions field.
func NewMentionValidatorHook(userLookup UserLookup, podLookup PodLookup, repo channelDomain.ChannelRepository) PostSendHook {
	return func(ctx context.Context, mc *MessageContext) error {
		if mc.Mentions == nil {
			return nil
		}

		orgID := mc.Channel.OrganizationID
		changed := false

		if len(mc.Mentions.UserIDs) > 0 && userLookup != nil {
			validIDs, err := userLookup.ValidateUserIDs(ctx, orgID, mc.Mentions.UserIDs)
			if err != nil {
				slog.Error("failed to validate mentioned user IDs", "error", err)
			} else if len(validIDs) != len(mc.Mentions.UserIDs) {
				mc.Mentions.UserIDs = validIDs
				changed = true
			}
		}

		if len(mc.Mentions.PodKeys) > 0 && podLookup != nil {
			validKeys, err := podLookup.GetPodsByKeys(ctx, orgID, mc.Mentions.PodKeys)
			if err != nil {
				slog.Error("failed to validate mentioned pod keys", "error", err)
			} else if len(validKeys) != len(mc.Mentions.PodKeys) {
				mc.Mentions.PodKeys = validKeys
				changed = true
			}
		}

		if changed && repo != nil {
			mc.Message.Mentions = channelDomain.MessageMentions{
				Pods:  mc.Mentions.PodKeys,
				Users: mc.Mentions.UserIDs,
			}
			if err := repo.UpdateMessageMentions(ctx, mc.Message.ID, mc.Message.Mentions); err != nil {
				slog.Error("failed to update message mentions after validation", "error", err)
			}
		}

		return nil
	}
}
