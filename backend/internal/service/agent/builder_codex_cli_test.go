package agent

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCodexCLIBuilder_BuildLaunchArgs_VersionAdaptation verifies the end-to-end
// integration of command_template rendering + version-specific arg adaptation.
//
// This is the critical path:
//   DB template (--ask-for-approval, latest syntax)
//     → BaseAgentBuilder.BuildLaunchArgs (render template)
//       → AdaptArgsForVersion (downgrade for old versions)
func TestCodexCLIBuilder_BuildLaunchArgs_VersionAdaptation(t *testing.T) {
	// Mirror the DB command_template from migration 000059
	codexAgentType := &agent.AgentType{
		Slug:          CodexCLISlug,
		LaunchCommand: "codex",
		CommandTemplate: agent.CommandTemplate{
			Args: []agent.ArgRule{
				{
					Condition: &agent.Condition{
						Field:    "approval_mode",
						Operator: "not_empty",
					},
					Args: []string{"--ask-for-approval", "{{.config.approval_mode}}"},
				},
			},
		},
	}

	tests := []struct {
		name         string
		approvalMode string
		agentVersion string
		wantArgs     []string
	}{
		{
			name:         "new version keeps --ask-for-approval",
			approvalMode: "suggest",
			agentVersion: "0.1.2025050100",
			wantArgs:     []string{"--ask-for-approval", "suggest"},
		},
		{
			name:         "old version downgrades to --approval-mode",
			approvalMode: "suggest",
			agentVersion: "0.1.2025040100",
			wantArgs:     []string{"--approval-mode", "suggest"},
		},
		{
			name:         "empty version (unknown) - no adaptation, keeps latest syntax",
			approvalMode: "suggest",
			agentVersion: "",
			wantArgs:     []string{"--ask-for-approval", "suggest"},
		},
		{
			name:         "exact threshold version - no adaptation",
			approvalMode: "suggest",
			agentVersion: "0.1.2025042500",
			wantArgs:     []string{"--ask-for-approval", "suggest"},
		},
		{
			name:         "empty approval_mode - condition filters out args",
			approvalMode: "",
			agentVersion: "0.1.2025050100",
			wantArgs:     nil,
		},
		{
			name:         "full-auto mode with old version",
			approvalMode: "full-auto",
			agentVersion: "0.1.2025030100",
			wantArgs:     []string{"--approval-mode", "full-auto"},
		},
		// Codex CLI >= 0.100.0 (Rust rewrite) - approval value mapping
		{
			name:         "rust codex maps suggest to on-request",
			approvalMode: "suggest",
			agentVersion: "0.101.0",
			wantArgs:     []string{"--ask-for-approval", "on-request"},
		},
		{
			name:         "rust codex maps auto-edit to on-failure",
			approvalMode: "auto-edit",
			agentVersion: "0.101.0",
			wantArgs:     []string{"--ask-for-approval", "on-failure"},
		},
		{
			name:         "rust codex maps full-auto to never",
			approvalMode: "full-auto",
			agentVersion: "0.100.0",
			wantArgs:     []string{"--ask-for-approval", "never"},
		},
		{
			name:         "rust codex passes through unknown values",
			approvalMode: "on-request",
			agentVersion: "0.101.0",
			wantArgs:     []string{"--ask-for-approval", "on-request"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewCodexCLIBuilder()

			config := agent.ConfigValues{}
			if tt.approvalMode != "" {
				config["approval_mode"] = tt.approvalMode
			}

			ctx := &BuildContext{
				Request:   &ConfigBuildRequest{},
				AgentType: codexAgentType,
				Config:    config,
				TemplateCtx: map[string]interface{}{
					"config": config,
				},
				AgentVersion: tt.agentVersion,
			}

			args, err := builder.BuildLaunchArgs(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantArgs, args)
		})
	}
}
