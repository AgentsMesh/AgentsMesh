package mcp

import (
	"testing"
)

func TestNewHTTPServer(t *testing.T) {
	server := NewHTTPServer(nil, 9090)

	if server == nil {
		t.Fatal("NewHTTPServer returned nil")
	}

	if server.port != 9090 {
		t.Errorf("port: got %v, want %v", server.port, 9090)
	}

	if len(server.tools) == 0 {
		t.Error("tools should be registered")
	}
}

func TestHTTPServerRegisterPod(t *testing.T) {
	server := NewHTTPServer(nil, 9090)

	ticketID := 123
	projectID := 456

	server.RegisterPod("test-pod", "test-org", &ticketID, &projectID, "claude")

	pod, ok := server.GetPod("test-pod")
	if !ok {
		t.Fatal("pod should be registered")
	}

	if pod.PodKey != "test-pod" {
		t.Errorf("PodKey: got %v, want %v", pod.PodKey, "test-pod")
	}

	if pod.TicketID == nil || *pod.TicketID != 123 {
		t.Errorf("TicketID: got %v, want 123", pod.TicketID)
	}

	if pod.AgentType != "claude" {
		t.Errorf("AgentType: got %v, want %v", pod.AgentType, "claude")
	}
}

func TestHTTPServerUnregisterPod(t *testing.T) {
	server := NewHTTPServer(nil, 9090)

	server.RegisterPod("test-pod", "test-org", nil, nil, "claude")

	_, ok := server.GetPod("test-pod")
	if !ok {
		t.Fatal("pod should be registered")
	}

	server.UnregisterPod("test-pod")

	_, ok = server.GetPod("test-pod")
	if ok {
		t.Error("pod should be unregistered")
	}
}

func TestHTTPServerPodCount(t *testing.T) {
	server := NewHTTPServer(nil, 9090)

	if server.PodCount() != 0 {
		t.Errorf("initial count should be 0, got %v", server.PodCount())
	}

	server.RegisterPod("pod-1", "test-org", nil, nil, "claude")
	if server.PodCount() != 1 {
		t.Errorf("count should be 1, got %v", server.PodCount())
	}

	server.RegisterPod("pod-2", "test-org", nil, nil, "claude")
	if server.PodCount() != 2 {
		t.Errorf("count should be 2, got %v", server.PodCount())
	}

	server.UnregisterPod("pod-1")
	if server.PodCount() != 1 {
		t.Errorf("count should be 1, got %v", server.PodCount())
	}
}

func TestHTTPServerPort(t *testing.T) {
	server := NewHTTPServer(nil, 9090)
	if server.Port() != 9090 {
		t.Errorf("Port: got %v, want %v", server.Port(), 9090)
	}
}

func TestHTTPServerGenerateMCPConfig(t *testing.T) {
	server := NewHTTPServer(nil, 9090)
	config := server.GenerateMCPConfig("test-pod")

	if config == nil {
		t.Fatal("config should not be nil")
	}

	mcpServers, ok := config["mcpServers"].(map[string]interface{})
	if !ok {
		t.Fatal("mcpServers should exist")
	}

	if _, ok := mcpServers["agentsmesh-collaboration"]; !ok {
		t.Error("agentsmesh-collaboration server should exist")
	}
}
