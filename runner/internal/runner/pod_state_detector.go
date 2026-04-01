package runner

import (
	"sync"

	"github.com/anthropics/agentsmesh/runner/internal/logger"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/detector"
)

// GetOrCreateStateDetector returns the state detector for this pod, creating one if needed.
func (p *Pod) GetOrCreateStateDetector() detector.StateDetector {
	d := p.getOrCreateStateDetectorInternal()
	if d == nil {
		return nil
	}
	return d
}

// SubscribeStateChange subscribes to state change events.
// Returns false if VirtualTerminal is not available.
func (p *Pod) SubscribeStateChange(id string, cb func(detector.StateChangeEvent)) bool {
	d := p.getOrCreateStateDetectorInternal()
	if d == nil {
		return false
	}
	d.Subscribe(id, cb)
	return true
}

// SubscribeAgentStatusBridge subscribes to state detection events and bridges them
// to the backend via the provided sendStatus function.
func (p *Pod) SubscribeAgentStatusBridge(sendStatus func(podKey, status string) error) {
	if p.vtProvider == nil {
		return
	}

	var statusMu sync.Mutex
	lastSentStatus := ""
	podKey := p.PodKey

	p.SubscribeStateChange("grpc-agent-status", func(event detector.StateChangeEvent) {
		var backendStatus string
		switch event.NewState {
		case detector.StateExecuting:
			backendStatus = "executing"
		case detector.StateWaiting:
			backendStatus = "waiting"
		case detector.StateNotRunning:
			backendStatus = "idle"
		default:
			return
		}
		statusMu.Lock()
		if backendStatus == lastSentStatus {
			statusMu.Unlock()
			return
		}
		lastSentStatus = backendStatus
		statusMu.Unlock()
		if err := sendStatus(podKey, backendStatus); err != nil {
			logger.Pod().Error("Failed to send agent status",
				"pod_key", podKey, "status", backendStatus, "error", err)
		}
	})
}

// UnsubscribeStateChange removes a state change subscription by ID.
func (p *Pod) UnsubscribeStateChange(id string) {
	p.stateDetectorMu.RLock()
	d := p.stateDetector
	p.stateDetectorMu.RUnlock()

	if d != nil {
		d.Unsubscribe(id)
	}
}

func (p *Pod) getOrCreateStateDetectorInternal() *ManagedStateDetector {
	p.stateDetectorMu.RLock()
	if p.stateDetector != nil {
		defer p.stateDetectorMu.RUnlock()
		return p.stateDetector
	}
	p.stateDetectorMu.RUnlock()

	p.stateDetectorMu.Lock()
	defer p.stateDetectorMu.Unlock()

	if p.stateDetector != nil {
		return p.stateDetector
	}

	if p.vtProvider != nil {
		if vt := p.vtProvider(); vt != nil {
			p.stateDetector = NewManagedStateDetector(vt)
		}
	}
	return p.stateDetector
}

// NotifyStateDetectorWithScreen notifies the state detector about new output.
func (p *Pod) NotifyStateDetectorWithScreen(bytes int, screenLines []string) {
	p.stateDetectorMu.RLock()
	detector := p.stateDetector
	p.stateDetectorMu.RUnlock()

	if detector != nil {
		detector.OnOutput(bytes)
		if screenLines != nil {
			detector.OnScreenUpdate(screenLines)
		}
	}
}

// StopStateDetector stops the state detector if running.
func (p *Pod) StopStateDetector() {
	p.stateDetectorMu.Lock()
	defer p.stateDetectorMu.Unlock()

	if p.stateDetector != nil {
		p.stateDetector.Stop()
		p.stateDetector = nil
	}
}
