package audit

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"net"
	"time"
)

// Actor type constants
const (
	ActorTypeUser   = "user"
	ActorTypeSystem = "system"
	ActorTypeRunner = "runner"
)

// Action constants
const (
	// Organization actions
	ActionOrgCreated = "organization.created"
	ActionOrgUpdated = "organization.updated"
	ActionOrgDeleted = "organization.deleted"

	// Team actions
	ActionTeamCreated     = "team.created"
	ActionTeamUpdated     = "team.updated"
	ActionTeamDeleted     = "team.deleted"
	ActionTeamMemberAdded = "team.member_added"

	// User actions
	ActionUserLogin    = "user.login"
	ActionUserLogout   = "user.logout"
	ActionUserCreated  = "user.created"
	ActionUserUpdated  = "user.updated"
	ActionUserInvited  = "user.invited"
	ActionUserRemoved  = "user.removed"
	ActionUserRoleChanged = "user.role_changed"

	// Runner actions
	ActionRunnerRegistered   = "runner.registered"
	ActionRunnerDeregistered = "runner.deregistered"
	ActionRunnerOnline       = "runner.online"
	ActionRunnerOffline      = "runner.offline"

	// Runner certificate actions
	ActionRunnerCertIssued   = "runner.certificate_issued"
	ActionRunnerCertRenewed  = "runner.certificate_renewed"
	ActionRunnerCertRevoked  = "runner.certificate_revoked"
	ActionRunnerCertRejected = "runner.certificate_rejected" // Revoked cert attempted connection

	// Runner auth actions
	ActionRunnerAuthRequested = "runner.auth_requested"  // Requested authorization URL
	ActionRunnerAuthApproved  = "runner.auth_approved"   // User approved authorization
	ActionRunnerTokenUsed     = "runner.token_used"      // Token registration used
	ActionRunnerReactivated   = "runner.reactivated"     // Reactivated

	// Pod actions
	ActionPodCreated    = "pod.created"
	ActionPodStarted    = "pod.started"
	ActionPodTerminated = "pod.terminated"

	// Channel actions
	ActionChannelCreated  = "channel.created"
	ActionChannelArchived = "channel.archived"

	// Ticket actions
	ActionTicketCreated   = "ticket.created"
	ActionTicketUpdated   = "ticket.updated"
	ActionTicketDeleted   = "ticket.deleted"
	ActionTicketAssigned  = "ticket.assigned"
	ActionTicketMRLinked  = "ticket.mr_linked"

	// Billing actions
	ActionSubscriptionCreated = "subscription.created"
	ActionSubscriptionUpdated = "subscription.updated"
	ActionPaymentReceived     = "payment.received"
	ActionPaymentFailed       = "payment.failed"

	// Git Provider actions
	ActionGitProviderAdded   = "git_provider.added"
	ActionGitProviderRemoved = "git_provider.removed"
	ActionRepoAdded          = "repository.added"
	ActionRepoRemoved        = "repository.removed"

	// Agent actions
	ActionAgentConfigured = "agent.configured"
	ActionAgentCredentialUpdated = "agent.credential_updated"
)

// Resource type constants
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

// Details represents audit log details as JSONB
type Details map[string]interface{}

// Scan implements sql.Scanner for Details
func (d *Details) Scan(value interface{}) error {
	if value == nil {
		*d = nil
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(bytes, d)
}

// Value implements driver.Valuer for Details
func (d Details) Value() (driver.Value, error) {
	if d == nil {
		return nil, nil
	}
	return json.Marshal(d)
}

// Log represents an audit log entry
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

// Entry creates a new audit log entry builder
func Entry(action string) *LogBuilder {
	return &LogBuilder{
		log: Log{
			Action:    action,
			CreatedAt: time.Now(),
		},
	}
}

// LogBuilder provides a fluent interface for building audit logs
type LogBuilder struct {
	log Log
}

// Organization sets the organization ID
func (b *LogBuilder) Organization(id int64) *LogBuilder {
	b.log.OrganizationID = &id
	return b
}

// Actor sets the actor information
func (b *LogBuilder) Actor(actorType string, actorID *int64) *LogBuilder {
	b.log.ActorType = actorType
	b.log.ActorID = actorID
	return b
}

// Resource sets the resource information
func (b *LogBuilder) Resource(resourceType string, resourceID *int64) *LogBuilder {
	b.log.ResourceType = resourceType
	b.log.ResourceID = resourceID
	return b
}

// Details sets the details
func (b *LogBuilder) Details(details Details) *LogBuilder {
	b.log.Details = details
	return b
}

// Request sets request information
func (b *LogBuilder) Request(ip net.IP, userAgent string) *LogBuilder {
	b.log.IPAddress = ip
	b.log.UserAgent = &userAgent
	return b
}

// Build returns the constructed log entry
func (b *LogBuilder) Build() *Log {
	return &b.log
}
