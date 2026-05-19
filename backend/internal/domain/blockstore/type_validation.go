package blockstore

func (spec BlockTypeSpec) ValidateRequiredFields(data JSONMap) string {
	for _, k := range spec.RequiredKeys() {
		if _, present := data[k]; !present {
			return k
		}
	}
	return ""
}

// ValidateRecord performs full Tier-1 validation against Columns. For each
// column it checks:
//   - Required columns must be present
//   - Select / multi_select values must be in Options (when Options non-empty)
//   - Type-level constraints (number is numeric, date parseable, etc.) are
//     enforced lightly — the frontend is the first line of defense; this is
//     the last-resort guard against malformed Agent writes.
//
// Returns ("", "") on success; otherwise (key, reason). Empty Columns means
// the type is not Tier-1-driven → only required-field check runs.
func (spec BlockTypeSpec) ValidateRecord(data JSONMap) (string, string) {
	if missing := spec.ValidateRequiredFields(data); missing != "" {
		return missing, "required"
	}
	if key, reason := validateEnumValues(spec.EnumValues, data); key != "" {
		return key, reason
	}
	if key, reason := validateNonEmptyArrays(spec.NonEmptyArrayKeys, data); key != "" {
		return key, reason
	}
	for _, col := range spec.Columns {
		if col.Deprecated || col.Computed != "" {
			continue
		}
		raw, present := data[col.Key]
		if !present {
			continue
		}
		if msg := validateColumnValue(col, raw); msg != "" {
			return col.Key, msg
		}
	}
	return "", ""
}

func validateEnumValues(enums map[string][]string, data JSONMap) (string, string) {
	for key, allowed := range enums {
		raw, present := data[key]
		if !present {
			continue
		}
		s, ok := raw.(string)
		if !ok {
			return key, "expected string"
		}
		matched := false
		for _, v := range allowed {
			if v == s {
				matched = true
				break
			}
		}
		if !matched {
			return key, "value not in allowed set"
		}
	}
	return "", ""
}

func validateNonEmptyArrays(keys []string, data JSONMap) (string, string) {
	for _, key := range keys {
		raw, present := data[key]
		if !present {
			continue
		}
		arr, ok := raw.([]any)
		if !ok {
			return key, "expected array"
		}
		if len(arr) == 0 {
			return key, "must be non-empty"
		}
	}
	return "", ""
}

func validateColumnValue(col ColumnSpec, raw any) string {
	switch col.Type {
	case ColumnTypeNumber:
		switch raw.(type) {
		case float64, float32, int, int32, int64:
			return ""
		default:
			return "expected number"
		}
	case ColumnTypeBoolean:
		if _, ok := raw.(bool); !ok {
			return "expected boolean"
		}
	case ColumnTypeSelect:
		s, ok := raw.(string)
		if !ok {
			return "expected string"
		}
		if len(col.Options) == 0 {
			return ""
		}
		for _, opt := range col.Options {
			if opt.Value == s {
				return ""
			}
		}
		return "value not in select options"
	case ColumnTypeMultiSelect:
		arr, ok := raw.([]any)
		if !ok {
			return "expected array"
		}
		if len(col.Options) == 0 {
			return ""
		}
		allowed := make(map[string]bool, len(col.Options))
		for _, opt := range col.Options {
			allowed[opt.Value] = true
		}
		for _, v := range arr {
			s, ok := v.(string)
			if !ok || !allowed[s] {
				return "value not in multi_select options"
			}
		}
	}
	return ""
}

// ValidateRequiredFields (package fn) resolves against the bootstrap registry.
// Deprecated: service callers should use Service.resolveTypeSpecInTx (or
// LookupTypeSpec directly for read-only paths) + spec.ValidateRequiredFields.
// Still used by non-service call sites and by tests.
func ValidateRequiredFields(t string, data JSONMap) string {
	spec, ok := LookupTypeSpec(t)
	if !ok {
		return ""
	}
	return spec.ValidateRequiredFields(data)
}

// IsChildAllowed (package fn) resolves against the bootstrap registry.
// Deprecated: service callers should use Service.resolveTypeSpecInTx (or
// LookupTypeSpec directly for read-only paths) + spec.IsChildAllowed.
func IsChildAllowed(parent, child string) bool {
	spec, ok := LookupTypeSpec(parent)
	if !ok {
		return false
	}
	return spec.IsChildAllowed(child)
}
