package loop

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/gitprovider"
	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	agentpodSvc "github.com/anthropics/agentsmesh/backend/internal/service/agentpod"
	ticketSvc "github.com/anthropics/agentsmesh/backend/internal/service/ticket"
)

// PodTerminator defines the minimal interface needed by LoopOrchestrator
// to terminate Pods (used for timeout handling).
type PodTerminator interface {
	TerminatePod(ctx context.Context, podKey string) error
}

// RepoQueryForLoop provides repository lookup for AgentFile Layer generation.
type RepoQueryForLoop interface {
	GetByID(ctx context.Context, id int64) (*gitprovider.Repository, error)
}

// LoopOrchestrator orchestrates the full lifecycle of a Loop run:
//   - TriggerRun:          atomic run record creation (FOR UPDATE + SSOT concurrency check)
//   - StartRun:            Pod creation + optional Autopilot setup
//   - HandleRunCompleted:  stats update, runtime state (last_pod_key), event publishing
//
// Architecture: Pod is the Single Source of Truth (SSOT) for execution status.
// The orchestrator creates LoopRun records and associates them with Pods,
// but does NOT maintain run status independently. Status is always derived
// from Pod state when queried.
type LoopOrchestrator struct {
	loopService    *LoopService
	loopRunService *LoopRunService
	eventBus       *eventbus.EventBus
	logger         *slog.Logger

	// External dependencies (injected after construction)
	podOrchestrator *agentpodSvc.PodOrchestrator
	autopilotSvc    *agentpodSvc.AutopilotControllerService
	podTerminator   PodTerminator // for terminating timed-out Pods
	ticketService   *ticketSvc.Service
	repoQuery       RepoQueryForLoop // for resolving RepositoryID → clone URL

	// HTTP client for webhook callbacks (reused across calls)
	httpClient *http.Client
}

// NewLoopOrchestrator creates a new LoopOrchestrator
func NewLoopOrchestrator(
	loopService *LoopService,
	loopRunService *LoopRunService,
	eventBus *eventbus.EventBus,
	logger *slog.Logger,
) *LoopOrchestrator {
	return &LoopOrchestrator{
		loopService:    loopService,
		loopRunService: loopRunService,
		eventBus:       eventBus,
		logger:         logger.With("component", "loop_orchestrator"),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
			// Disable redirects to prevent SSRF bypass via HTTP redirect to internal IPs.
			// The initial callback_url is validated against private ranges at create/update time,
			// but a redirect could point to an internal address, bypassing that check.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

// SetPodDependencies injects Pod-related dependencies after construction.
// Called from main.go after PodOrchestrator and PodCoordinator are available.
func (o *LoopOrchestrator) SetPodDependencies(
	podOrch *agentpodSvc.PodOrchestrator,
	autopilot *agentpodSvc.AutopilotControllerService,
	podTerminator PodTerminator,
	ticket *ticketSvc.Service,
	repoQuery RepoQueryForLoop,
) {
	o.podOrchestrator = podOrch
	o.autopilotSvc = autopilot
	o.podTerminator = podTerminator
	o.ticketService = ticket
	o.repoQuery = repoQuery
}
