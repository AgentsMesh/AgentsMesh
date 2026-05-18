package migrations

import (
	"strings"
	"testing"
)

// TestMigration000133AgentUsesLegacyColumns verifies that migration 000133
// adds the uses_legacy_columns column and backfills the Claude-family agents.
// This is a static SQL audit — it does not require running migrations against
// a database. The intent is to catch accidental edits to the embedded SQL.
func TestMigration000133AgentUsesLegacyColumns(t *testing.T) {
	up, err := FS.ReadFile("000133_agent_uses_legacy_columns.up.sql")
	if err != nil {
		t.Fatalf("read up migration: %v", err)
	}
	upStr := string(up)

	if !strings.Contains(upStr, "ADD COLUMN uses_legacy_columns") {
		t.Error("up migration must add uses_legacy_columns column")
	}
	if !strings.Contains(upStr, "claude-code") {
		t.Error("up migration must backfill claude-code")
	}
	if !strings.Contains(upStr, "'claude'") {
		t.Error("up migration must backfill 'claude' (Claude family)")
	}
	if !strings.Contains(upStr, "uses_legacy_columns = TRUE") {
		t.Error("up migration must SET uses_legacy_columns = TRUE")
	}

	down, err := FS.ReadFile("000133_agent_uses_legacy_columns.down.sql")
	if err != nil {
		t.Fatalf("read down migration: %v", err)
	}
	if !strings.Contains(string(down), "DROP COLUMN") {
		t.Error("down migration must drop the column")
	}
}
