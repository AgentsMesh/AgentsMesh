package mesh

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

// MeshRepository defines the data access interface for mesh topology operations.
type MeshRepository interface {
	// ListEnabledRunners returns all enabled runners for an organization.
	ListEnabledRunners(ctx context.Context, orgID int64) ([]*runner.Runner, error)

	// GetChannelPodKeys returns the pod keys participating in a channel.
	GetChannelPodKeys(ctx context.Context, channelID int64) ([]string, error)

	// CountChannelMessages returns the message count for a channel.
	CountChannelMessages(ctx context.Context, channelID int64) (int64, error)

	// ListPodsByTicketIDs returns pods matching any of the given ticket IDs.
	ListPodsByTicketIDs(ctx context.Context, ticketIDs []int64) ([]*agentpod.Pod, error)

	// CreateChannelPod adds a pod to a channel.
	CreateChannelPod(ctx context.Context, cp *ChannelPod) error

	// DeleteChannelPod removes a pod from a channel.
	DeleteChannelPod(ctx context.Context, channelID int64, podKey string) error

	// CreateChannelAccess records access to a channel.
	CreateChannelAccess(ctx context.Context, access *ChannelAccess) error
}
