package tokenusage

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// captureLogs returns a slog.Logger writing JSON records into the buffer
// so tests can assert on level + message + attribute combinations.
func captureLogs(t *testing.T) (*slog.Logger, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	h := slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(h), &buf
}

func TestLogEmptyUsage_ZeroPodStartedAtWarns(t *testing.T) {
	log, buf := captureLogs(t)
	logEmptyUsage(log, "codex", "/tmp/sandbox", time.Time{})

	out := buf.String()
	assert.Contains(t, out, `"level":"WARN"`, "zero podStartedAt must be Warn (caller bug signal)")
	assert.Contains(t, out, "caller passed zero podStartedAt")
	assert.NotContains(t, out, "pod_runtime_seconds", "no runtime should be logged when podStartedAt is zero")
}

func TestLogEmptyUsage_ShortPodIsDebug(t *testing.T) {
	log, buf := captureLogs(t)
	podStarted := time.Now().Add(-1 * time.Second)
	logEmptyUsage(log, "codex", "/tmp/sandbox", podStarted)

	out := buf.String()
	assert.Contains(t, out, `"level":"DEBUG"`, "pod < 5s with empty usage must be Debug (no signal)")
	assert.Contains(t, out, "Token usage empty after parse")
	assert.Contains(t, out, "pod_runtime_seconds")
}

func TestLogEmptyUsage_LongPodIsInfo(t *testing.T) {
	log, buf := captureLogs(t)
	podStarted := time.Now().Add(-30 * time.Second)
	logEmptyUsage(log, "codex", "/tmp/sandbox", podStarted)

	out := buf.String()
	assert.Contains(t, out, `"level":"INFO"`, "pod >= 5s with empty usage must be Info (suspicious)")
	assert.Contains(t, out, "Token usage empty after parse")
	assert.Contains(t, out, "pod_runtime_seconds")
}

func TestLogEmptyUsage_ContainsContextFields(t *testing.T) {
	log, buf := captureLogs(t)
	podStarted := time.Now().Add(-30 * time.Second)
	logEmptyUsage(log, "myagent", "/sandbox/path", podStarted)

	out := buf.String()
	assert.Contains(t, out, `"agent":"myagent"`)
	assert.Contains(t, out, `"sandbox_path":"/sandbox/path"`)
	// pod_runtime_seconds is a float — assert the field name, not exact value
	assert.True(t, strings.Contains(out, `"pod_runtime_seconds":`), "must include pod_runtime_seconds")
}
