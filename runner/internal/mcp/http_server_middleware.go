package mcp

import (
	"fmt"
)

// registerTools registers all collaboration tools.
func (s *HTTPServer) registerTools() {
	s.tools = []*MCPTool{
		// Pod interaction tools
		s.createGetPodSnapshotTool(),
		s.createSendPodInputTool(),
		s.createGetPodStatusTool(),

		// Discovery tools
		s.createListAvailablePodsTool(),
		s.createListRunnersTool(),
		s.createListRepositoriesTool(),

		// Binding tools
		s.createBindPodTool(),
		s.createAcceptBindingTool(),
		s.createRejectBindingTool(),
		s.createUnbindPodTool(),
		s.createGetBindingsTool(),
		s.createGetBoundPodsTool(),

		// Channel tools
		s.createSearchChannelsTool(),
		s.createCreateChannelTool(),
		s.createGetChannelTool(),
		s.createSendChannelMessageTool(),
		s.createGetChannelMessagesTool(),
		s.createGetChannelDocumentTool(),
		s.createUpdateChannelDocumentTool(),

		// Ticket tools
		s.createSearchTicketsTool(),
		s.createGetTicketTool(),
		s.createCreateTicketTool(),
		s.createUpdateTicketTool(),
		s.createDeleteTicketTool(),
		s.createPostCommentTool(),

		// Pod tools
		s.createCreatePodTool(),

		// Loop tools
		s.createListLoopsTool(),
		s.createTriggerLoopTool(),
	}
}

// GenerateMCPConfig generates the MCP configuration JSON for Claude Code.
func (s *HTTPServer) GenerateMCPConfig(podKey string) map[string]interface{} {
	return map[string]interface{}{
		"mcpServers": map[string]interface{}{
			"agentsmesh-collaboration": map[string]interface{}{
				"command": "curl",
				"args": []string{
					"-X", "POST",
					"-H", "Content-Type: application/json",
					"-H", fmt.Sprintf("X-Pod-Key: %s", podKey),
					fmt.Sprintf("http://localhost:%d/mcp", s.port),
					"-d", "@-",
				},
			},
		},
	}
}
