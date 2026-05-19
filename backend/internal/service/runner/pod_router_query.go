package runner

import (
	"context"
	"time"

	"github.com/google/uuid"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

const ObservePodTimeout = 10 * time.Second

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

type pendingPodQuery struct {
	resultCh chan *ObservePodResult
	timeout  time.Time
}

func (tr *PodRouter) registerQuery(requestID string) chan *ObservePodResult {
	resultCh := make(chan *ObservePodResult, 1)
	tr.pendingQueries.Store(requestID, &pendingPodQuery{
		resultCh: resultCh,
		timeout:  time.Now().Add(ObservePodTimeout),
	})
	return resultCh
}

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
		}
	}
}

func (tr *PodRouter) ObservePod(ctx context.Context, podKey string, lines int32, includeScreen bool) (*ObservePodResult, error) {
	runnerID, found := tr.GetRunnerID(podKey)
	if !found {
		return nil, ErrRunnerNotConnected
	}

	requestID := uuid.New().String()

	resultCh := tr.registerQuery(requestID)

	if err := tr.commandSender.SendObservePod(ctx, runnerID, requestID, podKey, lines, includeScreen); err != nil {
		tr.pendingQueries.Delete(requestID)
		return nil, err
	}

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

func initQuerySupport(tr *PodRouter, cm *RunnerConnectionManager, done chan struct{}) {
	tr.done = done

	cm.SetObservePodResultCallback(func(runnerID int64, data *runnerv1.ObservePodResult) {
		tr.completeQuery(data.RequestId, runnerID, data)
	})

	go tr.cleanupQueryLoop()
}
