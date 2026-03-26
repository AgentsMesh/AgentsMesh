package agent

import (
	"testing"
)

func TestClaudeCodeBuilder_HandleInitialPrompt(t *testing.T) {
	builder := NewClaudeCodeBuilder()

	t.Run("prepends prompt to args", func(t *testing.T) {
		ctx := &BuildContext{
			Request: &ConfigBuildRequest{
				InitialPrompt: "Fix the bug",
			},
		}
		args := []string{"--model", "opus"}

		result := builder.HandleInitialPrompt(ctx, args)

		if len(result) != 3 {
			t.Fatalf("Result length = %d, want 3", len(result))
		}
		if result[0] != "Fix the bug" {
			t.Errorf("First arg = %s, want 'Fix the bug'", result[0])
		}
		if result[1] != "--model" {
			t.Errorf("Second arg = %s, want '--model'", result[1])
		}
	})

	t.Run("returns args unchanged when no prompt", func(t *testing.T) {
		ctx := &BuildContext{
			Request: &ConfigBuildRequest{
				InitialPrompt: "",
			},
		}
		args := []string{"--model", "opus"}

		result := builder.HandleInitialPrompt(ctx, args)

		if len(result) != 2 {
			t.Errorf("Result length = %d, want 2", len(result))
		}
	})
}

func TestGeminiCLIBuilder_HandleInitialPrompt(t *testing.T) {
	builder := NewGeminiCLIBuilder()

	t.Run("appends prompt to args", func(t *testing.T) {
		ctx := &BuildContext{
			Request: &ConfigBuildRequest{
				InitialPrompt: "Fix the bug",
			},
		}
		args := []string{"--sandbox"}

		result := builder.HandleInitialPrompt(ctx, args)

		if len(result) != 2 {
			t.Fatalf("Result length = %d, want 2", len(result))
		}
		if result[0] != "--sandbox" {
			t.Errorf("First arg = %s, want '--sandbox'", result[0])
		}
		if result[1] != "Fix the bug" {
			t.Errorf("Last arg = %s, want 'Fix the bug'", result[1])
		}
	})

	t.Run("returns args unchanged when no prompt", func(t *testing.T) {
		ctx := &BuildContext{
			Request: &ConfigBuildRequest{
				InitialPrompt: "",
			},
		}
		args := []string{"--sandbox"}

		result := builder.HandleInitialPrompt(ctx, args)

		if len(result) != 1 {
			t.Errorf("Result length = %d, want 1", len(result))
		}
	})
}

func TestAiderBuilder_HandleInitialPrompt(t *testing.T) {
	builder := NewAiderBuilder()

	t.Run("ignores prompt", func(t *testing.T) {
		ctx := &BuildContext{
			Request: &ConfigBuildRequest{
				InitialPrompt: "Fix the bug",
			},
		}
		args := []string{"--model", "gpt-4"}

		result := builder.HandleInitialPrompt(ctx, args)

		// Aider should ignore the prompt and return args unchanged
		if len(result) != 2 {
			t.Fatalf("Result length = %d, want 2", len(result))
		}
		if result[0] != "--model" || result[1] != "gpt-4" {
			t.Errorf("Args should be unchanged, got %v", result)
		}
	})
}

func TestOpenCodeBuilder_HandleInitialPrompt(t *testing.T) {
	builder := NewOpenCodeBuilder()

	t.Run("passes prompt via --prompt flag", func(t *testing.T) {
		ctx := &BuildContext{
			Request: &ConfigBuildRequest{
				InitialPrompt: "Fix the bug",
			},
		}
		args := []string{"--model", "anthropic/claude-sonnet-4"}

		result := builder.HandleInitialPrompt(ctx, args)

		if len(result) != 4 {
			t.Fatalf("Result length = %d, want 4", len(result))
		}
		if result[0] != "--prompt" {
			t.Errorf("First arg = %s, want '--prompt'", result[0])
		}
		if result[1] != "Fix the bug" {
			t.Errorf("Second arg = %s, want 'Fix the bug'", result[1])
		}
		if result[2] != "--model" {
			t.Errorf("Third arg = %s, want '--model'", result[2])
		}
	})

	t.Run("returns args unchanged when no prompt", func(t *testing.T) {
		ctx := &BuildContext{
			Request: &ConfigBuildRequest{
				InitialPrompt: "",
			},
		}
		args := []string{"--model", "anthropic/claude-sonnet-4"}

		result := builder.HandleInitialPrompt(ctx, args)

		if len(result) != 2 {
			t.Errorf("Result length = %d, want 2", len(result))
		}
	})
}

func TestCodexCLIBuilder_HandleInitialPrompt(t *testing.T) {
	builder := NewCodexCLIBuilder()

	t.Run("prepends prompt to args", func(t *testing.T) {
		ctx := &BuildContext{
			Request: &ConfigBuildRequest{
				InitialPrompt: "Fix the bug",
			},
		}
		args := []string{"--approval-mode", "auto-edit"}

		result := builder.HandleInitialPrompt(ctx, args)

		if len(result) != 3 {
			t.Fatalf("Result length = %d, want 3", len(result))
		}
		if result[0] != "Fix the bug" {
			t.Errorf("First arg = %s, want 'Fix the bug'", result[0])
		}
	})
}
