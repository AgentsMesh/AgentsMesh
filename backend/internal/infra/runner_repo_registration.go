package infra

import (
	"context"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/infra/dberr"
	"gorm.io/gorm"
)

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

func (r *runnerRepository) DeleteClaimedPendingAuth(ctx context.Context, id int64) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("id = ? AND authorized = true", id).
		Delete(&runner.PendingAuth{})
	return result.RowsAffected, result.Error
}

func (r *runnerRepository) CleanupExpiredPendingAuths(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&runner.PendingAuth{})
	return result.RowsAffected, result.Error
}

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

		if err := issueCert(); err != nil {
			return err
		}

		if err := tx.Create(rn).Error; err != nil {
			if dberr.IsUniqueViolation(err) {
				return gorm.ErrDuplicatedKey
			}
			return err
		}

		cert.RunnerID = rn.ID
		if err := tx.Create(cert).Error; err != nil {
			return err
		}

		return tx.Model(&runner.Runner{}).
			Where("id = ?", rn.ID).
			Updates(map[string]interface{}{
				"cert_serial_number": cert.SerialNumber,
				"cert_expires_at":    cert.ExpiresAt,
			}).Error
	})
}

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

// ReactivateWithTokenAtomic claims the token, issues+stores a fresh cert, and
// links it to the runner in one transaction. The claim UPDATE locks the token
// row for the tx, so a concurrent purge blocks and any failure rolls the claim
// back — no external unclaim, no settle window. issueCert (local crypto)
// populates cert before it is written.
func (r *runnerRepository) ReactivateWithTokenAtomic(ctx context.Context, tokenID, runnerID int64, cert *runner.Certificate, issueCert func() error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		claim := tx.Model(&runner.ReactivationToken{}).
			Where("id = ? AND used_at IS NULL AND expires_at > ?", tokenID, now).
			Update("used_at", now)
		if claim.Error != nil {
			return claim.Error
		}
		if claim.RowsAffected == 0 {
			return runner.ErrReactivationUnavailable
		}

		if err := issueCert(); err != nil {
			return err
		}

		cert.RunnerID = runnerID
		if err := tx.Create(cert).Error; err != nil {
			return err
		}

		return tx.Model(&runner.Runner{}).
			Where("id = ?", runnerID).
			Updates(map[string]interface{}{
				"cert_serial_number": cert.SerialNumber,
				"cert_expires_at":    cert.ExpiresAt,
			}).Error
	})
}

// Consumed tokens go promptly: each pins its creator via a NO ACTION FK
// (runner_reactivation_tokens.created_by -> users), so retaining them widens the
// window in which deleting that user fails with 23503. A used_at is only ever set
// inside ReactivateWithTokenAtomic's committed tx, so `used_at IS NOT NULL` never
// races an in-flight claim.
func (r *runnerRepository) CleanupExpiredReactivationTokens(ctx context.Context) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at < ? OR used_at IS NOT NULL", time.Now()).
		Delete(&runner.ReactivationToken{})
	return result.RowsAffected, result.Error
}
