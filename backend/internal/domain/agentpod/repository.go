package agentpod

import (
	"context"
	"errors"
	"time"
)

var (
	ErrSandboxAlreadyResumed = errors.New("sandbox has already been resumed by another active pod")
)

type PodListQuery struct {
	Statuses      []string // empty = all statuses
	CreatedByID   int64    // >0 restricts to this creator
	GrantedUserID int64    // >0 also includes pods granted to this user
	RunnerID      int64    // >0 restricts to pods owned by this runner
	Limit         int
	Offset        int
}

type PodRepository interface {
	Create(ctx context.Context, pod *Pod) error
	GetByKey(ctx context.Context, podKey string) (*Pod, error)
	GetByID(ctx context.Context, podID int64) (*Pod, error)
	GetOrgAndCreator(ctx context.Context, podKey string) (orgID, creatorID int64, err error)
	GetTicketByID(ctx context.Context, ticketID int64) (slug, title string, err error)
	ListByOrg(ctx context.Context, orgID int64, q PodListQuery) ([]*Pod, int64, error)
	ListByTicket(ctx context.Context, ticketID int64) ([]*Pod, error)
	ListByRunner(ctx context.Context, runnerID int64, status string) ([]*Pod, error)
	ListByRunnerPaginated(ctx context.Context, runnerID int64, q PodListQuery) ([]*Pod, int64, error)
	ListActive(ctx context.Context, runnerID int64) ([]*Pod, error)
	GetActivePodBySourcePodKey(ctx context.Context, sourcePodKey string) (*Pod, error)
	ListActiveResumedBy(ctx context.Context, sourcePodKeys []string) (map[string]string, error)
	FindByBranchAndRepo(ctx context.Context, orgID, repoID int64, branchName string) (*Pod, error)
	UpdateByKey(ctx context.Context, podKey string, updates map[string]interface{}) (int64, error)
	UpdateByKeyAndStatus(ctx context.Context, podKey, status string, updates map[string]interface{}) error
	UpdateAgentStatus(ctx context.Context, podKey string, updates map[string]interface{}) error
	UpdateField(ctx context.Context, podKey, field string, value interface{}) error
	DecrementRunnerPods(ctx context.Context, runnerID int64) error
	ListActiveByRunner(ctx context.Context, runnerID int64) ([]*Pod, error)
	ListInitializingByRunner(ctx context.Context, runnerID int64) ([]*Pod, error)
	MarkOrphaned(ctx context.Context, pod *Pod, finishedAt time.Time) error
	MarkStaleAsDisconnected(ctx context.Context, threshold time.Time) (int64, error)
	CleanupStale(ctx context.Context, threshold time.Time) (int64, error)
	UpdateByKeyAndStatusCounted(ctx context.Context, podKey, status string, updates map[string]interface{}) (int64, error)
	UpdateTerminatedWithFallbackError(ctx context.Context, podKey string, updates map[string]interface{}, fallbackErrorCode string) error
	UpdateTerminatedIfActive(ctx context.Context, podKey string, updates map[string]interface{}, fallbackErrorCode string) (int64, error)
	UpdateByKeyAndActiveStatus(ctx context.Context, podKey string, updates map[string]interface{}) (int64, error)
	GetByKeyAndRunner(ctx context.Context, podKey string, runnerID int64) (*Pod, error)
	CountActiveByKeys(ctx context.Context, podKeys []string) (int, error)
	EnrichWithLoopInfo(ctx context.Context, pods []*Pod) error
	ListRunnersByRepo(ctx context.Context, orgID, repoID int64, limit int) ([]RunnerRepoHistory, error)
}

type RunnerRepoHistory struct {
	RunnerID int64 `gorm:"column:runner_id"`
	PodCount int   `gorm:"column:pod_count"`
}

type SettingsRepository interface {
	GetByUserID(ctx context.Context, userID int64) (*UserAgentPodSettings, error)
	Create(ctx context.Context, settings *UserAgentPodSettings) error
	Save(ctx context.Context, settings *UserAgentPodSettings) error
	DeleteByUserID(ctx context.Context, userID int64) error
}

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
	GetApprovalTimedOut(ctx context.Context, orgIDs []int64) ([]*AutopilotController, error)
}
