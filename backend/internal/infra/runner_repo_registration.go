package infra

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"gorm.io/gorm"
)

// --- Certificate ---

func (r *runnerRepository) CreateCertificate(ctx context.Context, cert *runner.Certificate) error {
	return r.db.WithContext(ctx).Create(cert).Error
}

func (r *runnerRepository) GetCertificateBySerial(ctx context.Context, serial string) (*runner.Certificate, error) {
	var cert runner.Certificate
	if err := r.db.WithContext(ctx).Where("serial_number = ?", serial).First(&cert).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &cert, nil
}

func (r *runnerRepository) RevokeCertificate(ctx context.Context, serial string, reason string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&runner.Certificate{}).
		Where("serial_number = ?", serial).
		Updates(map[string]interface{}{
			"revoked_at":        now,
			"revocation_reason": reason,
		}).Error
}

// --- PendingAuth ---

func (r *runnerRepository) CreatePendingAuth(ctx context.Context, pa *runner.PendingAuth) error {
	return r.db.WithContext(ctx).Create(pa).Error
}

func (r *runnerRepository) GetPendingAuthByKey(ctx context.Context, authKey string) (*runner.PendingAuth, error) {
	var pa runner.PendingAuth
	if err := r.db.WithContext(ctx).Where("auth_key = ?", authKey).First(&pa).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &pa, nil
}

func (r *runnerRepository) ClaimPendingAuth(ctx context.Context, id int64, orgID int64) (int64, error) {
	result := r.db.WithContext(ctx).Model(&runner.PendingAuth{}).
		Where("id = ? AND authorized = false AND expires_at > ?", id, time.Now()).
		Updates(map[string]interface{}{
			"authorized":      true,
			"organization_id": orgID,
		})
	return result.RowsAffected, result.Error
}

func (r *runnerRepository) UpdatePendingAuthRunnerID(ctx context.Context, id int64, runnerID int64) error {
	return r.db.WithContext(ctx).Model(&runner.PendingAuth{}).
		Where("id = ?", id).
		Update("runner_id", runnerID).Error
}

func (r *runnerRepository) DeleteClaimedPendingAuth(ctx context.Context, id int64) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("id = ? AND authorized = true", id).
		Delete(&runner.PendingAuth{})
	return result.RowsAffected, result.Error
}

func (r *runnerRepository) CleanupExpiredPendingAuths(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&runner.PendingAuth{}).Error
}

// --- Registration Token ---

func (r *runnerRepository) CreateRegistrationToken(ctx context.Context, token *runner.GRPCRegistrationToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *runnerRepository) GetRegistrationTokenByHash(ctx context.Context, hash string) (*runner.GRPCRegistrationToken, error) {
	var token runner.GRPCRegistrationToken
	if err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&token).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

func (r *runnerRepository) ListRegistrationTokensByOrg(ctx context.Context, orgID int64) ([]runner.GRPCRegistrationToken, error) {
	var tokens []runner.GRPCRegistrationToken
	if err := r.db.WithContext(ctx).
		Where("organization_id = ?", orgID).
		Order("created_at DESC").
		Find(&tokens).Error; err != nil {
		return nil, err
	}
	return tokens, nil
}

func (r *runnerRepository) DeleteRegistrationToken(ctx context.Context, tokenID, orgID int64) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("id = ? AND organization_id = ?", tokenID, orgID).
		Delete(&runner.GRPCRegistrationToken{})
	return result.RowsAffected, result.Error
}

func (r *runnerRepository) RegisterWithTokenAtomic(ctx context.Context, tokenID int64, rn *runner.Runner, cert *runner.Certificate, issueCert func() error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Atomic token usage update
		updateResult := tx.Model(&runner.GRPCRegistrationToken{}).
			Where("id = ? AND used_count < max_uses", tokenID).
			Where("expires_at > ?", time.Now()).
			Update("used_count", gorm.Expr("used_count + 1"))
		if updateResult.Error != nil {
			return updateResult.Error
		}
		if updateResult.RowsAffected == 0 {
			return runner.ErrTokenExhausted
		}

		// Issue certificate via callback (only reached if token claim succeeded)
		if err := issueCert(); err != nil {
			return err
		}

		// Create runner
		if err := tx.Create(rn).Error; err != nil {
			return err
		}

		// Save certificate (runner ID now set by Create)
		cert.RunnerID = rn.ID
		if err := tx.Create(cert).Error; err != nil {
			return err
		}

		// Update runner with certificate info
		return tx.Model(&runner.Runner{}).
			Where("id = ?", rn.ID).
			Updates(map[string]interface{}{
				"cert_serial_number": cert.SerialNumber,
				"cert_expires_at":    cert.ExpiresAt,
			}).Error
	})
}

// --- Reactivation Token ---

func (r *runnerRepository) CreateReactivationToken(ctx context.Context, token *runner.ReactivationToken) error {
	return r.db.WithContext(ctx).Create(token).Error
}

func (r *runnerRepository) GetReactivationTokenByHash(ctx context.Context, hash string) (*runner.ReactivationToken, error) {
	var token runner.ReactivationToken
	if err := r.db.WithContext(ctx).Where("token_hash = ?", hash).First(&token).Error; err != nil {
		if isNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

func (r *runnerRepository) ClaimReactivationToken(ctx context.Context, id int64) (int64, error) {
	now := time.Now()
	result := r.db.WithContext(ctx).Model(&runner.ReactivationToken{}).
		Where("id = ? AND used_at IS NULL AND expires_at > ?", id, now).
		Update("used_at", now)
	return result.RowsAffected, result.Error
}

func (r *runnerRepository) UnclaimReactivationToken(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Model(&runner.ReactivationToken{}).
		Where("id = ?", id).
		Update("used_at", nil).Error
}

func (r *runnerRepository) CleanupExpiredReactivationTokens(ctx context.Context) error {
	return r.db.WithContext(ctx).
		Where("expires_at < ? OR used_at IS NOT NULL", time.Now()).
		Delete(&runner.ReactivationToken{}).Error
}
