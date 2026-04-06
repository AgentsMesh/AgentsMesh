package mesh

import (
	"testing"
	"time"
)

// --- Test MeshNode ---

func TestMeshNodeStruct(t *testing.T) {
	now := time.Now()
	model := "opus"
	ticketID := int64(20)
	repoID := int64(5)
	position := &NodePosition{X: 100.5, Y: 200.5}

	node := MeshNode{
		PodKey:   "pod-123",
		Status:       "running",
		AgentStatus:  "executing",
		Model:        &model,
		TicketID:     &ticketID,
		RepositoryID: &repoID,
		CreatedByID:  50,
		RunnerID:     10,
		StartedAt:    &now,
		Position:     position,
	}

	if node.PodKey != "pod-123" {
		t.Errorf("expected PodKey 'pod-123', got %s", node.PodKey)
	}
	if node.Status != "running" {
		t.Errorf("expected Status 'running', got %s", node.Status)
	}
	if node.AgentStatus != "executing" {
		t.Errorf("expected AgentStatus 'executing', got %s", node.AgentStatus)
	}
	if *node.Model != "opus" {
		t.Errorf("expected Model 'opus', got %s", *node.Model)
	}
	if *node.TicketID != 20 {
		t.Errorf("expected TicketID 20, got %d", *node.TicketID)
	}
	if node.CreatedByID != 50 {
		t.Errorf("expected CreatedByID 50, got %d", node.CreatedByID)
	}
	if node.RunnerID != 10 {
		t.Errorf("expected RunnerID 10, got %d", node.RunnerID)
	}
	if node.Position.X != 100.5 {
		t.Errorf("expected Position.X 100.5, got %f", node.Position.X)
	}
}

func TestMeshNodeWithNilOptionalFields(t *testing.T) {
	node := MeshNode{
		PodKey:  "pod-456",
		Status:      "initializing",
		AgentStatus: "idle",
		CreatedByID: 50,
		RunnerID:    10,
	}

	if node.Model != nil {
		t.Error("expected Model to be nil")
	}
	if node.TicketID != nil {
		t.Error("expected TicketID to be nil")
	}
	if node.RepositoryID != nil {
		t.Error("expected RepositoryID to be nil")
	}
	if node.StartedAt != nil {
		t.Error("expected StartedAt to be nil")
	}
	if node.Position != nil {
		t.Error("expected Position to be nil")
	}
}

// --- Test NodePosition ---

func TestNodePositionStruct(t *testing.T) {
	pos := NodePosition{
		X: 150.25,
		Y: 300.75,
	}

	if pos.X != 150.25 {
		t.Errorf("expected X 150.25, got %f", pos.X)
	}
	if pos.Y != 300.75 {
		t.Errorf("expected Y 300.75, got %f", pos.Y)
	}
}

func TestNodePositionZeroValues(t *testing.T) {
	pos := NodePosition{}

	if pos.X != 0 {
		t.Errorf("expected X 0, got %f", pos.X)
	}
	if pos.Y != 0 {
		t.Errorf("expected Y 0, got %f", pos.Y)
	}
}

// --- Test MeshEdge ---

func TestMeshEdgeStruct(t *testing.T) {
	edge := MeshEdge{
		ID:            1,
		Source:        "pod-init",
		Target:        "pod-target",
		GrantedScopes: []string{"pod:read", "pod:write"},
		PendingScopes: []string{"file:read"},
		Status:        "active",
	}

	if edge.ID != 1 {
		t.Errorf("expected ID 1, got %d", edge.ID)
	}
	if edge.Source != "pod-init" {
		t.Errorf("expected Source 'pod-init', got %s", edge.Source)
	}
	if edge.Target != "pod-target" {
		t.Errorf("expected Target 'pod-target', got %s", edge.Target)
	}
	if len(edge.GrantedScopes) != 2 {
		t.Errorf("expected 2 GrantedScopes, got %d", len(edge.GrantedScopes))
	}
	if len(edge.PendingScopes) != 1 {
		t.Errorf("expected 1 PendingScopes, got %d", len(edge.PendingScopes))
	}
	if edge.Status != "active" {
		t.Errorf("expected Status 'active', got %s", edge.Status)
	}
}

