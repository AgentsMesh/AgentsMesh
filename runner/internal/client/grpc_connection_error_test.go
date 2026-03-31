package client

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestIsFatalStreamError_Unauthenticated_RunnerNotFound(t *testing.T) {
	err := status.Error(codes.Unauthenticated, "runner not found")

	fatal, msg := isFatalStreamError(err)

	assert.True(t, fatal)
	assert.Contains(t, msg, "deleted from the server")
	assert.Contains(t, msg, "re-register")
}

func TestIsFatalStreamError_Unauthenticated_Other(t *testing.T) {
	err := status.Error(codes.Unauthenticated, "certificate expired")

	fatal, msg := isFatalStreamError(err)

	assert.True(t, fatal)
	assert.Contains(t, msg, "Authentication failed")
	assert.Contains(t, msg, "certificate expired")
}

func TestIsFatalStreamError_PermissionDenied_Disabled(t *testing.T) {
	err := status.Error(codes.PermissionDenied, "runner is disabled")

	fatal, msg := isFatalStreamError(err)

	assert.True(t, fatal)
	assert.Contains(t, msg, "disabled by an administrator")
}

func TestIsFatalStreamError_PermissionDenied_Other(t *testing.T) {
	err := status.Error(codes.PermissionDenied, "org suspended")

	fatal, msg := isFatalStreamError(err)

	assert.True(t, fatal)
	assert.Contains(t, msg, "Permission denied")
	assert.Contains(t, msg, "org suspended")
}

func TestIsFatalStreamError_Unavailable(t *testing.T) {
	err := status.Error(codes.Unavailable, "connection refused")

	fatal, msg := isFatalStreamError(err)

	assert.False(t, fatal)
	assert.Empty(t, msg)
}

func TestIsFatalStreamError_NonGRPCError(t *testing.T) {
	err := errors.New("plain network error")

	fatal, msg := isFatalStreamError(err)

	assert.False(t, fatal)
	assert.Empty(t, msg)
}

func TestIsFatalStreamError_Internal(t *testing.T) {
	err := status.Error(codes.Internal, "internal server error")

	fatal, msg := isFatalStreamError(err)

	assert.False(t, fatal)
	assert.Empty(t, msg)
}
