package channel

import (
	"testing"
	"time"
)

// --- Test MessageContent ---

func TestMessageContentScanNil(t *testing.T) {
	var mc MessageContent
	err := mc.Scan(nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mc.Kind != "" {
		t.Error("expected zero-value MessageContent after nil scan")
	}
}

func TestMessageContentScanBytes(t *testing.T) {
	var mc MessageContent
	err := mc.Scan([]byte(`{"kind":"text","blocks":[]}`))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mc.Kind != "text" {
		t.Errorf("expected kind 'text', got %q", mc.Kind)
	}
}

func TestMessageContentScanString(t *testing.T) {
	var mc MessageContent
	err := mc.Scan(`{"kind":"attachment","attachment_key":"abc"}`)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if mc.Kind != "attachment" {
		t.Errorf("expected kind 'attachment', got %q", mc.Kind)
	}
}

func TestMessageContentScanInvalidType(t *testing.T) {
	var mc MessageContent
	err := mc.Scan(12345)
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}

func TestMessageContentValue(t *testing.T) {
	mc := MessageContent{Kind: "text", Blocks: []Block{{Type: "paragraph"}}}
	val, err := mc.Value()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val == nil {
		t.Error("expected non-nil driver value")
	}
}

// --- Test MessageMentions ---

func TestMessageMentionsScanNil(t *testing.T) {
	var mm MessageMentions
	err := mm.Scan(nil)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(mm.Pods) != 0 || len(mm.Users) != 0 {
		t.Error("expected zero-value MessageMentions after nil scan")
	}
}

func TestMessageMentionsScanBytes(t *testing.T) {
	var mm MessageMentions
	err := mm.Scan([]byte(`{"pods":["pod-1","pod-2"],"users":[10]}`))
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if len(mm.Pods) != 2 {
		t.Errorf("expected 2 pods, got %d", len(mm.Pods))
	}
	if len(mm.Users) != 1 || mm.Users[0] != 10 {
		t.Errorf("expected users [10], got %v", mm.Users)
	}
}

func TestMessageMentionsScanString(t *testing.T) {
	var mm MessageMentions
	err := mm.Scan(`{"channel":true}`)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !mm.Channel {
		t.Error("expected Channel to be true")
	}
}

func TestMessageMentionsScanInvalidType(t *testing.T) {
	var mm MessageMentions
	err := mm.Scan(99)
	if err == nil {
		t.Error("expected error for unsupported type")
	}
}

func TestMessageMentionsValueNil(t *testing.T) {
	var mm MessageMentions
	val, err := mm.Value()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val == nil {
		t.Error("expected non-nil driver value")
	}
}

func TestMessageMentionsValueValid(t *testing.T) {
	mm := MessageMentions{Pods: []string{"pod-x"}, Users: []int64{42}}
	val, err := mm.Value()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if val == nil {
		t.Error("expected non-nil driver value")
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
	mc := MessageContent{Kind: "text"}

	m := Message{
		ID:           1,
		ChannelID:    10,
		SenderPod:    &senderPod,
		SenderUserID: &senderUserID,
		MessageType:  MessageTypeText,
		Body:         "Hello, world!",
		Content:      &mc,
		Mentions:     MessageMentions{Pods: []string{"pod-a"}},
		CreatedAt:    now,
	}

	if m.ID != 1 {
		t.Errorf("expected ID 1, got %d", m.ID)
	}
	if m.ChannelID != 10 {
		t.Errorf("expected ChannelID 10, got %d", m.ChannelID)
	}
	if m.Body != "Hello, world!" {
		t.Errorf("expected Body 'Hello, world!', got %s", m.Body)
	}
	if m.MessageType != "text" {
		t.Errorf("expected MessageType 'text', got %s", m.MessageType)
	}
}

// --- Benchmarks ---

func BenchmarkMessageContentScan(b *testing.B) {
	data := []byte(`{"kind":"text","blocks":[{"type":"paragraph","elements":[{"type":"text","text":"hello"}]}]}`)
	for i := 0; i < b.N; i++ {
		var mc MessageContent
		mc.Scan(data)
	}
}

func BenchmarkMessageMentionsScan(b *testing.B) {
	data := []byte(`{"pods":["pod-1","pod-2"],"users":[10,20]}`)
	for i := 0; i < b.N; i++ {
		var mm MessageMentions
		mm.Scan(data)
	}
}
