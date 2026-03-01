package autopilot

import (
	"log/slog"
	"time"

	runnerv1 "github.com/AgentsMesh/AgentsMesh/proto/gen/go/runner/v1"
)

// GRPCEventReporter reports Autopilot events via gRPC connection.
type GRPCEventReporter struct {
	sendFunc func(msg *runnerv1.RunnerMessage) error
	log      *slog.Logger
}

// GRPCEventReporterConfig contains configuration for GRPCEventReporter.
type GRPCEventReporterConfig struct {
	SendFunc func(msg *runnerv1.RunnerMessage) error
	Logger   *slog.Logger
}

// NewGRPCEventReporter creates a new gRPC event reporter.
func NewGRPCEventReporter(sendFunc func(msg *runnerv1.RunnerMessage) error) *GRPCEventReporter {
	return &GRPCEventReporter{
		sendFunc: sendFunc,
		log:      slog.Default(),
	}
}

// NewGRPCEventReporterWithConfig creates a new gRPC event reporter with configuration.
func NewGRPCEventReporterWithConfig(cfg GRPCEventReporterConfig) *GRPCEventReporter {
	log := cfg.Logger
	if log == nil {
		log = slog.Default()
	}
	return &GRPCEventReporter{
		sendFunc: cfg.SendFunc,
		log:      log,
	}
}

// ReportAutopilotStatus reports an Autopilot status event.
func (r *GRPCEventReporter) ReportAutopilotStatus(event *runnerv1.AutopilotStatusEvent) {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_AutopilotStatus{
			AutopilotStatus: event,
		},
		Timestamp: time.Now().UnixMilli(),
	}
	if err := r.sendFunc(msg); err != nil {
		r.log.Warn("Failed to report autopilot status event",
			"error", err,
			"autopilot_key", event.AutopilotKey,
			"phase", event.Status.GetPhase())
	}
}

// ReportAutopilotIteration reports an Autopilot iteration event.
func (r *GRPCEventReporter) ReportAutopilotIteration(event *runnerv1.AutopilotIterationEvent) {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_AutopilotIteration{
			AutopilotIteration: event,
		},
		Timestamp: time.Now().UnixMilli(),
	}
	if err := r.sendFunc(msg); err != nil {
		r.log.Warn("Failed to report autopilot iteration event",
			"error", err,
			"autopilot_key", event.AutopilotKey,
			"iteration", event.Iteration,
			"phase", event.Phase)
	}
}

// ReportAutopilotCreated reports an Autopilot created event.
func (r *GRPCEventReporter) ReportAutopilotCreated(event *runnerv1.AutopilotCreatedEvent) {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_AutopilotCreated{
			AutopilotCreated: event,
		},
		Timestamp: time.Now().UnixMilli(),
	}
	if err := r.sendFunc(msg); err != nil {
		r.log.Warn("Failed to report autopilot created event",
			"error", err,
			"autopilot_key", event.AutopilotKey,
			"pod_key", event.PodKey)
	}
}

// ReportAutopilotTerminated reports an Autopilot terminated event.
func (r *GRPCEventReporter) ReportAutopilotTerminated(event *runnerv1.AutopilotTerminatedEvent) {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_AutopilotTerminated{
			AutopilotTerminated: event,
		},
		Timestamp: time.Now().UnixMilli(),
	}
	if err := r.sendFunc(msg); err != nil {
		r.log.Warn("Failed to report autopilot terminated event",
			"error", err,
			"autopilot_key", event.AutopilotKey,
			"reason", event.Reason)
	}
}

// ReportAutopilotThinking reports an Autopilot thinking event.
// This event exposes the Control Agent's decision-making process to the user.
func (r *GRPCEventReporter) ReportAutopilotThinking(event *runnerv1.AutopilotThinkingEvent) {
	msg := &runnerv1.RunnerMessage{
		Payload: &runnerv1.RunnerMessage_AutopilotThinking{
			AutopilotThinking: event,
		},
		Timestamp: time.Now().UnixMilli(),
	}
	if err := r.sendFunc(msg); err != nil {
		r.log.Warn("Failed to report autopilot thinking event",
			"error", err,
			"autopilot_key", event.AutopilotKey,
			"iteration", event.Iteration,
			"decision_type", event.DecisionType)
	}
}
