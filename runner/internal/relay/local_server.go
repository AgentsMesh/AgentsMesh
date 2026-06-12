package relay

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// LocalServer is the runner's always-on browser-facing WebSocket server.
//
// It exposes the same wire protocol as the cloud relay (binary frames defined
// in protocol.go) at ws://127.0.0.1:<port>/browser/relay so renderers can swap
// the URL with no transport changes. Each pod is registered explicitly via
// RegisterPod with an expected token; incoming connections are validated by
// exact-match against that token.
type LocalServer struct {
	httpServer *http.Server
	listener   net.Listener
	logger     *slog.Logger

	mu      sync.RWMutex
	pods    map[string]*localPodLane
	closed  bool
	urlOnce sync.Once
	url     string
}

// ReplyFunc writes one framed message back to a single browser connection.
type ReplyFunc func(msgType byte, payload []byte)

// RequestHandler handles an inbound message that must be answered on the
// originating connection only, rather than broadcast to every browser on the
// pod (see SetRequestHandler).
type RequestHandler func(payload []byte, reply ReplyFunc)

// localPodLane holds per-pod state for the local server.
type localPodLane struct {
	expectedTokens map[string]struct{}
	handlers       map[byte]func([]byte)
	reqHandlers    map[byte]RequestHandler
	conns          map[*websocket.Conn]struct{}
	mu             sync.RWMutex
	// writeMu serializes WriteMessage: Send (event/aggregator goroutine) and
	// SetRequestHandler replies (readLoop goroutine) can target the same conn,
	// and gorilla/websocket forbids concurrent writers on one connection.
	writeMu sync.Mutex
}

const (
	localReadTimeout  = 60 * time.Second
	localWriteTimeout = 10 * time.Second
	// Browser-side payloads are PTY input + ACP control messages; 1MiB is
	// well above any legitimate frame and bounds memory if an authenticated
	// peer tries to OOM the runner with a giant message.
	localReadLimitBytes = 1 << 20
)

var localUpgrader = websocket.Upgrader{
	ReadBufferSize:  64 * 1024,
	WriteBufferSize: 64 * 1024,
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}

// NewLocalServer constructs a LocalServer bound to 127.0.0.1 on an
// auto-picked port. The server is not running until Start is called.
func NewLocalServer(logger *slog.Logger) *LocalServer {
	if logger == nil {
		logger = slog.Default()
	}
	return &LocalServer{
		logger: logger.With("component", "local_relay_server"),
		pods:   make(map[string]*localPodLane),
	}
}

// Start binds 127.0.0.1:0 and serves until Stop is called.
// Returns the advertised URL after the listener is up.
func (s *LocalServer) Start(ctx context.Context) (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", fmt.Errorf("local relay listen: %w", err)
	}
	s.listener = listener

	mux := http.NewServeMux()
	mux.HandleFunc("/browser/relay", s.handleBrowser)

	s.httpServer = &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		_ = listener.Close()
		return "", errors.New("local relay listener returned non-TCP addr")
	}
	url := fmt.Sprintf("ws://127.0.0.1:%d/browser/relay", addr.Port)
	s.urlOnce.Do(func() { s.url = url })

	go func() {
		if err := s.httpServer.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			s.logger.Error("local relay server stopped with error", "error", err)
		}
	}()

	go func() {
		<-ctx.Done()
		s.Stop()
	}()

	s.logger.Info("Local relay server started", "url", url)
	return url, nil
}

// Stop closes the HTTP server and all active browser connections.
func (s *LocalServer) Stop() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	pods := s.pods
	s.pods = make(map[string]*localPodLane)
	s.mu.Unlock()

	for _, lane := range pods {
		lane.mu.Lock()
		for c := range lane.conns {
			_ = c.Close()
		}
		lane.conns = nil
		lane.mu.Unlock()
	}

	if s.httpServer != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.httpServer.Shutdown(shutdownCtx)
	}
}

// URL returns the advertised ws:// URL ("" before Start).
func (s *LocalServer) URL() string {
	return s.url
}
