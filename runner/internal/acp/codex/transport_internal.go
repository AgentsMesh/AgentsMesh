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
		if isApprovalRequest(msg.Method) {
			id, _ := msg.GetID()
			t.handleApprovalRequest(id, msg.Params)
		} else {
			t.tracker.RejectRequest(msg)
		}
	}
}

func isApprovalRequest(method string) bool {
	return method == "item/commandExecution/requestApproval" ||
		method == "item/fileChange/requestApproval"
}
