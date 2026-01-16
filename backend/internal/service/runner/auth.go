package runner

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"golang.org/x/crypto/bcrypt"
)

// AuthenticateRunner authenticates a runner by its auth token
func (s *Service) AuthenticateRunner(ctx context.Context, runnerID int64, authToken string) (*runner.Runner, error) {
	var r runner.Runner
	if err := s.db.WithContext(ctx).First(&r, runnerID).Error; err != nil {
		return nil, ErrRunnerNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(r.AuthTokenHash), []byte(authToken)); err != nil {
		return nil, ErrInvalidToken
	}

	return &r, nil
}

// ValidateRunnerAuth validates runner authentication by node_id and auth token
// Returns the runner if authentication is successful
func (s *Service) ValidateRunnerAuth(ctx context.Context, nodeID, authToken string) (*runner.Runner, error) {
	var r runner.Runner
	if err := s.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&r).Error; err != nil {
		return nil, ErrRunnerNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(r.AuthTokenHash), []byte(authToken)); err != nil {
		return nil, ErrInvalidAuth
	}

	if !r.IsEnabled {
		return nil, ErrRunnerDisabled
	}

	return &r, nil
}

// RegenerateAuthToken generates a new authentication token for a runner
func (s *Service) RegenerateAuthToken(ctx context.Context, runnerID int64) (string, error) {
	var r runner.Runner
	if err := s.db.WithContext(ctx).First(&r, runnerID).Error; err != nil {
		return "", ErrRunnerNotFound
	}

	// Generate new auth token
	authTokenBytes := make([]byte, 32)
	if _, err := rand.Read(authTokenBytes); err != nil {
		return "", err
	}
	authToken := hex.EncodeToString(authTokenBytes)

	authTokenHash, err := bcrypt.GenerateFromPassword([]byte(authToken), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	// Update the runner with the new token hash
	if err := s.db.WithContext(ctx).Model(&r).Update("auth_token_hash", string(authTokenHash)).Error; err != nil {
		return "", err
	}

	return authToken, nil
}
