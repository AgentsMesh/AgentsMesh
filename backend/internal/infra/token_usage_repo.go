package infra

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/tokenusage"
	"gorm.io/gorm"
)

// Compile-time interface check.
var _ tokenusage.Repository = (*tokenUsageRepository)(nil)

type tokenUsageRepository struct {
	db *gorm.DB
}

// NewTokenUsageRepository creates a new TokenUsageRepository backed by GORM.
func NewTokenUsageRepository(db *gorm.DB) tokenusage.Repository {
	return &tokenUsageRepository{db: db}
}

func (r *tokenUsageRepository) Create(ctx context.Context, record *tokenusage.TokenUsage) error {
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *tokenUsageRepository) CreateBatch(ctx context.Context, records []*tokenusage.TokenUsage) error {
	if len(records) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Create(records).Error
}

func (r *tokenUsageRepository) GetSummary(ctx context.Context, orgID int64, f tokenusage.AggregationFilter) (*tokenusage.UsageSummary, error) {
	var result tokenusage.UsageSummary
	q := r.db.WithContext(ctx).Model(&tokenusage.TokenUsage{}).
		Select(`COALESCE(SUM(input_tokens),0) as input_tokens,
			COALESCE(SUM(output_tokens),0) as output_tokens,
			COALESCE(SUM(cache_creation_tokens),0) as cache_creation_tokens,
			COALESCE(SUM(cache_read_tokens),0) as cache_read_tokens,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens),0) as total_tokens`).
		Where("organization_id = ? AND created_at >= ? AND created_at < ?", orgID, f.StartTime, f.EndTime)
	q = applyOptionalFilters(q, f, false)
	// Single-row aggregation — LIMIT 1 as a safety net.
	if err := q.Limit(1).Scan(&result).Error; err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *tokenUsageRepository) GetTimeSeries(ctx context.Context, orgID int64, f tokenusage.AggregationFilter) ([]tokenusage.TimeSeriesPoint, error) {
	granularity := validGranularity(f.Granularity)
	var results []tokenusage.TimeSeriesPoint
	q := r.db.WithContext(ctx).Model(&tokenusage.TokenUsage{}).
		Select("date_trunc(?, created_at) as period, SUM(input_tokens) as input_tokens, SUM(output_tokens) as output_tokens, SUM(cache_creation_tokens) as cache_creation_tokens, SUM(cache_read_tokens) as cache_read_tokens", granularity).
		Where("organization_id = ? AND created_at >= ? AND created_at < ?", orgID, f.StartTime, f.EndTime).
		Group("period").
		Order("period").
		Limit(1000) // Cap time series data points to prevent DoS on extreme date ranges.
	q = applyOptionalFilters(q, f, false)
	if err := q.Scan(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

func (r *tokenUsageRepository) GetByAgent(ctx context.Context, orgID int64, f tokenusage.AggregationFilter) ([]tokenusage.AgentUsage, error) {
	var results []tokenusage.AgentUsage
	q := r.db.WithContext(ctx).Model(&tokenusage.TokenUsage{}).
		Select(`agent_slug,
			SUM(input_tokens) as input_tokens,
			SUM(output_tokens) as output_tokens,
			SUM(cache_creation_tokens) as cache_creation_tokens,
			SUM(cache_read_tokens) as cache_read_tokens,
			SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens) as total_tokens`).
		Where("organization_id = ? AND created_at >= ? AND created_at < ?", orgID, f.StartTime, f.EndTime).
		Group("agent_slug").
		Order("total_tokens DESC").
		Limit(effectiveLimit(f.Limit))
	q = applyOptionalFilters(q, f, false)
	if err := q.Scan(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

func (r *tokenUsageRepository) GetByUser(ctx context.Context, orgID int64, f tokenusage.AggregationFilter) ([]tokenusage.UserUsage, error) {
	var results []tokenusage.UserUsage
	q := r.db.WithContext(ctx).Model(&tokenusage.TokenUsage{}).
		Select(`token_usages.user_id,
			COALESCE(u.name, u.username, 'unknown') as username,
			COALESCE(u.email, '') as email,
			SUM(token_usages.input_tokens) as input_tokens,
			SUM(token_usages.output_tokens) as output_tokens,
			SUM(token_usages.cache_creation_tokens) as cache_creation_tokens,
			SUM(token_usages.cache_read_tokens) as cache_read_tokens,
			SUM(token_usages.input_tokens + token_usages.output_tokens + token_usages.cache_creation_tokens + token_usages.cache_read_tokens) as total_tokens`).
		Joins("LEFT JOIN users u ON u.id = token_usages.user_id").
		Where("token_usages.organization_id = ? AND token_usages.created_at >= ? AND token_usages.created_at < ?", orgID, f.StartTime, f.EndTime).
		Group("token_usages.user_id, u.name, u.username, u.email").
		Order("total_tokens DESC").
		Limit(effectiveLimit(f.Limit))
	q = applyOptionalFilters(q, f, true)
	if err := q.Scan(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

func (r *tokenUsageRepository) GetByModel(ctx context.Context, orgID int64, f tokenusage.AggregationFilter) ([]tokenusage.ModelUsage, error) {
	var results []tokenusage.ModelUsage
	q := r.db.WithContext(ctx).Model(&tokenusage.TokenUsage{}).
		Select(`model,
			SUM(input_tokens) as input_tokens,
			SUM(output_tokens) as output_tokens,
			SUM(cache_creation_tokens) as cache_creation_tokens,
			SUM(cache_read_tokens) as cache_read_tokens,
			SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens) as total_tokens`).
		Where("organization_id = ? AND created_at >= ? AND created_at < ?", orgID, f.StartTime, f.EndTime).
		Group("model").
		Order("total_tokens DESC").
		Limit(effectiveLimit(f.Limit))
	q = applyOptionalFilters(q, f, false)
	if err := q.Scan(&results).Error; err != nil {
		return nil, err
	}
	return results, nil
}

// validGranularity returns a safe granularity value (whitelist).
func validGranularity(g string) string {
	switch g {
	case "day", "week", "month":
		return g
	default:
		return "day"
	}
}

// applyOptionalFilters applies optional WHERE clauses for agent, user, and model.
// When qualified is true, column names are prefixed with "token_usages." for JOIN queries.
func applyOptionalFilters(q *gorm.DB, f tokenusage.AggregationFilter, qualified bool) *gorm.DB {
	prefix := ""
	if qualified {
		prefix = "token_usages."
	}
	if f.AgentSlug != nil {
		q = q.Where(prefix+"agent_slug = ?", *f.AgentSlug)
	}
	if f.UserID != nil {
		q = q.Where(prefix+"user_id = ?", *f.UserID)
	}
	if f.Model != nil {
		q = q.Where(prefix+"model = ?", *f.Model)
	}
	return q
}

// effectiveLimit returns the limit to use, defaulting to 100 if not set.
func effectiveLimit(limit int) int {
	if limit > 0 {
		return limit
	}
	return 100
}
