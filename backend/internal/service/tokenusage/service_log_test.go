package tokenusage

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	runnerv1 "github.com/anthropics/agentsmesh/proto/gen/go/runner/v1"
	"github.com/stretchr/testify/assert"
)

func captureSvcLogs(t *testing.T) (*Service, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	return &Service{logger: slog.New(h)}, &buf
}

func reportWithStarted(secs int64, models int) *runnerv1.TokenUsageReport {
	r := &runnerv1.TokenUsageReport{PodStartedAtUnixSeconds: secs}
	for i := 0; i < models; i++ {
		r.Models = append(r.Models, &runnerv1.TokenModelUsage{Model: "m"})
	}
	return r
}

// PodStartedAtUnixSeconds == 0 must route to "legacy runner" Info, not be
// mistaken for a freshly-started pod.
func TestLogEmptyAfterFilter_ZeroIsLegacyInfo(t *testing.T) {
	svc, buf := captureSvcLogs(t)
	svc.logEmptyAfterFilter("pod-1", "codex", reportWithStarted(0, 1))
	out := buf.String()
	assert.Contains(t, out, `"level":"INFO"`)
	assert.Contains(t, out, "no pod_started_at")
	assert.NotContains(t, out, "pod_runtime_seconds")
}

// time.Time{}.Unix() returns -62135596800 from a buggy old Go runner; the
// backend must treat it as legacy, NOT compute a 64-billion-second runtime
// and emit a misleading long-pod Warn.
func TestLogEmptyAfterFilter_NegativeIsLegacyInfo(t *testing.T) {
	svc, buf := captureSvcLogs(t)
	svc.logEmptyAfterFilter("pod-2", "codex", reportWithStarted(-62135596800, 1))
	out := buf.String()
	assert.Contains(t, out, `"level":"INFO"`)
	assert.Contains(t, out, "no pod_started_at")
	assert.NotContains(t, out, `"level":"WARN"`)
}

func TestLogEmptyAfterFilter_ShortPodIsDebug(t *testing.T) {
	svc, buf := captureSvcLogs(t)
	svc.logEmptyAfterFilter("pod-3", "codex", reportWithStarted(time.Now().Add(-1*time.Second).Unix(), 1))
	out := buf.String()
	assert.Contains(t, out, `"level":"DEBUG"`)
	assert.Contains(t, out, "after filter")
}

func TestLogEmptyAfterFilter_LongPodIsWarn(t *testing.T) {
	svc, buf := captureSvcLogs(t)
	svc.logEmptyAfterFilter("pod-4", "codex", reportWithStarted(time.Now().Add(-30*time.Second).Unix(), 2))
	out := buf.String()
	assert.Contains(t, out, `"level":"WARN"`)
	assert.Contains(t, out, "after filter")
	assert.True(t, strings.Contains(out, `"pod_runtime_seconds":`))
}
