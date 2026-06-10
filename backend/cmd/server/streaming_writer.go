package main

import (
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

// streamWriteTimeout is the sliding per-write deadline for connect server-stream
// responses. It must exceed the longest keepalive interval (events: 25s) so a
// healthy stream — which writes at least one frame per interval — keeps pushing
// the deadline forward and is never killed. A dead/half-open client stops
// consuming; its next blocked write trips this deadline, the write errors, and
// the handler unwinds (releasing the goroutine + subscription) rather than
// blocking until the OS TCP keepalive (minutes-to-hours) notices.
const streamWriteTimeout = 60 * time.Second

// unsupportedDeadlineWarnInterval rate-limits the "deadline unsupported" warning.
// A persistent failure (e.g. HTTP/2, where SetWriteDeadline is unsupported)
// degrades EVERY stream, so the warning must keep surfacing — not be silenced
// for the whole process after a single log line.
const unsupportedDeadlineWarnInterval = time.Minute

var lastUnsupportedDeadlineWarnNanos atomic.Int64

// streamingResponseWriter gives a connect server-stream response a per-write
// sliding deadline so a long-lived stream is never aborted by the server's
// fixed WriteTimeout (the deadline re-arms each frame). It auto-detects the
// response kind from the Content-Type the handler sets: a UNARY response keeps
// the tighter server WriteTimeout, everything else slides. So there's no
// per-procedure allowlist to maintain — a newly-added server-stream is
// protected automatically — and unary RPCs (the bulk of traffic) are untouched.
type streamingResponseWriter struct {
	http.ResponseWriter
	sliding bool
	decided bool
}

func newStreamingResponseWriter(w http.ResponseWriter) http.ResponseWriter {
	return &streamingResponseWriter{ResponseWriter: w}
}

// isUnaryContentType reports whether a response Content-Type is a connect UNARY
// framing (application/proto or application/json). Everything else — connect
// streams (application/connect+*), gRPC / gRPC-Web (application/grpc*), or an
// unrecognized/empty type — is treated as streaming. Defaulting the unknown
// case to streaming is deliberate: a stream wrongly left under the short
// WriteTimeout would be killed mid-flight, whereas a unary wrongly given the
// sliding deadline only widens its (bounded) write-stall window.
func isUnaryContentType(ct string) bool {
	if i := strings.IndexByte(ct, ';'); i >= 0 { // strip "; charset=utf-8" etc.
		ct = ct[:i]
	}
	switch strings.TrimSpace(ct) {
	case "application/proto", "application/json":
		return true
	default:
		return false
	}
}

// decide latches the streaming decision once the handler has set Content-Type
// (at the first WriteHeader or first Write). Idempotent.
func (w *streamingResponseWriter) decide() {
	if w.decided {
		return
	}
	w.decided = true
	w.sliding = !isUnaryContentType(w.Header().Get("Content-Type"))
}

func (w *streamingResponseWriter) WriteHeader(statusCode int) {
	w.decide()
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *streamingResponseWriter) Write(b []byte) (int, error) {
	w.decide()
	if w.sliding {
		if err := http.NewResponseController(w.ResponseWriter).
			SetWriteDeadline(time.Now().Add(streamWriteTimeout)); err != nil {
			// HTTP/2 or a ResponseWriter wrapper without Unwrap → can't slide the
			// deadline; the stream falls back to the fixed WriteTimeout and will be
			// aborted. Warn (rate-limited) so the silent failure stays diagnosable.
			warnUnsupportedDeadline(err)
		}
	}
	return w.ResponseWriter.Write(b)
}

// warnUnsupportedDeadline logs at most once per unsupportedDeadlineWarnInterval.
// A lost CAS race just skips this round (a concurrent writer emitted it); the
// next interval re-arms, so a 100%-of-streams regression keeps reporting.
func warnUnsupportedDeadline(err error) {
	now := time.Now().UnixNano()
	last := lastUnsupportedDeadlineWarnNanos.Load()
	if now-last < int64(unsupportedDeadlineWarnInterval) {
		return
	}
	if !lastUnsupportedDeadlineWarnNanos.CompareAndSwap(last, now) {
		return
	}
	slog.Warn("streaming write deadline unsupported; long-lived streams may be aborted by WriteTimeout", "error", err)
}

// Flush satisfies http.Flusher (connect flushes each frame). Unwrap lets an
// outer http.ResponseController reach the underlying Flusher / deadline setter.
func (w *streamingResponseWriter) Flush() {
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

func (w *streamingResponseWriter) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}
