package agentpod

import (
	"context"
	"errors"
	"time"
)

var (
	// ErrSandboxAlreadyResumed indicates a unique constraint violation on source_pod_key.
	ErrSandboxAlreadyResumed = errors.New("sandbox has already been resumed by another active pod")
)

// PodRepository defines persistence operations for Pod entities.
type PodRepository interface {
	// Create persists a new Pod. Returns ErrSandboxAlreadyResumed on unique constraint violation.
	Create(ctx context.Context, pod *Pod) error
	// GetByKey returns a Pod by its pod_key with Runner, Agent, and Repository preloaded.
	GetByKey(ctx context.Context, podKey string) (*Pod, error)
	// GetByID returns a Pod by its ID with Runner, Agent, and Repository preloaded.
	GetByID(ctx context.Context, podID int64) (*Pod, error)
	// GetOrgAndCreator returns the organization_id and created_by_id for a pod.
	GetOrgAndCreator(ctx context.Context, podKey string) (orgID, creatorID int64, err error)
	// GetTicketByID returns a ticket's slug and title by ID (cross-domain read for pod creation).
	GetTicketByID(ctx context.Context, ticketID int64) (slug, title string, err error)
	// ListByOrg returns pods for an organization with optional status and creator filter and pagination.
	ListByOrg(ctx context.Context, orgID int64, statuses []string, createdByID int64, limit, offset int) ([]*Pod, int64, error)
	// ListByTicket returns pods for a ticket with associations preloaded.
	ListByTicket(ctx context.Context, ticketID int64) ([]*Pod, error)
	// ListByRunner returns pods for a runner with optional status filter.
	ListByRunner(ctx context.Context, runnerID int64, status string) ([]*Pod, error)
	// ListByRunnerPaginated returns pods for a runner with pagination.
	ListByRunnerPaginated(ctx context.Context, runnerID int64, status string, limit, offset int) ([]*Pod, int64, error)
	// ListActive returns active pods for a runner (initializing, running, paused, disconnected).
	ListActive(ctx context.Context, runnerID int64) ([]*Pod, error)
	// GetActivePodBySourcePodKey returns an active pod resumed from the given source pod key.
	GetActivePodBySourcePodKey(ctx context.Context, sourcePodKey string) (*Pod, error)
	// FindByBranchAndRepo finds the most recent pod by branch and repository.
	FindByBranchAndRepo(ctx context.Context, orgID, repoID int64, branchName string) (*Pod, error)
	// UpdateByKey updates pod fields by pod_key. Returns rows affected.
	UpdateByKey(ctx context.Context, podKey string, updates map[string]interface{}) (int64, error)
	// UpdateByKeyAndStatus updates pod fields where pod_key and status match.
	UpdateByKeyAndStatus(ctx context.Context, podKey, status string, updates map[string]interface{}) error
	// UpdateAgentStatus updates agent_status, last_activity, and optionally agent_pid.
	UpdateAgentStatus(ctx context.Context, podKey string, updates map[string]interface{}) error
	// UpdateField updates a single field by pod_key.
	UpdateField(ctx context.Context, podKey, field string, value interface{}) error
	// DecrementRunnerPods decrements current_pods for a runner.
	DecrementRunnerPods(ctx context.Context, runnerID int64) error
	// ListActiveByRunner returns active pods (running/initializing) for reconciliation.
	ListActiveByRunner(ctx context.Context, runnerID int64) ([]*Pod, error)
	// ListInitializingByRunner returns pods in "initializing" state for a runner.
	ListInitializingByRunner(ctx context.Context, runnerID int64) ([]*Pod, error)
	// MarkOrphaned marks a pod as orphaned.
	MarkOrphaned(ctx context.Context, pod *Pod, finishedAt time.Time) error
	// MarkStaleAsDisconnected marks initializing/running pods with stale activity as disconnected.
	MarkStaleAsDisconnected(ctx context.Context, threshold time.Time) (int64, error)
	// CleanupStale marks stale disconnected pods as terminated. Returns rows affected.
	CleanupStale(ctx context.Context, threshold time.Time) (int64, error)
	// UpdateByKeyAndStatusCounted is like UpdateByKeyAndStatus but returns rows affected.
	UpdateByKeyAndStatusCounted(ctx context.Context, podKey, status string, updates map[string]interface{}) (int64, error)
	// UpdateTerminatedWithFallbackError updates a terminated pod, setting error_code
	// only if not already set (uses COALESCE(NULLIF(error_code, ''), fallbackCode)).
	UpdateTerminatedWithFallbackError(ctx context.Context, podKey string, updates map[string]interface{}, fallbackErrorCode string) error
	// UpdateTerminatedIfActive is like UpdateTerminatedWithFallbackError but only
	// updates pods that are still in an active state. Returns rows affected.
	UpdateTerminatedIfActive(ctx context.Context, podKey string, updates map[string]interface{}, fallbackErrorCode string) (int64, error)
	// UpdateByKeyAndActiveStatus updates a pod only if it's in an active state.
	// Returns rows affected so the caller can detect if the pod was already terminal.
	UpdateByKeyAndActiveStatus(ctx context.Context, podKey string, updates map[string]interface{}) (int64, error)
	// GetByKeyAndRunner returns a pod by pod_key and runner_id (no preloads).
	GetByKeyAndRunner(ctx context.Context, podKey string, runnerID int64) (*Pod, error)
	// CountActiveByKeys counts how many of the given pod keys are in active status.
	CountActiveByKeys(ctx context.Context, podKeys []string) (int, error)
	// EnrichWithLoopInfo populates the Loop field on pods by joining loop_runs → loops.
	EnrichWithLoopInfo(ctx context.Context, pods []*Pod) error
}

