package backend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	c := NewClient("http://localhost:8080", "s", "r1", "ws://a", "us", 1000)
	if c == nil || c.baseURL != "http://localhost:8080" || c.IsRegistered() {
		t.Error("client init failed")
	}
}

func TestClient_Register(t *testing.T) {
	for _, tt := range []struct {
		name    string
		status  int
		wantErr bool
	}{
		{"ok", http.StatusOK, false}, {"err", http.StatusInternalServerError, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()
			c := NewClient(srv.URL, "s", "r1", "ws://a", "us", 1000)
			if err := c.Register(context.Background()); (err != nil) != tt.wantErr {
				t.Error("mismatch")
			}
		})
	}
	c := NewClient("http://127.0.0.1:1", "s", "r1", "ws://a", "us", 1000)
	if c.Register(context.Background()) == nil {
		t.Error("should fail")
	}
}

func TestClient_SendHeartbeat(t *testing.T) {
	for _, tt := range []struct {
		name    string
		status  int
		reg     bool
		wantErr bool
	}{
		{"ok", http.StatusOK, true, false}, {"not_reg", http.StatusOK, false, true},
		{"404", http.StatusNotFound, true, true}, {"500", http.StatusInternalServerError, true, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()
			c := NewClient(srv.URL, "s", "r1", "ws://a", "us", 1000)
			c.mu.Lock()
			c.registered = tt.reg
			c.mu.Unlock()
			if err := c.SendHeartbeat(context.Background(), 5); (err != nil) != tt.wantErr {
				t.Error("mismatch")
			}
		})
	}
	var req HeartbeatRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&req)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "s", "r1", "ws://a", "us", 1000)
	c.mu.Lock()
	c.registered = true
	c.mu.Unlock()
	_ = c.SendHeartbeat(context.Background(), 5)
	if req.Connections != 5 {
		t.Error("data wrong")
	}
	c2 := NewClient("http://127.0.0.1:1", "s", "r1", "ws://a", "us", 1000)
	c2.mu.Lock()
	c2.registered = true
	c2.mu.Unlock()
	if c2.SendHeartbeat(context.Background(), 5) == nil {
		t.Error("should fail")
	}
}

func TestClient_NotifySessionClosed(t *testing.T) {
	for _, tt := range []struct {
		name    string
		status  int
		wantErr bool
	}{
		{"ok", http.StatusOK, false}, {"err", http.StatusInternalServerError, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()
			c := NewClient(srv.URL, "s", "r1", "ws://a", "us", 1000)
			if err := c.NotifySessionClosed(context.Background(), "p1", "s1"); (err != nil) != tt.wantErr {
				t.Error("mismatch")
			}
		})
	}
	c := NewClient("http://127.0.0.1:1", "s", "r1", "ws://a", "us", 1000)
	if c.NotifySessionClosed(context.Background(), "p1", "s1") == nil {
		t.Error("should fail")
	}
}

func TestClient_StartHeartbeat(t *testing.T) {
	var count int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/internal/relays/heartbeat" {
			atomic.AddInt32(&count, 1)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "s", "r1", "ws://a", "us", 1000)
	c.mu.Lock()
	c.registered = true
	c.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	go c.StartHeartbeat(ctx, 40*time.Millisecond, func() int { return 1 })
	<-ctx.Done()
	if atomic.LoadInt32(&count) < 1 {
		t.Error("should heartbeat")
	}
}

func TestClient_StartHeartbeat_ReRegister(t *testing.T) {
	var hb, reg int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/internal/relays/heartbeat" {
			if atomic.AddInt32(&hb, 1) == 1 {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
		}
		if r.URL.Path == "/api/internal/relays/register" {
			atomic.AddInt32(&reg, 1)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "s", "r1", "ws://a", "us", 1000)
	c.mu.Lock()
	c.registered = true
	c.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	go c.StartHeartbeat(ctx, 50*time.Millisecond, func() int { return 1 })
	<-ctx.Done()
	if atomic.LoadInt32(&reg) < 1 {
		t.Error("should re-register")
	}
}

func TestClient_StartHeartbeat_ReRegisterFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	c := NewClient(srv.URL, "s", "r1", "ws://a", "us", 1000)
	c.mu.Lock()
	c.registered = true
	c.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	go c.StartHeartbeat(ctx, 30*time.Millisecond, func() int { return 1 })
	<-ctx.Done()
}

func TestRequestStructs(t *testing.T) {
	reg := RegisterRequest{RelayID: "r1", URL: "ws://x", Region: "us", Capacity: 100}
	hb := HeartbeatRequest{RelayID: "r1", Connections: 50, CPUUsage: 25.5, MemoryUsage: 60.0}
	sc := SessionClosedRequest{PodKey: "p1", SessionID: "s1"}
	if reg.RelayID != "r1" || hb.Connections != 50 || sc.PodKey != "p1" {
		t.Error("fields wrong")
	}
	data, _ := json.Marshal(reg)
	var dec RegisterRequest
	_ = json.Unmarshal(data, &dec)
	if dec.RelayID != reg.RelayID {
		t.Error("roundtrip failed")
	}
}

func TestClient_GetRelayURL(t *testing.T) {
	c := NewClient("http://localhost", "s", "r1", "ws://relay.test", "us", 1000)
	if got := c.GetRelayURL(); got != "ws://relay.test" {
		t.Errorf("GetRelayURL = %q, want %q", got, "ws://relay.test")
	}
}

func TestClient_Unregister(t *testing.T) {
	for _, tt := range []struct {
		name    string
		status  int
		wantErr bool
	}{
		{"ok", http.StatusOK, false},
		{"server_error", http.StatusInternalServerError, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer srv.Close()
			c := NewClient(srv.URL, "s", "r1", "ws://a", "us", 1000)
			c.mu.Lock()
			c.registered = true
			c.mu.Unlock()
			err := c.Unregister(context.Background(), "shutdown")
			if (err != nil) != tt.wantErr {
				t.Errorf("Unregister error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && c.IsRegistered() {
				t.Error("should be unregistered after success")
			}
		})
	}
	t.Run("network_error", func(t *testing.T) {
		c := NewClient("http://127.0.0.1:1", "s", "r1", "ws://a", "us", 1000)
		if c.Unregister(context.Background(), "shutdown") == nil {
			t.Error("should fail on network error")
		}
	})
}

func TestClient_Register_DNSCreated(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RegisterResponse{
			Status: "ok", URL: "wss://us-east-1.relay.example.com", DNSCreated: true,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "s", "r1", "", "us", 1000)
	if err := c.Register(context.Background()); err != nil {
		t.Fatalf("Register error: %v", err)
	}
	if got := c.GetRelayURL(); got != "wss://us-east-1.relay.example.com" {
		t.Errorf("GetRelayURL() = %q, want %q", got, "wss://us-east-1.relay.example.com")
	}
}

func TestClient_Register_DNSCreated_URLWithoutDNSFlag(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RegisterResponse{
			Status: "ok", URL: "wss://ignored.example.com", DNSCreated: false,
		})
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "s", "r1", "ws://original", "us", 1000)
	if err := c.Register(context.Background()); err != nil {
		t.Fatalf("Register error: %v", err)
	}
	if got := c.GetRelayURL(); got != "ws://original" {
		t.Errorf("GetRelayURL() = %q, want %q (should not be updated)", got, "ws://original")
	}
}
