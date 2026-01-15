package agent

import (
	"strings"
	"testing"

	"github.com/anthropics/agentmesh/backend/internal/domain/agent"
)

func TestConfigBuilder_buildFilesToCreate(t *testing.T) {
	db := setupConfigBuilderTestDB(t)
	provider := createTestProvider(db)
	builder := NewConfigBuilder(provider)

	tests := []struct {
		name          string
		filesTemplate agent.FilesTemplate
		config        agent.ConfigValues
		templateCtx   map[string]interface{}
		wantCount     int
		wantErr       bool
	}{
		{
			name:          "empty files template",
			filesTemplate: agent.FilesTemplate{},
			config:        agent.ConfigValues{},
			templateCtx:   map[string]interface{}{},
			wantCount:     0,
			wantErr:       false,
		},
		{
			name: "single file",
			filesTemplate: agent.FilesTemplate{
				{
					PathTemplate:    "/tmp/test.txt",
					ContentTemplate: "Hello World",
					Mode:            0644,
				},
			},
			config:      agent.ConfigValues{},
			templateCtx: map[string]interface{}{},
			wantCount:   1,
			wantErr:     false,
		},
		{
			name: "directory",
			filesTemplate: agent.FilesTemplate{
				{
					PathTemplate: "/tmp/testdir",
					IsDirectory:  true,
				},
			},
			config:      agent.ConfigValues{},
			templateCtx: map[string]interface{}{},
			wantCount:   1,
			wantErr:     false,
		},
		{
			name: "file with condition met",
			filesTemplate: agent.FilesTemplate{
				{
					Condition: &agent.Condition{
						Field:    "mcp_enabled",
						Operator: "eq",
						Value:    true,
					},
					PathTemplate:    "/tmp/mcp.json",
					ContentTemplate: `{"enabled":true}`,
					Mode:            0600,
				},
			},
			config:      agent.ConfigValues{"mcp_enabled": true},
			templateCtx: map[string]interface{}{},
			wantCount:   1,
			wantErr:     false,
		},
		{
			name: "file with condition not met",
			filesTemplate: agent.FilesTemplate{
				{
					Condition: &agent.Condition{
						Field:    "mcp_enabled",
						Operator: "eq",
						Value:    true,
					},
					PathTemplate:    "/tmp/mcp.json",
					ContentTemplate: `{"enabled":true}`,
				},
			},
			config:      agent.ConfigValues{"mcp_enabled": false},
			templateCtx: map[string]interface{}{},
			wantCount:   0,
			wantErr:     false,
		},
		{
			name: "file with content template",
			filesTemplate: agent.FilesTemplate{
				{
					PathTemplate:    "/tmp/config.json",
					ContentTemplate: `{"model":"{{.config.model}}"}`,
				},
			},
			config: agent.ConfigValues{"model": "opus"},
			templateCtx: map[string]interface{}{
				"config": map[string]interface{}{"model": "opus"},
			},
			wantCount: 1,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builder.buildFilesToCreate(tt.filesTemplate, tt.config, tt.templateCtx)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildFilesToCreate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != tt.wantCount {
				t.Errorf("buildFilesToCreate() count = %d, want %d", len(got), tt.wantCount)
			}
		})
	}
}

func TestConfigBuilder_buildFilesToCreate_ErrorPath(t *testing.T) {
	db := setupConfigBuilderTestDB(t)
	provider := createTestProvider(db)
	builder := NewConfigBuilder(provider)

	t.Run("returns error on content template render failure", func(t *testing.T) {
		filesTemplate := agent.FilesTemplate{
			{
				PathTemplate:    "/tmp/test.txt",
				ContentTemplate: "{{.invalid",
			},
		}
		config := agent.ConfigValues{}
		templateCtx := map[string]interface{}{}

		_, err := builder.buildFilesToCreate(filesTemplate, config, templateCtx)
		if err == nil {
			t.Error("Expected error for invalid content template")
		}
		if !strings.Contains(err.Error(), "failed to render content template") {
			t.Errorf("Error should contain 'failed to render content template', got: %v", err)
		}
	})

	t.Run("uses default mode 0644 when mode is 0", func(t *testing.T) {
		filesTemplate := agent.FilesTemplate{
			{
				PathTemplate:    "/tmp/test.txt",
				ContentTemplate: "content",
				Mode:            0,
			},
		}
		config := agent.ConfigValues{}
		templateCtx := map[string]interface{}{}

		files, err := builder.buildFilesToCreate(filesTemplate, config, templateCtx)
		if err != nil {
			t.Fatalf("buildFilesToCreate failed: %v", err)
		}

		if len(files) != 1 {
			t.Fatalf("Expected 1 file, got %d", len(files))
		}
		if files[0].Mode != 0644 {
			t.Errorf("Mode = %o, want 0644", files[0].Mode)
		}
	})
}
