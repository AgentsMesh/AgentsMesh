package channel

import (
	"testing"
	"time"
)

func TestNewChannelManager(t *testing.T) {
	m := NewChannelManager(200*time.Millisecond, 5, nil)
	if m == nil {
		t.Fatal("expected non-nil ChannelManager")
	}
	stats := m.Stats()
	if stats.ActiveChannels != 0 || stats.TotalSubscribers != 0 || stats.PendingPublishers != 0 || stats.PendingSubscribers != 0 {
		t.Fatalf("unexpected initial stats: %+v", stats)
	}
}

func TestNewChannelManagerWithConfig(t *testing.T) {
	cfg := testManagerConfig()
	cfg.MaxSubscribersPerPod = 7
	m := NewChannelManagerWithConfig(cfg, nil)
	if m == nil {
		t.Fatal("expected non-nil ChannelManager")
	}
	if m.config.MaxSubscribersPerPod != 7 {
		t.Fatalf("MaxSubscribersPerPod: got %d, want 7", m.config.MaxSubscribersPerPod)
	}
}

func TestChannelManager_Stats_Empty(t *testing.T) {
	m := NewChannelManagerWithConfig(testManagerConfig(), nil)
	stats := m.Stats()
	if stats.ActiveChannels != 0 {
		t.Fatalf("ActiveChannels: got %d, want 0", stats.ActiveChannels)
	}
	if stats.TotalSubscribers != 0 {
		t.Fatalf("TotalSubscribers: got %d, want 0", stats.TotalSubscribers)
	}
	if stats.PendingPublishers != 0 {
		t.Fatalf("PendingPublishers: got %d, want 0", stats.PendingPublishers)
	}
	if stats.PendingSubscribers != 0 {
		t.Fatalf("PendingSubscribers: got %d, want 0", stats.PendingSubscribers)
	}
}

func TestChannelManager_PublisherPending(t *testing.T) {
	m := NewChannelManagerWithConfig(testManagerConfig(), nil)
	pubServer, _ := createWSPair(t)

	if err := m.HandlePublisherConnect("pod-1", pubServer); err != nil {
		t.Fatalf("HandlePublisherConnect: %v", err)
	}

	stats := m.Stats()
	if stats.PendingPublishers != 1 {
		t.Fatalf("PendingPublishers: got %d, want 1", stats.PendingPublishers)
	}
	if stats.ActiveChannels != 0 {
		t.Fatalf("ActiveChannels: got %d, want 0", stats.ActiveChannels)
	}
}

func TestChannelManager_SubscriberPending(t *testing.T) {
	m := NewChannelManagerWithConfig(testManagerConfig(), nil)
	subServer, _ := createWSPair(t)

	if err := m.HandleSubscriberConnect("pod-1", "s1", subServer); err != nil {
		t.Fatalf("HandleSubscriberConnect: %v", err)
	}

	stats := m.Stats()
	if stats.PendingSubscribers != 1 {
		t.Fatalf("PendingSubscribers: got %d, want 1", stats.PendingSubscribers)
	}
	if stats.ActiveChannels != 0 {
		t.Fatalf("ActiveChannels: got %d, want 0", stats.ActiveChannels)
	}
}

func TestChannelManager_PubThenSub(t *testing.T) {
	m := NewChannelManagerWithConfig(testManagerConfig(), nil)
	pubServer, _ := createWSPair(t)
	subServer, _ := createWSPair(t)

	if err := m.HandlePublisherConnect("pod-1", pubServer); err != nil {
		t.Fatalf("HandlePublisherConnect: %v", err)
	}
	if err := m.HandleSubscriberConnect("pod-1", "s1", subServer); err != nil {
		t.Fatalf("HandleSubscriberConnect: %v", err)
	}

	stats := m.Stats()
	if stats.ActiveChannels != 1 {
		t.Fatalf("ActiveChannels: got %d, want 1", stats.ActiveChannels)
	}
	if stats.PendingPublishers != 0 {
		t.Fatalf("PendingPublishers: got %d, want 0", stats.PendingPublishers)
	}
	if stats.TotalSubscribers != 1 {
		t.Fatalf("TotalSubscribers: got %d, want 1", stats.TotalSubscribers)
	}
}

