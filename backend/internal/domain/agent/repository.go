package agent

import (
	"context"
	"time"
)

// AgentTypeRepository defines persistence operations for agent types
type AgentTypeRepository interface {
	// Builtin types
	ListBuiltinActive(ctx context.Context) ([]*AgentType, error)
	ListAllActive(ctx context.Context) ([]*AgentType, error)
	GetByID(ctx context.Context, id int64) (*AgentType, error)
	GetBySlug(ctx context.Context, slug string) (*AgentType, error)

	// Custom types
	ListCustomByOrg(ctx context.Context, orgID int64) ([]*CustomAgentType, error)
	GetCustomByID(ctx context.Context, id int64) (*CustomAgentType, error)
	CustomSlugExists(ctx context.Context, orgID int64, slug string) (bool, error)
	CreateCustom(ctx context.Context, custom *CustomAgentType) error
	UpdateCustom(ctx context.Context, id int64, updates map[string]interface{}) (*CustomAgentType, error)
	DeleteCustom(ctx context.Context, id int64) error
	CountLoopReferences(ctx context.Context, customID int64) (int64, error)
}

// CredentialProfileRepository defines persistence operations for credential profiles
type CredentialProfileRepository interface {
	Create(ctx context.Context, profile *UserAgentCredentialProfile) error
	GetWithAgentType(ctx context.Context, userID, profileID int64) (*UserAgentCredentialProfile, error)
	Delete(ctx context.Context, userID, profileID int64) (int64, error)

	ListActiveWithAgentType(ctx context.Context, userID int64) ([]*UserAgentCredentialProfile, error)
	ListByAgentType(ctx context.Context, userID, agentTypeID int64) ([]*UserAgentCredentialProfile, error)
	GetDefault(ctx context.Context, userID, agentTypeID int64) (*UserAgentCredentialProfile, error)

	NameExists(ctx context.Context, userID, agentTypeID int64, name string, excludeID *int64) (bool, error)
	UnsetDefaults(ctx context.Context, userID, agentTypeID int64) error
	Update(ctx context.Context, profile *UserAgentCredentialProfile, updates map[string]interface{}) error
	SetDefault(ctx context.Context, profile *UserAgentCredentialProfile) error
}

// UserConfigRepository defines persistence operations for user agent configs
type UserConfigRepository interface {
	GetByUserAndAgentType(ctx context.Context, userID, agentTypeID int64) (*UserAgentConfig, error)
	Upsert(ctx context.Context, userID, agentTypeID int64, configValues ConfigValues) error
	Delete(ctx context.Context, userID, agentTypeID int64) error
	ListByUser(ctx context.Context, userID int64) ([]*UserAgentConfig, error)
}

// MessageRepository defines persistence operations for agent messages
type MessageRepository interface {
	// CRUD
	Create(ctx context.Context, message *AgentMessage) error
	GetByID(ctx context.Context, id int64) (*AgentMessage, error)
	Save(ctx context.Context, message *AgentMessage) error
	Delete(ctx context.Context, message *AgentMessage) error

	// Status updates
	UpdateStatus(ctx context.Context, messageID int64, updates map[string]interface{}) error
	MarkAllRead(ctx context.Context, podKey string) (int64, error)

	// Queries
	GetMessages(ctx context.Context, podKey string, unreadOnly bool, messageTypes []string, limit, offset int) ([]*AgentMessage, error)
	GetUnreadMessages(ctx context.Context, podKey string, limit int) ([]*AgentMessage, error)
	GetUnreadCount(ctx context.Context, podKey string) (int64, error)
	GetConversation(ctx context.Context, correlationID string, limit int) ([]*AgentMessage, error)
	GetReplies(ctx context.Context, parentMessageID int64) ([]*AgentMessage, error)
	GetSentMessages(ctx context.Context, podKey string, limit, offset int) ([]*AgentMessage, error)
	GetMessagesBetween(ctx context.Context, podA, podB string, limit int) ([]*AgentMessage, error)

	// Retry/Dead Letter
	GetPendingRetries(ctx context.Context, before time.Time, limit int) ([]*AgentMessage, error)
	CreateDeadLetter(ctx context.Context, entry *DeadLetterEntry) error
	GetDeadLetters(ctx context.Context, limit, offset int) ([]*DeadLetterEntry, error)
	GetDeadLetterWithMessage(ctx context.Context, id int64) (*DeadLetterEntry, error)
	SaveDeadLetter(ctx context.Context, entry *DeadLetterEntry) error
	CleanupExpiredDeadLetters(ctx context.Context, olderThan time.Time) (int64, error)
}
