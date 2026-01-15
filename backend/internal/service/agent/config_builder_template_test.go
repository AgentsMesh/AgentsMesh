package agent

import (
	"strings"
	"testing"

	"github.com/anthropics/agentmesh/backend/internal/domain/agent"
)

func TestConfigBuilder_renderTemplate(t *testing.T) {
	db := setupConfigBuilderTestDB(t)
	provider := createTestProvider(db)
	builder := NewConfigBuilder(provider)

	tests := []struct {
		name     string
		template string
		ctx      map[string]interface{}
		want     string
		wantErr  bool
	}{
		{
			name:     "no template markers",
			template: "simple string",
			ctx:      nil,
			want:     "simple string",
			wantErr:  false,
		},
		{
			name:     "simple variable",
			template: "Hello {{.name}}",
			ctx:      map[string]interface{}{"name": "World"},
			want:     "Hello World",
			wantErr:  false,
		},
		{
			name:     "nested variable",
			template: "Model: {{.config.model}}",
			ctx:      map[string]interface{}{"config": map[string]interface{}{"model": "opus"}},
			want:     "Model: opus",
			wantErr:  false,
		},
		{
			name:     "multiple variables",
			template: "{{.key1}}-{{.key2}}",
			ctx:      map[string]interface{}{"key1": "a", "key2": "b"},
			want:     "a-b",
			wantErr:  false,
		},
		{
			name:     "invalid template syntax",
			template: "{{.invalid",
			ctx:      nil,
			want:     "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builder.renderTemplate(tt.template, tt.ctx)
			if (err != nil) != tt.wantErr {
				t.Errorf("renderTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("renderTemplate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfigBuilder_renderTemplate_ExecutionError(t *testing.T) {
	db := setupConfigBuilderTestDB(t)
	provider := createTestProvider(db)
	builder := NewConfigBuilder(provider)

	t.Run("handles missing keys gracefully", func(t *testing.T) {
		templateStr := "{{.missing}}"
		ctx := map[string]interface{}{}

		result, err := builder.renderTemplate(templateStr, ctx)
		if err != nil {
			t.Fatalf("renderTemplate failed: %v", err)
		}
		if result != "<no value>" {
			t.Errorf("result = %s, want '<no value>'", result)
		}
	})

	t.Run("handles nil context", func(t *testing.T) {
		templateStr := "static content"
		result, err := builder.renderTemplate(templateStr, nil)
		if err != nil {
			t.Fatalf("renderTemplate failed: %v", err)
		}
		if result != "static content" {
			t.Errorf("result = %s, want 'static content'", result)
		}
	})

	t.Run("handles complex nested template", func(t *testing.T) {
		templateStr := `{"port":{{.mcp_port}},"key":"{{.pod_key}}"}`
		ctx := map[string]interface{}{
			"mcp_port": 19000,
			"pod_key":  "test-123",
		}

		result, err := builder.renderTemplate(templateStr, ctx)
		if err != nil {
			t.Fatalf("renderTemplate failed: %v", err)
		}
		expected := `{"port":19000,"key":"test-123"}`
		if result != expected {
			t.Errorf("result = %s, want %s", result, expected)
		}
	})
}

func TestConfigBuilder_buildLaunchArgs(t *testing.T) {
	db := setupConfigBuilderTestDB(t)
	provider := createTestProvider(db)
	builder := NewConfigBuilder(provider)

	tests := []struct {
		name        string
		cmdTemplate agent.CommandTemplate
		config      agent.ConfigValues
		templateCtx map[string]interface{}
		wantArgs    []string
		wantErr     bool
	}{
		{
			name:        "empty command template",
			cmdTemplate: agent.CommandTemplate{},
			config:      agent.ConfigValues{},
			templateCtx: map[string]interface{}{},
			wantArgs:    nil,
			wantErr:     false,
		},
		{
			name: "simple args without condition",
			cmdTemplate: agent.CommandTemplate{
				Args: []agent.ArgRule{
					{Args: []string{"--verbose"}},
				},
			},
			config:      agent.ConfigValues{},
			templateCtx: map[string]interface{}{},
			wantArgs:    []string{"--verbose"},
			wantErr:     false,
		},
		{
			name: "args with template",
			cmdTemplate: agent.CommandTemplate{
				Args: []agent.ArgRule{
					{Args: []string{"--model", "{{.config.model}}"}},
				},
			},
			config: agent.ConfigValues{"model": "opus"},
			templateCtx: map[string]interface{}{
				"config": map[string]interface{}{"model": "opus"},
			},
			wantArgs: []string{"--model", "opus"},
			wantErr:  false,
		},
		{
			name: "args with condition met",
			cmdTemplate: agent.CommandTemplate{
				Args: []agent.ArgRule{
					{
						Condition: &agent.Condition{
							Field:    "enabled",
							Operator: "eq",
							Value:    true,
						},
						Args: []string{"--feature"},
					},
				},
			},
			config:      agent.ConfigValues{"enabled": true},
			templateCtx: map[string]interface{}{},
			wantArgs:    []string{"--feature"},
			wantErr:     false,
		},
		{
			name: "args with condition not met",
			cmdTemplate: agent.CommandTemplate{
				Args: []agent.ArgRule{
					{
						Condition: &agent.Condition{
							Field:    "enabled",
							Operator: "eq",
							Value:    true,
						},
						Args: []string{"--feature"},
					},
				},
			},
			config:      agent.ConfigValues{"enabled": false},
			templateCtx: map[string]interface{}{},
			wantArgs:    nil,
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := builder.buildLaunchArgs(tt.cmdTemplate, tt.config, tt.templateCtx)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildLaunchArgs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(got) != len(tt.wantArgs) {
				t.Errorf("buildLaunchArgs() = %v, want %v", got, tt.wantArgs)
				return
			}
			for i, arg := range tt.wantArgs {
				if got[i] != arg {
					t.Errorf("buildLaunchArgs()[%d] = %v, want %v", i, got[i], arg)
				}
			}
		})
	}
}

func TestConfigBuilder_buildLaunchArgs_ErrorPath(t *testing.T) {
	db := setupConfigBuilderTestDB(t)
	provider := createTestProvider(db)
	builder := NewConfigBuilder(provider)

	t.Run("returns error on template render failure", func(t *testing.T) {
		cmdTemplate := agent.CommandTemplate{
			Args: []agent.ArgRule{
				{Args: []string{"--model", "{{.invalid"}},
			},
		}
		config := agent.ConfigValues{}
		templateCtx := map[string]interface{}{}

		_, err := builder.buildLaunchArgs(cmdTemplate, config, templateCtx)
		if err == nil {
			t.Error("Expected error for invalid template")
		}
		if !strings.Contains(err.Error(), "failed to render arg template") {
			t.Errorf("Error should contain 'failed to render arg template', got: %v", err)
		}
	})

	t.Run("skips empty rendered args", func(t *testing.T) {
		cmdTemplate := agent.CommandTemplate{
			Args: []agent.ArgRule{
				{Args: []string{"--model", "{{.config.model}}"}},
			},
		}
		config := agent.ConfigValues{"model": ""}
		templateCtx := map[string]interface{}{
			"config": map[string]interface{}{"model": ""},
		}

		args, err := builder.buildLaunchArgs(cmdTemplate, config, templateCtx)
		if err != nil {
			t.Fatalf("buildLaunchArgs failed: %v", err)
		}

		found := false
		for _, arg := range args {
			if arg == "" {
				found = true
			}
		}
		if found {
			t.Error("Empty rendered args should be skipped")
		}
	})
}
