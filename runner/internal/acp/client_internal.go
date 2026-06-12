package acp

import (
	"bufio"
	"context"
	"io"
	"strings"
	"time"
)

// Stop gracefully shuts down the client and subprocess. processmgr.Handle.Stop
// implements the SIGTERM → wait → SIGKILL escalation that this function used
// to inline; the reapLoop inside processmgr is the single Wait point.
func (c *ACPClient) Stop() {
	c.stopOnce.Do(func() {
		c.cancel()

		if c.transport != nil {
			c.transport.Close()
		}

		if c.proc != nil {
			stopCtx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
			defer cancel()
			if err := c.proc.Stop(stopCtx); err != nil {
				c.logger.Warn("ACP subprocess stop reported error", "err", err)
			}
		}

		c.setState(StateStopped)
		close(c.done)
	})
}

// Done returns a channel that closes when the client stops.
func (c *ACPClient) Done() <-chan struct{} {
	return c.done
}

// watchExit fires the OnExit callback after the subprocess has been reaped by
// processmgr. cmd.Wait() is no longer called here — processmgr owns it — but
// the OnExit semantic (only on non-graceful exits) is preserved by checking
// the client's context.
func (c *ACPClient) watchExit() {
	if c.proc == nil {
		return
	}
	<-c.proc.Done()

	select {
	case <-c.ctx.Done():
		return
	default:
	}

	exitCode := 0
	if info, ok := c.proc.ExitInfo(); ok {
		exitCode = info.Code
	}
	c.logger.Info("ACP subprocess exited", "exit_code", exitCode)
	if c.cfg.Callbacks.OnExit != nil {
		c.cfg.Callbacks.OnExit(exitCode)
	}
}

// readStderr reads plain text lines from the agent's stderr and
// forwards them to the OnLog callback. Lines prefixed with
// "warn:" / "error:" are tagged with the matching level so the
// frontend (which suppresses level=stderr) can render them; we
// also feed warn/error into c.logs so late-subscribing snapshots
// surface them.
func (c *ACPClient) readStderr(r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		level, message := classifyStderrLine(line)
		c.addLog(LogEntry{Level: level, Message: message})
		if c.cfg.Callbacks.OnLog != nil {
			c.cfg.Callbacks.OnLog(level, message)
		}
	}
}

// classifyStderrLine inspects a stderr line for an explicit level prefix
// ("warn:" / "error:", case-insensitive). When none is found it falls back
// to the historical "stderr" level so existing consumers keep their tagging.
func classifyStderrLine(line string) (level, message string) {
	trimmed := strings.TrimLeft(line, " \t")
	lower := strings.ToLower(trimmed)
	switch {
	case strings.HasPrefix(lower, "error:"):
		return "error", strings.TrimSpace(trimmed[len("error:"):])
	case strings.HasPrefix(lower, "warn:"), strings.HasPrefix(lower, "warning:"):
		idx := strings.IndexByte(trimmed, ':')
		return "warn", strings.TrimSpace(trimmed[idx+1:])
	}
	return "stderr", line
}

// addMessage appends a content chunk to the message history,
// trimming oldest entries when the limit is exceeded.
func (c *ACPClient) addMessage(chunk ContentChunk) {
	c.messagesMu.Lock()
	defer c.messagesMu.Unlock()

	c.messages = append(c.messages, chunk)
	if len(c.messages) > c.maxMessages {
		c.messages = c.messages[len(c.messages)-c.maxMessages:]
	}
}

// upsertToolCall inserts or updates a tool call in the snapshot store.
func (c *ACPClient) upsertToolCall(update ToolCallUpdate) {
	c.toolCallsMu.Lock()
	defer c.toolCallsMu.Unlock()

	existing, ok := c.toolCalls[update.ToolCallID]
	if ok {
		existing.Status = update.Status
		if update.ArgumentsJSON != "" {
			existing.ArgumentsJSON = update.ArgumentsJSON
		}
	} else {
		c.toolCalls[update.ToolCallID] = &ToolCallSnapshot{
			ToolCallID:    update.ToolCallID,
			ToolName:      update.ToolName,
			Status:        update.Status,
			ArgumentsJSON: update.ArgumentsJSON,
		}
	}
}

