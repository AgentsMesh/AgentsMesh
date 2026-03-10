package client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestReconnectStrategy_ExponentialBackoff(t *testing.T) {
	s := NewReconnectStrategy(1*time.Second, 1*time.Minute)

	d1 := s.NextDelay()
	assert.Equal(t, 1*time.Second, d1)

	d2 := s.NextDelay()
	assert.Equal(t, 2*time.Second, d2)

	d3 := s.NextDelay()
	assert.Equal(t, 4*time.Second, d3)

	d4 := s.NextDelay()
	assert.Equal(t, 8*time.Second, d4)
}

func TestReconnectStrategy_MaxDelay(t *testing.T) {
	s := NewReconnectStrategy(1*time.Second, 4*time.Second)

	s.NextDelay() // 1s
	s.NextDelay() // 2s
	s.NextDelay() // 4s (max)
	d := s.NextDelay()
	assert.Equal(t, 4*time.Second, d, "should be capped at maxInterval")
}

func TestReconnectStrategy_Reset(t *testing.T) {
	s := NewReconnectStrategy(1*time.Second, 1*time.Minute)

	s.NextDelay()
	s.NextDelay()
	assert.Equal(t, 2, s.AttemptCount())

	s.Reset()
	assert.Equal(t, 0, s.AttemptCount())
	assert.Equal(t, 1*time.Second, s.CurrentInterval())

	// After reset, should start from initial again
	d := s.NextDelay()
	assert.Equal(t, 1*time.Second, d)
}

func TestReconnectStrategy_AttemptCount(t *testing.T) {
	s := NewReconnectStrategy(1*time.Second, 1*time.Minute)
	assert.Equal(t, 0, s.AttemptCount())

	s.NextDelay()
	assert.Equal(t, 1, s.AttemptCount())

	s.NextDelay()
	assert.Equal(t, 2, s.AttemptCount())
}

func TestReconnectStrategy_CurrentInterval(t *testing.T) {
	s := NewReconnectStrategy(5*time.Second, 30*time.Second)
	assert.Equal(t, 5*time.Second, s.CurrentInterval())

	s.NextDelay()
	assert.Equal(t, 10*time.Second, s.CurrentInterval())
}

func TestReconnectStrategy_ResetIdempotent(t *testing.T) {
	s := NewReconnectStrategy(1*time.Second, 1*time.Minute)

	// Reset without any attempts should not panic
	s.Reset()
	s.Reset()
	assert.Equal(t, 0, s.AttemptCount())
}
