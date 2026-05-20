package organization

import "github.com/anthropics/agentsmesh/backend/pkg/slugkit"

func (o *Organization) ValidateIdentifiers() error {
	return slugkit.ValidateIdentifier("organizations.slug", o.Slug)
}
