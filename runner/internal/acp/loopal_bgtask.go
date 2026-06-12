package acp

import (
	"encoding/json"
	"unicode/utf8"
)

const (
	maxBgOutput = 64 * 1024
	maxBgTasks  = 200
)

type loopalBgTask struct {
	ID              string `json:"id"`
	Description     string `json:"description"`
	Status          string `json:"status"`
	ExitCode        *int   `json:"exit_code"`
	Output          string `json:"output"`
	CreatedAtUnixMs uint64 `json:"created_at_unix_ms"`
}

// capString keeps the trailing max bytes of s on a UTF-8 rune boundary — the
// snapshot-cache counterpart of core loopal_session::cap_output (SSOT boundary).
func capString(s string, max int) string {
	if len(s) <= max {
		return s
	}
	cut := len(s) - max
	for cut < len(s) && !utf8.RuneStart(s[cut]) {
		cut++
	}
	return s[cut:]
}

// bgTask find-or-inserts a row. Output/completed events may arrive before their
// spawn (relay reordering) or after the spawn was evicted by the maxBgTasks cap;
// without an upsert they would be silently dropped, leaving a task stuck
// "Running" or never shown. Mirrors core loopal_session::bg_task_mut — keep the
// two fold rules behavior-compatible.
func (l *loopalState) bgTask(id string) *loopalBgTask {
	if t := l.bgTasks[id]; t != nil {
		return t
	}
	t := &loopalBgTask{ID: id, Status: "Running"}
	l.bgTasks[id] = t
	l.order = append(l.order, id)
	if len(l.order) > maxBgTasks {
		delete(l.bgTasks, l.order[0])
		l.order = l.order[1:]
	}
	return t
}

func (l *loopalState) applyBgSpawned(data json.RawMessage) {
	var d loopalBgTask
	if json.Unmarshal(data, &d) != nil || d.ID == "" {
		return
	}
	t := l.bgTask(d.ID)
	if t.Description == "" {
		t.Description = d.Description
	}
	if t.CreatedAtUnixMs == 0 {
		t.CreatedAtUnixMs = d.CreatedAtUnixMs
	}
}

func (l *loopalState) applyBgOutput(data json.RawMessage) {
	var d struct {
		ID    string `json:"id"`
		Delta string `json:"output_delta"`
	}
	if json.Unmarshal(data, &d) != nil || d.ID == "" {
		return
	}
	t := l.bgTask(d.ID)
	t.Output = capString(t.Output+d.Delta, maxBgOutput)
}

func (l *loopalState) applyBgCompleted(data json.RawMessage) {
	var d loopalBgTask
	if json.Unmarshal(data, &d) != nil || d.ID == "" {
		return
	}
	t := l.bgTask(d.ID)
	if d.Status != "" {
		t.Status = d.Status
	}
	if d.ExitCode != nil {
		t.ExitCode = d.ExitCode
	}
	if d.Output != "" {
		t.Output = capString(d.Output, maxBgOutput)
	}
}
