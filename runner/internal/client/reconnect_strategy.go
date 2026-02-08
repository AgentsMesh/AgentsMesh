package client

import (
	"time"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
)

// ReconnectStrategy manages reconnection with exponential backoff.
// Single Responsibility: Calculate reconnection delays.
type ReconnectStrategy struct {
	initialInterval time.Duration
	maxInterval     time.Duration
	currentInterval time.Duration
	attemptCount    int
}

// NewReconnectStrategy creates a new ReconnectStrategy.
func NewReconnectStrategy(initialInterval, maxInterval time.Duration) *ReconnectStrategy {
	return &ReconnectStrategy{
		initialInterval: initialInterval,
		maxInterval:     maxInterval,
		currentInterval: initialInterval,
		attemptCount:    0,
	}
}

// NextDelay returns the next reconnection delay and increments the attempt counter.
func (s *ReconnectStrategy) NextDelay() time.Duration {
	delay := s.currentInterval
	s.attemptCount++

	// Exponential backoff
	s.currentInterval *= 2
	if s.currentInterval > s.maxInterval {
		s.currentInterval = s.maxInterval
	}

	logger.GRPC().Debug("Reconnect delay calculated",
		"attempt", s.attemptCount,
		"delay", delay,
		"next_interval", s.currentInterval)

	return delay
}

// Reset resets the strategy to initial values (call on successful connection).
func (s *ReconnectStrategy) Reset() {
	if s.attemptCount > 0 {
		logger.GRPC().Debug("Reconnect strategy reset", "previous_attempts", s.attemptCount)
	}
	s.currentInterval = s.initialInterval
	s.attemptCount = 0
}

// AttemptCount returns the current reconnection attempt count.
func (s *ReconnectStrategy) AttemptCount() int {
	return s.attemptCount
}

// CurrentInterval returns the current reconnection interval.
func (s *ReconnectStrategy) CurrentInterval() time.Duration {
	return s.currentInterval
}