// SettingsRepository defines persistence operations for UserAgentPodSettings.
type SettingsRepository interface {
	GetByUserID(ctx context.Context, userID int64) (*UserAgentPodSettings, error)
	Create(ctx context.Context, settings *UserAgentPodSettings) error
	Save(ctx context.Context, settings *UserAgentPodSettings) error
	DeleteByUserID(ctx context.Context, userID int64) error
}

// AIProviderRepository defines persistence operations for UserAIProvider entities.
type AIProviderRepository interface {
	GetDefaultByType(ctx context.Context, userID int64, providerType string) (*UserAIProvider, error)
	GetEnabledByID(ctx context.Context, providerID int64) (*UserAIProvider, error)
	GetByID(ctx context.Context, providerID int64) (*UserAIProvider, error)
	ListByUser(ctx context.Context, userID int64) ([]*UserAIProvider, error)
	ListByUserAndType(ctx context.Context, userID int64, providerType string) ([]*UserAIProvider, error)
	Create(ctx context.Context, provider *UserAIProvider) error
	Save(ctx context.Context, provider *UserAIProvider) error
	Delete(ctx context.Context, providerID int64) error
	SetDefault(ctx context.Context, providerID int64) error
	ClearDefaults(ctx context.Context, userID int64, providerType string) error
}

// AutopilotRepository defines persistence operations for AutopilotController entities.
type AutopilotRepository interface {
	Create(ctx context.Context, controller *AutopilotController) error
	Save(ctx context.Context, controller *AutopilotController) error
	GetByOrgAndKey(ctx context.Context, orgID int64, key string) (*AutopilotController, error)
	GetByKey(ctx context.Context, key string) (*AutopilotController, error)
	GetActiveForPod(ctx context.Context, podKey string) (*AutopilotController, error)
	ListByOrg(ctx context.Context, orgID int64) ([]*AutopilotController, error)
	UpdateStatusByKey(ctx context.Context, key string, updates map[string]interface{}) error
	ListIterations(ctx context.Context, controllerID int64) ([]*AutopilotIteration, error)
	CreateIteration(ctx context.Context, iteration *AutopilotIteration) error
	// GetApprovalTimedOut returns autopilot controllers in waiting_approval phase
	// whose approval_request_at + approval_timeout_min has elapsed.
	// orgIDs filters to specific organizations; nil means all orgs.
	GetApprovalTimedOut(ctx context.Context, orgIDs []int64) ([]*AutopilotController, error)
}
