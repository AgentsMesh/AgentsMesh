package runner

import (
	"context"
	"time"

	"github.com/google/uuid"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// TerminalQueryTimeout is the default timeout for terminal observation queries
const TerminalQueryTimeout = 10 * time.Second

// ObserveTerminalQueryResult represents the result of a terminal observation query
type ObserveTerminalQueryResult struct {
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

// pendingTerminalQuery represents a pending terminal observation query request
type pendingTerminalQuery struct {
	resultCh chan *ObserveTerminalQueryResult
	timeout  time.Time
}

// registerQuery registers a pending query and returns a channel for the result.
func (tr *TerminalRouter) registerQuery(requestID string) chan *ObserveTerminalQueryResult {
	resultCh := make(chan *ObserveTerminalQueryResult, 1)
	tr.pendingQueries.Store(requestID, &pendingTerminalQuery{
		resultCh: resultCh,
		timeout:  time.Now().Add(TerminalQueryTimeout),
	})
	return resultCh
}

// completeQuery completes a pending query with the result from a runner callback.
func (tr *TerminalRouter) completeQuery(requestID string, runnerID int64, event *runnerv1.ObserveTerminalResult) {
	if v, ok := tr.pendingQueries.LoadAndDelete(requestID); ok {
		pq := v.(*pendingTerminalQuery)

		result := &ObserveTerminalQueryResult{
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

// ObserveTerminal sends an observe terminal command to the runner hosting the pod
// and waits for the async response. This is the single entry point for all callers
// (REST handler, MCP handler) — no external orchestration needed.
func (tr *TerminalRouter) ObserveTerminal(ctx context.Context, podKey string, lines int32, includeScreen bool) (*ObserveTerminalQueryResult, error) {
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
	if err := tr.commandSender.SendObserveTerminal(ctx, runnerID, requestID, podKey, lines, includeScreen); err != nil {
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
	case <-time.After(TerminalQueryTimeout):
		tr.pendingQueries.Delete(requestID)
		return &ObserveTerminalQueryResult{
			RequestID: requestID,
			RunnerID:  runnerID,
			Error:     "query timeout",
		}, nil
	}
}

// cleanupQueryLoop periodically cleans up expired terminal queries.
func (tr *TerminalRouter) cleanupQueryLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-tr.done:
			return
		case <-ticker.C:
			now := time.Now()
			tr.pendingQueries.Range(func(key, value any) bool {
				pq := value.(*pendingTerminalQuery)
				if now.After(pq.timeout) {
					if v, ok := tr.pendingQueries.LoadAndDelete(key); ok {
						pending := v.(*pendingTerminalQuery)
						select {
						case pending.resultCh <- &ObserveTerminalQueryResult{
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

// initQuerySupport sets up the observe terminal callback and starts the cleanup goroutine.
// pendingQueries is a zero-value sync.Map (ready to use without initialization).
func initQuerySupport(tr *TerminalRouter, cm *RunnerConnectionManager, done chan struct{}) {
	tr.done = done

	// Set up callback from connection manager for observe terminal responses
	cm.SetObserveTerminalResultCallback(func(runnerID int64, data *runnerv1.ObserveTerminalResult) {
		tr.completeQuery(data.RequestId, runnerID, data)
	})

	// Start cleanup goroutine for expired queries
	go tr.cleanupQueryLoop()
}
