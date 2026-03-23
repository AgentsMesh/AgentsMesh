package runner

import (
	"context"
	"time"

	"github.com/google/uuid"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// ObservePodTimeout is the default timeout for pod observation queries
const ObservePodTimeout = 10 * time.Second

// ObservePodResult represents the result of a pod observation query
type ObservePodResult struct {
	RequestID  string `json:"request_id"`
	RunnerID   int64  `json:"runner_id"`
	Output     string `json:"output"`
	Screen     string `json:"screen,omitempty"`
	CursorX    int    `json:"cursor_x"`
	CursorY    int    `json:"cursor_y"`
	TotalLines int    `json:"total_lines"`
	HasMore    bool   `json:"has_more"`
	Error      string `json:"error,omitempty"`
}

// pendingPodQuery represents a pending pod observation query request
type pendingPodQuery struct {
	resultCh chan *ObservePodResult
	timeout  time.Time
}

// registerQuery registers a pending query and returns a channel for the result.
func (tr *PodRouter) registerQuery(requestID string) chan *ObservePodResult {
	resultCh := make(chan *ObservePodResult, 1)
	tr.pendingQueries.Store(requestID, &pendingPodQuery{
		resultCh: resultCh,
		timeout:  time.Now().Add(ObservePodTimeout),
	})
	return resultCh
}

// completeQuery completes a pending query with the result from a runner callback.
func (tr *PodRouter) completeQuery(requestID string, runnerID int64, event *runnerv1.ObservePodResult) {
	if v, ok := tr.pendingQueries.LoadAndDelete(requestID); ok {
		pq := v.(*pendingPodQuery)

		result := &ObservePodResult{
			RequestID:  requestID,
			RunnerID:   runnerID,
			Output:     event.Output,
			Screen:     event.Screen,
			CursorX:    int(event.CursorX),
			CursorY:    int(event.CursorY),
			TotalLines: int(event.TotalLines),
			HasMore:    event.HasMore,
			Error:      event.Error,
		}

		select {
		case pq.resultCh <- result:
		default:
			// Channel full or closed, ignore
		}
	}
}

// ObservePod sends an observe pod command to the runner hosting the pod
// and waits for the async response. This is the single entry point for all callers
// (REST handler, MCP handler) — no external orchestration needed.
func (tr *PodRouter) ObservePod(ctx context.Context, podKey string, lines int32, includeScreen bool) (*ObservePodResult, error) {
	// Look up runner ID from pod-runner mapping
	runnerID, found := tr.GetRunnerID(podKey)
	if !found {
		return nil, ErrRunnerNotConnected
	}

	// Generate unique request ID
	requestID := uuid.New().String()

	// Register query and get result channel
	resultCh := tr.registerQuery(requestID)

	// Send command to runner
	if err := tr.commandSender.SendObservePod(ctx, runnerID, requestID, podKey, lines, includeScreen); err != nil {
		tr.pendingQueries.Delete(requestID)
		return nil, err
	}

	// Wait for result with timeout
	select {
	case result := <-resultCh:
		return result, nil
	case <-ctx.Done():
		tr.pendingQueries.Delete(requestID)
		return nil, ctx.Err()
	case <-time.After(ObservePodTimeout):
		tr.pendingQueries.Delete(requestID)
		return &ObservePodResult{
			RequestID: requestID,
			RunnerID:  runnerID,
			Error:     "query timeout",
		}, nil
	}
}

// cleanupQueryLoop periodically cleans up expired pod queries.
func (tr *PodRouter) cleanupQueryLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-tr.done:
			return
		case <-ticker.C:
			now := time.Now()
			tr.pendingQueries.Range(func(key, value any) bool {
				pq := value.(*pendingPodQuery)
				if now.After(pq.timeout) {
					if v, ok := tr.pendingQueries.LoadAndDelete(key); ok {
						pending := v.(*pendingPodQuery)
						select {
						case pending.resultCh <- &ObservePodResult{
							RequestID: key.(string),
							Error:     "query timeout",
						}:
						default:
						}
					}
				}
				return true
			})
		}
	}
}

// initQuerySupport sets up the observe pod callback and starts the cleanup goroutine.
// pendingQueries is a zero-value sync.Map (ready to use without initialization).
func initQuerySupport(tr *PodRouter, cm *RunnerConnectionManager, done chan struct{}) {
	tr.done = done

	// Set up callback from connection manager for observe pod responses
	cm.SetObservePodResultCallback(func(runnerID int64, data *runnerv1.ObservePodResult) {
		tr.completeQuery(data.RequestId, runnerID, data)
	})

	// Start cleanup goroutine for expired queries
	go tr.cleanupQueryLoop()
}
