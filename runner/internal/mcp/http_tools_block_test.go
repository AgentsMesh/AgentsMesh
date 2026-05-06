package mcp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Block Store MCP tool registration — regression coverage for the 12 tools
// (6 primitives + 2 sugar + memory.retrieve + block.list_types + 2 workspace
// discovery) that a pod's tools/list response must carry. A missing tool here
// means agents in pods stop seeing that capability; the check is cheap and
// the registration list is one-line-add drift-prone.
//
// Handler-level behaviour is covered by the end-to-end trigger-fire /
// block-crud e2e specs; this file only ensures the registration surface.

func TestRegisterTools_IncludesAllBlockTools(t *testing.T) {
	server := NewHTTPServer(nil, 9199)
	expected := []string{
		"block.create",
		"block.update",
		"block.delete",
		"block.add_ref",
		"block.remove_ref",
		"block.update_ref",
		"indicator.define",
		"trigger.define",
		"memory.retrieve",
		"block.list_types",
		"block.list_workspaces",
		"block.get_default_workspace",
	}
	have := make(map[string]bool, len(server.tools))
	for _, tool := range server.tools {
		have[tool.Name] = true
	}
	for _, name := range expected {
		assert.True(t, have[name], "block tool %q missing from registered set", name)
	}
}

func TestBlockTools_SchemaShape(t *testing.T) {
	// Spot-check the schema on one ops tool and one read tool; a regression
	// in how InputSchema is constructed (e.g. missing "required" list or
	// wrong property type) will show here before it breaks an agent at
	// runtime.
	srv := NewHTTPServer(nil, 9199)
	byName := map[string]*MCPTool{}
	for _, t := range srv.tools {
		byName[t.Name] = t
	}

	create := byName["block.create"]
	if assert.NotNil(t, create, "block.create missing") {
		schema := create.InputSchema
		assert.Equal(t, "object", schema["type"])
		required, _ := schema["required"].([]string)
		assert.Contains(t, required, "workspace_id")
		assert.Contains(t, required, "payload")
	}

	listTypes := byName["block.list_types"]
	if assert.NotNil(t, listTypes, "block.list_types missing") {
		schema := listTypes.InputSchema
		required, _ := schema["required"].([]string)
		assert.Contains(t, required, "workspace_id")
	}

	memoryRetrieve := byName["memory.retrieve"]
	if assert.NotNil(t, memoryRetrieve, "memory.retrieve missing") {
		schema := memoryRetrieve.InputSchema
		required, _ := schema["required"].([]string)
		assert.Contains(t, required, "workspace_id")
		assert.Contains(t, required, "query")
	}
}
