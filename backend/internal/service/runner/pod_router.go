package runner

import (
	"context"
	"fmt"
	"hash/fnv"
	"log/slog"
	"sync"

	"github.com/anthropics/agentsmesh/backend/internal/infra/eventbus"
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
)

type PodRouter struct {
	connectionManager *RunnerConnectionManager
	logger            *slog.Logger

	commandSender RunnerCommandSender

	oscDetector *OSCDetector

	shards [routerShards]*routerShard

	pendingQueries sync.Map      // map[requestID]*pendingPodQuery
	done           chan struct{} // signal channel for graceful shutdown of cleanup goroutine
}

func NewPodRouter(cm *RunnerConnectionManager, logger *slog.Logger) *PodRouter {
	tr := &PodRouter{
		connectionManager: cm,
		logger:            logger,
		commandSender:     NewNoOpCommandSender(logger), // Default to no-op
	}

	for i := 0; i < routerShards; i++ {
		tr.shards[i] = newRouterShard()
	}

	cm.SetOSCNotificationCallback(tr.handleOSCNotification)
	cm.SetOSCTitleCallback(tr.handleOSCTitle)

	initQuerySupport(tr, cm, make(chan struct{}))

	return tr
}

func (tr *PodRouter) getShard(podKey string) *routerShard {
	h := fnv.New32a()
	h.Write([]byte(podKey))
	return tr.shards[h.Sum32()%routerShards]
}

func (tr *PodRouter) SetCommandSender(sender RunnerCommandSender) {
	tr.commandSender = sender
	tr.logger.Info("command sender configured", "type", fmt.Sprintf("%T", sender))
}

func (tr *PodRouter) SetEventBus(eb *eventbus.EventBus) {
	if tr.oscDetector == nil {
		tr.oscDetector = &OSCDetector{}
	}
	tr.oscDetector.eventBus = eb
}

func (tr *PodRouter) SetPodInfoGetter(getter PodInfoGetter) {
	if tr.oscDetector == nil {
		tr.oscDetector = &OSCDetector{}
	}
	tr.oscDetector.podInfoGetter = getter
}

func (tr *PodRouter) SetNotifyFunc(fn NotifyFunc) {
	if tr.oscDetector == nil {
		tr.oscDetector = &OSCDetector{}
	}
	tr.oscDetector.notifyFunc = fn
}

func (tr *PodRouter) RegisterPod(podKey string, runnerID int64) {
	shard := tr.getShard(podKey)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	shard.podRunnerMap[podKey] = runnerID

	tr.logger.Debug("pod registered",
		"pod_key", podKey,
		"runner_id", runnerID)
}

func (tr *PodRouter) EnsurePodRegistered(podKey string, runnerID int64) {
	shard := tr.getShard(podKey)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	shard.podRunnerMap[podKey] = runnerID
	tr.logger.Debug("pod registered (ensure)", "pod_key", podKey, "runner_id", runnerID)
}

func (tr *PodRouter) UnregisterPod(podKey string) {
	shard := tr.getShard(podKey)

	shard.mu.Lock()
	delete(shard.podRunnerMap, podKey)
	shard.mu.Unlock()

	tr.logger.Debug("pod unregistered", "pod_key", podKey)
}

func (tr *PodRouter) handleOSCNotification(runnerID int64, data *runnerv1.OSCNotificationEvent) {
	podKey := data.PodKey

	tr.logger.Info("received OSC notification",
		"pod_key", podKey,
		"runner_id", runnerID,
		"title", data.Title,
		"body", data.Body,
	)

	if tr.oscDetector != nil && tr.oscDetector.eventBus != nil {
		tr.oscDetector.PublishNotification(context.Background(), podKey, data.Title, data.Body)
	}
}

func (tr *PodRouter) handleOSCTitle(runnerID int64, data *runnerv1.OSCTitleEvent) {
	podKey := data.PodKey

	tr.logger.Debug("received OSC title change",
		"pod_key", podKey,
		"runner_id", runnerID,
		"title", data.Title,
	)

	if tr.oscDetector != nil && tr.oscDetector.eventBus != nil {
		tr.oscDetector.PublishTitle(context.Background(), podKey, data.Title)
	}
}

func (tr *PodRouter) RoutePodInput(podKey string, data []byte) error {
	shard := tr.getShard(podKey)

	shard.mu.RLock()
	runnerID, ok := shard.podRunnerMap[podKey]
	shard.mu.RUnlock()

	if !ok {
		tr.logger.Warn("no runner for pod", "pod_key", podKey)
		return ErrRunnerNotConnected
	}

	return tr.commandSender.SendPodInput(context.Background(), runnerID, podKey, data)
}

func (tr *PodRouter) RoutePrompt(podKey string, prompt string) error {
	shard := tr.getShard(podKey)

	shard.mu.RLock()
	runnerID, ok := shard.podRunnerMap[podKey]
	shard.mu.RUnlock()

	if !ok {
		tr.logger.Warn("no runner for pod", "pod_key", podKey)
		return ErrRunnerNotConnected
	}

	return tr.commandSender.SendPrompt(context.Background(), runnerID, podKey, prompt)
}

func (tr *PodRouter) IsPodRegistered(podKey string) bool {
	shard := tr.getShard(podKey)

	shard.mu.RLock()
	defer shard.mu.RUnlock()
	_, ok := shard.podRunnerMap[podKey]
	return ok
}

func (tr *PodRouter) GetRunnerID(podKey string) (int64, bool) {
	shard := tr.getShard(podKey)

	shard.mu.RLock()
	defer shard.mu.RUnlock()
	id, ok := shard.podRunnerMap[podKey]
	return id, ok
}

func (tr *PodRouter) GetRegisteredPodCount() int {
	total := 0
	for i := 0; i < routerShards; i++ {
		shard := tr.shards[i]
		shard.mu.RLock()
		total += len(shard.podRunnerMap)
		shard.mu.RUnlock()
	}
	return total
}

func (tr *PodRouter) Stop() {
	close(tr.done)
}
