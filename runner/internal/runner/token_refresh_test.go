package runner

import (
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestPodWaitForNewToken_Success(t *testing.T) {
	pod := &Pod{}

	// Deliver token in goroutine
	go func() {
		time.Sleep(10 * time.Millisecond)
		pod.DeliverNewToken("new-token")
	}()

	// Wait for token
	token := pod.WaitForNewToken(100 * time.Millisecond)
	assert.Equal(t, "new-token", token)
}

func TestPodWaitForNewToken_Timeout(t *testing.T) {
	pod := &Pod{}

	start := time.Now()
	token := pod.WaitForNewToken(50 * time.Millisecond)
	elapsed := time.Since(start)

	assert.Empty(t, token)
	assert.True(t, elapsed >= 50*time.Millisecond)
	assert.True(t, elapsed < 100*time.Millisecond) // Should not wait too long
}

func TestPodWaitForNewToken_TokenDeliveredBeforeWait(t *testing.T) {
	pod := &Pod{}

	// Deliver token first
	pod.DeliverNewToken("pre-delivered-token")

	// Wait should receive it immediately
	start := time.Now()
	token := pod.WaitForNewToken(100 * time.Millisecond)
	elapsed := time.Since(start)

	assert.Equal(t, "pre-delivered-token", token)
	assert.True(t, elapsed < 50*time.Millisecond) // Should be fast
}

func TestPodDeliverNewToken_NoReceiver(t *testing.T) {
	pod := &Pod{}

	// Deliver token with no receiver - should not block
	done := make(chan struct{})
	go func() {
		pod.DeliverNewToken("token-1")
		close(done)
	}()

	select {
	case <-done:
		// Good, didn't block
	case <-time.After(100 * time.Millisecond):
		t.Fatal("DeliverNewToken blocked when no receiver")
	}

	// Channel should be created with buffer=1, so first token is stored
	// Second delivery should not block (drops the token)
	done2 := make(chan struct{})
	go func() {
		pod.DeliverNewToken("token-2")
		close(done2)
	}()

	select {
	case <-done2:
		// Good, didn't block
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Second DeliverNewToken blocked")
	}

	// Wait should get the first token (second was dropped)
	token := pod.WaitForNewToken(50 * time.Millisecond)
	assert.Equal(t, "token-1", token)
}

func TestPodDeliverNewToken_ChannelCreatedOnDemand(t *testing.T) {
	pod := &Pod{}

	// DeliverNewToken should create channel and not panic
	// We verify this by calling DeliverNewToken then WaitForNewToken
	pod.DeliverNewToken("token")

	// If channel was created, WaitForNewToken should receive the token
	token := pod.WaitForNewToken(50 * time.Millisecond)
	assert.Equal(t, "token", token)
}

func TestPodWaitForNewToken_ChannelCreatedOnDemand(t *testing.T) {
	pod := &Pod{}

	// WaitForNewToken should create channel on demand
	// Verify by starting a wait, then delivering a token
	resultCh := make(chan string, 1)
	go func() {
		token := pod.WaitForNewToken(100 * time.Millisecond)
		resultCh <- token
	}()

	// Give waiter time to start and create channel
	time.Sleep(10 * time.Millisecond)

	// Deliver token - this should succeed because channel was created
	pod.DeliverNewToken("delivered-token")

	// Waiter should receive the token
	select {
	case token := <-resultCh:
		assert.Equal(t, "delivered-token", token)
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for token")
	}
}

func TestPodTokenRefresh_ConcurrentAccess(t *testing.T) {
	pod := &Pod{}

	var wg sync.WaitGroup
	results := make(chan string, 10)

	// Multiple goroutines waiting for token
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			token := pod.WaitForNewToken(200 * time.Millisecond)
			if token != "" {
				results <- token
			}
		}()
	}

	// Give waiters time to start
	time.Sleep(10 * time.Millisecond)

	// Deliver one token
	pod.DeliverNewToken("single-token")

	wg.Wait()
	close(results)

	// Only one waiter should get the token
	count := 0
	for token := range results {
		assert.Equal(t, "single-token", token)
		count++
	}
	assert.Equal(t, 1, count, "Only one waiter should receive the token")
}

func TestPodTokenRefresh_MultipleDeliveries(t *testing.T) {
	pod := &Pod{}

	// Start waiting
	resultCh := make(chan string)
	go func() {
		token := pod.WaitForNewToken(200 * time.Millisecond)
		resultCh <- token
	}()

	// Give waiter time to start
	time.Sleep(10 * time.Millisecond)

	// Deliver multiple tokens quickly
	pod.DeliverNewToken("token-1")
	pod.DeliverNewToken("token-2") // Should be dropped (channel full)
	pod.DeliverNewToken("token-3") // Should be dropped (channel full)

	// Waiter should get first token
	token := <-resultCh
	assert.Equal(t, "token-1", token)
}
