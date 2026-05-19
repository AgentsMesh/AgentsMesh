package blockstoreschema

import (
	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
)

// Tool mirrors the structure that LLM providers expect for tool-use. Kept
// provider-neutral; adapters wrap this into Anthropic / OpenAI formats at
// call sites.
type Tool struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	InputSchema map[string]any `json:"input_schema"`
}

func AllTools() []Tool {
	return BuildToolsWithTypes(blockstore.BootstrapBlockTypes())
}

func BuildToolsWithTypes(typeKeys []string) []Tool {
	return []Tool{
		createBlockTool(typeKeys),
		updateBlockTool(),
		deleteBlockTool(),
		addRefTool(),
		removeRefTool(),
		updateRefTool(),
		memoryRetrieveTool(),
		defineIndicatorTool(),
		defineTriggerTool(),
	}
}

const ToolMemoryRetrieve = "memory.retrieve"

const ToolDefineIndicator = "indicator.define"

const ToolDefineTrigger = "trigger.define"

func schemaObject(properties map[string]any, required []string) map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": properties,
		"required":   required,
	}
}
