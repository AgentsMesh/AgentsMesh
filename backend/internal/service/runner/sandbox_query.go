package runner

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

const SandboxQueryTimeout = 30 * time.Second

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

type SandboxQueryResult struct {
	RequestID string           `json:"request_id"`
	RunnerID  int64            `json:"runner_id"`
	Sandboxes []*SandboxStatus `json:"sandboxes"`
	Error     string           `json:"error,omitempty"`
}

type pendingQuery struct {
	resultCh chan *SandboxQueryResult
	timeout  time.Time
}

type SandboxQueryService struct {
	pendingQueries sync.Map
	done           chan struct{}
	sender         SandboxQuerySender
}

func NewSandboxQueryService(cm *RunnerConnectionManager) *SandboxQueryService {
	s := &SandboxQueryService{
		done: make(chan struct{}),
	}

	if cm != nil {
		cm.SetSandboxesStatusCallback(func(runnerID int64, data *runnerv1.SandboxesStatusEvent) {
			s.CompleteQuery(data.RequestId, runnerID, data)
		})
	}

	go s.cleanupLoop()
	return s
}

func (s *SandboxQueryService) Stop() {
	close(s.done)
}

// SetSender must be called before QuerySandboxes — delayed injection avoids
// the construction-time cycle between this service and the command sender.
func (s *SandboxQueryService) SetSender(sender SandboxQuerySender) {
	s.sender = sender
}

func (s *SandboxQueryService) IsConnected(runnerID int64) bool {
	if s.sender == nil {
		return false
	}
	return s.sender.IsConnected(runnerID)
}

func (s *SandboxQueryService) RegisterQuery(requestID string) chan *SandboxQueryResult {
	return s.RegisterQueryWithTimeout(requestID, SandboxQueryTimeout)
}

func (s *SandboxQueryService) RegisterQueryWithTimeout(requestID string, timeout time.Duration) chan *SandboxQueryResult {
	resultCh := make(chan *SandboxQueryResult, 1)
	s.pendingQueries.Store(requestID, &pendingQuery{
		resultCh: resultCh,
		timeout:  time.Now().Add(timeout),
	})
	return resultCh
}

func (s *SandboxQueryService) CompleteQuery(requestID string, runnerID int64, event *runnerv1.SandboxesStatusEvent) {
	if v, ok := s.pendingQueries.LoadAndDelete(requestID); ok {
		pq := v.(*pendingQuery)

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
		}
	}
}

func (s *SandboxQueryService) QuerySandboxes(
	ctx context.Context,
	runnerID int64,
	podKeys []string,
) (*SandboxQueryResult, error) {
	if s.sender == nil {
		return nil, ErrCommandSenderNotSet
	}

	requestID := uuid.New().String()

	resultCh := s.RegisterQuery(requestID)

	if err := s.sender.SendQuerySandboxes(runnerID, requestID, podKeys); err != nil {
		s.pendingQueries.Delete(requestID)
		return nil, err
	}

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
