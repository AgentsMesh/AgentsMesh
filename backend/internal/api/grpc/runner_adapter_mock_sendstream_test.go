package grpc

import (
	"context"
	"sync"

	"google.golang.org/grpc/metadata"

	runnerv1 "github.com/AgentsMesh/AgentsMesh/proto/gen/go/runner/v1"
)

// mockSendStream is used for testing sendLoop
type mockSendStream struct {
	mu       sync.Mutex
	sendErr  error
	sentMsgs []*runnerv1.ServerMessage
}

func (m *mockSendStream) Send(msg *runnerv1.ServerMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.sendErr != nil {
		return m.sendErr
	}
	m.sentMsgs = append(m.sentMsgs, msg)
	return nil
}

func (m *mockSendStream) getSentMsgs() []*runnerv1.ServerMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Return a copy to avoid race
	result := make([]*runnerv1.ServerMessage, len(m.sentMsgs))
	copy(result, m.sentMsgs)
	return result
}

func (m *mockSendStream) Recv() (*runnerv1.RunnerMessage, error) {
	return nil, nil
}

func (m *mockSendStream) Context() context.Context {
	return context.Background()
}

func (m *mockSendStream) SetHeader(metadata.MD) error  { return nil }
func (m *mockSendStream) SendHeader(metadata.MD) error { return nil }
func (m *mockSendStream) SetTrailer(metadata.MD)       {}
func (m *mockSendStream) SendMsg(interface{}) error    { return nil }
func (m *mockSendStream) RecvMsg(interface{}) error    { return nil }
