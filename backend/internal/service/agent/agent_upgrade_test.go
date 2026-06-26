package agent

import (
	"context"
	"testing"
)

func TestGetUpgradeCommand(t *testing.T) {
	db := setupTestDB(t)
	svc := newTestAgentService(db)
	ctx := context.Background()

	t.Run("not configured returns ok=false", func(t *testing.T) {
		_, _, ok, err := svc.GetUpgradeCommand(ctx, "claude-code")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Fatal("expected ok=false for agent without upgrade_args")
		}
	})

	t.Run("configured returns argv + executable", func(t *testing.T) {
		if err := db.Exec(`UPDATE agents SET upgrade_manager = 'npm', upgrade_args = ? WHERE slug = 'claude-code'`,
			`["npm","i","-g","@anthropic-ai/claude-code@latest"]`).Error; err != nil {
			t.Fatalf("seed upgrade_args: %v", err)
		}
		exe, argv, ok, err := svc.GetUpgradeCommand(ctx, "claude-code")
		if err != nil || !ok {
			t.Fatalf("expected ok=true, got ok=%v err=%v", ok, err)
		}
		if exe != "claude" {
			t.Fatalf("executable = %q, want claude", exe)
		}
		want := []string{"npm", "i", "-g", "@anthropic-ai/claude-code@latest"}
		if len(argv) != len(want) {
			t.Fatalf("argv = %v, want %v", argv, want)
		}
		for i := range want {
			if argv[i] != want[i] {
				t.Fatalf("argv[%d] = %q, want %q", i, argv[i], want[i])
			}
		}
	})

	t.Run("non-existent slug returns ok=false without error", func(t *testing.T) {
		_, _, ok, err := svc.GetUpgradeCommand(ctx, "nonexistent")
		if err != nil {
			t.Fatalf("non-existent agent should not error, got %v", err)
		}
		if ok {
			t.Fatal("expected ok=false for non-existent agent")
		}
	})

	t.Run("malformed JSON returns error", func(t *testing.T) {
		if err := db.Exec(`UPDATE agents SET upgrade_manager = 'npm', upgrade_args = ? WHERE slug = 'codex-cli'`, `{not json`).Error; err != nil {
			t.Fatalf("seed bad upgrade_args: %v", err)
		}
		_, _, ok, err := svc.GetUpgradeCommand(ctx, "codex-cli")
		if err == nil || ok {
			t.Fatalf("expected error for malformed JSON, got ok=%v err=%v", ok, err)
		}
	})
}
