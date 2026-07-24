package runner

import (
	"context"
	"sync"

	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

type gatedUnregisterLocalBroker struct {
	mu sync.Mutex

	tokens          map[string]map[string]struct{}
	messageHandlers map[string]map[byte]func([]byte)
	requestHandlers map[string]map[byte]relay.RequestHandler

	unregisterStarted chan struct{}
	unregisterRelease chan struct{}
	onUnregister      func()
}

func newGatedUnregisterLocalBroker() *gatedUnregisterLocalBroker {
	return &gatedUnregisterLocalBroker{
		tokens:            make(map[string]map[string]struct{}),
		messageHandlers:   make(map[string]map[byte]func([]byte)),
		requestHandlers:   make(map[string]map[byte]relay.RequestHandler),
		unregisterStarted: make(chan struct{}),
		unregisterRelease: make(chan struct{}),
	}
}

func (b *gatedUnregisterLocalBroker) RegisterPod(podKey, token string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.tokens[podKey] == nil {
		b.tokens[podKey] = make(map[string]struct{})
	}
	b.tokens[podKey][token] = struct{}{}
}

func (b *gatedUnregisterLocalBroker) UnregisterPod(podKey string) {
	b.mu.Lock()
	delete(b.tokens, podKey)
	delete(b.messageHandlers, podKey)
	delete(b.requestHandlers, podKey)
	onUnregister := b.onUnregister
	b.mu.Unlock()
	if onUnregister != nil {
		onUnregister()
	}
	close(b.unregisterStarted)
	<-b.unregisterRelease
}

func (b *gatedUnregisterLocalBroker) SetMessageHandler(
	podKey string,
	msgType byte,
	handler func([]byte),
) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.messageHandlers[podKey] == nil {
		b.messageHandlers[podKey] = make(map[byte]func([]byte))
	}
	b.messageHandlers[podKey][msgType] = handler
}

func (b *gatedUnregisterLocalBroker) SetRequestHandler(
	podKey string,
	msgType byte,
	handler relay.RequestHandler,
) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.requestHandlers[podKey] == nil {
		b.requestHandlers[podKey] = make(map[byte]relay.RequestHandler)
	}
	b.requestHandlers[podKey][msgType] = handler
}

func (b *gatedUnregisterLocalBroker) Send(string, byte, []byte) error { return nil }
func (b *gatedUnregisterLocalBroker) Flush(context.Context, string) error {
	return nil
}
func (b *gatedUnregisterLocalBroker) IsPodConnected(string) bool { return false }
func (b *gatedUnregisterLocalBroker) URL() string                { return "ws://local" }

func (b *gatedUnregisterLocalBroker) registrationCounts(podKey string) (int, int, int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return len(b.tokens[podKey]), len(b.messageHandlers[podKey]), len(b.requestHandlers[podKey])
}

func (b *gatedUnregisterLocalBroker) messageHandler(podKey string, msgType byte) func([]byte) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.messageHandlers[podKey][msgType]
}

type gatedLocalHandlerContext struct {
	MessageHandlerContext
	local LocalRelayBroker
}

func (c gatedLocalHandlerContext) GetLocalRelayServer() LocalRelayBroker { return c.local }
