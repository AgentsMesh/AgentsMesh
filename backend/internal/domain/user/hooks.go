package user

import "github.com/anthropics/agentsmesh/backend/pkg/slugkit"

// ValidateIdentifiers enforces the slugkit contract on users.username.
// Wired through the GORM plugin in backend/internal/infra/gormvalidate
// — never call BeforeSave directly; this is invoked on every db.Create
// and db.Save / db.Update.
func (u *User) ValidateIdentifiers() error {
	return slugkit.ValidateIdentifier("users.username", u.Username)
}
