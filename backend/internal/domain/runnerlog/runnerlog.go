package runnerlog

import "time"

const (
	StatusPending    = "pending"
	StatusCollecting = "collecting"
	StatusUploading  = "uploading"
	StatusCompleted  = "completed"
	StatusFailed     = "failed"
)

var ValidStatuses = map[string]bool{
	StatusPending:    true,
	StatusCollecting: true,
	StatusUploading:  true,
	StatusCompleted:  true,
	StatusFailed:     true,
}

func IsTerminalStatus(s string) bool {
	return s == StatusCompleted || s == StatusFailed
}

type RunnerLog struct {
	ID             int64      `json:"id" gorm:"primaryKey"`
	OrganizationID int64      `json:"organization_id"`
	RunnerID       int64      `json:"runner_id"`
	RequestID      string     `json:"request_id" gorm:"uniqueIndex"`
	StorageKey     string     `json:"storage_key"`
	Status         string     `json:"status"`
	SizeBytes      int64      `json:"size_bytes"`
	ErrorMessage   string     `json:"error_message"`
	RequestedByID  int64      `json:"requested_by_id"`
	CreatedAt      time.Time  `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at"`
}

func (RunnerLog) TableName() string {
	return "runner_logs"
}
