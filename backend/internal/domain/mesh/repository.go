package mesh

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

type MeshRepository interface {
	ListEnabledRunners(ctx context.Context, orgID int64) ([]*runner.Runner, error)

	GetChannelPodKeys(ctx context.Context, channelID int64) ([]string, error)

	CountChannelMessages(ctx context.Context, channelID int64) (int64, error)

	ListPodsByTicketIDs(ctx context.Context, ticketIDs []int64) ([]*agentpod.Pod, error)

	CreateChannelPod(ctx context.Context, cp *ChannelPod) error

	DeleteChannelPod(ctx context.Context, channelID int64, podKey string) error

	CreateChannelAccess(ctx context.Context, access *ChannelAccess) error
}
