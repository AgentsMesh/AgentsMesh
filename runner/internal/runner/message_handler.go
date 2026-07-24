package runner

import (
	"fmt"
	"log/slog"
	"sync"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/client"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/relay"
)

// RunnerMessageHandler implements client.MessageHandler interface.
type RunnerMessageHandler struct {
	runner                  MessageHandlerContext
	podStore                PodStore
	conn                    client.Connection
	relayClientFactory      func(url, podKey, token string, logger *slog.Logger) relay.RelayClient
	perpetualSessionFactory perpetualSessionFactory
	perpetualRuntimeFactory perpetualRuntimeFactory

	// agentUpgradeMu serializes OnUpgradeAgent: concurrent package-manager
	// installs on one runner race on the manager's global cache lock.
	agentUpgradeMu sync.Mutex
}

// NewRunnerMessageHandler creates a new message handler.
func NewRunnerMessageHandler(runner MessageHandlerContext, store PodStore, conn client.Connection) *RunnerMessageHandler {
	logger.Runner().Debug("Creating message handler")
	handler := &RunnerMessageHandler{
		runner:   runner,
		podStore: store,
		conn:     conn,
		relayClientFactory: func(url, podKey, token string, logger *slog.Logger) relay.RelayClient {
			return relay.NewClient(runner.GetRunContext(), url, podKey, token, logger)
		},
	}
	handler.perpetualSessionFactory = defaultPerpetualSessionFactory
	handler.perpetualRuntimeFactory = handler.buildPerpetualPTYRuntime
	return handler
}

// OnTerminatePod handles terminate pod requests from server.
func (h *RunnerMessageHandler) OnTerminatePod(req client.TerminatePodRequest) error {
	log := logger.Pod()
	log.Info("Terminating pod", "pod_key", req.PodKey)

	if _, ok := h.podStore.Get(req.PodKey); !ok {
		log.Warn("Pod not found for termination", "pod_key", req.PodKey)
		return fmt.Errorf("pod not found: %s", req.PodKey)
	}

	h.cleanupPodExit(req.PodKey, -1, true)
	return nil
}

// OnUpdatePodPerpetual handles update_pod_perpetual command from server.
// Updates the pod's in-memory Perpetual flag so the next exit uses the correct behavior.
func (h *RunnerMessageHandler) OnUpdatePodPerpetual(cmd *runnerv1.UpdatePodPerpetualCommand) error {
	log := logger.Pod()
	pod, ok := h.podStore.Get(cmd.PodKey)
	if !ok {
		log.Warn("Pod not found for perpetual update", "pod_key", cmd.PodKey)
		return fmt.Errorf("pod not found: %s", cmd.PodKey)
	}
	pod.lifecycleMu.Lock()
	pod.Perpetual = cmd.Perpetual
	pod.lifecycleMu.Unlock()
	log.Info("Pod perpetual mode updated", "pod_key", cmd.PodKey, "perpetual", cmd.Perpetual)
	return nil
}

// OnListPods returns current pods.
func (h *RunnerMessageHandler) OnListPods() []client.PodInfo {
	pods := h.podStore.All()
	result := make([]client.PodInfo, 0, len(pods))

	for _, s := range pods {
		info := client.PodInfo{
			PodKey:      s.PodKey,
			Status:      s.GetStatus(),
			AgentStatus: h.getAgentStatusFromDetector(s),
		}
		s.WithActiveIO(func(io PodIO) { info.Pid = io.GetPID() })
		result = append(result, info)
	}

	return result
}

func (h *RunnerMessageHandler) getAgentStatusFromDetector(pod *Pod) string {
	status := "idle"
	pod.WithActiveIO(func(io PodIO) { status = io.GetAgentStatus() })
	return status
}

var _ client.MessageHandler = (*RunnerMessageHandler)(nil)

// OnGetLocalRelayURL returns the runner's advertised local relay URL.
// Surfaced in heartbeat so backend can route browser→runner traffic locally
// when the renderer happens to live on the same host.
func (h *RunnerMessageHandler) OnGetLocalRelayURL() string {
	srv := h.runner.GetLocalRelayServer()
	if srv == nil {
		return ""
	}
	return srv.URL()
}

// Note: OnSubscribePod, setupRelayClientHandlers, OnUnsubscribePod are in message_handler_relay.go
// Note: OnListRelayConnections, OnPodInput, OnQuerySandboxes, OnObservePod, OnSendPrompt are in message_handler_ops.go
