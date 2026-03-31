//go:build integration

package relay

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var integrationUpgrader = websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}

func wsURL(srv *httptest.Server) string {
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

// Tests: connect → send → writeLoop → server → readLoop → handleMessage → handler callback
func TestRelay_FullBidirectionalFlow_Integration(t *testing.T) {
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
			if msg == nil {
				continue
			}
			var replyType byte
			switch msg.Type {
			case MsgTypeOutput:
				replyType = MsgTypeInput
			case MsgTypeAcpEvent:
				replyType = MsgTypeAcpCommand
			default:
				continue
			}
			conn.WriteMessage(websocket.BinaryMessage, EncodeMessage(replyType, msg.Payload))
		}
	}))
	defer srv.Close()

	c := NewClient(wsURL(srv), "pod-1", "tok", nil)
	var inputPayload, acpPayload atomic.Value
	inputCh := make(chan struct{}, 1)
	acpCh := make(chan struct{}, 1)
	c.SetMessageHandler(MsgTypeInput, func(p []byte) { inputPayload.Store(string(p)); inputCh <- struct{}{} })
	c.SetMessageHandler(MsgTypeAcpCommand, func(p []byte) { acpPayload.Store(string(p)); acpCh <- struct{}{} })

	require.NoError(t, c.Connect())
	require.True(t, c.Start())
	defer c.Stop()

	require.NoError(t, c.Send(MsgTypeOutput, []byte("hello")))
	select {
	case <-inputCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for Input echo")
	}
	assert.Equal(t, "hello", inputPayload.Load())

	require.NoError(t, c.Send(MsgTypeAcpEvent, []byte("ev1")))
	select {
	case <-acpCh:
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for AcpCommand echo")
	}
	assert.Equal(t, "ev1", acpPayload.Load())
}

// Tests: readLoop disconnect → reconnectLoop → isHandshakeError → onTokenExpired → UpdateToken → reconnect
func TestRelay_ReconnectWithTokenRefresh_Integration(t *testing.T) {
	var connCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := connCount.Add(1)
		token := r.URL.Query().Get("token")
		if n == 1 {
			conn, err := integrationUpgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			conn.ReadMessage()
			conn.Close()
			return
		}
		if n == 2 && token != "new-token-123" {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		conn, err := integrationUpgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}))
	defer srv.Close()

	c := NewClient(wsURL(srv), "pod-1", "old-token", nil)
	var tokenRefreshed atomic.Bool
	c.SetTokenExpiredHandler(func() string { tokenRefreshed.Store(true); return "new-token-123" })
	reconnected := make(chan struct{}, 1)
	c.SetReconnectHandler(func() { reconnected <- struct{}{} })

	require.NoError(t, c.Connect())
	require.True(t, c.Start())
	defer c.Stop()

	c.Send(MsgTypeOutput, []byte("trigger"))

	select {
	case <-reconnected:
	case <-time.After(10 * time.Second):
		t.Fatal("timeout waiting for reconnect")
	}
	assert.True(t, tokenRefreshed.Load(), "token should have been refreshed")
	assert.True(t, c.IsConnected(), "client should be connected after reconnect")
}
