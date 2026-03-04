package channel

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// createWSPair creates a WebSocket connection pair (server-side conn, client-side conn)
// using httptest + gorilla/websocket. Uses sync.WaitGroup to ensure connection is ready.
// t.Cleanup auto-closes both connections.
func createWSPair(t *testing.T) (*websocket.Conn, *websocket.Conn) {
	t.Helper()
	var serverConn *websocket.Conn
	var wg sync.WaitGroup
	wg.Add(1)

	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Fatalf("upgrade failed: %v", err)
		}
		serverConn = c
		wg.Done()
	}))

	wsURL := "ws" + srv.URL[4:] // http -> ws
	clientConn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		srv.Close()
		t.Fatalf("dial failed: %v", err)
	}

	wg.Wait()
	srv.Close()

	t.Cleanup(func() {
		clientConn.Close()
		serverConn.Close()
	})

	return serverConn, clientConn
}

// testChannelConfig returns a ChannelConfig with short timeouts for testing
func testChannelConfig() ChannelConfig {
	return ChannelConfig{
		KeepAliveDuration:          200 * time.Millisecond,
		PublisherReconnectTimeout:  200 * time.Millisecond,
		SubscriberReconnectTimeout: 200 * time.Millisecond,
		OutputBufferSize:           1024,
		OutputBufferCount:          5,
	}
}

// testManagerConfig returns a ChannelManagerConfig with short timeouts for testing
func testManagerConfig() ChannelManagerConfig {
	return ChannelManagerConfig{
		KeepAliveDuration:          200 * time.Millisecond,
		MaxSubscribersPerPod:       3,
		PublisherReconnectTimeout:  200 * time.Millisecond,
		SubscriberReconnectTimeout: 200 * time.Millisecond,
		PendingConnectionTimeout:   500 * time.Millisecond,
		OutputBufferSize:           1024,
		OutputBufferCount:          5,
	}
}

// waitFor polls a condition function every 10ms until it returns true or timeout
func waitFor(t *testing.T, condition func() bool, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("waitFor: condition not met within timeout")
}
