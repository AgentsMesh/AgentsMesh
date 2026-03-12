package tokenusage

import (
	"context"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/tokenusage"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// Service handles token usage recording and querying.
type Service struct {
	repo   tokenusage.Repository
	logger *slog.Logger
}

// NewService creates a new token usage service.
func NewService(repo tokenusage.Repository, logger *slog.Logger) *Service {
	return &Service{repo: repo, logger: logger}
}

// RecordUsage persists token usage from a gRPC TokenUsageReport.
// It creates one record per model reported by the runner.
func (s *Service) RecordUsage(
	ctx context.Context,
	orgID int64,
	podID *int64,
	podKey string,
	userID int64,
	runnerID int64,
	agentTypeSlug string,
	report *runnerv1.TokenUsageReport,
) {
	if podKey == "" || orgID <= 0 || report == nil || len(report.Models) == 0 {
		return
	}

	now := time.Now()
	records := make([]*tokenusage.TokenUsage, 0, len(report.Models))
	for _, m := range report.Models {
		// Skip entries with negative token values (defensive).
		if m.InputTokens < 0 || m.OutputTokens < 0 || m.CacheCreationTokens < 0 || m.CacheReadTokens < 0 {
			s.logger.Warn("skipping token usage entry with negative values",
				"pod_key", podKey,
				"model", m.Model,
			)
			continue
		}
		// Enforce DB column length limits on string fields.
		if len(m.Model) > 100 {
			s.logger.Warn("skipping token usage entry: model name exceeds 100 bytes",
				"pod_key", podKey,
				"model_len", len(m.Model),
			)
			continue
		}
		uid, rid := userID, runnerID
		records = append(records, &tokenusage.TokenUsage{
			OrganizationID:      orgID,
			PodID:               podID,
			PodKey:              podKey,
			UserID:              &uid,
			RunnerID:            &rid,
			AgentTypeSlug:       agentTypeSlug,
			Model:               m.Model,
			InputTokens:         m.InputTokens,
			OutputTokens:        m.OutputTokens,
			CacheCreationTokens: m.CacheCreationTokens,
			CacheReadTokens:     m.CacheReadTokens,
			CreatedAt:           now,
		})
	}

	if len(records) == 0 {
		return
	}

	if err := s.repo.CreateBatch(ctx, records); err != nil {
		s.logger.Error("failed to record token usage batch",
			"pod_key", podKey,
			"models", len(records),
			"error", err,
		)
		return
	}

	s.logger.Debug("token usage recorded",
		"pod_key", podKey,
		"models", len(records),
		"org_id", orgID,
	)
}

// GetSummary returns aggregated totals for the given organization and filter.
func (s *Service) GetSummary(ctx context.Context, orgID int64, filter tokenusage.AggregationFilter) (*tokenusage.UsageSummary, error) {
	return s.repo.GetSummary(ctx, orgID, filter)
}

// GetTimeSeries returns time-bucketed aggregation for the given organization.
func (s *Service) GetTimeSeries(ctx context.Context, orgID int64, filter tokenusage.AggregationFilter) ([]tokenusage.TimeSeriesPoint, error) {
	return s.repo.GetTimeSeries(ctx, orgID, filter)
}

// GetByAgent returns usage grouped by agent type slug.
func (s *Service) GetByAgent(ctx context.Context, orgID int64, filter tokenusage.AggregationFilter) ([]tokenusage.AgentUsage, error) {
	return s.repo.GetByAgent(ctx, orgID, filter)
}

// GetByUser returns usage grouped by user ID.
func (s *Service) GetByUser(ctx context.Context, orgID int64, filter tokenusage.AggregationFilter) ([]tokenusage.UserUsage, error) {
	return s.repo.GetByUser(ctx, orgID, filter)
}

// GetByModel returns usage grouped by model name.
func (s *Service) GetByModel(ctx context.Context, orgID int64, filter tokenusage.AggregationFilter) ([]tokenusage.ModelUsage, error) {
	return s.repo.GetByModel(ctx, orgID, filter)
}
