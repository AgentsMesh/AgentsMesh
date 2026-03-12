package tokenusage

import "context"

// Repository defines the data-access contract for token usage records.
type Repository interface {
	Create(ctx context.Context, record *TokenUsage) error
	CreateBatch(ctx context.Context, records []*TokenUsage) error
	GetSummary(ctx context.Context, orgID int64, filter AggregationFilter) (*UsageSummary, error)
	GetTimeSeries(ctx context.Context, orgID int64, filter AggregationFilter) ([]TimeSeriesPoint, error)
	GetByAgent(ctx context.Context, orgID int64, filter AggregationFilter) ([]AgentUsage, error)
	GetByUser(ctx context.Context, orgID int64, filter AggregationFilter) ([]UserUsage, error)
	GetByModel(ctx context.Context, orgID int64, filter AggregationFilter) ([]ModelUsage, error)
}
