package blockstoreschema

func memoryRetrieveTool() Tool {
	return Tool{
		Name:        ToolMemoryRetrieve,
		Description: "Retrieve semantically relevant blocks (notes, tasks, comments) from the workspace. Use for long-term memory lookup before drafting a response.",
		InputSchema: schemaObject(map[string]any{
			"query":     map[string]any{"type": "string", "description": "Free-text query describing what you want to remember."},
			"k":         map[string]any{"type": "integer", "description": "Top-K hits to return (default 5, max 100)."},
			"type":      map[string]any{"type": "string", "description": "Optional: filter to one block type (e.g. 'task', 'comment')."},
			"min_score": map[string]any{"type": "number", "description": "Optional cosine score threshold (default 0)."},
		}, []string{"query"}),
	}
}

func defineIndicatorTool() Tool {
	columnSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"key":         map[string]any{"type": "string"},
			"label":       map[string]any{"type": "string"},
			"type":        map[string]any{"type": "string", "enum": []string{"text", "number", "boolean", "select", "multi_select", "date", "url", "user", "block_ref"}},
			"required":    map[string]any{"type": "boolean"},
			"default":     map[string]any{},
			"options":     map[string]any{"type": "array", "items": map[string]any{"type": "object"}},
			"description": map[string]any{"type": "string"},
		},
		"required": []string{"key", "type"},
	}
	return Tool{
		Name:        ToolDefineIndicator,
		Description: "Register (or revise) a schema-driven indicator type so records of this type can be created, listed, and aggregated. Equivalent to writing a block_type_def block — schema-safe wrapper.",
		InputSchema: schemaObject(map[string]any{
			"type_key":         map[string]any{"type": "string", "description": "Stable identifier (e.g. 'okr', 'incident')."},
			"label":            map[string]any{"type": "string", "description": "Human-readable label shown in UI."},
			"description":      map[string]any{"type": "string", "description": "One-line intent; displayed in slash menu / MCP hints."},
			"revision":         map[string]any{"type": "integer", "description": "Monotonic version. Later revision wins when resolving."},
			"default_view":     map[string]any{"type": "string"},
			"supported_views":  map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"allowed_children": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Whitelist of child types. Empty = any."},
			"columns":          map[string]any{"type": "array", "items": columnSchema, "description": "Field schema; renders the generic RecordEditor."},
		}, []string{"type_key", "columns"}),
	}
}

func defineTriggerTool() Tool {
	actionSchema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"kind":       map[string]any{"type": "string", "enum": []string{"webhook", "agent"}},
			"url":        map[string]any{"type": "string", "description": "Webhook URL (kind=webhook)"},
			"agent_slug": map[string]any{"type": "string", "description": "Agent to invoke (kind=agent)"},
			"headers":    map[string]any{"type": "object", "description": "Optional extra webhook headers"},
		},
		"required": []string{"kind"},
	}
	return Tool{
		Name:        ToolDefineTrigger,
		Description: "Register a reactive rule that fires when blocks of a given type are created / updated / deleted. Example: auto-notify when OKR progress < 0.3.",
		InputSchema: schemaObject(map[string]any{
			"name":        map[string]any{"type": "string", "description": "Stable identifier for logs."},
			"target_type": map[string]any{"type": "string", "description": "Block type to watch (e.g. 'okr', 'incident')."},
			"on":          map[string]any{"type": "string", "enum": []string{"create", "update", "delete"}},
			"predicate":   map[string]any{"type": "string", "description": "Optional condition (e.g. '{progress} < 0.3'). Empty = always fire."},
			"action":      actionSchema,
			"enabled":     map[string]any{"type": "boolean", "description": "Defaults to true."},
		}, []string{"name", "target_type", "on", "action"}),
	}
}