func TestChannelManager_SubThenPub(t *testing.T) {
	m := NewChannelManagerWithConfig(testManagerConfig(), nil)
	subServer, _ := createWSPair(t)
	pubServer, _ := createWSPair(t)

	if err := m.HandleSubscriberConnect("pod-1", "s1", subServer); err != nil {
		t.Fatalf("HandleSubscriberConnect: %v", err)
	}
	if err := m.HandlePublisherConnect("pod-1", pubServer); err != nil {
		t.Fatalf("HandlePublisherConnect: %v", err)
	}

	stats := m.Stats()
	if stats.ActiveChannels != 1 {
		t.Fatalf("ActiveChannels: got %d, want 1", stats.ActiveChannels)
	}
	if stats.PendingSubscribers != 0 {
		t.Fatalf("PendingSubscribers: got %d, want 0", stats.PendingSubscribers)
	}
	if stats.TotalSubscribers != 1 {
		t.Fatalf("TotalSubscribers: got %d, want 1", stats.TotalSubscribers)
	}
}

func TestChannelManager_GetChannel(t *testing.T) {
	m := NewChannelManagerWithConfig(testManagerConfig(), nil)
	pubServer, _ := createWSPair(t)
	subServer, _ := createWSPair(t)

	if err := m.HandlePublisherConnect("pod-1", pubServer); err != nil {
		t.Fatalf("HandlePublisherConnect: %v", err)
	}
	if err := m.HandleSubscriberConnect("pod-1", "s1", subServer); err != nil {
		t.Fatalf("HandleSubscriberConnect: %v", err)
	}

	ch := m.GetChannel("pod-1")
	if ch == nil {
		t.Fatal("expected non-nil channel for pod-1")
	}
	if ch.PodKey != "pod-1" {
		t.Fatalf("PodKey: got %q, want %q", ch.PodKey, "pod-1")
	}

	if m.GetChannel("pod-nonexistent") != nil {
		t.Fatal("expected nil for nonexistent pod")
	}
}

func TestChannelManager_CloseChannel(t *testing.T) {
	m := NewChannelManagerWithConfig(testManagerConfig(), nil)
	pubServer, _ := createWSPair(t)
	subServer, _ := createWSPair(t)

	if err := m.HandlePublisherConnect("pod-1", pubServer); err != nil {
		t.Fatalf("HandlePublisherConnect: %v", err)
	}
	if err := m.HandleSubscriberConnect("pod-1", "s1", subServer); err != nil {
		t.Fatalf("HandleSubscriberConnect: %v", err)
	}

	m.CloseChannel("pod-1")

	if m.GetChannel("pod-1") != nil {
		t.Fatal("expected nil channel after CloseChannel")
	}
	stats := m.Stats()
	if stats.ActiveChannels != 0 {
		t.Fatalf("ActiveChannels: got %d, want 0", stats.ActiveChannels)
	}
}

func TestChannelManager_AddSubscriber(t *testing.T) {
	m := NewChannelManagerWithConfig(testManagerConfig(), nil)
	pubServer, _ := createWSPair(t)
	sub1Server, _ := createWSPair(t)
	sub2Server, _ := createWSPair(t)

	if err := m.HandlePublisherConnect("pod-1", pubServer); err != nil {
		t.Fatalf("HandlePublisherConnect: %v", err)
	}
	if err := m.HandleSubscriberConnect("pod-1", "s1", sub1Server); err != nil {
		t.Fatalf("HandleSubscriberConnect s1: %v", err)
	}
	if err := m.HandleSubscriberConnect("pod-1", "s2", sub2Server); err != nil {
		t.Fatalf("HandleSubscriberConnect s2: %v", err)
	}

	stats := m.Stats()
	if stats.TotalSubscribers != 2 {
		t.Fatalf("TotalSubscribers: got %d, want 2", stats.TotalSubscribers)
	}
}

