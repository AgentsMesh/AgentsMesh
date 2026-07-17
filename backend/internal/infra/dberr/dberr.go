// Package dberr classifies driver-level database errors without committing the
// whole app to gorm's global TranslateError (which replaces *pgconn.PgError
// with a message-less sentinel and would break callers that string-match the
// original text).
package dberr

import "strings"

// IsUniqueViolation reports whether err is a unique-constraint violation. It
// spans both backends the codebase runs on: Postgres (SQLSTATE 23505) in
// production and SQLite ("UNIQUE constraint failed") in the testkit, so the
// same guard is exercised by unit tests and prod alike.
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "SQLSTATE 23505") ||
		strings.Contains(msg, "duplicate key") ||
		strings.Contains(msg, "UNIQUE constraint failed")
}
