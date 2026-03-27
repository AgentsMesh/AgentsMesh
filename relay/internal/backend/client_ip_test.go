package backend

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// errorReader is an io.Reader that always returns an error
type errorReader struct{}

func (e *errorReader) Read(p []byte) (int, error) {
	return 0, fmt.Errorf("simulated read error")
}

func TestClient_Register_WithAutoIP(t *testing.T) {
	var capturedReq RegisterRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&capturedReq)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RegisterResponse{Status: "ok"})
	}))
	defer srv.Close()

	c := NewClientWithConfig(ClientConfig{
		BaseURL: srv.URL, InternalAPISecret: "s",
		RelayID: "r1", RelayName: "us-east-1", RelayRegion: "us", RelayCapacity: 1000,
		AutoIP: true,
	})

	origTransport := c.httpClient.Transport
	c.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Host, "ipify") ||
			strings.Contains(r.URL.Host, "ifconfig") ||
			strings.Contains(r.URL.Host, "icanhazip") {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("5.6.7.8")),
			}, nil
		}
		if origTransport != nil {
			return origTransport.RoundTrip(r)
		}
		return http.DefaultTransport.RoundTrip(r)
	})

	if err := c.Register(context.Background()); err != nil {
		t.Fatalf("Register error: %v", err)
	}
	if capturedReq.IP != "5.6.7.8" {
		t.Errorf("IP = %q, want 5.6.7.8", capturedReq.IP)
	}
	if capturedReq.RelayName != "us-east-1" {
		t.Errorf("RelayName = %q, want us-east-1", capturedReq.RelayName)
	}
}

func TestClient_Register_WithAutoIP_Failure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(RegisterResponse{Status: "ok"})
	}))
	defer srv.Close()

	c := NewClientWithConfig(ClientConfig{
		BaseURL: srv.URL, InternalAPISecret: "s",
		RelayID: "r1", RelayName: "us-east-1", RelayRegion: "us", RelayCapacity: 1000,
		AutoIP: true,
	})

	c.httpClient.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Host, "ipify") ||
			strings.Contains(r.URL.Host, "ifconfig") ||
			strings.Contains(r.URL.Host, "icanhazip") {
			return nil, fmt.Errorf("connection refused")
		}
		return http.DefaultTransport.RoundTrip(r)
	})

	if err := c.Register(context.Background()); err != nil {
		t.Fatalf("Register should succeed even when IP detection fails: %v", err)
	}
	if c.relayIP != "" {
		t.Errorf("relayIP should be empty when detection fails, got %q", c.relayIP)
	}
}

func TestClient_DetectPublicIP(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		c := NewClient("http://localhost", "s", "r1", "ws://a", "us", 1000)
		c.httpClient = &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("1.2.3.4\n")),
				}, nil
			}),
		}
		ip, err := c.detectPublicIP(context.Background())
		if err != nil {
			t.Fatalf("detectPublicIP error: %v", err)
		}
		if ip != "1.2.3.4" {
			t.Errorf("ip = %q, want 1.2.3.4", ip)
		}
	})
	t.Run("all_fail", func(t *testing.T) {
		c := NewClient("http://localhost", "s", "r1", "ws://a", "us", 1000)
		c.httpClient = &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return nil, fmt.Errorf("connection refused")
			}),
		}
		_, err := c.detectPublicIP(context.Background())
		if err == nil {
			t.Error("should error when all services fail")
		}
	})
	t.Run("html_response_skipped", func(t *testing.T) {
		c := NewClient("http://localhost", "s", "r1", "ws://a", "us", 1000)
		c.httpClient = &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("<html>blocked</html>")),
				}, nil
			}),
		}
		_, err := c.detectPublicIP(context.Background())
		if err == nil {
			t.Error("should error when all responses contain HTML")
		}
	})
	t.Run("body_read_error", func(t *testing.T) {
		c := NewClient("http://localhost", "s", "r1", "ws://a", "us", 1000)
		c.httpClient = &http.Client{
			Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(&errorReader{}),
				}, nil
			}),
		}
		_, err := c.detectPublicIP(context.Background())
		if err == nil {
			t.Error("should error when body read fails")
		}
	})
}
