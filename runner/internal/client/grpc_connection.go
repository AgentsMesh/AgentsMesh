// Package client provides gRPC connection management for Runner.
package client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"sync"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/tls/certprovider"
	"google.golang.org/grpc/credentials/tls/certprovider/pemfile"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/security/advancedtls"
	"google.golang.org/grpc/status"
)

// GRPCProtocolVersion is the current gRPC protocol version.
const GRPCProtocolVersion = 2

// GRPCConnection manages the gRPC connection to the server with mTLS.
// Responsibilities: mTLS setup, bidirectional streaming, reconnection, message routing.
type GRPCConnection struct {
	// Connection configuration
	endpoint  string
	serverURL string // HTTP server URL for REST API calls (certificate renewal)
	nodeID    string
	orgSlug   string

	// mTLS certificate paths
	certFile string
	keyFile  string
	caFile   string

	// gRPC components
	conn   *grpc.ClientConn
	creds  credentials.TransportCredentials                            // advancedtls credentials for hot-reload
	client runnerv1.RunnerServiceClient                                // gRPC service client
	stream grpc.BidiStreamingClient[runnerv1.RunnerMessage, runnerv1.ServerMessage] // Bidirectional stream
	mu     sync.Mutex

	// Certificate providers for cleanup (prevent goroutine leaks)
	identityProvider certprovider.Provider
	rootProvider     certprovider.Provider

	// Message handling
	handler MessageHandler

	// Reconnection strategy
	reconnectStrategy *ReconnectStrategy

	// Heartbeat
	heartbeatInterval time.Duration

	// Initialization
	initTimeout       time.Duration
	initialized       bool
	availableAgents   []string
	initResultCh      chan *runnerv1.InitializeResult

	// Runner info
	runnerVersion string
	mcpPort       int

	// Lifecycle
	sendCh        chan *runnerv1.RunnerMessage // Send channel for messages (typed for safety)
	stopCh        chan struct{}
	stopOnce      sync.Once
	reconnectOnce sync.Once      // Ensures only one reconnection attempt
	reconnectCh   chan struct{}  // Signal to trigger reconnection

	// Certificate renewal
	certRenewalCheckInterval time.Duration
	certExpiryWarningDays    int
	certRenewalDays          int  // Days before expiry to trigger renewal (default 30)
	certUrgentDays           int  // Days before expiry for urgent reconnection (default 7)
}

// GRPCConnectionOption is a functional option for GRPCConnection.
type GRPCConnectionOption func(*GRPCConnection)

// WithGRPCHeartbeatInterval sets the heartbeat interval.
func WithGRPCHeartbeatInterval(d time.Duration) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.heartbeatInterval = d
	}
}

// WithGRPCInitTimeout sets the initialization timeout.
func WithGRPCInitTimeout(d time.Duration) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.initTimeout = d
	}
}

// WithGRPCRunnerVersion sets the runner version.
func WithGRPCRunnerVersion(version string) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.runnerVersion = version
	}
}

// WithGRPCMCPPort sets the MCP port.
func WithGRPCMCPPort(port int) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.mcpPort = port
	}
}

// WithGRPCServerURL sets the HTTP server URL for REST API calls.
func WithGRPCServerURL(serverURL string) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.serverURL = serverURL
	}
}

// WithGRPCCertRenewalDays sets the days before expiry to trigger renewal.
func WithGRPCCertRenewalDays(days int) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.certRenewalDays = days
	}
}

// WithGRPCCertUrgentDays sets the days before expiry for urgent reconnection.
func WithGRPCCertUrgentDays(days int) GRPCConnectionOption {
	return func(c *GRPCConnection) {
		c.certUrgentDays = days
	}
}