func TestMeshEdgeWithEmptyScopes(t *testing.T) {
	edge := MeshEdge{
		ID:            2,
		Source:        "pod-a",
		Target:        "pod-b",
		GrantedScopes: []string{},
		Status:        "pending",
	}

	if len(edge.GrantedScopes) != 0 {
		t.Errorf("expected 0 GrantedScopes, got %d", len(edge.GrantedScopes))
	}
	if edge.PendingScopes != nil && len(edge.PendingScopes) != 0 {
		t.Errorf("expected empty PendingScopes")
	}
}

// --- Test ChannelInfo ---

func TestChannelInfoStruct(t *testing.T) {
	desc := "Development channel"

	info := ChannelInfo{
		ID:           1,
		Name:         "dev-channel",
		Description:  &desc,
		PodKeys:  []string{"pod-1", "pod-2", "pod-3"},
		MessageCount: 150,
		IsArchived:   false,
	}

	if info.ID != 1 {
		t.Errorf("expected ID 1, got %d", info.ID)
	}
	if info.Name != "dev-channel" {
		t.Errorf("expected Name 'dev-channel', got %s", info.Name)
	}
	if *info.Description != "Development channel" {
		t.Errorf("expected Description 'Development channel', got %s", *info.Description)
	}
	if len(info.PodKeys) != 3 {
		t.Errorf("expected 3 PodKeys, got %d", len(info.PodKeys))
	}
	if info.MessageCount != 150 {
		t.Errorf("expected MessageCount 150, got %d", info.MessageCount)
	}
	if info.IsArchived {
		t.Error("expected IsArchived false")
	}
}

// --- Test MeshTopology ---

func TestMeshTopologyStruct(t *testing.T) {
	topology := MeshTopology{
		Nodes: []MeshNode{
			{PodKey: "pod-1", Status: "running"},
			{PodKey: "pod-2", Status: "running"},
		},
		Edges: []MeshEdge{
			{ID: 1, Source: "pod-1", Target: "pod-2", Status: "active"},
		},
		Channels: []ChannelInfo{
			{ID: 1, Name: "general", MessageCount: 50},
		},
	}

	if len(topology.Nodes) != 2 {
		t.Errorf("expected 2 Nodes, got %d", len(topology.Nodes))
	}
	if len(topology.Edges) != 1 {
		t.Errorf("expected 1 Edge, got %d", len(topology.Edges))
	}
	if len(topology.Channels) != 1 {
		t.Errorf("expected 1 Channel, got %d", len(topology.Channels))
	}
}

func TestMeshTopologyEmpty(t *testing.T) {
	topology := MeshTopology{
		Nodes:    []MeshNode{},
		Edges:    []MeshEdge{},
		Channels: []ChannelInfo{},
	}

	if len(topology.Nodes) != 0 {
		t.Errorf("expected 0 Nodes, got %d", len(topology.Nodes))
	}
	if len(topology.Edges) != 0 {
		t.Errorf("expected 0 Edges, got %d", len(topology.Edges))
	}
	if len(topology.Channels) != 0 {
		t.Errorf("expected 0 Channels, got %d", len(topology.Channels))
	}
}

// --- Test ChannelPod ---

func TestChannelPodTableName(t *testing.T) {
	cs := ChannelPod{}
	if cs.TableName() != "channel_pods" {
		t.Errorf("expected 'channel_pods', got %s", cs.TableName())
	}
}

func TestChannelPodStruct(t *testing.T) {
	now := time.Now()

	cs := ChannelPod{
		ID:         1,
		ChannelID:  10,
		PodKey: "pod-123",
		JoinedAt:   now,
	}

	if cs.ID != 1 {
		t.Errorf("expected ID 1, got %d", cs.ID)
	}
	if cs.ChannelID != 10 {
		t.Errorf("expected ChannelID 10, got %d", cs.ChannelID)
	}
	if cs.PodKey != "pod-123" {
		t.Errorf("expected PodKey 'pod-123', got %s", cs.PodKey)
	}
}

