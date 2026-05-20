package loop

import "github.com/anthropics/agentsmesh/backend/pkg/slugkit"

func (l *Loop) ValidateIdentifiers() error {
	if err := slugkit.ValidateIdentifier("loops.slug", l.Slug); err != nil {
		return err
	}
	return slugkit.ValidateIdentifier("loops.agent_slug", l.AgentSlug)
}
