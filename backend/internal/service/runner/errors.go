package runner

import (
	"errors"

	runnerDomain "github.com/anthropics/agentsmesh/backend/internal/domain/runner"
)

var (
	ErrRunnerNotFound      = errors.New("runner not found")
	ErrRunnerOffline       = errors.New("runner is offline")
	ErrInvalidToken        = errors.New("invalid registration token")
	ErrTokenExpired        = errors.New("registration token expired")
	ErrTokenExhausted      = runnerDomain.ErrTokenExhausted // shared with infra layer
	ErrRunnerAlreadyExists = errors.New("runner already exists")
	ErrRunnerDisabled      = errors.New("runner is disabled")
	ErrRunnerQuotaExceeded = errors.New("runner quota exceeded")
	ErrGRPCTokenNotFound   = errors.New("gRPC registration token not found")
	ErrNoRunnerForAgent    = errors.New("no available runner supports the requested agent")
	ErrRunnerHasLoopRefs   = errors.New("cannot delete: runner is referenced by one or more loops")

	ErrCertificateMismatch = errors.New("certificate mismatch")

	ErrAuthRequestNotFound          = errors.New("auth request not found")
	ErrAuthRequestExpired           = errors.New("auth request expired")
	ErrAuthRequestAlreadyAuthorized = errors.New("auth request already authorized")
)

var (
	ErrRunnerNotConnected   = errors.New("runner not connected")
	ErrRunnerNotInitialized = errors.New("runner not initialized")
	ErrConnectionClosed     = errors.New("connection closed")
	ErrSendBufferFull       = errors.New("send buffer is full")
	ErrCommandSenderNotSet  = errors.New("command sender not configured")
)
