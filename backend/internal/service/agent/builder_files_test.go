package agent

import (
	"testing"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/agent"
	runnerv1 "github.com/AgentsMesh/AgentsMesh/proto/gen/go/runner/v1"
)

func TestBaseAgentBuilder_BuildFilesToCreate(t *testing.T) {
	builder := NewBaseAgentBuilder("test")

	t.Run("builds files from template", func(t *testing.T) {
		ctx := &BuildContext{
			AgentType: &agent.AgentType{
				FilesTemplate: agent.FilesTemplate{
					{
						PathTemplate:    "/tmp/config.json",
						ContentTemplate: `{"key":"value"}`,
						Mode:            0600,
					},
				},
			},
			Config:      agent.ConfigValues{},
			TemplateCtx: map[string]interface{}{},
		}

		files, err := builder.BuildFilesToCreate(ctx)
		if err != nil {
			t.Fatalf("BuildFilesToCreate failed: %v", err)
		}

		if len(files) != 1 {
			t.Fatalf("Files length = %d, want 1", len(files))
		}
		if files[0].Path != "/tmp/config.json" {
			t.Errorf("Path = %s, want /tmp/config.json", files[0].Path)
		}
		if files[0].Content != `{"key":"value"}` {
			t.Errorf("Content = %s, unexpected", files[0].Content)
		}
		if files[0].Mode != 0600 {
			t.Errorf("Mode = %o, want 0600", files[0].Mode)
		}
	})

	t.Run("creates directories", func(t *testing.T) {
		ctx := &BuildContext{
			AgentType: &agent.AgentType{
				FilesTemplate: agent.FilesTemplate{
					{
						PathTemplate: "/tmp/mydir",
						IsDirectory:  true,
					},
				},
			},
			Config:      agent.ConfigValues{},
			TemplateCtx: map[string]interface{}{},
		}

		files, err := builder.BuildFilesToCreate(ctx)
		if err != nil {
			t.Fatalf("BuildFilesToCreate failed: %v", err)
		}

		if len(files) != 1 {
			t.Fatalf("Files length = %d, want 1", len(files))
		}
		if !files[0].IsDirectory {
			t.Error("IsDirectory should be true")
		}
	})

	t.Run("skips files when condition not met", func(t *testing.T) {
		ctx := &BuildContext{
			AgentType: &agent.AgentType{
				FilesTemplate: agent.FilesTemplate{
					{
						Condition: &agent.Condition{
							Field:    "mcp_enabled",
							Operator: "eq",
							Value:    true,
						},
						PathTemplate:    "/tmp/mcp.json",
						ContentTemplate: "{}",
					},
				},
			},
			Config:      agent.ConfigValues{"mcp_enabled": false},
			TemplateCtx: map[string]interface{}{},
		}

		files, err := builder.BuildFilesToCreate(ctx)
		if err != nil {
			t.Fatalf("BuildFilesToCreate failed: %v", err)
		}

		if len(files) != 0 {
			t.Errorf("Files should be empty when condition not met, got %v", files)
		}
	})

	t.Run("uses default mode when not specified", func(t *testing.T) {
		ctx := &BuildContext{
			AgentType: &agent.AgentType{
				FilesTemplate: agent.FilesTemplate{
					{
						PathTemplate:    "/tmp/config.json",
						ContentTemplate: "{}",
						Mode:            0, // Not specified
					},
				},
			},
			Config:      agent.ConfigValues{},
			TemplateCtx: map[string]interface{}{},
		}

		files, err := builder.BuildFilesToCreate(ctx)
		if err != nil {
			t.Fatalf("BuildFilesToCreate failed: %v", err)
		}

		if files[0].Mode != 0644 {
			t.Errorf("Mode = %o, want 0644 (default)", files[0].Mode)
		}
	})
}

func TestBaseAgentBuilder_PostProcess(t *testing.T) {
	builder := NewBaseAgentBuilder("test")

	t.Run("returns nil by default", func(t *testing.T) {
		ctx := &BuildContext{}
		cmd := &runnerv1.CreatePodCommand{}

		err := builder.PostProcess(ctx, cmd)
		if err != nil {
			t.Errorf("PostProcess should return nil, got %v", err)
		}
	})
}
