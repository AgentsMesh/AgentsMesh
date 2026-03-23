package codex

import (
	"github.com/anthropics/agentsmesh/runner/internal/acp"
)

func (t *Transport) dispatchMessage(msg *acp.JSONRPCMessage) {
	switch {
	case msg.IsResponse():
		t.tracker.HandleResponse(msg)
	case msg.IsNotification():
		t.handleNotification(msg.Method, msg.Params)
	case msg.IsRequest():
		t.tracker.RejectRequest(msg)
	}
}
