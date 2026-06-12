package runner

import (
	"reflect"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

func TestParseClaudeInitialConfig(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want acp.Configuration
	}{
		{
			name: "claude with permission-mode and model",
			args: []string{"-p", "--input-format", "stream-json", "--permission-mode", "bypassPermissions", "--model", "sonnet"},
			want: acp.Configuration{PermissionMode: "bypassPermissions", Model: "sonnet"},
		},
		{
			name: "claude with only permission-mode",
			args: []string{"-p", "--permission-mode", "plan"},
			want: acp.Configuration{PermissionMode: "plan"},
		},
		{
			name: "codex without claude flags",
			args: []string{"app-server"},
			want: acp.Configuration{},
		},
		{
			name: "empty args",
			args: nil,
			want: acp.Configuration{},
		},
		{
			name: "trailing flag without value is ignored",
			args: []string{"--permission-mode"},
			want: acp.Configuration{},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := parseClaudeInitialConfig(tc.args)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}
