package blockstore

func registerSystemSpecs(m map[string]BlockTypeSpec) {
	m[BlockTypeTypeDef] = BlockTypeSpec{
		Type:        BlockTypeTypeDef,
		DefaultView: "list",
		RequiredDataKey: []string{"type_key"},
		AllowedChildren: nil,
	}
	m[BlockTypeTriggerDef] = BlockTypeSpec{
		Type:           BlockTypeTriggerDef,
		DefaultView:    "list",
		SupportedViews: []string{"list"},
		RequiredDataKey: []string{"name", "target_type", "on", "action"},
		AllowedChildren: []string{},
	}
	m[BlockTypeAgentEvent] = BlockTypeSpec{
		Type:           BlockTypeAgentEvent,
		DefaultView:    "list",
		SupportedViews: []string{"list"},
		RequiredDataKey: []string{"agent_slug", "trigger_name", "op_kind", "fired_at"},
		AllowedChildren: []string{},
	}
	m[BlockTypeDocument] = BlockTypeSpec{
		Type:           BlockTypeDocument,
		DefaultView:    "document",
		SupportedViews: []string{"document"},
		AllowedChildren: []string{},
	}
}
