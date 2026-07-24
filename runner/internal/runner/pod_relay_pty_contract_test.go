package runner

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/anthropics/agentsmesh/runner/internal/terminal/vt"
)

func TestPTYSnapshotCurrentFixturesMatchProducerShape(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		resetAll bool
	}{
		{name: "current_requester", content: "requester-screen", resetAll: false},
		{name: "current_recovery", content: "recovery-screen", resetAll: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			terminal := vt.NewVirtualTerminal(20, 4, 100)
			terminal.Feed([]byte(test.content))
			podRelay := NewPTYPodRelay(
				"pod-1", nil, &PTYComponents{VirtualTerminal: terminal}, nil,
			)
			actual := decodeJSONContract(t, podRelay.materializeSnapshotEnvelope(test.resetAll))
			fixture := readSnapshotFixture(t, test.name)
			if !reflect.DeepEqual(actual, fixture) {
				t.Fatalf("snapshot contract mismatch\nactual:  %#v\nfixture: %#v", actual, fixture)
			}
		})
	}
}

func TestPTYSnapshotAltStateMatchesSharedFixture(t *testing.T) {
	terminal := vt.NewVirtualTerminal(20, 4, 100)
	terminal.Feed([]byte("normal-screen"))
	terminal.Feed([]byte("\x1b[?1049halt-screen\x1b[?25l\x1b[?2004h\x1b[31"))
	podRelay := NewPTYPodRelay("pod-1", nil, &PTYComponents{VirtualTerminal: terminal}, nil)
	actual := decodeJSONContract(t, podRelay.materializeSnapshot())
	fixture := readSnapshotFixture(t, "alt_terminal_state")

	if !reflect.DeepEqual(actual, fixture) {
		t.Fatalf("alternate snapshot contract mismatch\nactual:  %#v\nfixture: %#v", actual, fixture)
	}
}

func TestPTYSnapshotLegacyFixtureIntentionallyOmitsScope(t *testing.T) {
	legacy := readSnapshotFixture(t, "legacy_missing_scope")
	if _, exists := legacy["reset_all"]; exists {
		t.Fatal("legacy fixture unexpectedly carries reset_all")
	}
	if _, exists := legacy["snapshot_version"]; exists {
		t.Fatal("legacy fixture unexpectedly carries snapshot_version")
	}
	if _, exists := legacy["serialized_active_content"]; exists {
		t.Fatal("legacy fixture unexpectedly carries serialized_active_content")
	}
}

func readSnapshotFixture(t *testing.T, name string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(snapshotFixturePath(t, name))
	if err != nil {
		t.Fatal(err)
	}
	return decodeJSONContract(t, data)
}

func snapshotFixturePath(t *testing.T, name string) string {
	t.Helper()
	relative := filepath.Join("proto", "testdata", "terminal_snapshot", name+".json")
	if srcdir := os.Getenv("TEST_SRCDIR"); srcdir != "" {
		workspace := os.Getenv("TEST_WORKSPACE")
		if workspace == "" {
			workspace = "_main"
		}
		return filepath.Join(srcdir, workspace, relative)
	}

	directory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, statErr := os.Stat(filepath.Join(directory, "MODULE.bazel")); statErr == nil {
			return filepath.Join(directory, relative)
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			t.Fatal("could not locate repository root for terminal snapshot fixtures")
		}
		directory = parent
	}
}

func decodeJSONContract(t *testing.T, data []byte) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}
