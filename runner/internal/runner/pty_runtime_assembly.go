package runner

import (
	"reflect"

	"github.com/anthropics/agentsmesh/runner/internal/terminal"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/aggregator"
	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

// assemblePTYRuntime binds already-created PTY infrastructure to a Pod.
// Process creation and lifecycle policy remain the caller's responsibility.
func assemblePTYRuntime(
	pod *Pod,
	term *terminal.Terminal,
	virtualTerm *vt.VirtualTerminal,
	agg *aggregator.SmartAggregator,
	ptyLogger *aggregator.PTYLogger,
	localRelay LocalRelayBroker,
) PodRuntime {
	comps := &PTYComponents{
		Terminal:        term,
		VirtualTerminal: virtualTerm,
		Aggregator:      agg,
		PTYLogger:       ptyLogger,
	}

	term.SetReadOrderLocker(&comps.readMu)
	term.SetOutputHandler(NewPTYOutputHandler(pod.PodKey, comps, pod.NotifyStateDetectorWithScreen))
	ptyIO := NewPTYPodIO(pod.PodKey, comps, PTYPodIODeps{
		GetOrCreateDetector: pod.GetOrCreateStateDetector,
		SubscribeState:      pod.SubscribeStateChange,
		UnsubscribeState:    pod.UnsubscribeStateChange,
		GetPTYError:         pod.GetPTYError,
	})
	ptyRelay := NewPTYPodRelay(
		pod.PodKey,
		ptyIO,
		comps,
		normalizeLocalRelayBroker(localRelay),
	)
	return PodRuntime{
		IO:         ptyIO,
		Relay:      ptyRelay,
		vtProvider: func() *vt.VirtualTerminal { return virtualTerm },
	}
}

// normalizeLocalRelayBroker prevents a nil pointer wrapped in the broker
// interface from being mistaken for an active local destination.
func normalizeLocalRelayBroker(broker LocalRelayBroker) LocalRelayBroker {
	if broker == nil {
		return nil
	}
	value := reflect.ValueOf(broker)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		if value.IsNil() {
			return nil
		}
	}
	return broker
}
