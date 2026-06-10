package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// recordingResponseWriter records write deadlines, body, and flushes. It
// implements SetWriteDeadline + Flush so http.NewResponseController reaches them.
type recordingResponseWriter struct {
	header     http.Header
	body       []byte
	deadlines  []time.Time
	flushCount int
}

func (w *recordingResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}
func (w *recordingResponseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return len(b), nil
}
func (w *recordingResponseWriter) WriteHeader(int) {}
func (w *recordingResponseWriter) SetWriteDeadline(t time.Time) error {
	w.deadlines = append(w.deadlines, t)
	return nil
}
func (w *recordingResponseWriter) Flush() { w.flushCount++ }

// Each frame write must slide the deadline forward — this is what keeps a
// healthy long-lived stream from being killed by a fixed WriteTimeout.
func TestStreamingResponseWriterSlidesDeadlinePerWrite(t *testing.T) {
	rec := &recordingResponseWriter{}
	rec.Header().Set("Content-Type", "application/connect+proto") // streaming framing
	w := newStreamingResponseWriter(rec)
	before := time.Now()

	n, err := w.Write([]byte("frame-1"))
	if err != nil || n != 7 {
		t.Fatalf("Write = (%d,%v), want (7,nil)", n, err)
	}
	if string(rec.body) != "frame-1" {
		t.Fatalf("body = %q, want frame-1", rec.body)
	}
	if len(rec.deadlines) != 1 {
		t.Fatalf("got %d deadline updates, want 1", len(rec.deadlines))
	}
	if min := before.Add(streamWriteTimeout - time.Second); rec.deadlines[0].Before(min) {
		t.Fatalf("deadline %v not slid ~%v forward", rec.deadlines[0], streamWriteTimeout)
	}

	w.Write([]byte("frame-2")) //nolint:errcheck // forwarding verified above
	if len(rec.deadlines) != 2 {
		t.Fatalf("second write did not refresh deadline: %d updates", len(rec.deadlines))
	}
	if !rec.deadlines[1].After(rec.deadlines[0]) {
		t.Fatalf("second deadline %v not after first %v", rec.deadlines[1], rec.deadlines[0])
	}
}

// A unary response (application/proto) keeps the server's tighter WriteTimeout —
// the wrapper must NOT slide a deadline for it. Detected from Content-Type.
func TestStreamingResponseWriterSkipsDeadlineForUnary(t *testing.T) {
	rec := &recordingResponseWriter{}
	rec.Header().Set("Content-Type", "application/proto") // connect unary
	w := newStreamingResponseWriter(rec)

	w.Write([]byte("unary-body")) //nolint:errcheck // forwarding asserted below
	if len(rec.deadlines) != 0 {
		t.Fatalf("unary response set %d write deadline(s), want 0 (keeps server WriteTimeout)", len(rec.deadlines))
	}
	if string(rec.body) != "unary-body" {
		t.Fatalf("body = %q, want unary-body", rec.body)
	}
}

// isUnaryContentType must recognize connect unary framings and reject every
// streaming framing — incl. unknown/empty, which defaults to streaming so a
// stream is never wrongly left under the short WriteTimeout.
func TestIsUnaryContentType(t *testing.T) {
	unary := []string{"application/proto", "application/json", "application/json; charset=utf-8"}
	for _, ct := range unary {
		if !isUnaryContentType(ct) {
			t.Errorf("isUnaryContentType(%q) = false, want true", ct)
		}
	}
	streaming := []string{
		"application/connect+proto", "application/connect+json",
		"application/grpc", "application/grpc+proto", "application/grpc-web", "",
	}
	for _, ct := range streaming {
		if isUnaryContentType(ct) {
			t.Errorf("isUnaryContentType(%q) = true, want false (streaming/unknown)", ct)
		}
	}
}

// streamWriteTimeout must exceed the keepalive interval, else a healthy stream
// writing one keepalive per interval would still be killed between writes.
func TestStreamWriteTimeoutExceedsKeepalive(t *testing.T) {
	const keepaliveInterval = 25 * time.Second // mirrors events/sentinel_frames.go
	if streamWriteTimeout <= keepaliveInterval {
		t.Fatalf("streamWriteTimeout %v must exceed keepalive %v", streamWriteTimeout, keepaliveInterval)
	}
}

