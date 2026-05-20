// Command backfill-identifiers rewrites rows that violate the slugkit
// contract into compliant identifiers. Currently covers:
//
//   - users.username   (post-OAuth-bug regression for emails like
//     kudin.private@gmail.com that slipped through pre-Phase-1)
//   - channels.slug    (NULL pre-Phase-2; populated from channels.name)
//   - api_keys.slug    (NULL pre-Phase-2; populated from api_keys.name)
//
// Each rewrite/population is recorded in identifier_backfill_audit.
//
// Usage:
//
//	backfill-identifiers --dry-run     # report only, no writes (default)
//	backfill-identifiers --apply       # perform rewrites
//	backfill-identifiers --check       # count violations, exit 1 if any
//
// Phase 4 VALIDATE migrations must NOT deploy until --check reports zero
// violations. CI / deploy runbook should run --check as a hard gate.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func main() {
	dryRun := flag.Bool("dry-run", true, "report only, do not modify rows")
	apply := flag.Bool("apply", false, "perform rewrites (overrides --dry-run)")
	check := flag.Bool("check", false, "count violations only; exit 1 if any. For Phase 4 deploy gate")
	dsn := flag.String("dsn", os.Getenv("DATABASE_URL"), "Postgres DSN; defaults to $DATABASE_URL")
	flag.Parse()

	if *apply {
		*dryRun = false
	}
	if *dsn == "" {
		fmt.Fprintln(os.Stderr, "DATABASE_URL or --dsn required")
		os.Exit(2)
	}

	db, err := gorm.Open(postgres.Open(*dsn), &gorm.Config{Logger: logger.Default.LogMode(logger.Warn)})
	if err != nil {
		fmt.Fprintln(os.Stderr, "db open:", err)
		os.Exit(1)
	}

	ctx := context.Background()
	if *check {
		os.Exit(runCheck(ctx, db))
	}

	steps := []func(context.Context, *gorm.DB, bool) error{
		backfillUsersUsername,
		backfillChannelsSlug,
		backfillApiKeysSlug,
	}
	for _, step := range steps {
		if err := step(ctx, db, *dryRun); err != nil {
			fmt.Fprintln(os.Stderr, "backfill failed:", err)
			os.Exit(1)
		}
	}
}

// runCheck counts violations across every backfill target and returns the
// exit code (0 if clean, 1 if any violation, 2 on query error). Used as a
// Phase 4 deploy gate — CI must invoke this before applying VALIDATE
// CONSTRAINT migrations, otherwise the ALTER will scan-fail.
func runCheck(ctx context.Context, db *gorm.DB) int {
	total := 0
	for _, c := range violationCheckers() {
		n, err := c.count(ctx, db)
		if err != nil {
			slog.Error("violation check failed", "target", c.label, "error", err)
			return 2
		}
		slog.Info("violation check", "target", c.label, "count", n)
		total += n
	}
	if total > 0 {
		slog.Error("check failed", "total_violations", total,
			"hint", "run --apply during a maintenance window before deploying Phase 4 VALIDATE migrations")
		return 1
	}
	slog.Info("check passed", "violations", 0)
	return 0
}

type violationChecker struct {
	label string
	count func(ctx context.Context, db *gorm.DB) (int, error)
}

func violationCheckers() []violationChecker {
	return []violationChecker{
		{"users.username", countUsersUsernameViolations},
		{"channels.slug (NULL)", countChannelsSlugNullCount},
		{"api_keys.slug (NULL)", countApiKeysSlugNullCount},
	}
}

func countUsersUsernameViolations(ctx context.Context, db *gorm.DB) (int, error) {
	var n int64
	err := db.WithContext(ctx).Raw(
		`SELECT count(*) FROM users WHERE username !~ '^[a-z0-9]+(-[a-z0-9]+)*$' OR char_length(username) < 2 OR char_length(username) > 100`,
	).Scan(&n).Error
	return int(n), err
}

func countChannelsSlugNullCount(ctx context.Context, db *gorm.DB) (int, error) {
	var n int64
	err := db.WithContext(ctx).Raw(`SELECT count(*) FROM channels WHERE slug IS NULL`).Scan(&n).Error
	return int(n), err
}

func countApiKeysSlugNullCount(ctx context.Context, db *gorm.DB) (int, error) {
	var n int64
	err := db.WithContext(ctx).Raw(`SELECT count(*) FROM api_keys WHERE slug IS NULL`).Scan(&n).Error
	return int(n), err
}

