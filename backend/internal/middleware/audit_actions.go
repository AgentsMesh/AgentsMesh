package middleware

// AuditAction is a helper type for logging specific actions
type AuditAction string

const (
	// User actions
	AuditUserCreated    AuditAction = "users.created"
	AuditUserUpdated    AuditAction = "users.updated"
	AuditUserDeleted    AuditAction = "users.deleted"
	AuditUserLoggedIn   AuditAction = "users.logged_in"
	AuditUserLoggedOut  AuditAction = "users.logged_out"
	AuditUserRegistered AuditAction = "users.registered"

	// Organization actions
	AuditOrgCreated       AuditAction = "organizations.created"
	AuditOrgUpdated       AuditAction = "organizations.updated"
	AuditOrgDeleted       AuditAction = "organizations.deleted"
	AuditOrgMemberAdded   AuditAction = "organizations.member_added"
	AuditOrgMemberRemoved AuditAction = "organizations.member_removed"

	// Team actions
	AuditTeamCreated       AuditAction = "teams.created"
	AuditTeamUpdated       AuditAction = "teams.updated"
	AuditTeamDeleted       AuditAction = "teams.deleted"
	AuditTeamMemberAdded   AuditAction = "teams.member_added"
	AuditTeamMemberRemoved AuditAction = "teams.member_removed"

	// Runner actions
	AuditRunnerRegistered      AuditAction = "runners.registered"
	AuditRunnerDeleted         AuditAction = "runners.deleted"
	AuditRunnerOnline          AuditAction = "runners.online"
	AuditRunnerOffline         AuditAction = "runners.offline"
	AuditRunnerUpgradeStarted  AuditAction = "runners.upgrade_started"

	// Pod actions
	AuditPodCreated    AuditAction = "pods.created"
	AuditPodStarted    AuditAction = "pods.started"
	AuditPodTerminated AuditAction = "pods.terminated"
	AuditPodFailed     AuditAction = "pods.failed"

	// Channel actions
	AuditChannelCreated  AuditAction = "channels.created"
	AuditChannelArchived AuditAction = "channels.archived"
	AuditChannelJoined   AuditAction = "channels.joined"
	AuditChannelLeft     AuditAction = "channels.left"

	// Ticket actions
	AuditTicketCreated       AuditAction = "tickets.created"
	AuditTicketUpdated       AuditAction = "tickets.updated"
	AuditTicketDeleted       AuditAction = "tickets.deleted"
	AuditTicketStatusChanged AuditAction = "tickets.status_changed"

	// Git Provider actions
	AuditGitProviderCreated AuditAction = "git_providers.created"
	AuditGitProviderUpdated AuditAction = "git_providers.updated"
	AuditGitProviderDeleted AuditAction = "git_providers.deleted"

	// Repository actions
	AuditRepositoryCreated AuditAction = "repositories.created"
	AuditRepositoryDeleted AuditAction = "repositories.deleted"

	// Billing actions
	AuditSubscriptionCreated  AuditAction = "subscriptions.created"
	AuditSubscriptionUpdated  AuditAction = "subscriptions.updated"
	AuditSubscriptionCanceled AuditAction = "subscriptions.canceled"
)
