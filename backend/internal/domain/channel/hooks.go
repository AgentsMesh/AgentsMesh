package channel

import "github.com/anthropics/agentsmesh/backend/pkg/slugkit"

func (c *Channel) ValidateIdentifiers() error {
	if c.Slug == nil {
		return nil
	}
	return slugkit.ValidateIdentifier("channels.slug", *c.Slug)
}
