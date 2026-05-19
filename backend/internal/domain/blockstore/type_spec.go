package blockstore

type BlockTypeSpec struct {
	Type            string       // e.g. "task"
	Revision        int          // incremented when the spec shape changes (dynamic specs only)
	Label           string       // human-readable ("OKR", "Bug"); optional, defaults to Type
	Description     string       // one-line intent shown in slash menu / MCP descriptions
	DefaultView     string       // e.g. "document"
	SupportedViews  []string     // e.g. ["document", "list", "kanban"]
	RequiredDataKey []string     // legacy: presence-only check; empty for Columns-driven types
	AllowedChildren []string     // empty = any child block type; otherwise whitelist
	Columns         []ColumnSpec // Tier 1: schema-driven record fields

	EnumValues map[string][]string

	NonEmptyArrayKeys []string
}

type ColumnType string

const (
	ColumnTypeText        ColumnType = "text"
	ColumnTypeNumber      ColumnType = "number"
	ColumnTypeBoolean     ColumnType = "boolean"
	ColumnTypeSelect      ColumnType = "select"
	ColumnTypeMultiSelect ColumnType = "multi_select"
	ColumnTypeDate        ColumnType = "date"
	ColumnTypeURL         ColumnType = "url"
	ColumnTypeUser        ColumnType = "user"
	ColumnTypeBlockRef    ColumnType = "block_ref"
)

type ColumnSpec struct {
	Key         string         `json:"key"`
	Label       string         `json:"label,omitempty"`
	Type        ColumnType     `json:"type"`
	Required    bool           `json:"required,omitempty"`
	Default     any            `json:"default,omitempty"`
	Options     []SelectOption `json:"options,omitempty"`
	Description string         `json:"description,omitempty"`

	Computed string `json:"computed,omitempty"`

	Deprecated bool `json:"deprecated,omitempty"`
}

type SelectOption struct {
	Value string `json:"value"`
	Label string `json:"label,omitempty"`
	Color string `json:"color,omitempty"`
}

func (spec BlockTypeSpec) IsColumnsDriven() bool {
	return len(spec.Columns) > 0
}

func (spec BlockTypeSpec) IsChildAllowed(child string) bool {
	if len(spec.AllowedChildren) == 0 {
		return true
	}
	for _, allowed := range spec.AllowedChildren {
		if allowed == child {
			return true
		}
	}
	return false
}

func (spec BlockTypeSpec) RequiredKeys() []string {
	out := make([]string, 0, len(spec.RequiredDataKey)+len(spec.Columns))
	seen := make(map[string]bool, len(out))
	for _, k := range spec.RequiredDataKey {
		if !seen[k] {
			seen[k] = true
			out = append(out, k)
		}
	}
	for _, c := range spec.Columns {
		if c.Deprecated || c.Computed != "" {
			continue
		}
		if c.Required && !seen[c.Key] {
			seen[c.Key] = true
			out = append(out, c.Key)
		}
	}
	return out
}
