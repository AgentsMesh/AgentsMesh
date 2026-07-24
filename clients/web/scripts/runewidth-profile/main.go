package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

const (
	profilePath           = "clients/web/src/lib/terminal/runewidthUnicode17Ranges.ts"
	runewidthVersion      = "v0.0.21"
	runewidthUnicode      = "17"
	intervalsPerSourceRow = 4
)

type interval struct {
	start rune
	end   rune
}

func main() { os.Exit(run(os.Args[1:], os.Stderr)) }

func run(args []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("runewidth-profile", flag.ContinueOnError)
	flags.SetOutput(stderr)
	check := flags.Bool("check", false, "fail when the checked-in profile is stale")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	path, err := resolveProfilePath()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	want := generateProfile()
	if *check {
		got, readErr := os.ReadFile(path)
		if readErr != nil {
			fmt.Fprintln(stderr, readErr)
			return 1
		}
		if !bytes.Equal(got, want) {
			fmt.Fprintf(stderr, "%s is stale; run bazel run //clients/web/scripts/runewidth-profile:runewidth_profile\n", profilePath)
			return 1
		}
		return 0
	}
	if err := os.WriteFile(path, want, 0o644); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	return 0
}

func resolveProfilePath() (string, error) {
	if root := os.Getenv("BUILD_WORKSPACE_DIRECTORY"); root != "" {
		return filepath.Join(root, profilePath), nil
	}
	if runfiles := os.Getenv("TEST_SRCDIR"); runfiles != "" {
		workspace := os.Getenv("TEST_WORKSPACE")
		if workspace == "" {
			workspace = "_main"
		}
		return filepath.Join(runfiles, workspace, profilePath), nil
	}

	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, profilePath)
		if _, statErr := os.Stat(candidate); statErr == nil {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("cannot locate workspace profile %s", profilePath)
		}
		dir = parent
	}
}

func generateProfile() []byte {
	condition := runewidth.NewCondition()
	condition.EastAsianWidth = false

	zeroWidth := collectIntervals(condition, 0)
	doubleWidth := collectIntervals(condition, 2)

	var out bytes.Buffer
	fmt.Fprintf(&out, "// Code generated from github.com/mattn/go-runewidth %s (Unicode %s); DO NOT EDIT.\n", runewidthVersion, runewidthUnicode)
	out.WriteString("// Condition: EastAsianWidth=false; default width=1.\n\n")
	fmt.Fprintf(&out, "export const RUNEWIDTH_VERSION = %q;\n", runewidthVersion)
	fmt.Fprintf(&out, "export const RUNEWIDTH_UNICODE_VERSION = %q;\n\n", runewidthUnicode)
	writeIntervals(&out, "ZERO_WIDTH_RANGES", zeroWidth)
	out.WriteByte('\n')
	writeIntervals(&out, "DOUBLE_WIDTH_RANGES", doubleWidth)
	return out.Bytes()
}

func collectIntervals(condition *runewidth.Condition, width int) []interval {
	var result []interval
	inRange := false
	var start rune
	for codepoint := rune(0); codepoint <= utf8.MaxRune; codepoint++ {
		matches := condition.RuneWidth(codepoint) == width
		if matches && !inRange {
			start = codepoint
			inRange = true
		}
		if inRange && (!matches || codepoint == utf8.MaxRune) {
			end := codepoint - 1
			if matches {
				end = codepoint
			}
			result = append(result, interval{start: start, end: end})
			inRange = false
		}
	}
	return result
}

func writeIntervals(out *bytes.Buffer, name string, intervals []interval) {
	fmt.Fprintf(out, "export const %s = new Uint32Array([\n", name)
	for index, current := range intervals {
		if index%intervalsPerSourceRow == 0 {
			out.WriteString("  ")
		}
		fmt.Fprintf(out, "0x%X, 0x%X,", current.start, current.end)
		if index%intervalsPerSourceRow == intervalsPerSourceRow-1 || index == len(intervals)-1 {
			out.WriteByte('\n')
		} else {
			out.WriteByte(' ')
		}
	}
	out.WriteString("]);\n")
}
