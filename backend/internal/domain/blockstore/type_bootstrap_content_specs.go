package blockstore

func registerContentSpecs(m map[string]BlockTypeSpec) {
	m[BlockTypePage] = BlockTypeSpec{
		Type:            BlockTypePage,
		DefaultView:     "document",
		SupportedViews:  []string{"document", "list"},
		RequiredDataKey: []string{"title"},
		AllowedChildren: nil,
	}
	m[BlockTypeParagraph] = BlockTypeSpec{
		Type:            BlockTypeParagraph,
		DefaultView:     "document",
		SupportedViews:  []string{"document"},
		RequiredDataKey: nil,
		AllowedChildren: []string{BlockTypeParagraph},
	}
	m[BlockTypeTask] = BlockTypeSpec{
		Type:            BlockTypeTask,
		DefaultView:     "list",
		SupportedViews:  []string{"document", "list"},
		RequiredDataKey: []string{"title"},
		AllowedChildren: []string{BlockTypeTask, BlockTypeParagraph},
	}
	m[BlockTypeList] = BlockTypeSpec{
		Type:            BlockTypeList,
		DefaultView:     "list",
		SupportedViews:  []string{"document", "list"},
		RequiredDataKey: nil,
		AllowedChildren: nil,
	}
	m[BlockTypeView] = BlockTypeSpec{
		Type:            BlockTypeView,
		DefaultView:     "list",
		SupportedViews:  []string{"document", "list"},
		RequiredDataKey: []string{"source_type", "layout"},
		AllowedChildren: nil,
	}
	m[BlockTypeComment] = BlockTypeSpec{
		Type:            BlockTypeComment,
		DefaultView:     "list",
		SupportedViews:  []string{"list"},
		RequiredDataKey: []string{"text"},
		AllowedChildren: []string{},
	}
	m[BlockTypeHeading] = BlockTypeSpec{
		Type:            BlockTypeHeading,
		DefaultView:     "document",
		SupportedViews:  []string{"document"},
		RequiredDataKey: []string{"level"},
		AllowedChildren: []string{},
	}
	m[BlockTypeDivider] = BlockTypeSpec{
		Type:            BlockTypeDivider,
		DefaultView:     "document",
		SupportedViews:  []string{"document"},
		AllowedChildren: []string{},
	}
	m[BlockTypeCode] = BlockTypeSpec{
		Type:            BlockTypeCode,
		DefaultView:     "document",
		SupportedViews:  []string{"document"},
		RequiredDataKey: []string{"code"},
		AllowedChildren: []string{},
	}
	m[BlockTypeQuote] = BlockTypeSpec{
		Type:            BlockTypeQuote,
		DefaultView:     "document",
		SupportedViews:  []string{"document"},
		AllowedChildren: []string{BlockTypeParagraph, BlockTypeBulletedListItem, BlockTypeNumberedListItem},
	}
	m[BlockTypeCallout] = BlockTypeSpec{
		Type:            BlockTypeCallout,
		DefaultView:     "document",
		SupportedViews:  []string{"document"},
		AllowedChildren: []string{BlockTypeParagraph, BlockTypeBulletedListItem, BlockTypeNumberedListItem},
	}
	m[BlockTypeBulletedListItem] = BlockTypeSpec{
		Type:            BlockTypeBulletedListItem,
		DefaultView:     "document",
		SupportedViews:  []string{"document", "list"},
		AllowedChildren: nil,
	}
	m[BlockTypeNumberedListItem] = BlockTypeSpec{
		Type:            BlockTypeNumberedListItem,
		DefaultView:     "document",
		SupportedViews:  []string{"document", "list"},
		AllowedChildren: nil,
	}
	m[BlockTypeToggle] = BlockTypeSpec{
		Type:            BlockTypeToggle,
		DefaultView:     "document",
		SupportedViews:  []string{"document"},
		AllowedChildren: nil,
	}
	m[BlockTypeLinkToPage] = BlockTypeSpec{
		Type:            BlockTypeLinkToPage,
		DefaultView:     "document",
		SupportedViews:  []string{"document", "list"},
		RequiredDataKey: []string{"target_id"},
		AllowedChildren: []string{},
	}
	m[BlockTypeTable] = BlockTypeSpec{
		Type:            BlockTypeTable,
		DefaultView:     "document",
		SupportedViews:  []string{"document"},
		AllowedChildren: []string{},
	}
	m[BlockTypeColumnList] = BlockTypeSpec{
		Type:           BlockTypeColumnList,
		DefaultView:    "document",
		SupportedViews: []string{"document"},
		AllowedChildren: []string{BlockTypeColumn},
	}
	m[BlockTypeColumn] = BlockTypeSpec{
		Type:           BlockTypeColumn,
		DefaultView:    "document",
		SupportedViews: []string{"document"},
		AllowedChildren: nil,
	}
	m[BlockTypeMention] = BlockTypeSpec{
		Type:            BlockTypeMention,
		DefaultView:     "document",
		SupportedViews:  []string{"document", "list"},
		RequiredDataKey: []string{"user_id"},
		AllowedChildren: []string{},
	}
}