// NewGRPCConnection creates a new gRPC connection with mTLS.
func NewGRPCConnection(endpoint, nodeID, orgSlug, certFile, keyFile, caFile string, opts ...GRPCConnectionOption) *GRPCConnection {
	c := &GRPCConnection{
		endpoint:                 endpoint,
		nodeID:                   nodeID,
		orgSlug:                  orgSlug,
		certFile:                 certFile,
		keyFile:                  keyFile,
		caFile:                   caFile,
		heartbeatInterval:        30 * time.Second,
		initTimeout:              30 * time.Second,
		reconnectStrategy:        NewReconnectStrategy(5*time.Second, 5*time.Minute),
		sendCh:                   make(chan *runnerv1.RunnerMessage, 100),
		stopCh:                   make(chan struct{}),
		reconnectCh:              make(chan struct{}, 1),
		initResultCh:             make(chan *runnerv1.InitializeResult, 1),
		runnerVersion:            "1.0.0",
		mcpPort:                  19000,
		certRenewalCheckInterval: 24 * time.Hour,
		certExpiryWarningDays:    30,
		certRenewalDays:          30,  // Renew 30 days before expiry
		certUrgentDays:           7,   // Urgent reconnection 7 days before expiry
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// SetHandler sets the message handler.
func (c *GRPCConnection) SetHandler(handler MessageHandler) {
	c.handler = handler
}

// SetOrgSlug sets the organization slug.
func (c *GRPCConnection) SetOrgSlug(orgSlug string) {
	c.mu.Lock()
	c.orgSlug = orgSlug
	c.mu.Unlock()
}

// GetOrgSlug returns the organization slug.
func (c *GRPCConnection) GetOrgSlug() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.orgSlug
}

// IsInitialized returns whether the connection has completed initialization.
func (c *GRPCConnection) IsInitialized() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.initialized
}

// GetAvailableAgents returns the list of available agents.
func (c *GRPCConnection) GetAvailableAgents() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.availableAgents
}

