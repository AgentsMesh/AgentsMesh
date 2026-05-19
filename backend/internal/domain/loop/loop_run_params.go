package loop

import (
	"encoding/json"
	"time"
)

// PodStatusInfo holds Pod status info for SSOT resolution
type PodStatusInfo struct {
	PodKey     string
	Status     string
	FinishedAt *time.Time
}

type RunListFilter struct {
	LoopID int64
	Status string // Optional: filter by status (applied at DB level for finished runs)
	Limit  int
	Offset int
}

// TriggerRunAtomicParams contains parameters for atomically creating a loop run.
type TriggerRunAtomicParams struct {
	LoopID        int64
	TriggerType   string
	TriggerSource string
	TriggerParams json.RawMessage // Optional runtime variable overrides
}

type TriggerRunAtomicResult struct {
	Run     *LoopRun
	Loop    *Loop // the loop as read within the transaction (for event publishing)
	Skipped bool
	Reason  string
}
