package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/service/billing"
	"gorm.io/gorm"
)

func (s *Service) RequestAuthURL(ctx context.Context, req *RequestAuthURLRequest, frontendURL string) (*RequestAuthURLResponse, error) {
	if req.MachineKey == "" {
		return nil, fmt.Errorf("machine_key is required")
	}

	authKeyBytes := make([]byte, 32)
	if _, err := rand.Read(authKeyBytes); err != nil {
		return nil, fmt.Errorf("failed to generate auth key: %w", err)
	}
	authKey := hex.EncodeToString(authKeyBytes)

	expiresAt := time.Now().Add(15 * time.Minute)

	pendingAuth := &runner.PendingAuth{
		AuthKey:    authKey,
		MachineKey: req.MachineKey,
		ExpiresAt:  expiresAt,
	}

	if req.NodeID != "" {
		pendingAuth.NodeID = &req.NodeID
	}
	if len(req.Labels) > 0 {
		pendingAuth.Labels = runner.Labels(req.Labels)
	}

	if err := s.repo.CreatePendingAuth(ctx, pendingAuth); err != nil {
		return nil, fmt.Errorf("failed to create pending auth: %w", err)
	}

	return &RequestAuthURLResponse{
		AuthURL:   fmt.Sprintf("%s/runners/authorize?key=%s", frontendURL, authKey),
		AuthKey:   authKey,
		ExpiresIn: 900, // 15 minutes in seconds
	}, nil
}

func (s *Service) AuthorizeRunner(ctx context.Context, authKey string, orgID int64, userID int64, nodeID string) (*runner.Runner, error) {
	pendingAuth, err := s.repo.GetPendingAuthByKey(ctx, authKey)
	if err != nil {
		return nil, err
	}
	if pendingAuth == nil {
		return nil, ErrAuthRequestNotFound
	}

	if pendingAuth.IsExpired() {
		return nil, ErrAuthRequestExpired
	}

	finalNodeID, err := resolveNodeID(nodeID, pendingAuth.NodeID)
	if err != nil {
		return nil, err
	}

	// The exists-check is read-only and deliberately runs before the claim:
	// rejecting after claiming would leave authorized=true with no runner, and the
	// claim predicate only matches authorized=false — the auth key would be
	// unusable for the rest of its TTL with no way to retry.
	exists, err := s.repo.ExistsByNodeIDAndOrg(ctx, orgID, finalNodeID)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrRunnerAlreadyExists
	}

	// Resolve the limit here but enforce the count inside the tx (under a per-org
	// lock): a check-then-create here would let concurrent authorizes overshoot.
	runnerLimit := billing.UnlimitedQuota
	if s.billingService != nil {
		runnerLimit, err = s.billingService.ResourceLimit(ctx, orgID, "runners")
		if err != nil {
			return nil, ErrRunnerQuotaExceeded
		}
	}

	r := &runner.Runner{
		OrganizationID:     orgID,
		NodeID:             finalNodeID,
		Status:             runner.RunnerStatusOffline,
		MaxConcurrentPods:  5,
		Visibility:         runner.VisibilityOrganization,
		RegisteredByUserID: &userID,
	}

	claimed, err := s.repo.AuthorizeAndCreateRunner(ctx, pendingAuth.ID, orgID, r, runnerLimit)
	switch {
	case errors.Is(err, gorm.ErrDuplicatedKey):
		// A concurrent authorize won the (org, node_id) uniqueness race after our
		// pre-check; report it as the same conflict the pre-check would have.
		return nil, ErrRunnerAlreadyExists
	case err != nil:
		slog.ErrorContext(ctx, "authorize runner transaction failed", "auth_key_id", pendingAuth.ID, "error", err)
		return nil, fmt.Errorf("failed to authorize runner: %w", err)
	}
	if claimed == 0 {
		// The claim predicate (authorized=false AND expires_at>now) matched
		// nothing for one of two reasons; re-read to tell them apart rather than
		// always blaming a concurrent authorize. A purge may have removed an
		// expired row, so a missing row also means expired.
		if fresh, _ := s.repo.GetPendingAuthByKey(ctx, authKey); fresh == nil || fresh.IsExpired() {
			return nil, ErrAuthRequestExpired
		}
		return nil, ErrAuthRequestAlreadyAuthorized
	}

	return r, nil
}

func resolveNodeID(requested string, pendingNodeID *string) (string, error) {
	if requested != "" {
		return requested, nil
	}
	if pendingNodeID != nil && *pendingNodeID != "" {
		return *pendingNodeID, nil
	}
	nodeIDBytes := make([]byte, 8)
	if _, err := rand.Read(nodeIDBytes); err != nil {
		return "", fmt.Errorf("failed to generate node ID: %w", err)
	}
	return fmt.Sprintf("runner-%s", hex.EncodeToString(nodeIDBytes)), nil
}
