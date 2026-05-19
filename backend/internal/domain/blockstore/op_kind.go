package blockstore

const (
	OpCreateBlock = "createBlock"
	OpUpdateBlock = "updateBlock"
	OpDeleteBlock = "deleteBlock"
	OpAddRef      = "addRef"
	OpRemoveRef   = "removeRef"
	OpUpdateRef   = "updateRef"
)

const (
	RelNest        = "nest"
	RelMention     = "mention"
	RelEmbed       = "embed"
	RelDependsOn   = "depends_on"
	RelDerivedFrom = "derived_from"
	RelTag         = "tag"
	RelCommentsOn  = "comments_on" // Phase 4: a comment block anchored on another block
)

const (
	BlockTypePage      = "page"
	BlockTypeParagraph = "paragraph"
	BlockTypeTask      = "task"
	BlockTypeList      = "list"
	BlockTypeView      = "view"
	BlockTypeTypeDef   = "block_type_def"
	BlockTypeComment   = "comment"

	BlockTypeHeading            = "heading"
	BlockTypeDivider            = "divider"
	BlockTypeCode               = "code"
	BlockTypeQuote              = "quote"
	BlockTypeCallout            = "callout"
	BlockTypeBulletedListItem   = "bulleted_list_item"
	BlockTypeNumberedListItem   = "numbered_list_item"
	BlockTypeToggle             = "toggle"
	BlockTypeLinkToPage         = "link_to_page"
	BlockTypeImage              = "image"
	BlockTypeFile               = "file"
	BlockTypeVideo              = "video"
	BlockTypeEmbed              = "embed"
	BlockTypeBookmark           = "bookmark"
	BlockTypeAudio              = "audio"
	BlockTypeEquation           = "equation"
	BlockTypeChart              = "chart"
	BlockTypeSyncedBlock        = "synced_block"
	BlockTypeTable              = "table"
	BlockTypeMention            = "mention"

	// Phase 5: structured rich-text document backed by BlockNote. Holds the
	// full BlockNote AST in data.blocknote_ast plus a flattened plain text
	// mirror in block.text for search / embeddings.
	BlockTypeDocument = "document"

	BlockTypeColumnList = "column_list"
	BlockTypeColumn     = "column"

	BlockTypeTriggerDef = "trigger_def"

	BlockTypeAgentEvent = "agent_event"
)

func AllOpKinds() []string {
	return []string{
		OpCreateBlock, OpUpdateBlock, OpDeleteBlock,
		OpAddRef, OpRemoveRef, OpUpdateRef,
	}
}

func IsOrderedRel(rel string) bool {
	return rel == RelNest
}

// UniqueParentRels reports whether a rel enforces the single-parent invariant
// (one `to_id` may have at most one ref with this rel).
func IsUniqueParentRel(rel string) bool {
	return rel == RelNest
}
