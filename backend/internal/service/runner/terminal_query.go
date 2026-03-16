package runner

import (
	"context"
	"sync"
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

// TerminalQueryService handles terminal observation queries to runners
type TerminalQueryService struct {
	pendingQueries sync.Map      // map[requestID]*pendingTerminalQuery
	done           chan struct{} // signal channel for graceful shutdown
}

// NewTerminalQueryService creates a new terminal query service
func NewTerminalQueryService(cm *RunnerConnectionManager) *TerminalQueryService {
	s := &TerminalQueryService{
		done: make(chan struct{}),
	}

	// Set up callback from connection manager for observe terminal responses
	if cm != nil {
		cm.SetObserveTerminalResultCallback(func(runnerID int64, data *runnerv1.ObserveTerminalResult) {
			s.CompleteQuery(data.RequestId, runnerID, data)
		})
	}

	// Start cleanup goroutine for expired queries
	go s.cleanupLoop()
	return s
}

// Stop gracefully stops the terminal query service
func (s *TerminalQueryService) Stop() {
	close(s.done)
}

// RegisterQuery registers a pending query and returns a channel for the result
func (s *TerminalQueryService) RegisterQuery(requestID string) chan *ObserveTerminalQueryResult {
	resultCh := make(chan *ObserveTerminalQueryResult, 1)
	s.pendingQueries.Store(requestID, &pendingTerminalQuery{
		resultCh: resultCh,
		timeout:  time.Now().Add(TerminalQueryTimeout),
	})
	return resultCh
}

// CompleteQuery completes a pending query with the result
func (s *TerminalQueryService) CompleteQuery(requestID string, runnerID int64, event *runnerv1.ObserveTerminalResult) {
	if v, ok := s.pendingQueries.LoadAndDelete(requestID); ok {
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

// ObserveTerminal sends a terminal observation query to a runner and waits for the response
func (s *TerminalQueryService) ObserveTerminal(
	ctx context.Context,
	runnerID int64,
	podKey string,
	lines int32,
	includeScreen bool,
	sendFn func(runnerID int64, requestID, podKey string, lines int32, includeScreen bool) error,
) (*ObserveTerminalQueryResult, error) {
	// Generate unique request ID
	requestID := uuid.New().String()

	// Register query and get result channel
	resultCh := s.RegisterQuery(requestID)

	// Send query to runner
	if err := sendFn(runnerID, requestID, podKey, lines, includeScreen); err != nil {
		s.pendingQueries.Delete(requestID)
		return nil, err
	}

	// Wait for result with timeout
	select {
	case result := <-resultCh:
		return result, nil
	case <-ctx.Done():
		s.pendingQueries.Delete(requestID)
		return nil, ctx.Err()
	case <-time.After(TerminalQueryTimeout):
		s.pendingQueries.Delete(requestID)
		return &ObserveTerminalQueryResult{
			RequestID: requestID,
			RunnerID:  runnerID,
			Error:     "query timeout",
		}, nil
	}
}

// cleanupLoop periodically cleans up expired queries
func (s *TerminalQueryService) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			now := time.Now()
			s.pendingQueries.Range(func(key, value any) bool {
				pq := value.(*pendingTerminalQuery)
				if now.After(pq.timeout) {
					if v, ok := s.pendingQueries.LoadAndDelete(key); ok {
						pending := v.(*pendingTerminalQuery)
						// Send timeout error to channel
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
