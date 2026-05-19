package mesh

import "time"

type MeshNode struct {
	PodKey           string        `json:"pod_key"`
	Status           string        `json:"status"`
	AgentStatus      string        `json:"agent_status"`
	Model            *string       `json:"model,omitempty"`
	Title            *string       `json:"title,omitempty"`
	Alias            *string       `json:"alias,omitempty"`
	TicketID         *int64        `json:"ticket_id,omitempty"`
	TicketSlug       *string       `json:"ticket_slug,omitempty"`
	TicketTitle      *string       `json:"ticket_title,omitempty"`
	RepositoryID     *int64        `json:"repository_id,omitempty"`
	CreatedByID      int64         `json:"created_by_id"`
	RunnerID         int64         `json:"runner_id"`
	RunnerNodeID     string        `json:"runner_node_id"`
	RunnerStatus     string        `json:"runner_status"`
	StartedAt        *time.Time    `json:"started_at,omitempty"`
	Position         *NodePosition `json:"position,omitempty"`
}

type RunnerInfo struct {
	ID                int64  `json:"id"`
	NodeID            string `json:"node_id"`
	Status            string `json:"status"`
	MaxConcurrentPods int    `json:"max_concurrent_pods"`
	CurrentPods       int    `json:"current_pods"`
}

type NodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type MeshEdge struct {
	ID            int64    `json:"id"`
	Source        string   `json:"source"`         // Initiator pod key
	Target        string   `json:"target"`         // Target pod key
	GrantedScopes []string `json:"granted_scopes"`
	PendingScopes []string `json:"pending_scopes,omitempty"`
	Status        string   `json:"status"`
}

type ChannelInfo struct {
	ID           int64    `json:"id"`
	Name         string   `json:"name"`
	Description  *string  `json:"description,omitempty"`
	PodKeys      []string `json:"pod_keys"`
	MessageCount int      `json:"message_count"`
	IsArchived   bool     `json:"is_archived"`
}

type MeshTopology struct {
	Nodes    []MeshNode    `json:"nodes"`
	Edges    []MeshEdge    `json:"edges"`
	Channels []ChannelInfo `json:"channels"`
	Runners  []RunnerInfo  `json:"runners"`
}

type ChannelPod struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	ChannelID int64     `gorm:"not null;index" json:"channel_id"`
	PodKey    string    `gorm:"size:100;not null;index" json:"pod_key"`
	JoinedAt  time.Time `gorm:"not null;default:now()" json:"joined_at"`
}

func (ChannelPod) TableName() string {
	return "channel_pods"
}

type ChannelAccess struct {
	ID         int64     `gorm:"primaryKey" json:"id"`
	ChannelID  int64     `gorm:"not null;index" json:"channel_id"`
	PodKey     *string   `gorm:"size:100;index" json:"pod_key,omitempty"`
	UserID     *int64    `json:"user_id,omitempty"`
	LastAccess time.Time `gorm:"not null;default:now()" json:"last_access"`
}

func (ChannelAccess) TableName() string {
	return "channel_access"
}

type CreatePodForTicketRequest struct {
	OrganizationID int64  `json:"organization_id"`
	TicketID       int64  `json:"ticket_id"`
	RunnerID       int64  `json:"runner_id"`
	CreatedByID    int64  `json:"created_by_id"`
	Prompt         string `json:"prompt,omitempty"`
	Model          string `json:"model,omitempty"`
	PermissionMode string `json:"permission_mode,omitempty"`
}

type TicketPodInfo struct {
	TicketID int64         `json:"ticket_id"`
	Pods     []MeshNode `json:"pods"`
}

type BatchTicketPodsResponse struct {
	TicketPods map[int64][]MeshNode `json:"ticket_pods"`
}
