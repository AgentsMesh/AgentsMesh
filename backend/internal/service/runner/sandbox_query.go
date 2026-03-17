package runner

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// SandboxQueryTimeout is the default timeout for sandbox queries
const SandboxQueryTimeout = 30 * time.Second

// SandboxStatus represents the status of a sandbox on a runner
type SandboxStatus struct {
	PodKey                string `json:"pod_key"`
	Exists                bool   `json:"exists"`
	CanResume             bool   `json:"can_resume"`
	SandboxPath           string `json:"sandbox_path,omitempty"`
	RepositoryURL         string `json:"repository_url,omitempty"`
	BranchName            string `json:"branch_name,omitempty"`
	CurrentCommit         string `json:"current_commit,omitempty"`
	SizeBytes             int64  `json:"size_bytes,omitempty"`
	LastModified          int64  `json:"last_modified,omitempty"`
	HasUncommittedChanges bool   `json:"has_uncommitted_changes,omitempty"`
	Error                 string `json:"error,omitempty"`
}

// SandboxQueryResult represents the result of a sandbox query request
type SandboxQueryResult struct {
	RequestID string           `json:"request_id"`
	RunnerID  int64            `json:"runner_id"`
	Sandboxes []*SandboxStatus `json:"sandboxes"`
	Error     string           `json:"error,omitempty"`
}

// pendingQuery represents a pending sandbox query request
type pendingQuery struct {
	resultCh chan *SandboxQueryResult
	timeout  time.Time
}

// SandboxQueryService handles sandbox status queries to runners.
// It owns the full query lifecycle: send command → wait for async response → return result.
type SandboxQueryService struct {
	pendingQueries sync.Map      // map[requestID]*pendingQuery
	done           chan struct{} // signal channel for graceful shutdown
	sender         SandboxQuerySender // injected via SetSender (delayed wiring)
}

// NewSandboxQueryService creates a new sandbox query service
func NewSandboxQueryService(cm *RunnerConnectionManager) *SandboxQueryService {
	s := &SandboxQueryService{
		done: make(chan struct{}),
	}

	// Set up callback from connection manager for sandbox status responses
	if cm != nil {
		cm.SetSandboxesStatusCallback(func(runnerID int64, data *runnerv1.SandboxesStatusEvent) {
			s.CompleteQuery(data.RequestId, runnerID, data)
		})
	}

	// Start cleanup goroutine for expired queries
	// This only cleans up orphaned queries (e.g., when client disconnects)
	// Normal timeout is handled in QuerySandboxes via time.After
	go s.cleanupLoop()
	return s
}

// Stop gracefully stops the sandbox query service
func (s *SandboxQueryService) Stop() {
	close(s.done)
}

// SetSender sets the command sender for sandbox queries (delayed injection).
// Must be called before QuerySandboxes is used.
func (s *SandboxQueryService) SetSender(sender SandboxQuerySender) {
	s.sender = sender
}

// IsConnected checks if a runner is connected via the underlying sender.
func (s *SandboxQueryService) IsConnected(runnerID int64) bool {
	if s.sender == nil {
		return false
	}
	return s.sender.IsConnected(runnerID)
}

// RegisterQuery registers a pending query and returns a channel for the result
func (s *SandboxQueryService) RegisterQuery(requestID string) chan *SandboxQueryResult {
	return s.RegisterQueryWithTimeout(requestID, SandboxQueryTimeout)
}

// RegisterQueryWithTimeout registers a pending query with a custom timeout
func (s *SandboxQueryService) RegisterQueryWithTimeout(requestID string, timeout time.Duration) chan *SandboxQueryResult {
	resultCh := make(chan *SandboxQueryResult, 1)
	s.pendingQueries.Store(requestID, &pendingQuery{
		resultCh: resultCh,
		timeout:  time.Now().Add(timeout),
	})
	return resultCh
}

// CompleteQuery completes a pending query with the result
func (s *SandboxQueryService) CompleteQuery(requestID string, runnerID int64, event *runnerv1.SandboxesStatusEvent) {
	if v, ok := s.pendingQueries.LoadAndDelete(requestID); ok {
		pq := v.(*pendingQuery)

		// Convert proto to internal types
		sandboxes := make([]*SandboxStatus, len(event.Sandboxes))
		for i, sb := range event.Sandboxes {
			sandboxes[i] = &SandboxStatus{
				PodKey:                sb.PodKey,
				Exists:                sb.Exists,
				CanResume:             sb.CanResume,
				SandboxPath:           sb.SandboxPath,
				RepositoryURL:         sb.RepositoryUrl,
				BranchName:            sb.BranchName,
				CurrentCommit:         sb.CurrentCommit,
				SizeBytes:             sb.SizeBytes,
				LastModified:          sb.LastModified,
				HasUncommittedChanges: sb.HasUncommittedChanges,
				Error:                 sb.Error,
			}
		}

		result := &SandboxQueryResult{
			RequestID: requestID,
			RunnerID:  runnerID,
			Sandboxes: sandboxes,
		}

		select {
		case pq.resultCh <- result:
		default:
			// Channel full or closed, ignore
		}
	}
}

// QuerySandboxes sends a sandbox query to a runner and waits for the response.
// The sender must be configured via SetSender before calling this method.
func (s *SandboxQueryService) QuerySandboxes(
	ctx context.Context,
	runnerID int64,
	podKeys []string,
) (*SandboxQueryResult, error) {
	if s.sender == nil {
		return nil, ErrCommandSenderNotSet
	}

	// Generate unique request ID
	requestID := uuid.New().String()

	// Register query and get result channel
	resultCh := s.RegisterQuery(requestID)

	// Send query to runner
	if err := s.sender.SendQuerySandboxes(runnerID, requestID, podKeys); err != nil {
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
	case <-time.After(SandboxQueryTimeout):
		s.pendingQueries.Delete(requestID)
		return &SandboxQueryResult{
			RequestID: requestID,
			RunnerID:  runnerID,
			Error:     "query timeout",
		}, nil
	}
}

// cleanupLoop periodically cleans up expired queries
func (s *SandboxQueryService) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			now := time.Now()
			s.pendingQueries.Range(func(key, value any) bool {
				pq := value.(*pendingQuery)
				if now.After(pq.timeout) {
					if v, ok := s.pendingQueries.LoadAndDelete(key); ok {
						pending := v.(*pendingQuery)
						// Send timeout error to channel
						select {
						case pending.resultCh <- &SandboxQueryResult{
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
