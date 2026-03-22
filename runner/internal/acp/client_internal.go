package acp

import (
	"bufio"
	"io"
	"os/exec"
	"time"
)

// Stop gracefully shuts down the client and subprocess.
func (c *ACPClient) Stop() {
	c.stopOnce.Do(func() {
		c.cancel()

		if c.transport != nil {
			c.transport.Close()
		}

		if c.cmd != nil && c.cmd.Process != nil {
			select {
			case <-c.waitExitDone:
			case <-time.After(5 * time.Second):
				_ = c.cmd.Process.Kill()
				select {
				case <-c.waitExitDone:
				case <-time.After(2 * time.Second):
				}
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

// waitExit waits for the subprocess to exit and fires the OnExit callback.
// This is the sole owner of cmd.Wait() — Stop() must not call cmd.Wait().
func (c *ACPClient) waitExit() {
	defer close(c.waitExitDone)

	if c.cmd == nil || c.cmd.Process == nil {
		return
	}
	err := c.cmd.Wait()
	select {
	case <-c.ctx.Done():
		return // Normal shutdown via Stop(), don't fire OnExit
	default:
	}
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}
	c.logger.Info("ACP subprocess exited", "exit_code", exitCode)
	if c.cfg.Callbacks.OnExit != nil {
		c.cfg.Callbacks.OnExit(exitCode)
	}
}

// readStderr reads plain text lines from the agent's stderr and
// forwards them to the OnLog callback.
func (c *ACPClient) readStderr(r io.Reader) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if c.cfg.Callbacks.OnLog != nil {
			c.cfg.Callbacks.OnLog("stderr", line)
		}
	}
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
