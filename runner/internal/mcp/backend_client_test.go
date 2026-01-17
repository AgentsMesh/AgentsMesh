package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/mcp/tools"
)

func TestNewBackendClient(t *testing.T) {
	client := NewBackendClient("http://localhost:8080", "test-org", "test-pod")

	if client == nil {
		t.Fatal("NewBackendClient returned nil")
	}

	if client.baseURL != "http://localhost:8080" {
		t.Errorf("baseURL: got %v, want %v", client.baseURL, "http://localhost:8080")
	}

	if client.orgSlug != "test-org" {
		t.Errorf("orgSlug: got %v, want %v", client.orgSlug, "test-org")
	}

	if client.podKey != "test-pod" {
		t.Errorf("podKey: got %v, want %v", client.podKey, "test-pod")
	}

	if client.httpClient == nil {
		t.Error("httpClient should not be nil")
	}
}

func TestSetPodKey(t *testing.T) {
	client := NewBackendClient("http://localhost:8080", "test-org", "old-pod")
	client.SetPodKey("new-pod")

	if client.podKey != "new-pod" {
		t.Errorf("podKey: got %v, want %v", client.podKey, "new-pod")
	}
}

func TestObserveTerminal(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: got %v, want GET", r.Method)
		}

		if r.Header.Get("X-Pod-Key") != "test-pod" {
			t.Errorf("X-Pod-Key: got %v, want test-pod", r.Header.Get("X-Pod-Key"))
		}

		resp := tools.TerminalOutput{
			PodKey: "target-pod",
			Output:     "test output",
			CursorX:    10,
			CursorY:    5,
			TotalLines: 100,
			HasMore:    true,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	result, err := client.ObserveTerminal(context.Background(), "target-pod", 50, false, true)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Output != "test output" {
		t.Errorf("Output: got %v, want test output", result.Output)
	}

	if result.CursorX != 10 {
		t.Errorf("CursorX: got %v, want 10", result.CursorX)
	}
}