// applyToolCallResult merges a tool result into an existing tool call entry.
func (c *ACPClient) applyToolCallResult(result ToolCallResult) {
	c.toolCallsMu.Lock()
	defer c.toolCallsMu.Unlock()

	tc, ok := c.toolCalls[result.ToolCallID]
	if !ok {
		// Result arrived before update (shouldn't happen, but handle gracefully)
		s := result.Success
		c.toolCalls[result.ToolCallID] = &ToolCallSnapshot{
			ToolCallID:   result.ToolCallID,
			ToolName:     result.ToolName,
			Status:       "completed",
			Success:      &s,
			ResultText:   result.ResultText,
			ErrorMessage: result.ErrorMessage,
		}
		return
	}
	s := result.Success
	tc.Success = &s
	tc.ResultText = result.ResultText
	tc.ErrorMessage = result.ErrorMessage
	tc.Status = "completed"
}

// setPlan replaces the current plan.
func (c *ACPClient) setPlan(steps []PlanStep) {
	c.planMu.Lock()
	defer c.planMu.Unlock()
	c.plan = steps
}

// addThinking appends a thinking chunk to the snapshot history, trimming
// oldest entries when the cap is exceeded. The frontend tracks a finer-grained
// "complete" flag via incremental events; the snapshot only carries raw text.
func (c *ACPClient) addThinking(update ThinkingUpdate) {
	c.thinkingsMu.Lock()
	defer c.thinkingsMu.Unlock()
	c.thinkings = append(c.thinkings, update)
	if len(c.thinkings) > c.maxThinkings {
		c.thinkings = c.thinkings[len(c.thinkings)-c.maxThinkings:]
	}
}

// addLog appends a log entry to the snapshot history (warn/error only — info
// is discarded to avoid log-flood polluting late subscribers).
func (c *ACPClient) addLog(entry LogEntry) {
	if entry.Level != "warn" && entry.Level != "error" {
		return
	}
	c.logsMu.Lock()
	defer c.logsMu.Unlock()
	c.logs = append(c.logs, entry)
	if len(c.logs) > c.maxLogs {
		c.logs = c.logs[len(c.logs)-c.maxLogs:]
	}
}

// applyConfiguration merges a ConfigUpdate into the internal Configuration,
// returning the merged result for broadcast. Empty fields in the update are
// treated as "no change"; non-empty fields replace the existing value.
func (c *ACPClient) applyConfiguration(update ConfigUpdate) Configuration {
	c.configMu.Lock()
	defer c.configMu.Unlock()
	if update.PermissionMode != "" {
		c.configuration.PermissionMode = update.PermissionMode
	}
	if update.Model != "" {
		c.configuration.Model = update.Model
	}
	return c.configuration
}

// Configuration returns a snapshot of the current configuration.
func (c *ACPClient) Configuration() Configuration {
	c.configMu.RLock()
	defer c.configMu.RUnlock()
	return c.configuration
}

// SupportedPermissionModes returns the permission-mode wire values the agent
// advertised at initialize (nil if none). pod_io_acp validates
// set_permission_mode against this instead of a hardcoded Claude allowlist.
func (c *ACPClient) SupportedPermissionModes() []string {
	c.configMu.RLock()
	defer c.configMu.RUnlock()
	return c.configuration.SupportedPermissionModes
}

// SeedConfiguration merges the initial configuration captured at pod creation
// into the live Configuration, before any control_request flows. Field-level
// (empty = unchanged) so a later capture (client.go Start writing
// SupportedPermissionModes) isn't clobbered regardless of call ordering.
// Callers must invoke OnConfigChange themselves to broadcast the seeded values.
func (c *ACPClient) SeedConfiguration(cfg Configuration) {
	c.configMu.Lock()
	defer c.configMu.Unlock()
	if cfg.PermissionMode != "" {
		c.configuration.PermissionMode = cfg.PermissionMode
	}
	if cfg.Model != "" {
		c.configuration.Model = cfg.Model
	}
	if len(cfg.SupportedPermissionModes) > 0 {
		c.configuration.SupportedPermissionModes = cfg.SupportedPermissionModes
	}
}
