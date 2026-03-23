package client

import (
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

// TestPriorityChannelUnderLoad verifies control messages are never blocked by terminal output.
func TestPriorityChannelUnderLoad(t *testing.T) {
	conn := NewGRPCConnection("localhost:9443", "test-node", "test-org", "", "", "")

	// Fill terminal channel completely
	for i := 0; i < cap(conn.terminalCh); i++ {
		conn.terminalCh <- &runnerv1.RunnerMessage{}
	}

	// Verify terminal channel is full
	if len(conn.terminalCh) != cap(conn.terminalCh) {
		t.Fatalf("Terminal channel should be full: %d/%d", len(conn.terminalCh), cap(conn.terminalCh))
	}

	// Control channel should still accept messages
	start := time.Now()
	for i := 0; i < 50; i++ {
		select {
		case conn.controlCh <- &runnerv1.RunnerMessage{}:
			// Success
		default:
			t.Fatalf("Control channel blocked at message %d when terminal is full", i)
		}
	}
	elapsed := time.Since(start)

	// Should complete almost instantly (< 10ms)
	if elapsed > 10*time.Millisecond {
		t.Errorf("Control channel took too long: %v (expected < 10ms)", elapsed)
	}

	t.Logf("✅ Control channel accepted 50 messages in %v while terminal channel was full", elapsed)
}

// TestTerminalChannelNonBlocking verifies sendTerminal never blocks.
func TestTerminalChannelNonBlocking(t *testing.T) {
	conn := NewGRPCConnection("localhost:9443", "test-node", "test-org", "", "", "")

	// Fill terminal channel
	for i := 0; i < cap(conn.terminalCh); i++ {
		conn.terminalCh <- &runnerv1.RunnerMessage{}
	}

	// sendTerminal should return immediately (non-blocking drop)
	start := time.Now()
	for i := 0; i < 1000; i++ {
		msg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_PodOutput{
				PodOutput: &runnerv1.PodOutputEvent{
					PodKey: "test-pod",
					Data:   []byte("test data"),
				},
			},
		}
		// Note: sendTerminal requires stream to be connected, so we test the channel directly
		select {
		case conn.terminalCh <- msg:
			t.Fatal("Should not accept when full")
		default:
			// Expected: drop silently
		}
	}
	elapsed := time.Since(start)

	// 1000 drops should complete in < 50ms (generous for CI containers with resource contention)
	if elapsed > 50*time.Millisecond {
		t.Errorf("Terminal drop took too long: %v (expected < 50ms)", elapsed)
	}

	t.Logf("✅ 1000 terminal messages dropped in %v (non-blocking)", elapsed)
}

// TestConcurrentChannelAccess verifies thread-safety under concurrent load.
func TestConcurrentChannelAccess(t *testing.T) {
	conn := NewGRPCConnection("localhost:9443", "test-node", "test-org", "", "", "")

	var wg sync.WaitGroup
	var controlSent, terminalSent, terminalDropped int64

	numWriters := 20
	messagesPerWriter := 500

	// Drain channels in background
	stopDrain := make(chan struct{})
	go func() {
		for {
			select {
			case <-stopDrain:
				return
			case <-conn.controlCh:
			case <-conn.terminalCh:
			default:
				time.Sleep(100 * time.Microsecond)
			}
		}
	}()

	// Concurrent control message writers
	for i := 0; i < numWriters/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < messagesPerWriter; j++ {
				select {
				case conn.controlCh <- &runnerv1.RunnerMessage{}:
					atomic.AddInt64(&controlSent, 1)
				default:
					// Control channel full (rare)
				}
			}
		}()
	}

	// Concurrent terminal message writers
	for i := 0; i < numWriters/2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < messagesPerWriter; j++ {
				select {
				case conn.terminalCh <- &runnerv1.RunnerMessage{}:
					atomic.AddInt64(&terminalSent, 1)
				default:
					atomic.AddInt64(&terminalDropped, 1)
				}
			}
		}()
	}

	wg.Wait()
	close(stopDrain)

	t.Logf("✅ Concurrent access test completed:")
	t.Logf("   Control sent: %d", controlSent)
	t.Logf("   Terminal sent: %d, dropped: %d", terminalSent, terminalDropped)
}