func TestSendTerminalText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method: got %v, want POST", r.Method)
		}

		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["input"] != "hello world" {
			t.Errorf("input: got %v, want hello world", body["input"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	err := client.SendTerminalText(context.Background(), "target-pod", "hello world")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSendTerminalKey(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		// The client converts keys to escape sequences and sends them as "input"
		input, ok := body["input"].(string)
		if !ok || input == "" {
			t.Errorf("input: got %v, want non-empty string", body["input"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	err := client.SendTerminalKey(context.Background(), "target-pod", []string{"ctrl+c", "enter"})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListAvailablePods(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"pods": []tools.AvailablePod{
				{PodKey: "pod-1", Status: tools.PodStatusRunning},
				{PodKey: "pod-2", Status: tools.PodStatusRunning},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	pods, err := client.ListAvailablePods(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pods) != 2 {
		t.Errorf("pods count: got %v, want 2", len(pods))
	}
}

func TestRequestBinding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["target_pod"] != "target-pod" {
			t.Errorf("target_pod: got %v, want target-pod", body["target_pod"])
		}

		resp := map[string]interface{}{
			"binding": tools.Binding{
				ID:           1,
				InitiatorPod: "test-pod",
				TargetPod:    "target-pod",
				Status:       tools.BindingStatusPending,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	binding, err := client.RequestBinding(context.Background(), "target-pod", []tools.BindingScope{tools.ScopeTerminalRead})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if binding.Status != tools.BindingStatusPending {
		t.Errorf("status: got %v, want pending", binding.Status)
	}
}

func TestAcceptBinding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["binding_id"].(float64) != 1 {
			t.Errorf("binding_id: got %v, want 1", body["binding_id"])
		}

		resp := map[string]interface{}{
			"binding": tools.Binding{
				ID:     1,
				Status: tools.BindingStatusActive,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	binding, err := client.AcceptBinding(context.Background(), 1)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if binding.Status != tools.BindingStatusActive {
		t.Errorf("status: got %v, want active", binding.Status)
	}
}

func TestRejectBinding(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["reason"] != "not allowed" {
			t.Errorf("reason: got %v, want not allowed", body["reason"])
		}

		resp := map[string]interface{}{
			"binding": tools.Binding{
				ID:     1,
				Status: tools.BindingStatusRejected,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	binding, err := client.RejectBinding(context.Background(), 1, "not allowed")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if binding.Status != tools.BindingStatusRejected {
		t.Errorf("status: got %v, want rejected", binding.Status)
	}
}

func TestUnbindPod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	err := client.UnbindPod(context.Background(), "target-pod")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetBindings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"bindings": []tools.Binding{
				{ID: 1, Status: tools.BindingStatusActive},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	bindings, err := client.GetBindings(context.Background(), nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(bindings) != 1 {
		t.Errorf("bindings count: got %v, want 1", len(bindings))
	}
}

func TestGetBindingsWithStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("status") != "active" {
			t.Errorf("status param: got %v, want active", r.URL.Query().Get("status"))
		}

		resp := map[string]interface{}{
			"bindings": []tools.Binding{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	status := tools.BindingStatusActive
	_, err := client.GetBindings(context.Background(), &status)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetBoundPods(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Backend returns pod keys as strings, not AvailablePod objects
		resp := map[string]interface{}{
			"pods":  []string{"bound-pod-1", "bound-pod-2"},
			"count": 2,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	pods, err := client.GetBoundPods(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(pods) != 2 {
		t.Errorf("pods count: got %v, want 2", len(pods))
	}

	if pods[0] != "bound-pod-1" {
		t.Errorf("first pod: got %v, want bound-pod-1", pods[0])
	}
}

func TestSearchChannels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("name") != "test" {
			t.Errorf("name param: got %v, want test", r.URL.Query().Get("name"))
		}

		resp := map[string]interface{}{
			"channels": []tools.Channel{
				{ID: 1, Name: "test-channel"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	channels, err := client.SearchChannels(context.Background(), "test", nil, nil, nil, 0, 20)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(channels) != 1 {
		t.Errorf("channels count: got %v, want 1", len(channels))
	}
}

func TestCreateChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["name"] != "new-channel" {
			t.Errorf("name: got %v, want new-channel", body["name"])
		}

		resp := map[string]interface{}{
			"channel": tools.Channel{
				ID:   1,
				Name: "new-channel",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	channel, err := client.CreateChannel(context.Background(), "new-channel", "description", nil, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if channel.Name != "new-channel" {
		t.Errorf("name: got %v, want new-channel", channel.Name)
	}
}

func TestGetChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"channel": tools.Channel{
				ID:   1,
				Name: "test-channel",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	channel, err := client.GetChannel(context.Background(), 1)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if channel.ID != 1 {
		t.Errorf("ID: got %v, want 1", channel.ID)
	}
}

func TestSendMessage(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["content"] != "Hello" {
			t.Errorf("content: got %v, want Hello", body["content"])
		}

		resp := map[string]interface{}{
			"message": tools.ChannelMessage{
				ID:      1,
				Content: "Hello",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	msg, err := client.SendMessage(context.Background(), 1, "Hello", tools.ChannelMessageTypeText, nil, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg.Content != "Hello" {
		t.Errorf("content: got %v, want Hello", msg.Content)
	}
}

func TestGetMessages(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"messages": []tools.ChannelMessage{
				{ID: 1, Content: "Message 1"},
				{ID: 2, Content: "Message 2"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	messages, err := client.GetMessages(context.Background(), 1, nil, nil, nil, 50)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(messages) != 2 {
		t.Errorf("messages count: got %v, want 2", len(messages))
	}
}

func TestGetDocument(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"document": "test document content",
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	doc, err := client.GetDocument(context.Background(), 1)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if doc != "test document content" {
		t.Errorf("document: got %v, want test document content", doc)
	}
}

func TestUpdateDocument(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["document"] != "updated content" {
			t.Errorf("document: got %v, want updated content", body["document"])
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	err := client.UpdateDocument(context.Background(), 1, "updated content")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSearchTickets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"tickets": []tools.Ticket{
				{ID: 1, Identifier: "AM-1", Title: "Test Ticket"},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	tickets, err := client.SearchTickets(context.Background(), nil, nil, nil, nil, nil, nil, "", 20, 1)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(tickets) != 1 {
		t.Errorf("tickets count: got %v, want 1", len(tickets))
	}
}

func TestGetTicket(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"ticket": tools.Ticket{
				ID:         1,
				Identifier: "AM-123",
				Title:      "Test Ticket",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	ticket, err := client.GetTicket(context.Background(), "AM-123")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ticket.Identifier != "AM-123" {
		t.Errorf("identifier: got %v, want AM-123", ticket.Identifier)
	}
}

func TestCreateTicket(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["title"] != "New Ticket" {
			t.Errorf("title: got %v, want New Ticket", body["title"])
		}

		resp := map[string]interface{}{
			"ticket": tools.Ticket{
				ID:         1,
				Identifier: "AM-1",
				Title:      "New Ticket",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	ticket, err := client.CreateTicket(context.Background(), 1, "New Ticket", "Description", tools.TicketTypeTask, tools.TicketPriorityMedium, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ticket.Title != "New Ticket" {
		t.Errorf("title: got %v, want New Ticket", ticket.Title)
	}
}

func TestUpdateTicket(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)

		if body["title"] != "Updated Title" {
			t.Errorf("title: got %v, want Updated Title", body["title"])
		}

		resp := map[string]interface{}{
			"ticket": tools.Ticket{
				ID:    1,
				Title: "Updated Title",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	title := "Updated Title"
	ticket, err := client.UpdateTicket(context.Background(), "AM-1", &title, nil, nil, nil, nil)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if ticket.Title != "Updated Title" {
		t.Errorf("title: got %v, want Updated Title", ticket.Title)
	}
}

func TestCreatePod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body tools.PodCreateRequest
		json.NewDecoder(r.Body).Decode(&body)

		if body.InitialPrompt != "Hello" {
			t.Errorf("initial_prompt: got %v, want Hello", body.InitialPrompt)
		}

		resp := map[string]interface{}{
			"pod": map[string]interface{}{
				"pod_key": "new-pod",
				"status":  "created",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	resp, err := client.CreatePod(context.Background(), &tools.PodCreateRequest{
		InitialPrompt: "Hello",
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.PodKey != "new-pod" {
		t.Errorf("pod_key: got %v, want new-pod", resp.PodKey)
	}
}

func TestRequestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	_, err := client.GetTicket(context.Background(), "AM-1")

	if err == nil {
		t.Error("expected error for 500 response")
	}
}

func TestBackendClientImplementsInterface(t *testing.T) {
	var _ tools.CollaborationClient = (*BackendClient)(nil)
}

func TestListRunners(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method: got %v, want GET", r.Method)
		}

		expectedPath := "/api/v1/orgs/test-org/pod/runners"
		if r.URL.Path != expectedPath {
			t.Errorf("path: got %v, want %v", r.URL.Path, expectedPath)
		}

		resp := map[string]interface{}{
			"runners": []tools.RunnerSummary{
				{
					ID:                1,
					NodeID:            "dev-machine",
					Description:       "Development runner",
					Status:            "online",
					CurrentPods:       2,
					MaxConcurrentPods: 5,
					AvailableAgents: []tools.AgentTypeSummary{
						{
							ID:          1,
							Slug:        "claude-code",
							Name:        "Claude Code",
							Description: "AI coding assistant",
							Config: []tools.ConfigFieldSummary{
								{
									Name:     "model",
									Type:     "select",
									Default:  "claude-sonnet-4-20250514",
									Options:  []string{"claude-sonnet-4-20250514", "claude-opus-4-20250514"},
									Required: true,
								},
							},
							UserConfig: map[string]interface{}{
								"model": "claude-opus-4-20250514",
							},
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	runners, err := client.ListRunners(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runners) != 1 {
		t.Fatalf("runners count: got %v, want 1", len(runners))
	}

	runner := runners[0]
	if runner.ID != 1 {
		t.Errorf("runner ID: got %v, want 1", runner.ID)
	}
	if runner.NodeID != "dev-machine" {
		t.Errorf("runner NodeID: got %v, want dev-machine", runner.NodeID)
	}
	if runner.Status != "online" {
		t.Errorf("runner Status: got %v, want online", runner.Status)
	}
	if runner.CurrentPods != 2 {
		t.Errorf("runner CurrentPods: got %v, want 2", runner.CurrentPods)
	}
	if runner.MaxConcurrentPods != 5 {
		t.Errorf("runner MaxConcurrentPods: got %v, want 5", runner.MaxConcurrentPods)
	}

	// Verify available agents
	if len(runner.AvailableAgents) != 1 {
		t.Fatalf("available agents count: got %v, want 1", len(runner.AvailableAgents))
	}

	agent := runner.AvailableAgents[0]
	if agent.ID != 1 {
		t.Errorf("agent ID: got %v, want 1", agent.ID)
	}
	if agent.Slug != "claude-code" {
		t.Errorf("agent Slug: got %v, want claude-code", agent.Slug)
	}
	if agent.Name != "Claude Code" {
		t.Errorf("agent Name: got %v, want Claude Code", agent.Name)
	}

	// Verify config fields
	if len(agent.Config) != 1 {
		t.Fatalf("config fields count: got %v, want 1", len(agent.Config))
	}

	config := agent.Config[0]
	if config.Name != "model" {
		t.Errorf("config Name: got %v, want model", config.Name)
	}
	if config.Type != "select" {
		t.Errorf("config Type: got %v, want select", config.Type)
	}
	if !config.Required {
		t.Error("config Required: expected true")
	}
	if len(config.Options) != 2 {
		t.Errorf("config Options count: got %v, want 2", len(config.Options))
	}

	// Verify user config
	if agent.UserConfig == nil {
		t.Fatal("user config should not be nil")
	}
	if agent.UserConfig["model"] != "claude-opus-4-20250514" {
		t.Errorf("user config model: got %v, want claude-opus-4-20250514", agent.UserConfig["model"])
	}
}

func TestListRunnersEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"runners": []tools.RunnerSummary{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	runners, err := client.ListRunners(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runners) != 0 {
		t.Errorf("runners count: got %v, want 0", len(runners))
	}
}

func TestListRunnersMultipleAgents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"runners": []tools.RunnerSummary{
				{
					ID:                1,
					NodeID:            "multi-agent-runner",
					Status:            "online",
					CurrentPods:       0,
					MaxConcurrentPods: 10,
					AvailableAgents: []tools.AgentTypeSummary{
						{ID: 1, Slug: "claude-code", Name: "Claude Code"},
						{ID: 2, Slug: "aider", Name: "Aider"},
						{ID: 3, Slug: "codex", Name: "Codex CLI"},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewBackendClient(server.URL, "test-org", "test-pod")
	runners, err := client.ListRunners(context.Background())

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(runners) != 1 {
		t.Fatalf("runners count: got %v, want 1", len(runners))
	}

	if len(runners[0].AvailableAgents) != 3 {
		t.Errorf("available agents count: got %v, want 3", len(runners[0].AvailableAgents))
	}

	// Verify agent slugs
	expectedSlugs := []string{"claude-code", "aider", "codex"}
	for i, agent := range runners[0].AvailableAgents {
		if agent.Slug != expectedSlugs[i] {
			t.Errorf("agent[%d] Slug: got %v, want %v", i, agent.Slug, expectedSlugs[i])
		}
	}
}
