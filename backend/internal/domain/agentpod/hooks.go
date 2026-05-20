package agentpod

import "github.com/anthropics/agentsmesh/backend/pkg/slugkit"

// ValidateIdentifiers validates pod_key (backend-generated, format-stable)
// and agent_slug (reference to an existing agents row). Both checked
// defensively to catch future regressions in pod key generation.
func (p *Pod) ValidateIdentifiers() error {
	if err := slugkit.ValidateIdentifier("pods.pod_key", p.PodKey); err != nil {
		return err
	}
	return slugkit.ValidateIdentifier("pods.agent_slug", p.AgentSlug)
}