// --- Test ChannelAccess ---

func TestChannelAccessTableName(t *testing.T) {
	ca := ChannelAccess{}
	if ca.TableName() != "channel_access" {
		t.Errorf("expected 'channel_access', got %s", ca.TableName())
	}
}

func TestChannelAccessStruct(t *testing.T) {
	now := time.Now()
	podKey := "pod-123"
	userID := int64(50)

	ca := ChannelAccess{
		ID:         1,
		ChannelID:  10,
		PodKey: &podKey,
		UserID:     &userID,
		LastAccess: now,
	}

	if ca.ID != 1 {
		t.Errorf("expected ID 1, got %d", ca.ID)
	}
	if ca.ChannelID != 10 {
		t.Errorf("expected ChannelID 10, got %d", ca.ChannelID)
	}
	if *ca.PodKey != "pod-123" {
		t.Errorf("expected PodKey 'pod-123', got %s", *ca.PodKey)
	}
	if *ca.UserID != 50 {
		t.Errorf("expected UserID 50, got %d", *ca.UserID)
	}
}

// --- Test CreatePodForTicketRequest ---

func TestCreatePodForTicketRequestStruct(t *testing.T) {
	req := CreatePodForTicketRequest{
		OrganizationID: 100,
		TicketID:       20,
		RunnerID:       5,
		CreatedByID:    50,
		Prompt:         "Start working on ticket",
		Model:          "opus",
		PermissionMode: "bypassPermissions",
	}

	if req.OrganizationID != 100 {
		t.Errorf("expected OrganizationID 100, got %d", req.OrganizationID)
	}
	if req.TicketID != 20 {
		t.Errorf("expected TicketID 20, got %d", req.TicketID)
	}
	if req.Model != "opus" {
		t.Errorf("expected Model 'opus', got %s", req.Model)
	}
}

// --- Test TicketPodInfo ---

func TestTicketPodInfoStruct(t *testing.T) {
	info := TicketPodInfo{
		TicketID: 20,
		Pods: []MeshNode{
			{PodKey: "pod-1", Status: "running"},
			{PodKey: "pod-2", Status: "completed"},
		},
	}

	if info.TicketID != 20 {
		t.Errorf("expected TicketID 20, got %d", info.TicketID)
	}
	if len(info.Pods) != 2 {
		t.Errorf("expected 2 Pods, got %d", len(info.Pods))
	}
}

// --- Test BatchTicketPodsResponse ---

func TestBatchTicketPodsResponseStruct(t *testing.T) {
	resp := BatchTicketPodsResponse{
		TicketPods: map[int64][]MeshNode{
			1: {{PodKey: "pod-1", Status: "running"}},
			2: {{PodKey: "pod-2", Status: "running"}, {PodKey: "pod-3", Status: "completed"}},
		},
	}

	if len(resp.TicketPods) != 2 {
		t.Errorf("expected 2 ticket entries, got %d", len(resp.TicketPods))
	}
	if len(resp.TicketPods[1]) != 1 {
		t.Errorf("expected 1 pod for ticket 1, got %d", len(resp.TicketPods[1]))
	}
	if len(resp.TicketPods[2]) != 2 {
		t.Errorf("expected 2 pods for ticket 2, got %d", len(resp.TicketPods[2]))
	}
}

// --- Benchmark Tests ---

func BenchmarkChannelPodTableName(b *testing.B) {
	cs := ChannelPod{}
	for i := 0; i < b.N; i++ {
		cs.TableName()
	}
}

func BenchmarkChannelAccessTableName(b *testing.B) {
	ca := ChannelAccess{}
	for i := 0; i < b.N; i++ {
		ca.TableName()
	}
}

func BenchmarkMeshTopologyCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = MeshTopology{
			Nodes:    []MeshNode{{PodKey: "pod-1", Status: "running"}},
			Edges:    []MeshEdge{{ID: 1, Source: "pod-1", Target: "pod-2"}},
			Channels: []ChannelInfo{{ID: 1, Name: "general"}},
		}
	}
}
