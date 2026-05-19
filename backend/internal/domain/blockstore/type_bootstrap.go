package blockstore

var bootstrapTypeSpecs = buildBootstrapSpecs()

func buildBootstrapSpecs() map[string]BlockTypeSpec {
	m := make(map[string]BlockTypeSpec, 32)
	registerContentSpecs(m)
	registerMediaSpecs(m)
	registerSystemSpecs(m)
	return m
}

func LookupTypeSpec(t string) (BlockTypeSpec, bool) {
	spec, ok := bootstrapTypeSpecs[t]
	return spec, ok
}

func BootstrapBlockTypes() []string {
	return []string{
		BlockTypePage, BlockTypeParagraph, BlockTypeTask,
		BlockTypeList, BlockTypeView, BlockTypeTypeDef, BlockTypeComment,
		BlockTypeHeading, BlockTypeDivider, BlockTypeCode,
		BlockTypeQuote, BlockTypeCallout,
		BlockTypeBulletedListItem, BlockTypeNumberedListItem, BlockTypeToggle,
		BlockTypeLinkToPage,
		BlockTypeImage, BlockTypeFile, BlockTypeVideo,
		BlockTypeEmbed, BlockTypeBookmark,
		BlockTypeAudio, BlockTypeEquation, BlockTypeChart, BlockTypeSyncedBlock,
		BlockTypeTable, BlockTypeMention,
		BlockTypeTriggerDef, BlockTypeAgentEvent, BlockTypeDocument,
		BlockTypeColumnList, BlockTypeColumn,
	}
}
