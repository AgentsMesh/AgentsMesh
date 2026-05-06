package mcp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

// Workspace discovery tools — verify the registration shape and that handlers
// forward to the BlockStoreClient. Keeps the workspace-id resolution loop
// honest: agents see exactly two zero-arg discovery tools, no surprise
// required fields.

func TestBlockWorkspaceTools_SchemaIsZeroArg(t *testing.T) {
	srv := NewHTTPServer(nil, 9199)
	byName := map[string]*MCPTool{}
	for _, tool := range srv.tools {
		byName[tool.Name] = tool
	}

	for _, name := range []string{"block.list_workspaces", "block.get_default_workspace"} {
		tool := byName[name]
		require.NotNil(t, tool, "%s must be registered", name)
		schema := tool.InputSchema
		assert.Equal(t, "object", schema["type"], "%s.type", name)
		props, ok := schema["properties"].(map[string]interface{})
		require.True(t, ok, "%s.properties must be a map", name)
		assert.Empty(t, props, "%s must have no properties — agents call it with {}", name)
		_, hasRequired := schema["required"]
		assert.False(t, hasRequired, "%s must not declare required fields", name)
	}
}

func TestBlockWorkspaceTools_HandlersForward(t *testing.T) {
	srv := NewHTTPServer(nil, 9199)
	byName := map[string]*MCPTool{}
	for _, tool := range srv.tools {
		byName[tool.Name] = tool
	}

	t.Run("list_workspaces forwards to BlockListWorkspaces", func(t *testing.T) {
		stub := &workspaceClientStub{listResult: map[string]interface{}{"workspaces": []any{}}}
		tool := byName["block.list_workspaces"]
		require.NotNil(t, tool)
		got, err := tool.Handler(context.Background(), stub, map[string]interface{}{})
		require.NoError(t, err)
		assert.True(t, stub.listCalled, "BlockListWorkspaces must be invoked")
		assert.Equal(t, stub.listResult, got)
	})

	t.Run("get_default_workspace forwards to BlockGetDefaultWorkspace", func(t *testing.T) {
		stub := &workspaceClientStub{defaultResult: map[string]interface{}{"id": "uuid-x", "slug": "default"}}
		tool := byName["block.get_default_workspace"]
		require.NotNil(t, tool)
		got, err := tool.Handler(context.Background(), stub, map[string]interface{}{})
		require.NoError(t, err)
		assert.True(t, stub.defaultCalled, "BlockGetDefaultWorkspace must be invoked")
		assert.Equal(t, stub.defaultResult, got)
	})
}

// workspaceClientStub embeds an unimplemented client and overrides only the
// two methods under test. Other CollaborationClient methods will panic if
// invoked, surfacing accidental coupling.
type workspaceClientStub struct {
	tools.CollaborationClient
	listCalled    bool
	listResult    map[string]interface{}
	defaultCalled bool
	defaultResult map[string]interface{}
}

func (s *workspaceClientStub) BlockListWorkspaces(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	s.listCalled = true
	return s.listResult, nil
}

func (s *workspaceClientStub) BlockGetDefaultWorkspace(ctx context.Context, args map[string]interface{}) (map[string]interface{}, error) {
	s.defaultCalled = true
	return s.defaultResult, nil
}
