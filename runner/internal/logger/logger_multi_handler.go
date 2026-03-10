package logger

import (
	"context"
	"log/slog"
)

// multiHandler dispatches log records to multiple handlers with different levels.
// File handler always logs Debug+, stderr handler follows configured level.
type multiHandler struct {
	fileHandler   slog.Handler // File: Debug level (always)
	stderrHandler slog.Handler // Stderr: configured level
	fileLevel     slog.Level   // Debug
	stderrLevel   slog.Level   // configured level
}

func (h *multiHandler) Enabled(ctx context.Context, level slog.Level) bool {
	// Enabled if either handler accepts this level
	return level >= h.fileLevel || level >= h.stderrLevel
}

func (h *multiHandler) Handle(ctx context.Context, r slog.Record) error {
	// File always logs Debug+ (not Trace)
	if h.fileHandler != nil && r.Level >= h.fileLevel {
		if err := h.fileHandler.Handle(ctx, r); err != nil {
			return err
		}
	}
	// Stderr follows configured level
	if h.stderrHandler != nil && r.Level >= h.stderrLevel {
		if err := h.stderrHandler.Handle(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

func (h *multiHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	newHandler := &multiHandler{
		fileLevel:   h.fileLevel,
		stderrLevel: h.stderrLevel,
	}
	if h.fileHandler != nil {
		newHandler.fileHandler = h.fileHandler.WithAttrs(attrs)
	}
	if h.stderrHandler != nil {
		newHandler.stderrHandler = h.stderrHandler.WithAttrs(attrs)
	}
	return newHandler
}

func (h *multiHandler) WithGroup(name string) slog.Handler {
	newHandler := &multiHandler{
		fileLevel:   h.fileLevel,
		stderrLevel: h.stderrLevel,
	}
	if h.fileHandler != nil {
		newHandler.fileHandler = h.fileHandler.WithGroup(name)
	}
	if h.stderrHandler != nil {
		newHandler.stderrHandler = h.stderrHandler.WithGroup(name)
	}
	return newHandler
}