func TestStreamingResponseWriterFlushForwards(t *testing.T) {
	rec := &recordingResponseWriter{}
	f, ok := newStreamingResponseWriter(rec).(http.Flusher)
	if !ok {
		t.Fatal("streamingResponseWriter must satisfy http.Flusher (connect flushes each frame)")
	}
	f.Flush()
	if rec.flushCount != 1 {
		t.Fatalf("flushCount = %d, want 1", rec.flushCount)
	}
}

func TestStreamingResponseWriterUnwrap(t *testing.T) {
	rec := &recordingResponseWriter{}
	w := &streamingResponseWriter{ResponseWriter: rec}
	if w.Unwrap() != rec {
		t.Fatal("Unwrap must return the wrapped ResponseWriter")
	}
}

// A ResponseWriter without SetWriteDeadline (HTTP/2, opaque wrapper) must not
// panic — Write still forwards; the deadline refresh degrades to a no-op.
type plainResponseWriter struct {
	header http.Header
	body   []byte
}

func (w *plainResponseWriter) Header() http.Header {
	if w.header == nil {
		w.header = http.Header{}
	}
	return w.header
}
func (w *plainResponseWriter) Write(b []byte) (int, error) {
	w.body = append(w.body, b...)
	return len(b), nil
}
func (w *plainResponseWriter) WriteHeader(int) {}

func TestStreamingResponseWriterUnsupportedDeadlineStillWrites(t *testing.T) {
	rec := &plainResponseWriter{}
	rec.Header().Set("Content-Type", "application/connect+proto") // streaming → attempts SetWriteDeadline
	w := newStreamingResponseWriter(rec)
	n, err := w.Write([]byte("x"))
	if err != nil || n != 1 {
		t.Fatalf("Write = (%d,%v), want (1,nil)", n, err)
	}
	if string(rec.body) != "x" {
		t.Fatalf("body = %q, want x", rec.body)
	}
}

// routeConnectOrREST must wrap EVERY Connect response in the sliding-deadline
// writer (so any server-stream — current or future — survives the server
// WriteTimeout) and leave REST responses untouched. No per-procedure allowlist
// means a newly-added stream is protected with no extra registration.
func TestRouteConnectOrRESTWrapsConnectResponses(t *testing.T) {
	var connectW, restW http.ResponseWriter
	connectH := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { connectW = w })
	restH := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { restW = w })
	h := routeConnectOrREST(connectH, restH)

	h.ServeHTTP(&recordingResponseWriter{}, httptest.NewRequest(http.MethodPost, "/proto.events.v1.EventsService/Subscribe", nil))
	if _, ok := connectW.(*streamingResponseWriter); !ok {
		t.Fatalf("connect path writer = %T, want *streamingResponseWriter", connectW)
	}
	if restW != nil {
		t.Fatal("connect path must not reach the REST handler")
	}

	rec := &recordingResponseWriter{}
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/v1/health", nil))
	if restW != http.ResponseWriter(rec) {
		t.Fatalf("REST path writer = %v, want the raw (unwrapped) writer", restW)
	}
}

// End-to-end over a real http.Server with a short WriteTimeout: a streaming
// handler that writes well past the timeout must survive because each write
// slides the deadline. Without the wrapper the server would reset the
// connection mid-stream and the client read would fail / truncate.
func TestStreamingResponseWriterSurvivesServerWriteTimeout(t *testing.T) {
	const (
		writeTimeout = 300 * time.Millisecond // widened from a tight 150ms so a
		// loaded CI runner's scheduling delay before the first frame can't trip
		// the server WriteTimeout ahead of the wrapper's first deadline slide.
		frames   = 6
		frameGap = 100 * time.Millisecond // total ~600ms ≫ writeTimeout
	)

	handler := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/connect+proto") // streaming framing
		sw := newStreamingResponseWriter(w)
		for i := 0; i < frames; i++ {
			if _, err := sw.Write([]byte("frame\n")); err != nil {
				return
			}
			if f, ok := sw.(http.Flusher); ok {
				f.Flush()
			}
			time.Sleep(frameGap)
		}
	})

	srv := httptest.NewUnstartedServer(handler)
	srv.Config.WriteTimeout = writeTimeout
	srv.Start()
	defer srv.Close()

	resp, err := http.Get(srv.URL) //nolint:noctx // test client
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body (stream killed by WriteTimeout?): %v", err)
	}
	if got := strings.Count(string(body), "frame"); got != frames {
		t.Fatalf("received %d frames, want %d — sliding deadline did not outlive WriteTimeout", got, frames)
	}
}