// Connect establishes a gRPC connection with mTLS using advancedtls for certificate hot-reloading.
func (c *GRPCConnection) Connect() error {
	// Create advancedtls credentials with file-based certificate reloading
	creds, err := c.createAdvancedTLSCredentials()
	if err != nil {
		return fmt.Errorf("failed to create TLS credentials: %w", err)
	}

	// gRPC dial options
	dialOpts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithKeepaliveParams(keepalive.ClientParameters{
			Time:                30 * time.Second,
			Timeout:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	// Connect to server
	conn, err := grpc.Dial(c.endpoint, dialOpts...)
	if err != nil {
		return fmt.Errorf("failed to dial gRPC server: %w", err)
	}

	// Create gRPC service client
	client := runnerv1.NewRunnerServiceClient(conn)

	c.mu.Lock()
	c.conn = conn
	c.client = client
	c.creds = creds
	c.initialized = false
	c.mu.Unlock()

	logger.GRPC().Info("Connected to server with advancedtls", "endpoint", c.endpoint, "org", c.orgSlug)
	return nil
}

// createAdvancedTLSCredentials creates TLS credentials using advancedtls package.
// This enables automatic certificate hot-reloading when certificate files are updated.
func (c *GRPCConnection) createAdvancedTLSCredentials() (credentials.TransportCredentials, error) {
	// Create identity certificate provider with file watching
	// This provider will automatically reload certificates when files change
	identityProvider, err := pemfile.NewProvider(pemfile.Options{
		CertFile:        c.certFile,
		KeyFile:         c.keyFile,
		RefreshDuration: 1 * time.Hour, // Check for file changes every hour
	})
	if err != nil {
		logger.GRPC().Warn("Failed to create pemfile identity provider, using fallback", "error", err)
		return c.createFallbackTLSCredentials()
	}

	// Create root certificate provider with file watching for CA
	// This allows CA certificate rotation if needed
	rootProvider, err := pemfile.NewProvider(pemfile.Options{
		RootFile:        c.caFile,
		RefreshDuration: 24 * time.Hour, // CA changes are rare, check daily
	})
	if err != nil {
		logger.GRPC().Warn("Failed to create pemfile root provider, using static CA", "error", err)
		// Fall back to static CA if file watching fails
		caCert, readErr := os.ReadFile(c.caFile)
		if readErr != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", readErr)
		}
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		// Save identity provider for cleanup, rootProvider is nil in this path
		c.identityProvider = identityProvider
		c.rootProvider = nil

		// Use static root certificates
		options := &advancedtls.Options{
			IdentityOptions: advancedtls.IdentityCertificateOptions{
				IdentityProvider: identityProvider,
			},
			RootOptions: advancedtls.RootCertificateOptions{
				RootCertificates: caPool,
			},
			MinTLSVersion: tls.VersionTLS13,
			MaxTLSVersion: tls.VersionTLS13,
			// Only verify certificate chain, not hostname (server may use IP)
			VerificationType: advancedtls.CertVerification,
		}
		return advancedtls.NewClientCreds(options)
	}

	// Save providers for cleanup to prevent goroutine leaks
	c.identityProvider = identityProvider
	c.rootProvider = rootProvider

	// Create advancedtls client options with both providers
	options := &advancedtls.Options{
		IdentityOptions: advancedtls.IdentityCertificateOptions{
			IdentityProvider: identityProvider,
		},
		RootOptions: advancedtls.RootCertificateOptions{
			RootProvider: rootProvider,
		},
		MinTLSVersion: tls.VersionTLS13,
		MaxTLSVersion: tls.VersionTLS13,
		// Only verify certificate chain, not hostname (server may use IP)
		VerificationType: advancedtls.CertVerification,
	}

	return advancedtls.NewClientCreds(options)
}

// createFallbackTLSCredentials creates standard TLS credentials as fallback.
// Used when advancedtls is not available or fails to initialize.
func (c *GRPCConnection) createFallbackTLSCredentials() (credentials.TransportCredentials, error) {
	// Load client certificate
	cert, err := tls.LoadX509KeyPair(c.certFile, c.keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	// Load CA certificate (only trust AgentMesh CA, not system CAs)
	caCert, err := os.ReadFile(c.caFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to parse CA certificate")
	}

	// TLS config - only trust our private CA
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS13,
	}

	return credentials.NewTLS(tlsConfig), nil
}

// Start starts the connection management loop.
func (c *GRPCConnection) Start() {
	go c.connectionLoop()
}

// Stop stops the connection and releases resources.
func (c *GRPCConnection) Stop() {
	c.stopOnce.Do(func() {
		close(c.stopCh)
		c.mu.Lock()
		if c.conn != nil {
			c.conn.Close()
			c.conn = nil
		}
		// Close certificate providers to prevent goroutine leaks
		if c.identityProvider != nil {
			c.identityProvider.Close()
			c.identityProvider = nil
		}
		if c.rootProvider != nil {
			c.rootProvider.Close()
			c.rootProvider = nil
		}
		c.mu.Unlock()
	})
}

// send queues a message for sending through the sendCh channel.
// This method is thread-safe and should be used by all public Send* methods
// to avoid direct stream.Send() calls which are not goroutine-safe.
// Returns error if connection is closed, stopped, or channel is full.
func (c *GRPCConnection) send(msg *runnerv1.RunnerMessage) error {
	c.mu.Lock()
	if c.stream == nil {
		c.mu.Unlock()
		return fmt.Errorf("stream not connected")
	}
	c.mu.Unlock()

	select {
	case c.sendCh <- msg:
		return nil
	case <-c.stopCh:
		return fmt.Errorf("connection stopped")
	default:
		return fmt.Errorf("send buffer full")
	}
}

// SendPodCreated sends a pod_created event to the server.
func (c *GRPCConnection) SendPodCreated(podKey string, pid int32) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_PodCreated{
			PodCreated: &runnerv1.PodCreatedEvent{
				PodKey: podKey,
				Pid:    pid,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.send(msg)
}

// SendPodTerminated sends a pod_terminated event to the server.
func (c *GRPCConnection) SendPodTerminated(podKey string, exitCode int32, errorMsg string) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_PodTerminated{
			PodTerminated: &runnerv1.PodTerminatedEvent{
				PodKey:       podKey,
				ExitCode:     exitCode,
				ErrorMessage: errorMsg,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.send(msg)
}

// SendTerminalOutput sends terminal output to the server.
func (c *GRPCConnection) SendTerminalOutput(podKey string, data []byte) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_TerminalOutput{
			TerminalOutput: &runnerv1.TerminalOutputEvent{
				PodKey: podKey,
				Data:   data,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.send(msg)
}

// SendAgentStatus sends an agent status change event to the server.
func (c *GRPCConnection) SendAgentStatus(podKey string, status string) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_AgentStatus{
			AgentStatus: &runnerv1.AgentStatusEvent{
				PodKey: podKey,
				Status: status,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.send(msg)
}

// SendPtyResized sends a PTY resize event to the server.
func (c *GRPCConnection) SendPtyResized(podKey string, cols, rows int32) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_PtyResized{
			PtyResized: &runnerv1.PtyResizedEvent{
				PodKey: podKey,
				Cols:   cols,
				Rows:   rows,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.send(msg)
}

// SendError sends an error event to the server.
func (c *GRPCConnection) SendError(podKey, code, message string) error {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_Error{
			Error: &runnerv1.ErrorEvent{
				PodKey:  podKey,
				Code:    code,
				Message: message,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	return c.send(msg)
}

// QueueLength returns the current send queue length.
func (c *GRPCConnection) QueueLength() int {
	return len(c.sendCh)
}

// QueueCapacity returns the send queue capacity.
func (c *GRPCConnection) QueueCapacity() int {
	return cap(c.sendCh)
}

// connectionLoop manages the connection lifecycle with auto-reconnection.
func (c *GRPCConnection) connectionLoop() {
	for {
		select {
		case <-c.stopCh:
			logger.GRPC().Info("Connection loop stopped")
			return
		default:
		}

		// Try to connect
		if err := c.Connect(); err != nil {
			delay := c.reconnectStrategy.NextDelay()
			logger.GRPC().Warn("Failed to connect, will retry",
				"attempt", c.reconnectStrategy.AttemptCount(),
				"error", err,
				"retry_in", delay)

			select {
			case <-c.stopCh:
				return
			case <-time.After(delay):
			}
			continue
		}

		// Reset reconnect strategy on successful connection
		c.reconnectStrategy.Reset()

		// Run the connection (blocks until disconnected)
		c.runConnection()

		// Check if we should stop
		select {
		case <-c.stopCh:
			return
		default:
		}

		// Wait before reconnecting
		logger.GRPC().Info("Connection closed, will attempt to reconnect")
		select {
		case <-c.stopCh:
			return
		case <-time.After(c.reconnectStrategy.CurrentInterval()):
		}
	}
}

// runConnection establishes the bidirectional stream and handles messages.
func (c *GRPCConnection) runConnection() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Add org_slug to metadata for organization routing
	ctx = metadata.AppendToOutgoingContext(ctx, "x-org-slug", c.orgSlug)

	logger.GRPC().Debug("Establishing bidirectional stream", "org", c.orgSlug)

	// Create bidirectional stream
	stream, err := c.client.Connect(ctx)
	if err != nil {
		logger.GRPC().Error("Failed to establish stream", "error", err)
		return
	}

	c.mu.Lock()
	c.stream = stream
	c.mu.Unlock()

	logger.GRPC().Info("Bidirectional stream established")

	done := make(chan struct{})
	readLoopDone := make(chan struct{}) // Signal when readLoop exits

	// Start write loop
	go c.writeLoop(ctx, done)

	// IMPORTANT: Start read loop BEFORE initialization
	// The read loop must be running to receive the initialize_result response
	go c.readLoop(ctx, readLoopDone)

	// Perform initialization (blocks until handshake completes or times out)
	if err := c.performInitialization(ctx); err != nil {
		logger.GRPC().Error("Initialization failed", "error", err)
		close(done)
		return
	}

	// Start heartbeat loop (only after successful initialization)
	go c.heartbeatLoop(ctx, done)

	// Start certificate renewal checker
	go c.certRenewalChecker(ctx, done)

	// Monitor for reconnection signal (certificate renewal)
	go func() {
		select {
		case <-c.reconnectCh:
			logger.GRPC().Info("Reconnection requested for certificate renewal")
			cancel() // Cancel context to trigger reconnection
		case <-done:
			return
		case <-c.stopCh:
			return
		}
	}()

	// Wait for context cancellation, stop signal, or readLoop exit
	select {
	case <-ctx.Done():
		logger.GRPC().Debug("Context cancelled, closing connection")
	case <-c.stopCh:
		logger.GRPC().Debug("Stop signal received, closing connection")
	case <-readLoopDone:
		logger.GRPC().Debug("Read loop exited, closing connection")
	}

	// Signal other goroutines to stop
	close(done)
}

// performInitialization performs the three-phase initialization handshake.
func (c *GRPCConnection) performInitialization(ctx context.Context) error {
	logger.GRPC().Debug("Starting initialization handshake...")

	// Phase 1: Send initialize request
	hostname, _ := os.Hostname()
	initReq := &runnerv1.InitializeRequest{
		ProtocolVersion: GRPCProtocolVersion,
		RunnerInfo: &runnerv1.RunnerInfo{
			Version:  c.runnerVersion,
			NodeId:   c.nodeID,
			McpPort:  int32(c.mcpPort),
			Os:       runtime.GOOS,
			Arch:     runtime.GOARCH,
			Hostname: hostname,
		},
	}

	// Send initialize request via stream
	msg := &runnerv1.RunnerMessage{
		Payload:   &runnerv1.RunnerMessage_Initialize{Initialize: initReq},
		Timestamp: time.Now().UnixMilli(),
	}
	if err := c.stream.Send(msg); err != nil {
		return fmt.Errorf("failed to send initialize: %w", err)
	}
	logger.GRPC().Debug("Sent initialize request", "version", c.runnerVersion, "mcp_port", c.mcpPort)

	// Phase 2: Wait for initialize_result
	select {
	case result := <-c.initResultCh:
		logger.GRPC().Debug("Received initialize_result",
			"server_version", result.ServerInfo.Version,
			"agent_types", len(result.AgentTypes))

		// Phase 3: Check available agents and send initialized
		availableAgents := c.checkAvailableAgents(result.AgentTypes)
		c.mu.Lock()
		c.availableAgents = availableAgents
		c.mu.Unlock()

		// Send initialized confirmation via stream
		confirmMsg := &runnerv1.RunnerMessage{
			Payload: &runnerv1.RunnerMessage_Initialized{
				Initialized: &runnerv1.InitializedConfirm{
					AvailableAgents: availableAgents,
				},
			},
			Timestamp: time.Now().UnixMilli(),
		}
		if err := c.stream.Send(confirmMsg); err != nil {
			return fmt.Errorf("failed to send initialized: %w", err)
		}
		logger.GRPC().Debug("Sent initialized", "available_agents", availableAgents)

		c.mu.Lock()
		c.initialized = true
		c.mu.Unlock()

		logger.GRPC().Info("Initialization completed successfully")
		return nil

	case <-time.After(c.initTimeout):
		return fmt.Errorf("timeout waiting for initialize_result after %v", c.initTimeout)

	case <-ctx.Done():
		return fmt.Errorf("context cancelled during initialization")

	case <-c.stopCh:
		return fmt.Errorf("connection stopped during initialization")
	}
}

// checkAvailableAgents checks which agents are available on this runner.
func (c *GRPCConnection) checkAvailableAgents(agentTypes []*runnerv1.AgentTypeInfo) []string {
	var available []string
	log := logger.GRPC()

	for _, agent := range agentTypes {
		if agent.Command == "" {
			log.Debug("Agent has no command defined, skipping", "agent", agent.Slug)
			continue
		}

		// Check if executable exists in PATH
		path, err := exec.LookPath(agent.Command)
		if err != nil {
			log.Debug("Agent command not found in PATH", "agent", agent.Slug, "command", agent.Command)
			continue
		}

		log.Debug("Agent command found", "agent", agent.Slug, "path", path)
		available = append(available, agent.Slug)
	}

	return available
}

// readLoop reads messages from the gRPC stream.
// The done channel is closed when the loop exits to notify other goroutines.
func (c *GRPCConnection) readLoop(ctx context.Context, done chan<- struct{}) {
	defer close(done) // Signal exit to other goroutines
	log := logger.GRPC()
	for {
		msg, err := c.stream.Recv()
		if err != nil {
			if err == io.EOF {
				log.Info("Stream ended (EOF)")
				return
			}
			if status.Code(err) == codes.Canceled {
				log.Debug("Stream cancelled")
			} else {
				log.Error("Stream error", "error", err)
			}
			return
		}
		c.handleServerMessage(msg)
	}
}

// handleServerMessage dispatches received server messages to appropriate handlers.
func (c *GRPCConnection) handleServerMessage(msg *runnerv1.ServerMessage) {
	switch payload := msg.Payload.(type) {
	case *runnerv1.ServerMessage_InitializeResult:
		c.handleInitializeResult(payload.InitializeResult)

	case *runnerv1.ServerMessage_CreatePod:
		c.handleCreatePod(payload.CreatePod)

	case *runnerv1.ServerMessage_TerminatePod:
		c.handleTerminatePod(payload.TerminatePod)

	case *runnerv1.ServerMessage_TerminalInput:
		c.handleTerminalInput(payload.TerminalInput)

	case *runnerv1.ServerMessage_TerminalResize:
		c.handleTerminalResize(payload.TerminalResize)

	case *runnerv1.ServerMessage_SendPrompt:
		c.handleSendPrompt(payload.SendPrompt)

	default:
		logger.GRPC().Warn("Unknown server message type")
	}
}

// handleInitializeResult handles initialize_result from server.
func (c *GRPCConnection) handleInitializeResult(result *runnerv1.InitializeResult) {
	logger.GRPC().Debug("Received initialize_result", "version", result.ServerInfo.Version)
	// Convert to internal type and send to channel
	select {
	case c.initResultCh <- result:
	default:
		logger.GRPC().Warn("Initialize result channel full, dropping")
	}
}

// handleCreatePod handles create_pod command from server.
func (c *GRPCConnection) handleCreatePod(cmd *runnerv1.CreatePodCommand) {
	log := logger.GRPC()
	log.Info("Received create_pod", "pod_key", cmd.PodKey)
	if c.handler == nil {
		log.Warn("No handler set, ignoring create_pod")
		return
	}

	// Convert proto to internal type
	req := CreatePodRequest{
		PodKey:        cmd.PodKey,
		LaunchCommand: cmd.LaunchCommand,
		LaunchArgs:    cmd.LaunchArgs,
		EnvVars:       cmd.EnvVars,
	}

	// Convert files_to_create
	if len(cmd.FilesToCreate) > 0 {
		for _, f := range cmd.FilesToCreate {
			req.FilesToCreate = append(req.FilesToCreate, FileToCreate{
				PathTemplate: f.Path,
				Content:      f.Content,
				Mode:         int(f.Mode),
				IsDirectory:  f.IsDirectory,
			})
		}
	}

	// Convert work_dir_config
	if cmd.WorkDirConfig != nil {
		req.WorkDirConfig = &WorkDirConfig{
			Type:       cmd.WorkDirConfig.Type,
			Branch:     cmd.WorkDirConfig.BranchName,
			LocalPath:  cmd.WorkDirConfig.Path,
		}
	}

	if err := c.handler.OnCreatePod(req); err != nil {
		log.Error("Failed to create pod", "pod_key", cmd.PodKey, "error", err)
		c.sendError(cmd.PodKey, "create_pod_failed", err.Error())
	}
}

// handleTerminatePod handles terminate_pod command from server.
func (c *GRPCConnection) handleTerminatePod(cmd *runnerv1.TerminatePodCommand) {
	log := logger.GRPC()
	log.Info("Received terminate_pod", "pod_key", cmd.PodKey, "force", cmd.Force)
	if c.handler == nil {
		log.Warn("No handler set, ignoring terminate_pod")
		return
	}

	req := TerminatePodRequest{PodKey: cmd.PodKey}
	if err := c.handler.OnTerminatePod(req); err != nil {
		log.Error("Failed to terminate pod", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleTerminalInput handles terminal_input command from server.
func (c *GRPCConnection) handleTerminalInput(cmd *runnerv1.TerminalInputCommand) {
	if c.handler == nil {
		return
	}

	req := TerminalInputRequest{
		PodKey: cmd.PodKey,
		Data:   cmd.Data, // gRPC uses native bytes, no encoding needed
	}
	if err := c.handler.OnTerminalInput(req); err != nil {
		logger.GRPC().Error("Failed to send terminal input to pod", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleTerminalResize handles terminal_resize command from server.
func (c *GRPCConnection) handleTerminalResize(cmd *runnerv1.TerminalResizeCommand) {
	if c.handler == nil {
		return
	}

	req := TerminalResizeRequest{
		PodKey: cmd.PodKey,
		Cols:   uint16(cmd.Cols),
		Rows:   uint16(cmd.Rows),
	}
	if err := c.handler.OnTerminalResize(req); err != nil {
		logger.GRPC().Error("Failed to resize terminal for pod", "pod_key", cmd.PodKey, "error", err)
	}
}

// handleSendPrompt handles send_prompt command from server.
func (c *GRPCConnection) handleSendPrompt(cmd *runnerv1.SendPromptCommand) {
	logger.GRPC().Debug("Received send_prompt", "pod_key", cmd.PodKey)
	// TODO: Implement prompt sending when handler supports it
}

// sendError sends an error event back to the server (internal use).
func (c *GRPCConnection) sendError(podKey, code, message string) {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_Error{
			Error: &runnerv1.ErrorEvent{
				PodKey:  podKey,
				Code:    code,
				Message: message,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}
	if err := c.send(msg); err != nil {
		logger.GRPC().Error("Failed to send error", "error", err)
	}
}

// writeLoop sends messages to the gRPC stream.
// This is the ONLY goroutine that calls stream.Send() to ensure thread-safety.
func (c *GRPCConnection) writeLoop(ctx context.Context, done <-chan struct{}) {
	for {
		select {
		case <-c.stopCh:
			return
		case <-done:
			return
		case <-ctx.Done():
			return
		case msg := <-c.sendCh:
			c.mu.Lock()
			stream := c.stream
			c.mu.Unlock()

			if stream != nil {
				if err := stream.Send(msg); err != nil {
					logger.GRPC().Error("Failed to send message", "error", err)
					return
				}
			}
		}
	}
}

// heartbeatLoop sends periodic heartbeats.
func (c *GRPCConnection) heartbeatLoop(ctx context.Context, done <-chan struct{}) {
	ticker := time.NewTicker(c.heartbeatInterval)
	defer ticker.Stop()

	// Send initial heartbeat
	c.sendHeartbeat()

	for {
		select {
		case <-c.stopCh:
			return
		case <-done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.sendHeartbeat()
		}
	}
}

// sendHeartbeat sends a heartbeat message.
func (c *GRPCConnection) sendHeartbeat() {
	var pods []*runnerv1.PodInfo

	if c.handler != nil {
		// Convert from internal PodInfo to proto PodInfo
		internalPods := c.handler.OnListPods()
		for _, p := range internalPods {
			pods = append(pods, &runnerv1.PodInfo{
				PodKey:      p.PodKey,
				Status:      p.Status,
				AgentStatus: p.ClaudeStatus,
			})
		}
	}

	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_Heartbeat{
			Heartbeat: &runnerv1.HeartbeatData{
				NodeId: c.nodeID,
				Pods:   pods,
			},
		},
		Timestamp: time.Now().UnixMilli(),
	}

	logger.GRPC().Debug("Sending heartbeat", "pods", len(pods))

	if err := c.send(msg); err != nil {
		logger.GRPC().Error("Failed to send heartbeat", "error", err)
	}
}

// certRenewalChecker periodically checks certificate expiry and triggers renewal.
func (c *GRPCConnection) certRenewalChecker(ctx context.Context, done <-chan struct{}) {
	ticker := time.NewTicker(c.certRenewalCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-done:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.checkCertificateExpiry()
		}
	}
}

// checkCertificateExpiry checks if the certificate needs renewal.
func (c *GRPCConnection) checkCertificateExpiry() {
	log := logger.GRPC()
	daysUntilExpiry, err := c.getCertDaysUntilExpiry()
	if err != nil {
		log.Error("Failed to check certificate expiry", "error", err)
		return
	}

	log.Debug("Certificate expiry check", "days_until_expiry", daysUntilExpiry)

	// Check if renewal is needed (30 days before expiry by default)
	if daysUntilExpiry <= float64(c.certRenewalDays) {
		log.Info("Certificate expires soon, triggering renewal", "days_until_expiry", daysUntilExpiry)

		// Attempt to renew certificate via REST API
		if err := c.renewCertificate(); err != nil {
			log.Error("Certificate renewal failed", "error", err)
			// Don't return here - still check for urgent reconnection
		} else {
			log.Info("Certificate renewed successfully, advancedtls will auto-reload")
		}
	}

	// Check if urgent reconnection is needed (7 days before expiry by default)
	// This ensures long-lived connections use the new certificate
	if daysUntilExpiry <= float64(c.certUrgentDays) {
		log.Warn("Certificate expiring urgently, triggering reconnection", "days_until_expiry", daysUntilExpiry)
		c.triggerReconnect()
	}
}

// getCertDaysUntilExpiry returns the number of days until the certificate expires.
func (c *GRPCConnection) getCertDaysUntilExpiry() (float64, error) {
	cert, err := tls.LoadX509KeyPair(c.certFile, c.keyFile)
	if err != nil {
		return 0, fmt.Errorf("failed to load certificate: %w", err)
	}

	if len(cert.Certificate) == 0 {
		return 0, fmt.Errorf("no certificate found")
	}

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return 0, fmt.Errorf("failed to parse certificate: %w", err)
	}

	return time.Until(x509Cert.NotAfter).Hours() / 24, nil
}

// IsCertificateExpired checks if the certificate has expired.
// Returns true if the certificate is expired, along with error details if any.
func (c *GRPCConnection) IsCertificateExpired() (bool, error) {
	daysUntilExpiry, err := c.getCertDaysUntilExpiry()
	if err != nil {
		return false, err
	}
	return daysUntilExpiry <= 0, nil
}

// CertificateExpiryInfo returns detailed information about certificate expiry.
type CertificateExpiryInfo struct {
	DaysUntilExpiry float64
	ExpiresAt       time.Time
	IsExpired       bool
	NeedsRenewal    bool
	NeedsUrgent     bool
}

// GetCertificateExpiryInfo returns detailed certificate expiry information.
func (c *GRPCConnection) GetCertificateExpiryInfo() (*CertificateExpiryInfo, error) {
	cert, err := tls.LoadX509KeyPair(c.certFile, c.keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	if len(cert.Certificate) == 0 {
		return nil, fmt.Errorf("no certificate found")
	}

	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return nil, fmt.Errorf("failed to parse certificate: %w", err)
	}

	daysUntilExpiry := time.Until(x509Cert.NotAfter).Hours() / 24

	return &CertificateExpiryInfo{
		DaysUntilExpiry: daysUntilExpiry,
		ExpiresAt:       x509Cert.NotAfter,
		IsExpired:       daysUntilExpiry <= 0,
		NeedsRenewal:    daysUntilExpiry <= float64(c.certRenewalDays),
		NeedsUrgent:     daysUntilExpiry <= float64(c.certUrgentDays),
	}, nil
}

// renewCertificate calls the Backend REST API to renew the certificate.
// The new certificate is saved to files, and advancedtls will automatically reload them.
func (c *GRPCConnection) renewCertificate() error {
	if c.serverURL == "" {
		return fmt.Errorf("server URL not configured, cannot renew certificate")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Call the renewal API using mTLS
	// Note: This requires a working TLS connection with the current certificate
	result, err := RenewCertificate(ctx, RenewalRequest{
		ServerURL: c.serverURL,
		CertFile:  c.certFile,
		KeyFile:   c.keyFile,
		CAFile:    c.caFile,
	})
	if err != nil {
		return fmt.Errorf("renewal API call failed: %w", err)
	}

	// Save new certificate to files
	// advancedtls FileWatcherCertificateProvider will detect the change and reload
	if err := os.WriteFile(c.certFile, []byte(result.Certificate), 0600); err != nil {
		return fmt.Errorf("failed to save new certificate: %w", err)
	}

	if err := os.WriteFile(c.keyFile, []byte(result.PrivateKey), 0600); err != nil {
		return fmt.Errorf("failed to save new private key: %w", err)
	}

	logger.GRPC().Info("New certificate saved",
		"expires_at", time.Unix(result.ExpiresAt, 0).Format(time.RFC3339))

	return nil
}

// triggerReconnect signals the connection loop to reconnect.
// This is used to ensure long-lived connections use the new certificate.
func (c *GRPCConnection) triggerReconnect() {
	select {
	case c.reconnectCh <- struct{}{}:
		logger.GRPC().Info("Reconnection triggered")
	default:
		// Reconnection already pending
	}
}

// ==================== gRPC Error Handling ====================

// isRetryableError returns true if the gRPC error is retryable.
func isRetryableError(err error) bool {
	st, ok := status.FromError(err)
	if !ok {
		return false
	}

	switch st.Code() {
	case codes.Unavailable, codes.ResourceExhausted, codes.Aborted:
		return true
	default:
		return false
	}
}

// Ensure GRPCConnection implements Connection interface.
var _ Connection = (*GRPCConnection)(nil)
