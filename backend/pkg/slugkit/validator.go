package slugkit

// IdentifierValidator is implemented by domain models whose identifier
// fields must pass slugkit.Validate before persistence. Pair with the
// GORM plugin at backend/internal/infra/gormvalidate to wire validation
// into db.Create / db.Save / db.Update — domain packages stay free of
// gorm imports while still benefiting from Layer 2 defense.
//
// The contract: ValidateIdentifiers returns nil iff every identifier
// field on the receiver satisfies its slug rule, OR is empty when the
// column is nullable (use ValidateIdentifier for the per-field check).
type IdentifierValidator interface {
	ValidateIdentifiers() error
}
