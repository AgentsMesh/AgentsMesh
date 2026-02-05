package tasks

import (
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

// Redis key constants
const (
	WatchingSetKey        = "pipelines:watching"
	PipelineKeyPrefix     = "pipeline:"
	PollerLockKey         = "poller:lock"
	PollerHeartbeatKey    = "poller:heartbeat"
	LockTimeoutSeconds    = 60
	HeartbeatTTLSeconds   = 30
	CompletedPipelineTTL  = 24 * time.Hour
	RecentUpdateThreshold = 5 * time.Second
)

// Terminal pipeline statuses
var TerminalStatuses = map[string]bool{
	"success":  true,
	"failed":   true,
	"canceled": true,
	"skipped":  true,
}

// WatchedPipeline represents a pipeline being watched
type WatchedPipeline struct {
	ProjectID    string                 `json:"project_id"`
	PipelineID   string                 `json:"pipeline_id"`
	TaskType     string                 `json:"task_type"`
	TaskID       int64                  `json:"task_id"`
	Status       string                 `json:"status"`
	WebURL       string                 `json:"web_url,omitempty"`
	UpdatedAt    time.Time              `json:"updated_at"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
	ArtifactPath string                 `json:"artifact_path,omitempty"`
	JobName      string                 `json:"job_name,omitempty"`
	ResultJSON   string                 `json:"result_json,omitempty"`
}

// PipelineWatcher manages pipeline status watching in Redis
type PipelineWatcher struct {
	redis  *redis.Client
	logger *slog.Logger
}

// NewPipelineWatcher creates a new pipeline watcher
func NewPipelineWatcher(redisClient *redis.Client, logger *slog.Logger) *PipelineWatcher {
	return &PipelineWatcher{
		redis:  redisClient,
		logger: logger,
	}
}
