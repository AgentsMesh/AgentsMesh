package acp

import (
	"encoding/json"
	"sync"
)

const maxTopology = 200

type loopalAgentNode struct {
	Name    string `json:"name"`
	AgentID string `json:"agent_id"`
	Parent  string `json:"parent,omitempty"`
	Model   string `json:"model,omitempty"`
}

// loopalState caches Loopal control-panel state from _loopal/* events so a
// freshly-subscribed (or page-refreshed) browser can rebuild the panels via a
// loopal.snapshot. Parallel to the ACP session accumulators in ACPClient —
// same snapshot-on-resubscribe rationale.
//
// SSOT boundary: the Rust core (loopal_dispatch) is the authoritative reducer;
// this Go accumulator is ONLY a snapshot cache for resubscribe, because the
// browser's wasm state is lost on refresh and Loopal never re-pushes. The two
// reducers MUST stay behavior-compatible — change both when the fold rules move.
type loopalState struct {
	mu       sync.Mutex
	order    []string
	bgTasks  map[string]*loopalBgTask
	crons    json.RawMessage
	tasks    json.RawMessage
	mcp      json.RawMessage
	topology []loopalAgentNode
	goal     json.RawMessage
	// session params (mode/thinking/model) must be cached too: loopal emits them
	// once at cold-start, so a browser that subscribes later only sees them via
	// the snapshot, not as live events.
	mode     string
	thinking string
	model    string
}

func newLoopalState() *loopalState {
	return &loopalState{bgTasks: make(map[string]*loopalBgTask)}
}

func extractField(data json.RawMessage, key string) json.RawMessage {
	var m map[string]json.RawMessage
	if json.Unmarshal(data, &m) == nil {
		if v, ok := m[key]; ok {
			return v
		}
	}
	return nil
}

func extractStringField(data json.RawMessage, key string) string {
	if v := extractField(data, key); v != nil {
		var s string
		if json.Unmarshal(v, &s) == nil {
			return s
		}
	}
	return ""
}

func (l *loopalState) apply(kind string, data json.RawMessage) {
	l.mu.Lock()
	defer l.mu.Unlock()
	switch kind {
	case "bgTask.spawned":
		l.applyBgSpawned(data)
	case "bgTask.output":
		l.applyBgOutput(data)
	case "bgTask.completed":
		l.applyBgCompleted(data)
	case "crons":
		if v := extractField(data, "crons"); v != nil {
			l.crons = v
		}
	case "tasks":
		if v := extractField(data, "tasks"); v != nil {
			l.tasks = v
		}
	case "mcp":
		if v := extractField(data, "servers"); v != nil {
			l.mcp = v
		}
	case "goal":
		if v := extractField(data, "goal"); v != nil {
			l.goal = v
		}
	case "mode":
		if v := extractStringField(data, "mode"); v != "" {
			l.mode = v
		}
	case "thinking":
		if v := extractStringField(data, "thinking"); v != "" {
			l.thinking = v
		}
	case "model":
		if v := extractStringField(data, "model"); v != "" {
			l.model = v
		}
	case "topology.spawn":
		l.applyTopologySpawn(data)
	}
}

func (l *loopalState) applyTopologySpawn(data json.RawMessage) {
	var s struct {
		Spawn struct {
			Name    string `json:"name"`
			AgentID string `json:"agent_id"`
			Model   string `json:"model"`
			Parent  struct {
				Agent string `json:"agent"`
			} `json:"parent"`
		} `json:"spawn"`
	}
	if json.Unmarshal(data, &s) != nil || s.Spawn.AgentID == "" {
		return
	}
	for _, n := range l.topology {
		if n.AgentID == s.Spawn.AgentID {
			return
		}
	}
	l.topology = append(l.topology, loopalAgentNode{
		Name:    s.Spawn.Name,
		AgentID: s.Spawn.AgentID,
		Parent:  s.Spawn.Parent.Agent,
		Model:   s.Spawn.Model,
	})
	if len(l.topology) > maxTopology {
		l.topology = l.topology[1:]
	}
}

// snapshot returns a loopal.snapshot relay payload (flat {type, bg_tasks,
// crons?, tasks?, mcp?, topology?, goal?, mode?, thinking?, model?}), or nil
// when no Loopal state has accumulated.
func (l *loopalState) snapshot() []byte {
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.order) == 0 && l.crons == nil && l.tasks == nil && l.mcp == nil &&
		len(l.topology) == 0 && l.goal == nil && l.mode == "" && l.thinking == "" && l.model == "" {
		return nil
	}
	bg := make([]*loopalBgTask, 0, len(l.order))
	for _, id := range l.order {
		bg = append(bg, l.bgTasks[id])
	}
	out := map[string]any{"type": "loopal.snapshot", "bg_tasks": bg}
	if l.crons != nil {
		out["crons"] = l.crons
	}
	if l.tasks != nil {
		out["tasks"] = l.tasks
	}
	if l.mcp != nil {
		out["mcp"] = l.mcp
	}
	if len(l.topology) > 0 {
		out["topology"] = l.topology
	}
	if l.goal != nil {
		out["goal"] = l.goal
	}
	if l.mode != "" {
		out["mode"] = l.mode
	}
	if l.thinking != "" {
		out["thinking"] = l.thinking
	}
	if l.model != "" {
		out["model"] = l.model
	}
	data, err := json.Marshal(out)
	if err != nil {
		return nil
	}
	return data
}

// LoopalSnapshot returns the cached Loopal control-panel snapshot payload for
// resubscribing browsers, or nil when nothing has accumulated.
func (c *ACPClient) LoopalSnapshot() []byte {
	return c.loopal.snapshot()
}

func (c *ACPClient) applyLoopal(kind string, data json.RawMessage) {
	c.loopal.apply(kind, data)
}
