package runner

import "errors"

// Service errors - business logic errors
var (
	ErrRunnerNotFound      = errors.New("runner not found")
	ErrRunnerOffline       = errors.New("runner is offline")
	ErrInvalidToken        = errors.New("invalid registration token")
	ErrTokenExpired        = errors.New("registration token expired")
	ErrTokenExhausted      = errors.New("registration token usage exhausted")
	ErrRunnerAlreadyExists = errors.New("runner already exists")
	ErrRunnerDisabled      = errors.New("runner is disabled")
	ErrRunnerQuotaExceeded = errors.New("runner quota exceeded")
)

// Connection errors - runner connection related errors
var (
	ErrRunnerNotConnected   = errors.New("runner not connected")
	ErrRunnerNotInitialized = errors.New("runner not initialized")
	ErrConnectionClosed     = errors.New("connection closed")
	ErrSendBufferFull       = errors.New("send buffer is full")
	ErrCommandSenderNotSet  = errors.New("command sender not configured")
)
