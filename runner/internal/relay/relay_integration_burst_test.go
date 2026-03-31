//go:build integration

package relay

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests: writeLoop buffering, readLoop throughput, message integrity under load
func TestRelay_MultiMessageBurst_Integration(t *testing.T) {
	const burstSize = 50
	var mu sync.Mutex
	serverReceived := make([][]byte, 0, burstSize)
	allReceived := make(chan struct{})

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := integrationUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			msg, _ := DecodeMessage(data)
			if msg == nil || msg.Type != MsgTypeOutput {
				continue
			}
			mu.Lock()
			serverReceived = append(serverReceived, msg.Payload)
			done := len(serverReceived) == burstSize
			mu.Unlock()
			if done {
				close(allReceived)
				mu.Lock()
				for _, p := range serverReceived {
					conn.WriteMessage(websocket.BinaryMessage, EncodeMessage(MsgTypeInput, p))
				}
				mu.Unlock()
				for {
					if _, _, err := conn.ReadMessage(); err != nil {
						return
					}
				}
			}
		}
	}))
	defer srv.Close()

	c := NewClient(wsURL(srv), "pod-1", "tok", nil)
	var clientMu sync.Mutex
	clientReceived := make([]string, 0, burstSize)
	clientDone := make(chan struct{})
	c.SetMessageHandler(MsgTypeInput, func(p []byte) {
		clientMu.Lock()
		clientReceived = append(clientReceived, string(p))
		done := len(clientReceived) == burstSize
		clientMu.Unlock()
		if done {
			close(clientDone)
		}
	})

	require.NoError(t, c.Connect())
	require.True(t, c.Start())
	defer c.Stop()

	for i := 0; i < burstSize; i++ {
		require.NoError(t, c.Send(MsgTypeOutput, []byte{byte(i)}))
	}

	select {
	case <-allReceived:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout: server didn't receive all messages")
	}
	mu.Lock()
	assert.Len(t, serverReceived, burstSize)
	for i, p := range serverReceived {
		assert.Equal(t, []byte{byte(i)}, p, "message %d mismatch", i)
	}
	mu.Unlock()

	select {
	case <-clientDone:
	case <-time.After(5 * time.Second):
		t.Fatal("timeout: client didn't receive all echoed messages")
	}
	clientMu.Lock()
	assert.Len(t, clientReceived, burstSize)
	clientMu.Unlock()
}

// Tests: real-world pattern where Relay sends a full snapshot after reconnect
func TestRelay_SnapshotAfterReconnect_Integration(t *testing.T) {
	var connCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := connCount.Add(1)
		conn, err := integrationUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		if n == 1 {
			conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseGoingAway, ""))
			return
		}
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	c := NewClient(wsURL(srv), "pod-1", "tok", nil)
	snapshotSent := make(chan struct{}, 1)
	c.SetReconnectHandler(func() {
		c.Send(MsgTypeSnapshot, []byte("full-state"))
		snapshotSent <- struct{}{}
	})

	require.NoError(t, c.Connect())
	require.True(t, c.Start())
	defer c.Stop()

	select {
	case <-snapshotSent:
	case <-time.After(10 * time.Second):
		t.Fatal("timeout: snapshot not sent after reconnect")
	}
	assert.True(t, c.IsConnected())
	assert.GreaterOrEqual(t, int(connCount.Load()), 2, "should have connected at least twice")
}
