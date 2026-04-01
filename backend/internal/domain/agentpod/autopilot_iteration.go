package agentpod

import (
	"time"
)

// AutopilotIteration represents a single iteration record
type AutopilotIteration struct {
	ID                    int64 `gorm:"primaryKey" json:"id"`
	AutopilotControllerID int64 `gorm:"not null;index" json:"autopilot_controller_id"`
	Iteration             int32 `gorm:"not null" json:"iteration"`

	// Phase progression
	Phase string `gorm:"size:50;not null" json:"phase"` // started, control_running, action_sent, completed, error

	// Decision details
	Summary      *string `gorm:"type:text" json:"summary,omitempty"`
	FilesChanged *string `gorm:"type:text" json:"files_changed,omitempty"` // JSON array of file paths
	ErrorMessage *string `gorm:"type:text" json:"error_message,omitempty"`

	// Timing
	DurationMs int64     `json:"duration_ms,omitempty"`
	CreatedAt  time.Time `gorm:"not null;default:now()" json:"created_at"`
}

func (AutopilotIteration) TableName() string {
	return "autopilot_iterations"
}

// Iteration phase constants
const (
	IterationPhaseStarted        = "started"
	IterationPhaseControlRunning = "control_running"
	IterationPhaseActionSent     = "action_sent"
	IterationPhaseCompleted      = "completed"
	IterationPhaseError          = "error"
)
