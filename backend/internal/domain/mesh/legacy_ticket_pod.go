package mesh

// Legacy ticket-pod defaults captured here because mesh.CreatePodForTicket is
// the only API path that still creates pods *without* going through the
// AgentFile SSOT flow. Historically it always produced Claude pods with
// hardcoded defaults; making the convention explicit means readers do not have
// to guess why a particular agent / model / permission mode shows up downstream.
//
// New pod creation flows must go through PodOrchestrator and supply config via
// AgentFile Layer — do NOT add new consumers of these constants.
const (
	LegacyTicketPodAgentSlug      = "claude-code"
	LegacyTicketPodModel          = "opus"
	LegacyTicketPodPermissionMode = "bypassPermissions"
)
