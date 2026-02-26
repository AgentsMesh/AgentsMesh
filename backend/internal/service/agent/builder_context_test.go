package agent

import (
	"testing"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agent"
)

func TestBuildContext(t *testing.T) {
	t.Run("NewBuildContext creates context correctly", func(t *testing.T) {
		req := &ConfigBuildRequest{
			AgentTypeID: 1,
			UserID:      2,
		}
		agentType := &agent.AgentType{
			Slug: "test-agent",
		}
		config := agent.ConfigValues{"key": "value"}
		creds := agent.EncryptedCredentials{"secret": "hidden"}
		templateCtx := map[string]interface{}{"template": "data"}

		ctx := NewBuildContext(req, agentType, config, creds, true, templateCtx, "1.2.3")

		if ctx.Request != req {
			t.Error("Request not set correctly")
		}
		if ctx.AgentType != agentType {
			t.Error("AgentType not set correctly")
		}
		if ctx.Config["key"] != "value" {
			t.Error("Config not set correctly")
		}
		if ctx.Credentials["secret"] != "hidden" {
			t.Error("Credentials not set correctly")
		}
		if !ctx.IsRunnerHost {
			t.Error("IsRunnerHost not set correctly")
		}
		if ctx.TemplateCtx["template"] != "data" {
			t.Error("TemplateCtx not set correctly")
		}
		if ctx.AgentVersion != "1.2.3" {
			t.Error("AgentVersion not set correctly")
		}
	})
}
