package slugkit

import "fmt"

// ValidateIdentifier wraps Validate with field-name context, intended for
// GORM BeforeSave hooks. Empty values pass through so nullable identifier
// columns (e.g. Loop.AgentSlug omitempty) don't false-positive; non-empty
// required fields are enforced via NOT NULL at the DB layer.
func ValidateIdentifier(fieldName, value string) error {
	if value == "" {
		return nil
	}
	if err := Validate(value); err != nil {
		return fmt.Errorf("%s: %w", fieldName, err)
	}
	return nil
}
