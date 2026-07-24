package client

import (
	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/safego"
)

func subscribePodRequest(cmd *runnerv1.SubscribePodCommand) SubscribePodRequest {
	return SubscribePodRequest{
		PodKey:          cmd.PodKey,
		RelayURL:        cmd.RelayUrl,
		RunnerToken:     cmd.RunnerToken,
		LocalToken:      cmd.LocalToken,
		IncludeSnapshot: cmd.IncludeSnapshot,
		SnapshotHistory: cmd.SnapshotHistory,
	}
}

func (c *GRPCConnection) handleSubscribeRequest(req SubscribePodRequest) {
	log := logger.GRPC()
	log.Info("Received subscribe_pod", "pod_key", req.PodKey, "relay_url", req.RelayURL)
	if c.handler == nil {
		log.Warn("No handler set, ignoring subscribe_pod")
		return
	}
	if err := c.handler.OnSubscribePod(req); err != nil {
		log.Error("Failed to subscribe pod", "pod_key", req.PodKey, "error", err)
	}
}

func (c *GRPCConnection) admitCreatePod(cmd *runnerv1.CreatePodCommand) {
	intent, invalidate := c.podRelayIntents.Begin(cmd.PodKey)
	if invalidate {
		c.handleUnsubscribePod(&runnerv1.UnsubscribePodCommand{PodKey: cmd.PodKey})
	}
	c.handlerWg.Add(1)
	c.podQueue.Enqueue(cmd.PodKey, func() {
		defer c.handlerWg.Done()
		drainStarted := false
		defer func() {
			if !drainStarted {
				c.podRelayIntents.AbortIf(intent)
			}
		}()
		if !c.handleCreatePod(cmd) {
			return
		}
		drainStarted = c.startPodRelayIntentDrain(intent)
	})
}

func (c *GRPCConnection) admitTerminatePod(cmd *runnerv1.TerminatePodCommand) {
	if c.podRelayIntents.Cancel(cmd.PodKey) {
		c.handleUnsubscribePod(&runnerv1.UnsubscribePodCommand{PodKey: cmd.PodKey})
	}
	c.handlerWg.Add(1)
	c.podQueue.Enqueue(cmd.PodKey, func() {
		defer c.handlerWg.Done()
		c.handleTerminatePod(cmd)
		c.podQueue.Remove(cmd.PodKey)
	})
}

func (c *GRPCConnection) admitSubscribePod(cmd *runnerv1.SubscribePodCommand) {
	req := subscribePodRequest(cmd)
	intent, admission, teardownEpoch := c.podRelayIntents.OfferSubscribe(req)
	switch admission {
	case relayIntentStart:
		c.startPodRelayIntentDrain(intent)
	case relayIntentInvalidate:
		c.teardownRelayIntent(intent, teardownEpoch, req.PodKey)
	}
}

func (c *GRPCConnection) admitUnsubscribePod(cmd *runnerv1.UnsubscribePodCommand) {
	intent, admission, teardownEpoch := c.podRelayIntents.OfferUnsubscribe(cmd.PodKey)
	switch admission {
	case relayIntentDirect:
		c.handleUnsubscribePod(cmd)
	case relayIntentInvalidate:
		c.teardownRelayIntent(intent, teardownEpoch, cmd.PodKey)
	}
}

func (c *GRPCConnection) teardownRelayIntent(
	intent *podRelayIntent,
	teardownEpoch uint64,
	podKey string,
) {
	// Runner teardown runs without the intent mutex. The deferred release wakes
	// the drain only after this invalidation's transport teardown has returned.
	defer intent.endTeardown(teardownEpoch)
	c.handleUnsubscribePod(&runnerv1.UnsubscribePodCommand{PodKey: podKey})
}

func (c *GRPCConnection) startPodRelayIntentDrain(intent *podRelayIntent) bool {
	drainEpoch, claimed := intent.claimDrain()
	if !claimed {
		return false
	}
	c.handlerWg.Add(1)
	safego.Go("drain-pod-relay-intent", func() {
		defer c.handlerWg.Done()
		c.podRelayIntents.Drain(intent, drainEpoch, c.handleSubscribeRequest)
	})
	return true
}
