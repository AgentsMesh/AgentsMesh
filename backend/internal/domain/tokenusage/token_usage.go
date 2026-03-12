package tokenusage

import "time"

// TokenUsage represents a single model's token consumption within a pod session.
type TokenUsage struct {
	ID                  int64      `gorm:"primaryKey" json:"id"`
	OrganizationID      int64      `gorm:"not null" json:"organization_id"`
	PodID               *int64     `json:"pod_id"`
	PodKey              string     `gorm:"size:100;not null" json:"pod_key"`
	UserID              *int64     `json:"user_id"`
	RunnerID            *int64     `json:"runner_id"`
	AgentTypeSlug       string     `gorm:"size:50;not null" json:"agent_type_slug"`
	Model               string     `gorm:"size:100" json:"model"`
	InputTokens         int64      `gorm:"not null;default:0" json:"input_tokens"`
	OutputTokens        int64      `gorm:"not null;default:0" json:"output_tokens"`
	CacheCreationTokens int64      `gorm:"not null;default:0" json:"cache_creation_tokens"`
	CacheReadTokens     int64      `gorm:"not null;default:0" json:"cache_read_tokens"`
	SessionStartedAt    *time.Time `json:"session_started_at,omitempty"`
	SessionEndedAt      *time.Time `json:"session_ended_at,omitempty"`
	CreatedAt           time.Time  `gorm:"not null" json:"created_at"`
}

func (TokenUsage) TableName() string { return "token_usages" }

// AggregationFilter defines query parameters for token usage aggregation.
type AggregationFilter struct {
	StartTime     time.Time
	EndTime       time.Time
	AgentTypeSlug *string
	UserID        *int64
	Model         *string
	Granularity   string // "day", "week", "month"
	Limit         int    // Max rows for grouped queries (0 = default 100)
}

// UsageSummary holds aggregated totals.
type UsageSummary struct {
	InputTokens         int64 `json:"input_tokens"`
	OutputTokens        int64 `json:"output_tokens"`
	CacheCreationTokens int64 `json:"cache_creation_tokens"`
	CacheReadTokens     int64 `json:"cache_read_tokens"`
	TotalTokens         int64 `json:"total_tokens"`
}

// TimeSeriesPoint holds aggregated totals for a single time period.
type TimeSeriesPoint struct {
	Period              time.Time `json:"period"`
	InputTokens         int64     `json:"input_tokens"`
	OutputTokens        int64     `json:"output_tokens"`
	CacheCreationTokens int64     `json:"cache_creation_tokens"`
	CacheReadTokens     int64     `json:"cache_read_tokens"`
}

// AgentUsage holds aggregated totals for a single agent type.
type AgentUsage struct {
	AgentTypeSlug       string `json:"agent_type"`
	InputTokens         int64  `json:"input_tokens"`
	OutputTokens        int64  `json:"output_tokens"`
	CacheCreationTokens int64  `json:"cache_creation_tokens"`
	CacheReadTokens     int64  `json:"cache_read_tokens"`
	TotalTokens         int64  `json:"total_tokens"`
}

// UserUsage holds aggregated totals for a single user.
type UserUsage struct {
	UserID              int64  `json:"user_id"`
	Username            string `json:"username"`
	Email               string `json:"email"`
	InputTokens         int64  `json:"input_tokens"`
	OutputTokens        int64  `json:"output_tokens"`
	CacheCreationTokens int64  `json:"cache_creation_tokens"`
	CacheReadTokens     int64  `json:"cache_read_tokens"`
	TotalTokens         int64  `json:"total_tokens"`
}

// ModelUsage holds aggregated totals for a single model.
type ModelUsage struct {
	Model               string `json:"model"`
	InputTokens         int64  `json:"input_tokens"`
	OutputTokens        int64  `json:"output_tokens"`
	CacheCreationTokens int64  `json:"cache_creation_tokens"`
	CacheReadTokens     int64  `json:"cache_read_tokens"`
	TotalTokens         int64  `json:"total_tokens"`
}