// TestQueueUsageAccuracy verifies QueueUsage returns correct ratio.
func TestQueueUsageAccuracy(t *testing.T) {
	conn := NewGRPCConnection("localhost:9443", "test-node", "test-org", "", "", "")

	// Empty queue
	usage := conn.QueueUsage()
	if usage != 0.0 {
		t.Errorf("Empty queue usage should be 0.0, got %f", usage)
	}

	// Fill 50%
	halfCap := cap(conn.terminalCh) / 2
	for i := 0; i < halfCap; i++ {
		conn.terminalCh <- &runnerv1.RunnerMessage{}
	}
	usage = conn.QueueUsage()
	if usage < 0.49 || usage > 0.51 {
		t.Errorf("50%% filled queue usage should be ~0.5, got %f", usage)
	}

	// Fill to 100%
	for i := halfCap; i < cap(conn.terminalCh); i++ {
		conn.terminalCh <- &runnerv1.RunnerMessage{}
	}
	usage = conn.QueueUsage()
	if usage != 1.0 {
		t.Errorf("Full queue usage should be 1.0, got %f", usage)
	}

	t.Logf("✅ QueueUsage accuracy verified: 0%% → 50%% → 100%%")
}

// TestHighThroughputOutput simulates multiple pods with high output.
func TestHighThroughputOutput(t *testing.T) {
	conn := NewGRPCConnection("localhost:9443", "test-node", "test-org", "", "", "")

	var wg sync.WaitGroup
	var totalSent, totalDropped int64

	numPods := 5
	outputsPerPod := 10000

	// Simulate slow consumer
	stopConsumer := make(chan struct{})
	go func() {
		for {
			select {
			case <-stopConsumer:
				return
			case <-conn.terminalCh:
				time.Sleep(100 * time.Microsecond) // Slow consumer
			}
		}
	}()

	start := time.Now()

	// Simulate multiple pods sending high-volume output
	for pod := 0; pod < numPods; pod++ {
		wg.Add(1)
		go func(podID int) {
			defer wg.Done()
			for i := 0; i < outputsPerPod; i++ {
				select {
				case conn.terminalCh <- &runnerv1.RunnerMessage{}:
					atomic.AddInt64(&totalSent, 1)
				default:
					atomic.AddInt64(&totalDropped, 1)
				}
			}
		}(pod)
	}

	wg.Wait()
	elapsed := time.Since(start)
	close(stopConsumer)

	total := totalSent + totalDropped
	dropRate := float64(totalDropped) / float64(total) * 100

	t.Logf("✅ High throughput test (%d pods × %d outputs):", numPods, outputsPerPod)
	t.Logf("   Total: %d, Sent: %d, Dropped: %d (%.1f%%)", total, totalSent, totalDropped, dropRate)
	t.Logf("   Elapsed: %v, Throughput: %.0f msg/sec", elapsed, float64(total)/elapsed.Seconds())

	// Should handle at least 100k msg/sec
	throughput := float64(total) / elapsed.Seconds()
	if throughput < 100000 {
		t.Errorf("Throughput too low: %.0f msg/sec (expected > 100k)", throughput)
	}
}

// TestControlChannelNeverStarved verifies control messages always get through.
func TestControlChannelNeverStarved(t *testing.T) {
	conn := NewGRPCConnection("localhost:9443", "test-node", "test-org", "", "", "")

	var wg sync.WaitGroup
	var controlSuccess, controlFailed int64

	// Heavy terminal load
	stopTerminal := make(chan struct{})
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stopTerminal:
					return
				case conn.terminalCh <- &runnerv1.RunnerMessage{}:
				default:
					runtime.Gosched()
				}
			}
		}()
	}

	// Drain terminal slowly
	go func() {
		for {
			select {
			case <-stopTerminal:
				return
			case <-conn.terminalCh:
				time.Sleep(1 * time.Millisecond)
			}
		}
	}()

	// Simulate heartbeats (every 100ms for 2 seconds)
	start := time.Now()
	for i := 0; i < 20; i++ {
		select {
		case conn.controlCh <- &runnerv1.RunnerMessage{}:
			atomic.AddInt64(&controlSuccess, 1)
		default:
			atomic.AddInt64(&controlFailed, 1)
		}
		time.Sleep(100 * time.Millisecond)
	}
	elapsed := time.Since(start)

	close(stopTerminal)
	wg.Wait()

	// Drain remaining control messages
	for len(conn.controlCh) > 0 {
		<-conn.controlCh
	}

	t.Logf("✅ Control channel starvation test:")
	t.Logf("   Success: %d, Failed: %d (over %v)", controlSuccess, controlFailed, elapsed)

	// All heartbeats should succeed
	if controlFailed > 0 {
		t.Errorf("Control messages should never fail: %d failed", controlFailed)
	}
}
