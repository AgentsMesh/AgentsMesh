package channel

import (
	"testing"
	"time"

	"github.com/lib/pq"
)

// --- Test Channel ---

func TestChannelTableName(t *testing.T) {
	c := Channel{}
	if c.TableName() != "channels" {
		t.Errorf("expected 'channels', got %s", c.TableName())
	}
}

func TestChannelStruct(t *testing.T) {
	now := time.Now()
	desc := "Test channel"
	doc := "Shared doc content"
	repoID := int64(5)
	ticketID := int64(20)
	createdByPod := "pod-123"
	createdByUserID := int64(50)

	c := Channel{
		ID:               1,
		OrganizationID:   100,
		Name:             "Development",
		Description:      &desc,
		Document:         &doc,
		RepositoryID:     &repoID,
		TicketID:         &ticketID,
		CreatedByPod: &createdByPod,
		CreatedByUserID:  &createdByUserID,
		IsArchived:       false,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	if c.ID != 1 {
		t.Errorf("expected ID 1, got %d", c.ID)
	}
	if c.Name != "Development" {
		t.Errorf("expected Name 'Development', got %s", c.Name)
	}
	if *c.Description != "Test channel" {
		t.Errorf("expected Description 'Test channel', got %s", *c.Description)
	}
}

// --- Test Message Type Constants ---

func TestMessageTypeConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{MessageTypeText, "text"},
		{MessageTypeSystem, "system"},
		{MessageTypeCode, "code"},
		{MessageTypeCommand, "command"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("expected '%s', got '%s'", tt.expected, tt.constant)
		}
	}
}

// --- Test MessageMetadata ---

func TestMessageMetadataScanNil(t *testing.T) {
	var mm MessageMetadata
	err := mm.Scan(nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mm != nil {
		t.Error("expected nil MessageMetadata")
	}
}

func TestMessageMetadataScanValid(t *testing.T) {
	var mm MessageMetadata
	err := mm.Scan([]byte(`{"language":"go","filename":"main.go"}`))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mm["language"] != "go" {
		t.Errorf("expected language 'go', got %v", mm["language"])
	}
}

func TestMessageMetadataScanInvalidType(t *testing.T) {
	var mm MessageMetadata
	err := mm.Scan("not bytes")
	if err == nil {
		t.Error("expected error for invalid type")
	}
}

func TestMessageMetadataValueNil(t *testing.T) {
	var mm MessageMetadata
	val, err := mm.Value()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val != nil {
		t.Error("expected nil value")
	}
}

func TestMessageMetadataValueValid(t *testing.T) {
	mm := MessageMetadata{"language": "go"}
	val, err := mm.Value()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val == nil {
		t.Error("expected non-nil value")
	}
}

// --- Test Message ---

func TestMessageTableName(t *testing.T) {
	m := Message{}
	if m.TableName() != "channel_messages" {
		t.Errorf("expected 'channel_messages', got %s", m.TableName())
	}
}

func TestMessageStruct(t *testing.T) {
	now := time.Now()
	senderPod := "pod-sender"
	senderUserID := int64(50)

	m := Message{
		ID:           1,
		ChannelID:    10,
		SenderPod:    &senderPod,
		SenderUserID: &senderUserID,
		MessageType:   MessageTypeText,
		Content:       "Hello, world!",
		Metadata:      MessageMetadata{"mention": "@user"},
		CreatedAt:     now,
	}

	if m.ID != 1 {
		t.Errorf("expected ID 1, got %d", m.ID)
	}
	if m.ChannelID != 10 {
		t.Errorf("expected ChannelID 10, got %d", m.ChannelID)
	}
	if m.Content != "Hello, world!" {
		t.Errorf("expected Content 'Hello, world!', got %s", m.Content)
	}
	if m.MessageType != "text" {
		t.Errorf("expected MessageType 'text', got %s", m.MessageType)
	}
}

// --- Test Binding Status Constants ---

func TestBindingStatusConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{BindingStatusPending, "pending"},
		{BindingStatusActive, "active"},
		{BindingStatusRejected, "rejected"},
		{BindingStatusInactive, "inactive"},
		{BindingStatusExpired, "expired"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("expected '%s', got '%s'", tt.expected, tt.constant)
		}
	}
}

// --- Test Binding Scope Constants ---

func TestBindingScopeConstants(t *testing.T) {
	if BindingScopeTerminalRead != "terminal:read" {
		t.Errorf("expected 'terminal:read', got %s", BindingScopeTerminalRead)
	}
	if BindingScopeTerminalWrite != "terminal:write" {
		t.Errorf("expected 'terminal:write', got %s", BindingScopeTerminalWrite)
	}
}

func TestValidBindingScopes(t *testing.T) {
	if !ValidBindingScopes[BindingScopeTerminalRead] {
		t.Error("expected terminal:read to be valid")
	}
	if !ValidBindingScopes[BindingScopeTerminalWrite] {
		t.Error("expected terminal:write to be valid")
	}
	if ValidBindingScopes["invalid:scope"] {
		t.Error("expected invalid:scope to be invalid")
	}
}

// --- Test Binding Policy Constants ---

func TestBindingPolicyConstants(t *testing.T) {
	tests := []struct {
		constant string
		expected string
	}{
		{BindingPolicySameUserAuto, "same_user_auto"},
		{BindingPolicySameProjectAuto, "same_project_auto"},
		{BindingPolicyExplicitOnly, "explicit_only"},
	}

	for _, tt := range tests {
		if tt.constant != tt.expected {
			t.Errorf("expected '%s', got '%s'", tt.expected, tt.constant)
		}
	}
}

// --- Test PodBinding ---

func TestPodBindingTableName(t *testing.T) {
	pb := PodBinding{}
	if pb.TableName() != "pod_bindings" {
		t.Errorf("expected 'pod_bindings', got %s", pb.TableName())
	}
}

