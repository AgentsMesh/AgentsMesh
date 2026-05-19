package blockstore

func registerMediaSpecs(m map[string]BlockTypeSpec) {
	m[BlockTypeImage] = BlockTypeSpec{
		Type:            BlockTypeImage,
		DefaultView:     "document",
		SupportedViews:  []string{"document", "gallery"},
		RequiredDataKey: []string{"url"},
		AllowedChildren: []string{},
	}
	m[BlockTypeFile] = BlockTypeSpec{
		Type:            BlockTypeFile,
		DefaultView:     "document",
		SupportedViews:  []string{"document", "list"},
		RequiredDataKey: []string{"url", "name"},
		AllowedChildren: []string{},
	}
	m[BlockTypeVideo] = BlockTypeSpec{
		Type:            BlockTypeVideo,
		DefaultView:     "document",
		SupportedViews:  []string{"document"},
		RequiredDataKey: []string{"url"},
		AllowedChildren: []string{},
	}
	m[BlockTypeEmbed] = BlockTypeSpec{
		Type:            BlockTypeEmbed,
		DefaultView:     "document",
		SupportedViews:  []string{"document"},
		RequiredDataKey: []string{"url"},
		AllowedChildren: []string{},
	}
	m[BlockTypeBookmark] = BlockTypeSpec{
		Type:            BlockTypeBookmark,
		DefaultView:     "document",
		SupportedViews:  []string{"document"},
		RequiredDataKey: []string{"url"},
		AllowedChildren: []string{},
	}
	m[BlockTypeAudio] = BlockTypeSpec{
		Type:            BlockTypeAudio,
		DefaultView:     "document",
		SupportedViews:  []string{"document"},
		RequiredDataKey: []string{"url"},
		AllowedChildren: []string{},
	}
	m[BlockTypeEquation] = BlockTypeSpec{
		Type:            BlockTypeEquation,
		DefaultView:     "document",
		SupportedViews:  []string{"document"},
		RequiredDataKey: []string{"latex"},
		AllowedChildren: []string{},
	}
	m[BlockTypeChart] = BlockTypeSpec{
		Type:           BlockTypeChart,
		DefaultView:    "document",
		SupportedViews: []string{"document"},
		RequiredDataKey: []string{"type", "series"},
		EnumValues: map[string][]string{
			"type": {"bar", "line", "pie", "area", "scatter", "radar"},
		},
		NonEmptyArrayKeys: []string{"series"},
		AllowedChildren:   []string{},
	}
	m[BlockTypeSyncedBlock] = BlockTypeSpec{
		Type:            BlockTypeSyncedBlock,
		DefaultView:     "document",
		SupportedViews:  []string{"document"},
		RequiredDataKey: []string{"source_id"},
		AllowedChildren: []string{},
	}
}
