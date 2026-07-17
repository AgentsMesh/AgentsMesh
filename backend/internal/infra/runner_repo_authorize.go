package infra

import (
	"context"
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/runner"
	"github.com/anthropics/agentsmesh/backend/internal/infra/dberr"
	"gorm.io/gorm"
)

// AuthorizeAndCreateRunner claims the pending auth, creates the runner, and links
// them atomically. The returned count is the claim's: 0 means another
// authorization won the race and nothing was written.
//
// One transaction rather than three statements, because there is no compensating
// action available afterwards. Claiming flips authorized=true, and the claim
// predicate only matches authorized=false — so any later failure used to burn the
// auth key for the rest of its TTL with no way to retry. And a runner created
// before a failed link could never be handed a certificate, yet still counted
// against the org's quota forever. Rolling back is the unclaim.
// runnerQuotaLockClass namespaces the per-org advisory lock. The two-argument
// advisory-lock space is separate from the single-argument one blockstore uses,
// so these keys never collide with it regardless of value.
const runnerQuotaLockClass = 0x52554e52 // "RUNR"

func (r *runnerRepository) AuthorizeAndCreateRunner(
	ctx context.Context, pendingAuthID, orgID int64, rn *runner.Runner, runnerLimit int,
) (int64, error) {
	var claimed int64

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Enforce the runner quota inside the tx, serialized per org, so a
		// concurrent authorize of a different node cannot overshoot the limit the
		// service checked before the tx. runnerLimit < 0 means unlimited.
		if runnerLimit >= 0 {
			if tx.Name() == "postgres" {
				if err := tx.Exec("SELECT pg_advisory_xact_lock(?, ?)", runnerQuotaLockClass, int32(orgID)).Error; err != nil {
					return err
				}
			}
			var count int64
			if err := tx.Model(&runner.Runner{}).Where("organization_id = ?", orgID).Count(&count).Error; err != nil {
				return err
			}
			if count >= int64(runnerLimit) {
				return runner.ErrRunnerQuotaExceeded
			}
		}

		claim := tx.Model(&runner.PendingAuth{}).
			Where("id = ? AND authorized = false AND expires_at > ?", pendingAuthID, time.Now()).
			Updates(map[string]interface{}{
				"authorized":      true,
				"organization_id": orgID,
			})
		if claim.Error != nil {
			return claim.Error
		}
		claimed = claim.RowsAffected
		if claimed == 0 {
			return nil
		}

		if err := tx.Create(rn).Error; err != nil {
			if dberr.IsUniqueViolation(err) {
				return gorm.ErrDuplicatedKey
			}
			return err
		}

		link := tx.Model(&runner.PendingAuth{}).
			Where("id = ?", pendingAuthID).
			Update("runner_id", rn.ID)
		if link.Error != nil {
			return link.Error
		}
		if link.RowsAffected == 0 {
			return fmt.Errorf("pending auth %d vanished inside its own transaction", pendingAuthID)
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return claimed, nil
}
