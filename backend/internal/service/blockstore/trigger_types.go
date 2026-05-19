package blockstoreservice

import "github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"

type TriggerDef struct {
	Name       string             `json:"name"`
	TargetType string             `json:"target_type"`
	On         string             `json:"on"`
	Predicate  string             `json:"predicate,omitempty"`
	Action     TriggerAction      `json:"action"`
	Enabled    bool               `json:"enabled"`
	Meta       blockstore.JSONMap `json:"-"`
	// CreatedBy carries the trigger author through to side-effect writes (e.g. agent_event)
	// so ACL checks resolve against a real user — "权限跟着人走".
	CreatedBy int64 `json:"-"`
}

type TriggerAction struct {
	Kind      string            `json:"kind"`
	URL       string            `json:"url,omitempty"`
	AgentSlug string            `json:"agent_slug,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
}
