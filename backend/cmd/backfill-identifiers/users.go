package main

import (
	"context"
	"fmt"
	"log/slog"

	"gorm.io/gorm"

	"github.com/anthropics/agentsmesh/backend/pkg/slugkit"
)

type userRow struct {
	ID       int64
	Username string
}

// backfillUsersUsername rewrites every users.username that violates slugkit
// rules. It loads the full username set once and resolves collisions in
// memory so the inner loop avoids one SELECT per candidate.
func backfillUsersUsername(ctx context.Context, db *gorm.DB, dryRun bool) error {
	var rows []userRow
	err := db.WithContext(ctx).Raw(
		`SELECT id, username FROM users WHERE username !~ '^[a-z0-9]+(-[a-z0-9]+)*$' OR char_length(username) < 2 OR char_length(username) > 100`,
	).Scan(&rows).Error
	if err != nil {
		return fmt.Errorf("scan users: %w", err)
	}

	taken, err := loadUsernameSet(ctx, db)
	if err != nil {
		return fmt.Errorf("load usernames: %w", err)
	}

	slog.Info("scan complete", "table", "users.username", "violations", len(rows), "dry_run", dryRun)
	for _, r := range rows {
		newSlug, err := deriveUsername(ctx, r, taken)
		if err != nil {
			slog.Error("derive failed", "user_id", r.ID, "old", r.Username, "error", err)
			return err
		}
		slog.Info("rewrite", "table", "users", "row_id", r.ID, "old", r.Username, "new", newSlug, "dry_run", dryRun)
		taken[newSlug] = struct{}{}
		delete(taken, r.Username)
		if dryRun {
			continue
		}
		if err := applyUserRewrite(ctx, db, r, newSlug); err != nil {
			return err
		}
	}
	return nil
}

// loadUsernameSet pulls every existing username into memory. Safe here
// because this is a one-shot backfill CLI (single invocation, large
// transaction window, runs against the live DB once per release); do NOT
// copy this pattern into service-layer hot paths.
func loadUsernameSet(ctx context.Context, db *gorm.DB) (map[string]struct{}, error) {
	var all []string
	if err := db.WithContext(ctx).Raw(`SELECT username FROM users`).Scan(&all).Error; err != nil {
		return nil, err
	}
	set := make(map[string]struct{}, len(all))
	for _, u := range all {
		set[u] = struct{}{}
	}
	return set, nil
}

func deriveUsername(ctx context.Context, r userRow, taken map[string]struct{}) (string, error) {
	check := slugkit.FromExistsCheck(func(_ context.Context, candidate string) (bool, error) {
		if candidate == r.Username {
			return false, nil // the row's own value is not a collision with itself
		}
		_, exists := taken[candidate]
		return exists, nil
	})
	if s, ok := slugkit.TrySeeds(ctx, []string{r.Username, fmt.Sprintf("user-%d", r.ID)}, check); ok {
		return s, nil
	}
	return "", fmt.Errorf("could not derive unique username for id=%d", r.ID)
}

func applyUserRewrite(ctx context.Context, db *gorm.DB, r userRow, newSlug string) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(`UPDATE users SET username = ? WHERE id = ?`, newSlug, r.ID).Error; err != nil {
			return fmt.Errorf("update users: %w", err)
		}
		return auditRewrite(tx, "users", "username", r.ID, r.Username, newSlug)
	})
}
