package blockstoreservice

import (
	"encoding/json"

	"github.com/anthropics/agentsmesh/backend/internal/domain/blockstore"
)

// BlockACL is block.meta.acl: visibility ∈ {workspace,private,org}; nil falls back to workspace.
type BlockACL struct {
	Visibility   string  `json:"visibility"`
	AllowedUsers []int64 `json:"allowed_users"`
}

func extractACL(meta blockstore.JSONMap) BlockACL {
	raw, ok := meta["acl"]
	if !ok {
		return BlockACL{}
	}
	buf, err := json.Marshal(raw)
	if err != nil {
		return BlockACL{}
	}
	var out BlockACL
	_ = json.Unmarshal(buf, &out)
	return out
}

func (acl BlockACL) allows(actorUserID, createdBy int64) bool {
	switch acl.Visibility {
	case "", "workspace", "org":
		return true
	case "private":
		if actorUserID == createdBy {
			return true
		}
		for _, u := range acl.AllowedUsers {
			if u == actorUserID {
				return true
			}
		}
		return false
	default:
		return false
	}
}
