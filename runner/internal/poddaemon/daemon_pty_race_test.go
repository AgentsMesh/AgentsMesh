package poddaemon

import (
	"net"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestConcurrentResizeGetSize verifies that Resize and GetSize can be called
// concurrently without data race. This test is meaningful with -race flag.
func TestConcurrentResizeGetSize(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer serverConn.Close()
	d := newDaemonPTY(clientConn, 42, 80, 24)
	defer d.Close()

	// Consume resize messages on server side to prevent blocking
	go func() {
		for {
			_, _, err := ReadMessage(serverConn)
			if err != nil {
				return
			}
		}
	}()

	var wg sync.WaitGroup
	const iterations = 100

	// Writer goroutine: rapidly resize
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			_ = d.Resize(80+i%40, 24+i%20)
		}
	}()

	// Reader goroutine: rapidly get size
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < iterations; i++ {
			cols, rows, err := d.GetSize()
			assert.NoError(t, err)
			assert.Greater(t, cols, 0)
			assert.Greater(t, rows, 0)
		}
	}()

	wg.Wait()
}
