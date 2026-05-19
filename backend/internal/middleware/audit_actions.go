package middleware

type AuditAction string

const (
	AuditUserCreated    AuditAction = "users.created"
	AuditUserUpdated    AuditAction = "users.updated"
	AuditUserDeleted    AuditAction = "users.deleted"
	AuditUserLoggedIn   AuditAction = "users.logged_in"
	AuditUserLoggedOut  AuditAction = "users.logged_out"
	AuditUserRegistered AuditAction = "users.registered"

	AuditOrgCreated       AuditAction = "organizations.created"
	AuditOrgUpdated       AuditAction = "organizations.updated"
	AuditOrgDeleted       AuditAction = "organizations.deleted"
	AuditOrgMemberAdded   AuditAction = "organizations.member_added"
	AuditOrgMemberRemoved AuditAction = "organizations.member_removed"

	AuditTeamCreated       AuditAction = "teams.created"
	AuditTeamUpdated       AuditAction = "teams.updated"
	AuditTeamDeleted       AuditAction = "teams.deleted"
	AuditTeamMemberAdded   AuditAction = "teams.member_added"
	AuditTeamMemberRemoved AuditAction = "teams.member_removed"

	AuditRunnerRegistered      AuditAction = "runners.registered"
	AuditRunnerDeleted         AuditAction = "runners.deleted"
	AuditRunnerOnline          AuditAction = "runners.online"
	AuditRunnerOffline         AuditAction = "runners.offline"
	AuditRunnerUpgradeStarted  AuditAction = "runners.upgrade_started"

	AuditPodCreated    AuditAction = "pods.created"
	AuditPodStarted    AuditAction = "pods.started"
	AuditPodTerminated AuditAction = "pods.terminated"
	AuditPodFailed     AuditAction = "pods.failed"

	AuditChannelCreated  AuditAction = "channels.created"
	AuditChannelArchived AuditAction = "channels.archived"
	AuditChannelJoined   AuditAction = "channels.joined"
	AuditChannelLeft     AuditAction = "channels.left"

	AuditTicketCreated       AuditAction = "tickets.created"
	AuditTicketUpdated       AuditAction = "tickets.updated"
	AuditTicketDeleted       AuditAction = "tickets.deleted"
	AuditTicketStatusChanged AuditAction = "tickets.status_changed"

	AuditGitProviderCreated AuditAction = "git_providers.created"
	AuditGitProviderUpdated AuditAction = "git_providers.updated"
	AuditGitProviderDeleted AuditAction = "git_providers.deleted"

	AuditRepositoryCreated AuditAction = "repositories.created"
	AuditRepositoryDeleted AuditAction = "repositories.deleted"

	AuditSubscriptionCreated  AuditAction = "subscriptions.created"
	AuditSubscriptionUpdated  AuditAction = "subscriptions.updated"
	AuditSubscriptionCanceled AuditAction = "subscriptions.canceled"
)
