package runner

import (
	"context"
	"errors"
	"time"
)

// Repository-level errors used by both infra and service layers.
var (
	ErrTokenExhausted = errors.New("registration token usage exhausted")
)

// HeartbeatUpdate represents a batch heartbeat update for a single runner.
type HeartbeatUpdate struct {
	RunnerID    int64
	CurrentPods int
	Status      string
	Version     string
	Timestamp   time.Time
}

// RunnerRepository defines the data-access contract for the runner aggregate.
type RunnerRepository interface {
	// --- Runner CRUD ---
	GetByID(ctx context.Context, id int64) (*Runner, error)
	GetByNodeID(ctx context.Context, nodeID string) (*Runner, error)
	GetByNodeIDAndOrgID(ctx context.Context, nodeID string, orgID int64) (*Runner, error)
	ExistsByNodeIDAndOrg(ctx context.Context, orgID int64, nodeID string) (bool, error)
	Create(ctx context.Context, r *Runner) error
	UpdateFields(ctx context.Context, runnerID int64, updates map[string]interface{}) error
	// UpdateFieldsCAS updates fields only when casField matches casValue. Returns rows affected.
	UpdateFieldsCAS(ctx context.Context, runnerID int64, casField string, casValue interface{}, updates map[string]interface{}) (int64, error)
	Delete(ctx context.Context, runnerID int64) error

	// --- Runner Queries ---
	ListByOrg(ctx context.Context, orgID, userID int64) ([]*Runner, error)
	ListAvailable(ctx context.Context, orgID, userID int64) ([]*Runner, error)
	ListAvailableOrdered(ctx context.Context, orgID, userID int64) ([]*Runner, error)
	ListAvailableForAgent(ctx context.Context, orgID, userID int64, agentJSON string) ([]*Runner, error)

	// --- Runner Pod Count ---
	IncrementPods(ctx context.Context, runnerID int64) error
	DecrementPods(ctx context.Context, runnerID int64) error
	MarkOfflineRunners(ctx context.Context, threshold time.Time) error
	// SetPodCount sets the runner's current_pods to the given count.
	SetPodCount(ctx context.Context, runnerID int64, count int) error

	// --- Heartbeat Batch ---
	// BatchUpdateHeartbeats updates runner heartbeat fields in batch.
	// Each update is independent; one failure does not abort the entire batch.
	// Returns the number of successfully updated runners.
	BatchUpdateHeartbeats(ctx context.Context, items []HeartbeatUpdate) (int, error)

	// --- Cross-domain Helpers ---
	GetOrgSlug(ctx context.Context, orgID int64) (string, error)
	CountLoopsByRunner(ctx context.Context, runnerID int64) (int64, error)

	// --- Certificate ---
	CreateCertificate(ctx context.Context, cert *Certificate) error
	GetCertificateBySerial(ctx context.Context, serial string) (*Certificate, error)
	RevokeCertificate(ctx context.Context, serial string, reason string) error

	// --- PendingAuth ---
	CreatePendingAuth(ctx context.Context, pa *PendingAuth) error
	GetPendingAuthByKey(ctx context.Context, authKey string) (*PendingAuth, error)
	ClaimPendingAuth(ctx context.Context, id int64, orgID int64) (int64, error)
	UpdatePendingAuthRunnerID(ctx context.Context, id int64, runnerID int64) error
	DeleteClaimedPendingAuth(ctx context.Context, id int64) (int64, error)
	CleanupExpiredPendingAuths(ctx context.Context) error

	// --- Registration Token ---
	CreateRegistrationToken(ctx context.Context, token *GRPCRegistrationToken) error
	GetRegistrationTokenByHash(ctx context.Context, hash string) (*GRPCRegistrationToken, error)
	ListRegistrationTokensByOrg(ctx context.Context, orgID int64) ([]GRPCRegistrationToken, error)
	DeleteRegistrationToken(ctx context.Context, tokenID, orgID int64) (int64, error)
	// RegisterWithTokenAtomic atomically claims token usage, creates runner, saves certificate, and updates runner cert info.
	// issueCert is called after the token claim succeeds; it must populate cert fields (SerialNumber, etc.).
	RegisterWithTokenAtomic(ctx context.Context, tokenID int64, r *Runner, cert *Certificate, issueCert func() error) error

	// --- Reactivation Token ---
	CreateReactivationToken(ctx context.Context, token *ReactivationToken) error
	GetReactivationTokenByHash(ctx context.Context, hash string) (*ReactivationToken, error)
	ClaimReactivationToken(ctx context.Context, id int64) (int64, error)
	UnclaimReactivationToken(ctx context.Context, id int64) error
	CleanupExpiredReactivationTokens(ctx context.Context) error
}
