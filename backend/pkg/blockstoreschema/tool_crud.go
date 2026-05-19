package blockstoreschema

import "github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"

func createBlockTool(typeKeys []string) Tool {
	return Tool{
		Name:        blockstore.OpCreateBlock,
		Description: "Create a new block inside a workspace. Returns the new block's id.",
		InputSchema: schemaObject(map[string]any{
			"id":   map[string]any{"type": "string", "format": "uuid", "description": "Optional pre-assigned block id (uuid_v4 or v7)."},
			"type": map[string]any{"type": "string", "enum": typeKeys, "description": "Block type key."},
			"data": map[string]any{"type": "object", "description": "Type-specific JSON payload."},
			"text": map[string]any{"type": "string", "description": "Plain-text summary used for full-text search."},
			"meta": map[string]any{"type": "object", "description": "Metadata (ACL, tags, extensions)."},
		}, []string{"type"}),
	}
}

func updateBlockTool() Tool {
	return Tool{
		Name:        blockstore.OpUpdateBlock,
		Description: "Update an existing block's data / text / meta. Pass expected_updated_at for optimistic concurrency.",
		InputSchema: schemaObject(map[string]any{
			"id":                  map[string]any{"type": "string", "format": "uuid"},
			"data":                map[string]any{"type": "object"},
			"text":                map[string]any{"type": "string"},
			"meta":                map[string]any{"type": "object"},
			"expected_updated_at": map[string]any{"type": "string", "format": "date-time"},
		}, []string{"id"}),
	}
}

func deleteBlockTool() Tool {
	return Tool{
		Name:        blockstore.OpDeleteBlock,
		Description: "Soft-delete a block. Incoming refs are not cascaded; consumers should treat dangling tos as tombstones.",
		InputSchema: schemaObject(map[string]any{
			"id": map[string]any{"type": "string", "format": "uuid"},
		}, []string{"id"}),
	}
}

func addRefTool() Tool {
	return Tool{
		Name:        blockstore.OpAddRef,
		Description: "Create a relationship from one block to another. Use rel='nest' (with order_key) to place a child under a parent; use other rels for mentions, dependencies, etc.",
		InputSchema: schemaObject(map[string]any{
			"from": map[string]any{"type": "string", "format": "uuid"},
			"to":   map[string]any{"type": "string", "format": "uuid"},
			"rel": map[string]any{
				"type":        "string",
				"description": "Relation type. 'nest' requires order_key and enforces single parent.",
				"examples":    []string{blockstore.RelNest, blockstore.RelMention, blockstore.RelEmbed, blockstore.RelDependsOn},
			},
			"order_key": map[string]any{"type": "string", "description": "Fractional index (required for rel='nest')."},
			"anchor":    map[string]any{"type": "string", "description": "Optional inner-position anchor."},
		}, []string{"from", "to", "rel"}),
	}
}

func removeRefTool() Tool {
	return Tool{
		Name:        blockstore.OpRemoveRef,
		Description: "Remove a block-to-block ref by its id.",
		InputSchema: schemaObject(map[string]any{
			"ref_id": map[string]any{"type": "integer"},
		}, []string{"ref_id"}),
	}
}

func updateRefTool() Tool {
	return Tool{
		Name:        blockstore.OpUpdateRef,
		Description: "Reposition or re-annotate an existing ref (change parent, order_key, anchor, or meta). For rel='nest' this is the canonical 'move' operation.",
		InputSchema: schemaObject(map[string]any{
			"ref_id":    map[string]any{"type": "integer"},
			"from":      map[string]any{"type": "string", "format": "uuid"},
			"order_key": map[string]any{"type": "string"},
			"anchor":    map[string]any{"type": "string"},
			"meta":      map[string]any{"type": "object", "description": "Merge / replace the ref's meta (e.g. comment resolved=true)."},
		}, []string{"ref_id"}),
	}
}
