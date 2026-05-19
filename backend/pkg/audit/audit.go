package audit

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net"
	"time"
)

const (
	ActorTypeUser   = "user"
	ActorTypeSystem = "system"
	ActorTypeRunner = "runner"
)

const (
	ActionOrgCreated = "organization.created"
	ActionOrgUpdated = "organization.updated"
	ActionOrgDeleted = "organization.deleted"

	ActionTeamCreated     = "team.created"
	ActionTeamUpdated     = "team.updated"
	ActionTeamDeleted     = "team.deleted"
	ActionTeamMemberAdded = "team.member_added"

	ActionUserLogin    = "user.login"
	ActionUserLogout   = "user.logout"
	ActionUserCreated  = "user.created"
	ActionUserUpdated  = "user.updated"
	ActionUserInvited  = "user.invited"
	ActionUserRemoved  = "user.removed"
	ActionUserRoleChanged = "user.role_changed"

	ActionRunnerRegistered   = "runner.registered"
	ActionRunnerDeregistered = "runner.deregistered"
	ActionRunnerOnline       = "runner.online"
	ActionRunnerOffline      = "runner.offline"

	ActionRunnerCertIssued   = "runner.certificate_issued"
	ActionRunnerCertRenewed  = "runner.certificate_renewed"
	ActionRunnerCertRevoked  = "runner.certificate_revoked"
	ActionRunnerCertRejected = "runner.certificate_rejected"

	ActionRunnerAuthRequested = "runner.auth_requested"
	ActionRunnerAuthApproved  = "runner.auth_approved"
	ActionRunnerTokenUsed     = "runner.token_used"
	ActionRunnerReactivated   = "runner.reactivated"

	ActionPodCreated    = "pod.created"
	ActionPodStarted    = "pod.started"
	ActionPodTerminated = "pod.terminated"

	ActionChannelCreated  = "channel.created"
	ActionChannelArchived = "channel.archived"

	ActionTicketCreated   = "ticket.created"
	ActionTicketUpdated   = "ticket.updated"
	ActionTicketDeleted   = "ticket.deleted"
	ActionTicketAssigned  = "ticket.assigned"
	ActionTicketMRLinked  = "ticket.mr_linked"

	ActionSubscriptionCreated = "subscription.created"
	ActionSubscriptionUpdated = "subscription.updated"
	ActionPaymentReceived     = "payment.received"
	ActionPaymentFailed       = "payment.failed"

	ActionGitProviderAdded   = "git_provider.added"
	ActionGitProviderRemoved = "git_provider.removed"
	ActionRepoAdded          = "repository.added"
	ActionRepoRemoved        = "repository.removed"

	ActionAgentConfigured = "agent.configured"
	ActionAgentCredentialUpdated = "agent.credential_updated"
)

const (
	ResourceOrganization = "organization"
	ResourceTeam         = "team"
	ResourceUser         = "user"
	ResourceRunner       = "runner"
	ResourcePod          = "pod"
	ResourceChannel      = "channel"
	ResourceTicket       = "ticket"
	ResourceSubscription = "subscription"
	ResourceGitProvider  = "git_provider"
	ResourceRepository   = "repository"
	ResourceAgent        = "agent"
)

type Details map[string]interface{}

func (d *Details) Scan(value interface{}) error {
	if value == nil {
		*d = nil
		return nil
	}
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New("unsupported type for Scan")
	}
	return json.Unmarshal(bytes, d)
}

func (d Details) Value() (driver.Value, error) {
	if d == nil {
		return nil, nil
	}
	return json.Marshal(d)
}

type Log struct {
	ID             int64  `gorm:"primaryKey" json:"id"`
	OrganizationID *int64 `gorm:"index" json:"organization_id,omitempty"`

	ActorID   *int64 `json:"actor_id,omitempty"`
	ActorType string `gorm:"size:50;not null" json:"actor_type"`

	Action       string `gorm:"size:100;not null;index" json:"action"`
	ResourceType string `gorm:"size:50;not null" json:"resource_type"`
	ResourceID   *int64 `json:"resource_id,omitempty"`

	Details   Details `gorm:"type:jsonb" json:"details,omitempty"`
	IPAddress net.IP  `gorm:"type:inet" json:"ip_address,omitempty"`
	UserAgent *string `gorm:"type:text" json:"user_agent,omitempty"`

	CreatedAt time.Time `gorm:"not null;default:now();index" json:"created_at"`
}

func (Log) TableName() string {
	return "audit_logs"
}

func Entry(action string) *LogBuilder {
	return &LogBuilder{
		log: Log{
			Action:    action,
			CreatedAt: time.Now(),
		},
	}
}

type LogBuilder struct {
	log Log
}

func (b *LogBuilder) Organization(id int64) *LogBuilder {
	b.log.OrganizationID = &id
	return b
}

func (b *LogBuilder) Actor(actorType string, actorID *int64) *LogBuilder {
	b.log.ActorType = actorType
	b.log.ActorID = actorID
	return b
}

func (b *LogBuilder) Resource(resourceType string, resourceID *int64) *LogBuilder {
	b.log.ResourceType = resourceType
	b.log.ResourceID = resourceID
	return b
}

func (b *LogBuilder) Details(details Details) *LogBuilder {
	b.log.Details = details
	return b
}

func (b *LogBuilder) Request(ip net.IP, userAgent string) *LogBuilder {
	b.log.IPAddress = ip
	b.log.UserAgent = &userAgent
	return b
}

func (b *LogBuilder) Build() *Log {
	return &b.log
}
