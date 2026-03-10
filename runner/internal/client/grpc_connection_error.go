// Package client provides gRPC connection management for Runner.
package client

import (
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// setFatalError records a fatal error that should stop reconnection attempts.
func (c *GRPCConnection) setFatalError(err error) {
	c.fatalErrMu.Lock()
	c.fatalErr = err
	c.fatalErrMu.Unlock()
}

// getFatalError returns the fatal error if one has been recorded.
func (c *GRPCConnection) getFatalError() error {
	c.fatalErrMu.Lock()
	defer c.fatalErrMu.Unlock()
	return c.fatalErr
}

// isFatalStreamError checks if a gRPC stream error is fatal (should not retry).
// Returns true and a user-friendly message if the error is fatal.
func isFatalStreamError(err error) (bool, string) {
	st, ok := status.FromError(err)
	if !ok {
		return false, ""
	}

	switch st.Code() {
	case codes.Unauthenticated:
		msg := st.Message()
		if strings.Contains(msg, "runner not found") {
			return true, "This runner has been deleted from the server. Please re-register with: agentsmesh-runner register --server <SERVER> --token <TOKEN> --force"
		}
		return true, "Authentication failed: " + msg + ". Please re-register with: agentsmesh-runner register --server <SERVER> --token <TOKEN> --force"

	case codes.PermissionDenied:
		msg := st.Message()
		if strings.Contains(msg, "runner is disabled") {
			return true, "This runner has been disabled by an administrator. Please contact your organization admin to re-enable it."
		}
		return true, "Permission denied: " + msg

	default:
		return false, ""
	}
}
