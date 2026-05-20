package main

import (
	"context"
	"fmt"
	"log/slog"

	"gorm.io/gorm"

	"github.com/anthropics/agentsmesh/backend/pkg/slugkit"
)

type orgScopedRow struct {
	ID    int64
	OrgID int64
	Name  string
}

func backfillChannelsSlug(ctx context.Context, db *gorm.DB, dryRun bool) error {
	return backfillNullableSlug(ctx, db, dryRun, "channels")
}

func backfillApiKeysSlug(ctx context.Context, db *gorm.DB, dryRun bool) error {
	return backfillNullableSlug(ctx, db, dryRun, "api_keys")
}

// backfillNullableSlug populates a NULL slug column from the row's name,
// guaranteeing uniqueness within (organization_id, slug). Used for channels
// and api_keys, both of which followed the "name doubles as identifier"
// anti-pattern before Phase 2.
func backfillNullableSlug(ctx context.Context, db *gorm.DB, dryRun bool, table string) error {
	var rows []orgScopedRow
	scanSQL := fmt.Sprintf(`SELECT id, organization_id AS org_id, name FROM %s WHERE slug IS NULL`, table)
	if err := db.WithContext(ctx).Raw(scanSQL).Scan(&rows).Error; err != nil {
		return fmt.Errorf("scan %s: %w", table, err)
	}

	slog.Info("scan complete", "table", table+".slug", "null_rows", len(rows), "dry_run", dryRun)
	for _, r := range rows {
		newSlug, err := deriveOrgScopedSlug(ctx, db, table, r)
		if err != nil {
			slog.Error("derive failed", "table", table, "row_id", r.ID, "name", r.Name, "error", err)
			return err
		}
		slog.Info("populate", "table", table, "row_id", r.ID, "name", r.Name, "slug", newSlug, "dry_run", dryRun)
		if dryRun {
			continue
		}
		if err := applyOrgScopedSlug(ctx, db, table, r.ID, r.Name, newSlug); err != nil {
			return err
		}
	}
	return nil
}

func deriveOrgScopedSlug(ctx context.Context, db *gorm.DB, table string, r orgScopedRow) (string, error) {
	existsSQL := fmt.Sprintf(`SELECT count(*) FROM %s WHERE organization_id = ? AND slug = ?`, table)
	check := slugkit.FromExistsCheck(func(ctx context.Context, candidate string) (bool, error) {
		var n int64
		err := db.WithContext(ctx).Raw(existsSQL, r.OrgID, candidate).Scan(&n).Error
		return n > 0, err
	})
	if s, ok := slugkit.TrySeeds(ctx, []string{r.Name, fmt.Sprintf("%s-%d", table, r.ID)}, check); ok {
		return s, nil
	}
	return "", fmt.Errorf("could not derive %s.slug for id=%d", table, r.ID)
}
