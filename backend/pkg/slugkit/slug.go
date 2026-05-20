package slugkit

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// Slug is a typed identifier guaranteed to satisfy Validate (via factories).
// All new UNIQUE identifier columns SHOULD be typed as Slug; raw string
// columns are tech debt that relies on Layer 1-3 (DB CHECK, GORM hook,
// service helper) for safety.
type Slug string

func NewFromTrusted(s string) (Slug, error) {
	if err := Validate(s); err != nil {
		return "", err
	}
	return Slug(s), nil
}

// MustNewForTest is for test fixtures only; panics on invalid input.
func MustNewForTest(s string) Slug {
	sl, err := NewFromTrusted(s)
	if err != nil {
		panic(fmt.Errorf("slugkit.MustNewForTest(%q): %w", s, err))
	}
	return sl
}

func (s Slug) String() string { return string(s) }

// Scan does NOT re-Validate inbound DB values. The DB CHECK constraint is
// the authoritative source of truth for stored slug shape; re-running
// Validate at read time would (a) duplicate that guarantee and (b) break
// any legitimate window where a CHECK is being introduced or relaxed.
// Reads stay cheap; writes go through Validate at the API boundary.
func (s *Slug) Scan(value interface{}) error {
	if value == nil {
		*s = ""
		return nil
	}
	switch v := value.(type) {
	case string:
		*s = Slug(v)
		return nil
	case []byte:
		*s = Slug(v)
		return nil
	}
	return fmt.Errorf("slugkit.Slug: cannot scan %T", value)
}

func (s Slug) Value() (driver.Value, error) { return string(s), nil }

func (s Slug) MarshalJSON() ([]byte, error) { return json.Marshal(string(s)) }

// UnmarshalJSON validates inbound JSON at the API boundary so malformed
// identifiers fail at request parsing, not deep in the handler.
func (s *Slug) UnmarshalJSON(b []byte) error {
	var raw string
	if err := json.Unmarshal(b, &raw); err != nil {
		return err
	}
	sl, err := NewFromTrusted(raw)
	if err != nil {
		return err
	}
	*s = sl
	return nil
}
