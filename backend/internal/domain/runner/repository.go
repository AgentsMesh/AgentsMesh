package runner

import (
	"context"
	"errors"
	"time"
)

var (
	ErrTokenExhausted = errors.New("registration token usage exhausted")
)

type HeartbeatUpdate struct {
	RunnerID    int64
	CurrentPods int
	Status      string
	Version     string
	Timestamp   time.Time
}

type RunnerRepository interface {
	GetByID(ctx context.Context, id int64) (*Runner, error)
	GetByNodeID(ctx context.Context, nodeID string) (*Runner, error)
	GetByNodeIDAndOrgID(ctx context.Context, nodeID string, orgID int64) (*Runner, error)
	ExistsByNodeIDAndOrg(ctx context.Context, orgID int64, nodeID string) (bool, error)
	Create(ctx context.Context, r *Runner) error
	UpdateFields(ctx context.Context, runnerID int64, updates map[string]interface{}) error
	UpdateFieldsCAS(ctx context.Context, runnerID int64, casField string, casValue interface{}, updates map[string]interface{}) (int64, error)
	Delete(ctx context.Context, runnerID int64) error

	ListByOrg(ctx context.Context, orgID, userID int64) ([]*Runner, error)
	ListAvailable(ctx context.Context, orgID, userID int64) ([]*Runner, error)
	ListAvailableOrdered(ctx context.Context, orgID, userID int64) ([]*Runner, error)
	ListAvailableForAgent(ctx context.Context, orgID, userID int64, agentJSON string) ([]*Runner, error)

	IncrementPods(ctx context.Context, runnerID int64) error
	DecrementPods(ctx context.Context, runnerID int64) error
	MarkOfflineRunners(ctx context.Context, threshold time.Time) error
	SetPodCount(ctx context.Context, runnerID int64, count int) error

	BatchUpdateHeartbeats(ctx context.Context, items []HeartbeatUpdate) (int, error)

	GetOrgSlug(ctx context.Context, orgID int64) (string, error)
	CountLoopsByRunner(ctx context.Context, runnerID int64) (int64, error)

	CreateCertificate(ctx context.Context, cert *Certificate) error
	GetCertificateBySerial(ctx context.Context, serial string) (*Certificate, error)
	RevokeCertificate(ctx context.Context, serial string, reason string) error

	CreatePendingAuth(ctx context.Context, pa *PendingAuth) error
	GetPendingAuthByKey(ctx context.Context, authKey string) (*PendingAuth, error)
	ClaimPendingAuth(ctx context.Context, id int64, orgID int64) (int64, error)
	UpdatePendingAuthRunnerID(ctx context.Context, id int64, runnerID int64) error
	DeleteClaimedPendingAuth(ctx context.Context, id int64) (int64, error)
	CleanupExpiredPendingAuths(ctx context.Context) error

	CreateRegistrationToken(ctx context.Context, token *GRPCRegistrationToken) error
	GetRegistrationTokenByHash(ctx context.Context, hash string) (*GRPCRegistrationToken, error)
	ListRegistrationTokensByOrg(ctx context.Context, orgID int64) ([]GRPCRegistrationToken, error)
	DeleteRegistrationToken(ctx context.Context, tokenID, orgID int64) (int64, error)
	// RegisterWithTokenAtomic atomically claims token usage, creates runner, saves certificate, and updates runner cert info.
	// issueCert is called after the token claim succeeds; it must populate cert fields (SerialNumber, etc.).
	RegisterWithTokenAtomic(ctx context.Context, tokenID int64, r *Runner, cert *Certificate, issueCert func() error) error

	CreateReactivationToken(ctx context.Context, token *ReactivationToken) error
	GetReactivationTokenByHash(ctx context.Context, hash string) (*ReactivationToken, error)
	ClaimReactivationToken(ctx context.Context, id int64) (int64, error)
	UnclaimReactivationToken(ctx context.Context, id int64) error
	CleanupExpiredReactivationTokens(ctx context.Context) error
}