func TestChannelManager_MaxSubscribers(t *testing.T) {
	cfg := testManagerConfig()
	cfg.MaxSubscribersPerPod = 3
	m := NewChannelManagerWithConfig(cfg, nil)

	pubServer, _ := createWSPair(t)
	if err := m.HandlePublisherConnect("pod-1", pubServer); err != nil {
		t.Fatalf("HandlePublisherConnect: %v", err)
	}

	// Add 3 subscribers (max)
	for i := 1; i <= 3; i++ {
		srvConn, _ := createWSPair(t)
		if err := m.HandleSubscriberConnect("pod-1", "s"+string(rune('0'+i)), srvConn); err != nil {
			t.Fatalf("HandleSubscriberConnect s%d: %v", i, err)
		}
	}

	// 4th subscriber should fail
	s4Server, _ := createWSPair(t)
	err := m.HandleSubscriberConnect("pod-1", "s4", s4Server)
	if err == nil {
		t.Fatal("expected MaxSubscribersError for 4th subscriber")
	}
	if _, ok := err.(*MaxSubscribersError); !ok {
		t.Fatalf("expected *MaxSubscribersError, got %T: %v", err, err)
	}
}

func TestChannelManager_PubReconnect(t *testing.T) {
	m := NewChannelManagerWithConfig(testManagerConfig(), nil)

	pubServer, pubClient := createWSPair(t)
	subServer, _ := createWSPair(t)

	if err := m.HandlePublisherConnect("pod-1", pubServer); err != nil {
		t.Fatalf("HandlePublisherConnect: %v", err)
	}
	if err := m.HandleSubscriberConnect("pod-1", "s1", subServer); err != nil {
		t.Fatalf("HandleSubscriberConnect: %v", err)
	}

	ch := m.GetChannel("pod-1")
	if ch == nil {
		t.Fatal("expected channel to exist")
	}

	// Close the publisher client to trigger disconnect
	_ = pubClient.Close()
	waitFor(t, func() bool {
		return ch.IsPublisherDisconnected()
	}, 2*time.Second)

	// Reconnect publisher via manager
	newPubServer, _ := createWSPair(t)
	if err := m.HandlePublisherConnect("pod-1", newPubServer); err != nil {
		t.Fatalf("HandlePublisherConnect reconnect: %v", err)
	}

	// Verify same channel is reused
	ch2 := m.GetChannel("pod-1")
	if ch2 != ch {
		t.Fatal("expected same channel instance after publisher reconnect")
	}
	if ch.IsPublisherDisconnected() {
		t.Fatal("expected IsPublisherDisconnected false after reconnect")
	}
}

func TestChannelManager_Stats(t *testing.T) {
	m := NewChannelManagerWithConfig(testManagerConfig(), nil)

	// Create an active channel: pod-1
	pub1Server, _ := createWSPair(t)
	sub1Server, _ := createWSPair(t)
	sub1bServer, _ := createWSPair(t)
	if err := m.HandlePublisherConnect("pod-1", pub1Server); err != nil {
		t.Fatalf("HandlePublisherConnect pod-1: %v", err)
	}
	if err := m.HandleSubscriberConnect("pod-1", "s1", sub1Server); err != nil {
		t.Fatalf("HandleSubscriberConnect pod-1 s1: %v", err)
	}
	if err := m.HandleSubscriberConnect("pod-1", "s1b", sub1bServer); err != nil {
		t.Fatalf("HandleSubscriberConnect pod-1 s1b: %v", err)
	}

	// Create a pending publisher: pod-2
	pub2Server, _ := createWSPair(t)
	if err := m.HandlePublisherConnect("pod-2", pub2Server); err != nil {
		t.Fatalf("HandlePublisherConnect pod-2: %v", err)
	}

	// Create a pending subscriber: pod-3
	sub3Server, _ := createWSPair(t)
	if err := m.HandleSubscriberConnect("pod-3", "s3", sub3Server); err != nil {
		t.Fatalf("HandleSubscriberConnect pod-3: %v", err)
	}

	stats := m.Stats()
	if stats.ActiveChannels != 1 {
		t.Fatalf("ActiveChannels: got %d, want 1", stats.ActiveChannels)
	}
	if stats.TotalSubscribers != 2 {
		t.Fatalf("TotalSubscribers: got %d, want 2", stats.TotalSubscribers)
	}
	if stats.PendingPublishers != 1 {
		t.Fatalf("PendingPublishers: got %d, want 1", stats.PendingPublishers)
	}
	if stats.PendingSubscribers != 1 {
		t.Fatalf("PendingSubscribers: got %d, want 1", stats.PendingSubscribers)
	}
}

func TestMaxSubscribersError_Error(t *testing.T) {
	err := &MaxSubscribersError{Max: 10}
	want := "maximum subscribers per pod reached"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
}