func TestPodBindingHasScope(t *testing.T) {
	pb := &PodBinding{
		GrantedScopes: pq.StringArray{BindingScopeTerminalRead, BindingScopeTerminalWrite},
	}

	if !pb.HasScope(BindingScopeTerminalRead) {
		t.Error("expected HasScope(terminal:read) = true")
	}
	if !pb.HasScope(BindingScopeTerminalWrite) {
		t.Error("expected HasScope(terminal:write) = true")
	}
	if pb.HasScope("invalid:scope") {
		t.Error("expected HasScope(invalid:scope) = false")
	}
}

func TestPodBindingHasPendingScope(t *testing.T) {
	pb := &PodBinding{
		PendingScopes: pq.StringArray{BindingScopeTerminalWrite},
	}

	if !pb.HasPendingScope(BindingScopeTerminalWrite) {
		t.Error("expected HasPendingScope(terminal:write) = true")
	}
	if pb.HasPendingScope(BindingScopeTerminalRead) {
		t.Error("expected HasPendingScope(terminal:read) = false")
	}
}

func TestPodBindingIsActive(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{BindingStatusActive, true},
		{BindingStatusPending, false},
		{BindingStatusRejected, false},
		{BindingStatusInactive, false},
	}

	for _, tt := range tests {
		pb := &PodBinding{Status: tt.status}
		if pb.IsActive() != tt.expected {
			t.Errorf("status %s: expected IsActive() = %v", tt.status, tt.expected)
		}
	}
}

func TestPodBindingIsPending(t *testing.T) {
	tests := []struct {
		status   string
		expected bool
	}{
		{BindingStatusPending, true},
		{BindingStatusActive, false},
		{BindingStatusRejected, false},
	}

	for _, tt := range tests {
		pb := &PodBinding{Status: tt.status}
		if pb.IsPending() != tt.expected {
			t.Errorf("status %s: expected IsPending() = %v", tt.status, tt.expected)
		}
	}
}

func TestPodBindingCanObserve(t *testing.T) {
	tests := []struct {
		name          string
		status        string
		grantedScopes []string
		expected      bool
	}{
		{"active with read", BindingStatusActive, []string{BindingScopeTerminalRead}, true},
		{"active without read", BindingStatusActive, []string{BindingScopeTerminalWrite}, false},
		{"pending with read", BindingStatusPending, []string{BindingScopeTerminalRead}, false},
		{"active with no scopes", BindingStatusActive, []string{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := &PodBinding{
				Status:        tt.status,
				GrantedScopes: pq.StringArray(tt.grantedScopes),
			}
			if pb.CanObserve() != tt.expected {
				t.Errorf("expected CanObserve() = %v, got %v", tt.expected, pb.CanObserve())
			}
		})
	}
}

func TestPodBindingCanControl(t *testing.T) {
	tests := []struct {
		name          string
		status        string
		grantedScopes []string
		expected      bool
	}{
		{"active with write", BindingStatusActive, []string{BindingScopeTerminalWrite}, true},
		{"active without write", BindingStatusActive, []string{BindingScopeTerminalRead}, false},
		{"pending with write", BindingStatusPending, []string{BindingScopeTerminalWrite}, false},
		{"active with both", BindingStatusActive, []string{BindingScopeTerminalRead, BindingScopeTerminalWrite}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pb := &PodBinding{
				Status:        tt.status,
				GrantedScopes: pq.StringArray(tt.grantedScopes),
			}
			if pb.CanControl() != tt.expected {
				t.Errorf("expected CanControl() = %v, got %v", tt.expected, pb.CanControl())
			}
		})
	}
}

func TestPodBindingStruct(t *testing.T) {
	now := time.Now()
	reason := "User declined"

	pb := PodBinding{
		ID:              1,
		OrganizationID:  100,
		InitiatorPod:    "pod-init",
		TargetPod:       "pod-target",
		GrantedScopes:   pq.StringArray{BindingScopeTerminalRead},
		PendingScopes:   pq.StringArray{BindingScopeTerminalWrite},
		Status:          BindingStatusPending,
		RequestedAt:     &now,
		RejectionReason: &reason,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if pb.ID != 1 {
		t.Errorf("expected ID 1, got %d", pb.ID)
	}
	if pb.InitiatorPod != "pod-init" {
		t.Errorf("expected InitiatorPod 'pod-init', got %s", pb.InitiatorPod)
	}
	if pb.TargetPod != "pod-target" {
		t.Errorf("expected TargetPod 'pod-target', got %s", pb.TargetPod)
	}
}

// --- Benchmark Tests ---

func BenchmarkPodBindingHasScope(b *testing.B) {
	pb := &PodBinding{
		GrantedScopes: pq.StringArray{BindingScopeTerminalRead, BindingScopeTerminalWrite},
	}
	for i := 0; i < b.N; i++ {
		pb.HasScope(BindingScopeTerminalRead)
	}
}

func BenchmarkPodBindingIsActive(b *testing.B) {
	pb := &PodBinding{Status: BindingStatusActive}
	for i := 0; i < b.N; i++ {
		pb.IsActive()
	}
}

func BenchmarkPodBindingCanObserve(b *testing.B) {
	pb := &PodBinding{
		Status:        BindingStatusActive,
		GrantedScopes: pq.StringArray{BindingScopeTerminalRead},
	}
	for i := 0; i < b.N; i++ {
		pb.CanObserve()
	}
}

func BenchmarkMessageMetadataScan(b *testing.B) {
	data := []byte(`{"language":"go","filename":"main.go"}`)
	for i := 0; i < b.N; i++ {
		var mm MessageMetadata
		mm.Scan(data)
	}
}
