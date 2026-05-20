package agent

import "github.com/anthropics/agentsmesh/backend/pkg/slugkit"

// ValidateIdentifiers guards builtin Agent.Slug only. CustomAgent will
// land here once the REST POST /agents/custom path is fully funneled
// through slugkit.SanitizeAndValidate.
func (a *Agent) ValidateIdentifiers() error {
	return slugkit.ValidateIdentifier("agents.slug", a.Slug)
}
