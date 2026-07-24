package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

func writeTestProfile(t *testing.T, root string, contents []byte) string {
	t.Helper()
	path := filepath.Join(root, profilePath)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, contents, 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func runCommand(t *testing.T, args ...string) (int, string) {
	t.Helper()
	var stderr bytes.Buffer
	return run(args, &stderr), stderr.String()
}

func TestMainSuccessPaths(t *testing.T) {
	t.Run("check current profile", func(t *testing.T) {
		root := t.TempDir()
		path := writeTestProfile(t, root, generateProfile())
		t.Setenv("BUILD_WORKSPACE_DIRECTORY", root)

		status, stderr := runCommand(t, "--check")
		if status != 0 || stderr != "" {
			t.Fatalf("run returned status %d and stderr %q, want success", status, stderr)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if want := generateProfile(); !bytes.Equal(got, want) {
			t.Fatal("check mode changed the current profile")
		}
	})

	t.Run("write profile", func(t *testing.T) {
		root := t.TempDir()
		path := filepath.Join(root, profilePath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("BUILD_WORKSPACE_DIRECTORY", root)

		status, stderr := runCommand(t)
		if status != 0 || stderr != "" {
			t.Fatalf("run returned status %d and stderr %q, want success", status, stderr)
		}

		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if want := generateProfile(); !bytes.Equal(got, want) {
			t.Fatal("write mode did not persist the generated profile")
		}
	})
}

func TestMainFailurePaths(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		prepare    func(*testing.T)
		wantStatus int
		wantStderr string
	}{
		{
			name: "workspace cannot be resolved",
			prepare: func(t *testing.T) {
				t.Setenv("BUILD_WORKSPACE_DIRECTORY", "")
				t.Setenv("TEST_SRCDIR", "")
				t.Chdir(t.TempDir())
			},
			wantStatus: 1,
			wantStderr: "cannot locate workspace profile",
		},
		{
			name: "profile cannot be read",
			args: []string{"--check"},
			prepare: func(t *testing.T) {
				root := t.TempDir()
				t.Setenv("BUILD_WORKSPACE_DIRECTORY", root)
			},
			wantStatus: 1,
			wantStderr: "no such file or directory",
		},
		{
			name: "profile is stale",
			args: []string{"--check"},
			prepare: func(t *testing.T) {
				root := t.TempDir()
				writeTestProfile(t, root, []byte("stale\n"))
				t.Setenv("BUILD_WORKSPACE_DIRECTORY", root)
			},
			wantStatus: 1,
			wantStderr: "is stale; run bazel run",
		},
		{
			name: "profile cannot be written",
			prepare: func(t *testing.T) {
				root := t.TempDir()
				t.Setenv("BUILD_WORKSPACE_DIRECTORY", root)
			},
			wantStatus: 1,
			wantStderr: "no such file or directory",
		},
		{
			name:       "invalid argument",
			args:       []string{"--unknown"},
			prepare:    func(*testing.T) {},
			wantStatus: 2,
			wantStderr: "flag provided but not defined",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.prepare(t)
			status, stderr := runCommand(t, test.args...)
			if status != test.wantStatus || !strings.Contains(stderr, test.wantStderr) {
				t.Fatalf("run returned status %d and stderr %q, want status %d containing %q", status, stderr, test.wantStatus, test.wantStderr)
			}
		})
	}
}

func TestResolveProfilePathSources(t *testing.T) {
	t.Run("workspace directory takes precedence", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("BUILD_WORKSPACE_DIRECTORY", root)
		t.Setenv("TEST_SRCDIR", filepath.Join(root, "ignored-runfiles"))
		got, err := resolveProfilePath()
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(root, profilePath); got != want {
			t.Fatalf("resolveProfilePath() = %q, want %q", got, want)
		}
	})

	t.Run("runfiles use default workspace", func(t *testing.T) {
		root := t.TempDir()
		t.Setenv("BUILD_WORKSPACE_DIRECTORY", "")
		t.Setenv("TEST_SRCDIR", root)
		t.Setenv("TEST_WORKSPACE", "")
		got, err := resolveProfilePath()
		if err != nil {
			t.Fatal(err)
		}
		if want := filepath.Join(root, "_main", profilePath); got != want {
			t.Fatalf("resolveProfilePath() = %q, want %q", got, want)
		}
	})

	t.Run("current directory walks to workspace", func(t *testing.T) {
		root := t.TempDir()
		path := writeTestProfile(t, root, []byte("profile"))
		nested := filepath.Join(root, "nested", "directory")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("BUILD_WORKSPACE_DIRECTORY", "")
		t.Setenv("TEST_SRCDIR", "")
		t.Chdir(nested)

		got, err := resolveProfilePath()
		if err != nil {
			t.Fatal(err)
		}
		if got != path {
			t.Fatalf("resolveProfilePath() = %q, want %q", got, path)
		}
	})
}

func TestGeneratedProfileIsCurrent(t *testing.T) {
	path, err := resolveProfilePath()
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := generateProfile(); !bytes.Equal(got, want) {
		t.Fatalf("%s is stale; run the runewidth_profile target", profilePath)
	}
}

func TestGeneratedIntervalsAreSortedAndDisjoint(t *testing.T) {
	condition := runewidth.NewCondition()
	condition.EastAsianWidth = false
	for _, width := range []int{0, 2} {
		intervals := collectIntervals(condition, width)
		for index, current := range intervals {
			if current.start > current.end {
				t.Fatalf("width %d interval %d is reversed: %#x..%#x", width, index, current.start, current.end)
			}
			if index > 0 && intervals[index-1].end >= current.start {
				t.Fatalf("width %d intervals %d and %d overlap or are unsorted", width, index-1, index)
			}
		}
	}
}

func TestCollectIntervalsClosesRangeAtMaxRune(t *testing.T) {
	condition := runewidth.NewCondition()
	condition.EastAsianWidth = false
	intervals := collectIntervals(condition, condition.RuneWidth(utf8.MaxRune))
	if len(intervals) == 0 {
		t.Fatal("collectIntervals returned no ranges for the maximum rune width")
	}
	if got := intervals[len(intervals)-1].end; got != utf8.MaxRune {
		t.Fatalf("last interval ends at %U, want %U", got, utf8.MaxRune)
	}
}

func TestPTYRenderFixtureRuneWidths(t *testing.T) {
	condition := runewidth.NewCondition()
	condition.EastAsianWidth = false
	cases := map[rune]int{
		'😀': 2,
		'🫠': 2,
		'☰': 2,
		'❤': 1,
		'界': 2,
		'·': 1,
		'́': 0,
		'️': 0,
	}
	for codepoint, want := range cases {
		if got := condition.RuneWidth(codepoint); got != want {
			t.Errorf("RuneWidth(%U) = %d, want %d", codepoint, got, want)
		}
	}
}
