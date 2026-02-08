package logger

import (
	"context"
	"log/slog"
)

// Module creates a logger with a module prefix.
// Usage: log := logger.Module("grpc")
//
//	log.Info("Connected", "endpoint", addr)
func Module(name string) *slog.Logger {
	return Default().With("module", name)
}

// Pre-defined module loggers for common components.
// These provide convenient access to module-specific loggers.

// GRPC returns a logger for gRPC-related logging.
func GRPC() *slog.Logger {
	return Module("grpc")
}

// Runner returns a logger for runner core logging.
func Runner() *slog.Logger {
	return Module("runner")
}

// Terminal returns a logger for terminal/PTY logging.
func Terminal() *slog.Logger {
	return Module("terminal")
}

// MCP returns a logger for MCP-related logging.
func MCP() *slog.Logger {
	return Module("mcp")
}

// Workspace returns a logger for workspace/git operations.
func Workspace() *slog.Logger {
	return Module("workspace")
}

// Sandbox returns a logger for sandbox operations.
func Sandbox() *slog.Logger {
	return Module("sandbox")
}

// Tray returns a logger for system tray operations.
func Tray() *slog.Logger {
	return Module("tray")
}

// Service returns a logger for system service operations.
func Service() *slog.Logger {
	return Module("service")
}

// Console returns a logger for web console operations.
func Console() *slog.Logger {
	return Module("console")
}

// Monitor returns a logger for agent monitor operations.
func Monitor() *slog.Logger {
	return Module("monitor")
}

// Pod returns a logger for pod operations.
func Pod() *slog.Logger {
	return Module("pod")
}

// Plugin returns a logger for plugin operations.
func Plugin() *slog.Logger {
	return Module("plugin")
}

// Autopilot returns a logger for AutopilotController operations.
func Autopilot() *slog.Logger {
	return Module("autopilot")
}

// Trace logs a message at Trace level using the default logger.
// Use for high-frequency, low-level debugging that would be too verbose for Debug.
func Trace(msg string, args ...any) {
	slog.Log(context.Background(), LevelTrace, msg, args...)
}

// LogTrace logs a message at Trace level using the given logger.
// Use for high-frequency, low-level debugging that would be too verbose for Debug.
func LogTrace(l *slog.Logger, msg string, args ...any) {
	l.Log(context.Background(), LevelTrace, msg, args...)
}

// TraceLogger wraps a slog.Logger with a convenient Trace method.
type TraceLogger struct {
	*slog.Logger
}

// Trace logs a message at Trace level.
func (l TraceLogger) Trace(msg string, args ...any) {
	l.Log(context.Background(), LevelTrace, msg, args...)
}

// WithTrace wraps a slog.Logger to add the Trace convenience method.
func WithTrace(l *slog.Logger) TraceLogger {
	return TraceLogger{Logger: l}
}

// Module-specific loggers with Trace support

// TerminalTrace returns a logger for terminal/PTY logging with Trace support.
func TerminalTrace() TraceLogger {
	return WithTrace(Module("terminal"))
}

// GRPCTrace returns a logger for gRPC-related logging with Trace support.
func GRPCTrace() TraceLogger {
	return WithTrace(Module("grpc"))
}

// RunnerTrace returns a logger for runner core logging with Trace support.
func RunnerTrace() TraceLogger {
	return WithTrace(Module("runner"))
}

// AutopilotTrace returns a logger for AutopilotController operations with Trace support.
func AutopilotTrace() TraceLogger {
	return WithTrace(Module("autopilot"))
}
