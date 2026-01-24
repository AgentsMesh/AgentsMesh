package terminal

import (
	"bytes"
	"testing"
)

// Helper to build sync output frame (incremental - no clear screen)
func buildSyncFrame(content string) []byte {
	return append(append(syncOutputStartSeq, []byte(content)...), syncOutputEndSeq...)
}

// Helper to build full redraw frame (contains ESC[2J - triggers discard of previous frames)
func buildFullRedrawFrame(content string) []byte {
	// Full redraw frames contain ESC[2J (clear screen) followed by actual content
	frameContent := append(eraseScreenSeq, []byte(content)...)
	return append(append(syncOutputStartSeq, frameContent...), syncOutputEndSeq...)
}

// Helper to build large frame (>1KB - treated as full redraw)
func buildLargeFrame(content string) []byte {
	// Pad content to exceed 1KB threshold
	padding := make([]byte, 1025-len(content))
	for i := range padding {
		padding[i] = 'x'
	}
	fullContent := append([]byte(content), padding...)
	return append(append(syncOutputStartSeq, fullContent...), syncOutputEndSeq...)
}

func TestFrameDetector_AnalyzeFrameBoundaries_NoFrames(t *testing.T) {
	fd := NewFrameDetector()

	// Empty data
	result := fd.AnalyzeFrameBoundaries(nil)
	if result.HasSyncFrames || result.CompleteEnd != -1 || result.IncompleteStart != -1 {
		t.Error("Expected empty result for nil data")
	}

	// Plain text
	result = fd.AnalyzeFrameBoundaries([]byte("hello world"))
	if result.HasSyncFrames {
		t.Error("Should not detect sync frames in plain text")
	}
}

func TestFrameDetector_AnalyzeFrameBoundaries_SingleCompleteFrame(t *testing.T) {
	fd := NewFrameDetector()

	frame := buildSyncFrame("content")
	result := fd.AnalyzeFrameBoundaries(frame)

	if !result.HasSyncFrames {
		t.Error("Should detect sync frames")
	}
	if result.CompleteEnd != len(frame) {
		t.Errorf("Expected CompleteEnd=%d, got %d", len(frame), result.CompleteEnd)
	}
	if result.IncompleteStart != -1 {
		t.Errorf("Expected no incomplete frame, got IncompleteStart=%d", result.IncompleteStart)
	}
}

func TestFrameDetector_AnalyzeFrameBoundaries_MultipleCompleteFrames(t *testing.T) {
	fd := NewFrameDetector()

	frame1 := buildSyncFrame("frame1")
	frame2 := buildSyncFrame("frame2")
	data := append(frame1, frame2...)

	result := fd.AnalyzeFrameBoundaries(data)

	if !result.HasSyncFrames {
		t.Error("Should detect sync frames")
	}
	if result.CompleteEnd != len(data) {
		t.Errorf("Expected CompleteEnd=%d, got %d", len(data), result.CompleteEnd)
	}
	if result.IncompleteStart != -1 {
		t.Errorf("Expected no incomplete frame, got IncompleteStart=%d", result.IncompleteStart)
	}
}

func TestFrameDetector_AnalyzeFrameBoundaries_IncompleteFrame(t *testing.T) {
	fd := NewFrameDetector()

	// Frame start without end
	data := append(syncOutputStartSeq, []byte("incomplete content")...)
	result := fd.AnalyzeFrameBoundaries(data)

	if !result.HasSyncFrames {
		t.Error("Should detect sync frames")
	}
	if result.CompleteEnd != -1 {
		t.Errorf("Expected no complete frame, got CompleteEnd=%d", result.CompleteEnd)
	}
	if result.IncompleteStart != 0 {
		t.Errorf("Expected IncompleteStart=0, got %d", result.IncompleteStart)
	}
}

func TestFrameDetector_AnalyzeFrameBoundaries_CompleteAndIncomplete(t *testing.T) {
	fd := NewFrameDetector()

	// Complete frame followed by incomplete frame
	completeFrame := buildSyncFrame("complete")
	incompleteStart := append(syncOutputStartSeq, []byte("incomplete")...)
	data := append(completeFrame, incompleteStart...)

	result := fd.AnalyzeFrameBoundaries(data)

	if !result.HasSyncFrames {
		t.Error("Should detect sync frames")
	}
	if result.CompleteEnd != len(completeFrame) {
		t.Errorf("Expected CompleteEnd=%d, got %d", len(completeFrame), result.CompleteEnd)
	}
	if result.IncompleteStart != len(completeFrame) {
		t.Errorf("Expected IncompleteStart=%d, got %d", len(completeFrame), result.IncompleteStart)
	}
}

func TestFrameDetector_AnalyzeFrameBoundaries_OrphanEnd(t *testing.T) {
	fd := NewFrameDetector()

	// End sequence without matching start (orphan)
	data := append([]byte("prefix"), syncOutputEndSeq...)
	result := fd.AnalyzeFrameBoundaries(data)

	// Should detect end sequence exists
	if !result.HasSyncFrames {
		t.Error("Should detect sync frames (even orphan ends)")
	}
	// But no complete frame
	if result.CompleteEnd != -1 {
		t.Errorf("Should not find complete frame with orphan end, got CompleteEnd=%d", result.CompleteEnd)
	}
}

func TestFrameDetector_AnalyzeFrameBoundaries_ClearScreen(t *testing.T) {
	fd := NewFrameDetector()

	// Data with clear screen but no sync frames
	data := append([]byte("old content"), clearScreenSeq...)
	data = append(data, []byte("new content")...)

	result := fd.AnalyzeFrameBoundaries(data)

	if result.HasSyncFrames {
		t.Error("Should not detect sync frames")
	}
	expectedPos := len("old content")
	if result.ClearScreenPos != expectedPos {
		t.Errorf("Expected ClearScreenPos=%d, got %d", expectedPos, result.ClearScreenPos)
	}
}

func TestFrameDetector_DiscardOldFrames_MultipleCompleteFrames(t *testing.T) {
	fd := NewFrameDetector()

	// Content-aware discard: only discards when there's a full redraw frame
	// Small incremental frames are preserved
	frame1 := buildSyncFrame("old frame 1")
	frame2 := buildSyncFrame("old frame 2")
	frame3 := buildFullRedrawFrame("latest frame") // Full redraw frame triggers discard
	data := append(append(frame1, frame2...), frame3...)

	buffer := bytes.NewBuffer(data)
	discarded := fd.DiscardOldFrames(buffer)

	// Should discard first two frames because frame3 is a full redraw
	expectedDiscarded := len(frame1) + len(frame2)
	if discarded != expectedDiscarded {
		t.Errorf("Expected to discard %d bytes, discarded %d", expectedDiscarded, discarded)
	}

	// Buffer should contain only frame3
	if !bytes.Equal(buffer.Bytes(), frame3) {
		t.Errorf("Buffer should contain only latest frame, got %q", buffer.String())
	}
}

func TestFrameDetector_DiscardOldFrames_PreservesIncrementalFrames(t *testing.T) {
	fd := NewFrameDetector()

	// All incremental frames (no clear screen) - should ALL be preserved
	frame1 := buildSyncFrame("incremental 1")
	frame2 := buildSyncFrame("incremental 2")
	frame3 := buildSyncFrame("incremental 3")
	data := append(append(frame1, frame2...), frame3...)

	buffer := bytes.NewBuffer(data)
	discarded := fd.DiscardOldFrames(buffer)

	// Should NOT discard anything - all incremental frames preserved
	if discarded != 0 {
		t.Errorf("Should not discard incremental frames, discarded %d bytes", discarded)
	}

	// Buffer should contain all frames
	if !bytes.Equal(buffer.Bytes(), data) {
		t.Errorf("All incremental frames should be preserved")
	}
}

func TestFrameDetector_DiscardOldFrames_KeepsIncompleteFrame(t *testing.T) {
	fd := NewFrameDetector()

	// Use full redraw frame to trigger discard of older frames
	frame1 := buildSyncFrame("complete 1")
	frame2 := buildFullRedrawFrame("complete 2") // Full redraw triggers discard of frame1
	incomplete := append(syncOutputStartSeq, []byte("incomplete content")...)
	data := append(append(frame1, frame2...), incomplete...)

	buffer := bytes.NewBuffer(data)
	discarded := fd.DiscardOldFrames(buffer)

	// Should discard frame1 because frame2 is a full redraw
	expectedDiscarded := len(frame1)
	if discarded != expectedDiscarded {
		t.Errorf("Expected to discard %d bytes, discarded %d", expectedDiscarded, discarded)
	}

	// Buffer should contain frame2 + incomplete
	expected := append(frame2, incomplete...)
	if !bytes.Equal(buffer.Bytes(), expected) {
		t.Errorf("Buffer should contain last complete + incomplete frame\nExpected: %q\nGot: %q",
			expected, buffer.Bytes())
	}
}

func TestFrameDetector_DiscardOldFrames_OnlyIncomplete(t *testing.T) {
	fd := NewFrameDetector()

	incomplete := append(syncOutputStartSeq, []byte("only incomplete")...)
	buffer := bytes.NewBuffer(incomplete)

	discarded := fd.DiscardOldFrames(buffer)

	// Should not discard anything - incomplete frame is preserved
	if discarded != 0 {
		t.Errorf("Should not discard incomplete frame, discarded %d bytes", discarded)
	}
	if !bytes.Equal(buffer.Bytes(), incomplete) {
		t.Error("Incomplete frame should be preserved")
	}
}

func TestFrameDetector_DiscardOldFrames_ClearScreenFallback(t *testing.T) {
	fd := NewFrameDetector()

	data := append([]byte("old content"), clearScreenSeq...)
	data = append(data, []byte("new content")...)

	buffer := bytes.NewBuffer(data)
	discarded := fd.DiscardOldFrames(buffer)

	// Should discard content before clear screen
	expectedDiscarded := len("old content")
	if discarded != expectedDiscarded {
		t.Errorf("Expected to discard %d bytes, discarded %d", expectedDiscarded, discarded)
	}

	expected := append(clearScreenSeq, []byte("new content")...)
	if !bytes.Equal(buffer.Bytes(), expected) {
		t.Errorf("Expected %q, got %q", expected, buffer.Bytes())
	}
}

func TestFrameDetector_DiscardOldFrames_EmptyBuffer(t *testing.T) {
	fd := NewFrameDetector()
	buffer := &bytes.Buffer{}

	discarded := fd.DiscardOldFrames(buffer)

	if discarded != 0 {
		t.Errorf("Should not discard from empty buffer, discarded %d", discarded)
	}
}

func TestFrameDetector_DiscardOldFrames_PlainText(t *testing.T) {
	fd := NewFrameDetector()
	data := []byte("plain text without any frame markers")

	buffer := bytes.NewBuffer(data)
	discarded := fd.DiscardOldFrames(buffer)

	// Should not discard anything - no frame markers
	if discarded != 0 {
		t.Errorf("Should not discard plain text, discarded %d bytes", discarded)
	}
}

func TestFrameDetector_FindFlushBoundary_AllComplete(t *testing.T) {
	fd := NewFrameDetector()

	frame1 := buildSyncFrame("frame 1")
	frame2 := buildSyncFrame("frame 2")
	data := append(frame1, frame2...)

	flushEnd, keepFrom := fd.FindFlushBoundary(data)

	// All frames complete - flush everything
	if flushEnd != len(data) {
		t.Errorf("Expected flushEnd=%d, got %d", len(data), flushEnd)
	}
	if keepFrom != len(data) {
		t.Errorf("Expected keepFrom=%d, got %d", len(data), keepFrom)
	}
}

func TestFrameDetector_FindFlushBoundary_WithIncomplete(t *testing.T) {
	fd := NewFrameDetector()

	complete := buildSyncFrame("complete")
	incomplete := append(syncOutputStartSeq, []byte("incomplete")...)
	data := append(complete, incomplete...)

	flushEnd, keepFrom := fd.FindFlushBoundary(data)

	// Should flush up to incomplete frame start
	expectedFlush := len(complete)
	if flushEnd != expectedFlush {
		t.Errorf("Expected flushEnd=%d, got %d", expectedFlush, flushEnd)
	}
	if keepFrom != expectedFlush {
		t.Errorf("Expected keepFrom=%d, got %d", expectedFlush, keepFrom)
	}
}

func TestFrameDetector_FindFlushBoundary_OnlyIncomplete(t *testing.T) {
	fd := NewFrameDetector()

	incomplete := append(syncOutputStartSeq, []byte("incomplete")...)

	flushEnd, keepFrom := fd.FindFlushBoundary(incomplete)

	// Should not flush incomplete frame
	if flushEnd != 0 {
		t.Errorf("Expected flushEnd=0, got %d", flushEnd)
	}
	if keepFrom != 0 {
		t.Errorf("Expected keepFrom=0, got %d", keepFrom)
	}
}

func TestFrameDetector_FindFlushBoundary_NoSyncFrames(t *testing.T) {
	fd := NewFrameDetector()

	data := []byte("plain text")

	flushEnd, keepFrom := fd.FindFlushBoundary(data)

	// No sync frames - flush everything
	if flushEnd != len(data) {
		t.Errorf("Expected flushEnd=%d, got %d", len(data), flushEnd)
	}
	if keepFrom != len(data) {
		t.Errorf("Expected keepFrom=%d, got %d", len(data), keepFrom)
	}
}

func TestFindAllPositions(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		seq      []byte
		expected []int
	}{
		{
			name:     "no match",
			data:     []byte("hello world"),
			seq:      []byte("xyz"),
			expected: nil,
		},
		{
			name:     "single match",
			data:     []byte("hello world"),
			seq:      []byte("world"),
			expected: []int{6},
		},
		{
			name:     "multiple matches",
			data:     []byte("abcabcabc"),
			seq:      []byte("abc"),
			expected: []int{0, 3, 6},
		},
		{
			name:     "overlapping matches",
			data:     []byte("aaa"),
			seq:      []byte("aa"),
			expected: []int{0, 1},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := findAllPositions(tc.data, tc.seq)
			if len(result) != len(tc.expected) {
				t.Errorf("Expected %v, got %v", tc.expected, result)
				return
			}
			for i, pos := range result {
				if pos != tc.expected[i] {
					t.Errorf("Expected position[%d]=%d, got %d", i, tc.expected[i], pos)
				}
			}
		})
	}
}

// TestFrameDetector_RealWorldScenario simulates real Claude Code output patterns
func TestFrameDetector_RealWorldScenario(t *testing.T) {
	fd := NewFrameDetector()

	// Simulate: small incremental frames (animation/spinner) followed by a full redraw
	// In real Claude Code:
	// - Incremental frames: small updates like spinner animation, typing effects
	// - Full redraw frames: when UI layout changes (contains ESC[2J or is large)
	frame1 := buildSyncFrame("Tool: Search\nSearching...")          // Incremental
	frame2 := buildSyncFrame("Tool: Search\nSearching..")           // Incremental
	frame3 := buildFullRedrawFrame("Tool: Search\nFound 5 files\n") // Full redraw
	incomplete := append(syncOutputStartSeq, []byte("Tool: Read\nfile1.go contents:\npackage main")...)

	data := append(append(append(frame1, frame2...), frame3...), incomplete...)

	// DiscardOldFrames should discard frame1 and frame2 (before the full redraw frame3)
	buffer := bytes.NewBuffer(data)
	discarded := fd.DiscardOldFrames(buffer)

	expectedDiscarded := len(frame1) + len(frame2)
	if discarded != expectedDiscarded {
		t.Errorf("Expected to discard %d bytes, discarded %d", expectedDiscarded, discarded)
	}

	// FindFlushBoundary should only flush frame3, keep incomplete
	remaining := buffer.Bytes()
	flushEnd, _ := fd.FindFlushBoundary(remaining)

	expectedFlushEnd := len(frame3)
	if flushEnd != expectedFlushEnd {
		t.Errorf("Expected flushEnd=%d, got %d", expectedFlushEnd, flushEnd)
	}
}

// TestFrameDetector_RealWorldScenario_AllIncremental tests that all incremental frames are preserved
func TestFrameDetector_RealWorldScenario_AllIncremental(t *testing.T) {
	fd := NewFrameDetector()

	// All incremental frames - common during animations
	frame1 := buildSyncFrame("Spinner: /")
	frame2 := buildSyncFrame("Spinner: -")
	frame3 := buildSyncFrame("Spinner: \\")
	frame4 := buildSyncFrame("Spinner: |")

	data := append(append(append(frame1, frame2...), frame3...), frame4...)

	// All incremental frames should be preserved
	buffer := bytes.NewBuffer(data)
	discarded := fd.DiscardOldFrames(buffer)

	if discarded != 0 {
		t.Errorf("Should not discard incremental frames, discarded %d bytes", discarded)
	}

	// FindFlushBoundary should flush all frames (all complete)
	flushEnd, _ := fd.FindFlushBoundary(buffer.Bytes())
	if flushEnd != len(data) {
		t.Errorf("Expected to flush all data, flushEnd=%d, data len=%d", flushEnd, len(data))
	}
}

// TestFrameDetector_FrameCountPreservation tests that frame pairs are preserved correctly
func TestFrameDetector_FrameCountPreservation(t *testing.T) {
	fd := NewFrameDetector()

	// Build data: 9 incremental frames + 1 full redraw frame at the end
	// Only the last full redraw should trigger discarding of previous frames
	var data []byte
	incrementalCount := 9
	for i := 0; i < incrementalCount; i++ {
		data = append(data, buildSyncFrame("incremental frame")...)
	}
	// Add a full redraw frame at the end
	fullRedrawFrame := buildFullRedrawFrame("full redraw content")
	data = append(data, fullRedrawFrame...)

	// Count starts and ends in original data
	originalStarts := len(findAllPositions(data, syncOutputStartSeq))
	originalEnds := len(findAllPositions(data, syncOutputEndSeq))
	totalFrames := incrementalCount + 1

	if originalStarts != totalFrames || originalEnds != totalFrames {
		t.Fatalf("Original data should have %d starts and ends, got %d/%d",
			totalFrames, originalStarts, originalEnds)
	}

	// After DiscardOldFrames, should keep only the last full redraw frame
	buffer := bytes.NewBuffer(data)
	fd.DiscardOldFrames(buffer)

	remaining := buffer.Bytes()
	remainingStarts := len(findAllPositions(remaining, syncOutputStartSeq))
	remainingEnds := len(findAllPositions(remaining, syncOutputEndSeq))

	// Should have 1 complete frame (the full redraw frame)
	if remainingStarts != 1 {
		t.Errorf("Expected 1 start sequence remaining, got %d", remainingStarts)
	}
	if remainingEnds != 1 {
		t.Errorf("Expected 1 end sequence remaining, got %d", remainingEnds)
	}

	// Starts and ends should match
	if remainingStarts != remainingEnds {
		t.Errorf("Frame starts (%d) and ends (%d) should match", remainingStarts, remainingEnds)
	}
}

// TestFrameDetector_AllIncrementalFramesPreserved tests that all incremental frames are kept
func TestFrameDetector_AllIncrementalFramesPreserved(t *testing.T) {
	fd := NewFrameDetector()

	// Build data with only incremental frames - all should be preserved
	var data []byte
	frameCount := 10
	for i := 0; i < frameCount; i++ {
		data = append(data, buildSyncFrame("incremental frame")...)
	}

	buffer := bytes.NewBuffer(data)
	discarded := fd.DiscardOldFrames(buffer)

	// No frames should be discarded
	if discarded != 0 {
		t.Errorf("Should not discard incremental frames, discarded %d bytes", discarded)
	}

	remaining := buffer.Bytes()
	remainingStarts := len(findAllPositions(remaining, syncOutputStartSeq))
	remainingEnds := len(findAllPositions(remaining, syncOutputEndSeq))

	// All frames should be preserved
	if remainingStarts != frameCount {
		t.Errorf("Expected %d start sequences, got %d", frameCount, remainingStarts)
	}
	if remainingEnds != frameCount {
		t.Errorf("Expected %d end sequences, got %d", frameCount, remainingEnds)
	}
}

// TestFrameDetector_DiscardWithSyncFrames_OnlyComplete tests discardWithSyncFrames edge cases
func TestFrameDetector_DiscardWithSyncFrames_OnlyComplete(t *testing.T) {
	fd := NewFrameDetector()

	// Single complete frame - should not discard
	frame := buildSyncFrame("content")
	buffer := bytes.NewBuffer(frame)
	discarded := fd.DiscardOldFrames(buffer)

	if discarded != 0 {
		t.Errorf("Should not discard single complete frame, discarded %d", discarded)
	}
}

// TestFrameDetector_FindFlushBoundary_Empty tests empty data
func TestFrameDetector_FindFlushBoundary_Empty(t *testing.T) {
	fd := NewFrameDetector()

	flushEnd, keepFrom := fd.FindFlushBoundary(nil)
	if flushEnd != 0 || keepFrom != 0 {
		t.Errorf("Empty data should return 0,0, got %d,%d", flushEnd, keepFrom)
	}

	flushEnd, keepFrom = fd.FindFlushBoundary([]byte{})
	if flushEnd != 0 || keepFrom != 0 {
		t.Errorf("Empty slice should return 0,0, got %d,%d", flushEnd, keepFrom)
	}
}

// TestFrameDetector_AnalyzeFrameBoundaries_OnlyEnds tests data with only end sequences
func TestFrameDetector_AnalyzeFrameBoundaries_OnlyEnds(t *testing.T) {
	fd := NewFrameDetector()

	// Only end sequences (orphans)
	data := append(syncOutputEndSeq, syncOutputEndSeq...)
	result := fd.AnalyzeFrameBoundaries(data)

	if !result.HasSyncFrames {
		t.Error("Should detect sync frames (even ends only)")
	}
	if result.CompleteEnd != -1 {
		t.Errorf("Should not find complete frame with ends only, got %d", result.CompleteEnd)
	}
	if result.IncompleteStart != -1 {
		t.Errorf("Should not find incomplete start with ends only, got %d", result.IncompleteStart)
	}
}

// TestFrameDetector_DiscardOldFrames_SingleFrame tests single frame scenarios
func TestFrameDetector_DiscardOldFrames_SingleFrame(t *testing.T) {
	fd := NewFrameDetector()

	// Single complete frame at position 0
	frame := buildSyncFrame("single")
	buffer := bytes.NewBuffer(frame)
	discarded := fd.DiscardOldFrames(buffer)

	// Should not discard - it's the only frame
	if discarded != 0 {
		t.Errorf("Should not discard only frame, discarded %d", discarded)
	}
	if !bytes.Equal(buffer.Bytes(), frame) {
		t.Error("Frame should be unchanged")
	}
}

// TestFrameDetector_NestedFrames tests handling of nested/overlapping frames
func TestFrameDetector_NestedFrames(t *testing.T) {
	fd := NewFrameDetector()

	// Unusual case: start, start, end, end
	// This shouldn't happen in practice but test robustness
	data := append(syncOutputStartSeq, syncOutputStartSeq...)
	data = append(data, []byte("content")...)
	data = append(data, syncOutputEndSeq...)
	data = append(data, syncOutputEndSeq...)

	result := fd.AnalyzeFrameBoundaries(data)

	// Should handle without panic
	if !result.HasSyncFrames {
		t.Error("Should detect sync frames")
	}
}

// TestFrameDetector_MixedContent tests mixed content scenarios
func TestFrameDetector_MixedContent(t *testing.T) {
	fd := NewFrameDetector()

	// Text before a full redraw frame - prefix should be discarded
	data := []byte("prefix text")
	data = append(data, buildFullRedrawFrame("frame content")...)
	data = append(data, []byte("suffix text")...)

	buffer := bytes.NewBuffer(data)
	discarded := fd.DiscardOldFrames(buffer)

	// Should discard prefix because there's a full redraw frame
	if discarded != len("prefix text") {
		t.Errorf("Expected to discard prefix, discarded %d bytes", discarded)
	}
}

// TestFrameDetector_MixedContent_IncrementalFrame tests that prefix is preserved with incremental frames
func TestFrameDetector_MixedContent_IncrementalFrame(t *testing.T) {
	fd := NewFrameDetector()

	// Text before an incremental frame - nothing should be discarded
	data := []byte("prefix text")
	data = append(data, buildSyncFrame("frame content")...)
	data = append(data, []byte("suffix text")...)

	buffer := bytes.NewBuffer(data)
	discarded := fd.DiscardOldFrames(buffer)

	// Should NOT discard anything - incremental frames don't trigger discard
	if discarded != 0 {
		t.Errorf("Expected not to discard with incremental frame, discarded %d bytes", discarded)
	}
}

// TestFrameDetector_DiscardOldFrames_NoStartPositions tests edge case with no starts
func TestFrameDetector_DiscardOldFrames_NoStartPositions(t *testing.T) {
	fd := NewFrameDetector()

	// Only clear screen, no sync frames
	data := append(clearScreenSeq, []byte("content")...)
	buffer := bytes.NewBuffer(data)

	// Should use clear screen fallback but position is 0, so nothing to discard
	discarded := fd.DiscardOldFrames(buffer)
	if discarded != 0 {
		t.Errorf("Should not discard when clear screen is at position 0, discarded %d", discarded)
	}
}

// ============================================================================
// StripRedundantSequencesInFrames Tests
// ============================================================================

// TestFrameDetector_StripRedundantSequences_NoSyncFrames tests that non-sync data is unchanged
func TestFrameDetector_StripRedundantSequences_NoSyncFrames(t *testing.T) {
	fd := NewFrameDetector()

	// Plain text with clear screen - should NOT be stripped (outside sync frame)
	data := append([]byte("before"), eraseScreenSeq...)
	data = append(data, cursorHomeSeq...)
	data = append(data, []byte("after")...)

	result := fd.StripRedundantSequencesInFrames(data)

	// Should be unchanged - no sync frames present
	if !bytes.Equal(result, data) {
		t.Errorf("Should not modify data without sync frames\nOriginal: %q\nResult: %q", data, result)
	}
}

// TestFrameDetector_StripRedundantSequences_InSyncFrame tests stripping inside sync frame
func TestFrameDetector_StripRedundantSequences_InSyncFrame(t *testing.T) {
	fd := NewFrameDetector()

	// Sync frame containing ESC[2J and ESC[H
	frameContent := append(eraseScreenSeq, cursorHomeSeq...)
	frameContent = append(frameContent, []byte("real content")...)
	frame := buildSyncFrame(string(frameContent))

	result := fd.StripRedundantSequencesInFrames(frame)

	// ESC[2J and ESC[H should be stripped from inside frame
	if bytes.Contains(result, eraseScreenSeq) {
		t.Error("ESC[2J should be stripped from sync frame")
	}
	if bytes.Contains(result, cursorHomeSeq) {
		t.Error("ESC[H should be stripped from sync frame")
	}

	// Frame boundaries should be preserved
	if !bytes.Contains(result, syncOutputStartSeq) {
		t.Error("Sync frame start should be preserved")
	}
	if !bytes.Contains(result, syncOutputEndSeq) {
		t.Error("Sync frame end should be preserved")
	}

	// Real content should be preserved
	if !bytes.Contains(result, []byte("real content")) {
		t.Error("Real content should be preserved")
	}
}

// TestFrameDetector_StripRedundantSequences_MultipleFrames tests multiple frames
func TestFrameDetector_StripRedundantSequences_MultipleFrames(t *testing.T) {
	fd := NewFrameDetector()

	// Frame 1: with clear screen
	frame1Content := append(eraseScreenSeq, []byte("frame1")...)
	frame1 := buildSyncFrame(string(frame1Content))

	// Frame 2: with cursor home
	frame2Content := append(cursorHomeSeq, []byte("frame2")...)
	frame2 := buildSyncFrame(string(frame2Content))

	data := append(frame1, frame2...)
	result := fd.StripRedundantSequencesInFrames(data)

	// Both sequences should be stripped
	if bytes.Contains(result, eraseScreenSeq) {
		t.Error("ESC[2J should be stripped")
	}
	if bytes.Contains(result, cursorHomeSeq) {
		t.Error("ESC[H should be stripped")
	}

	// Content should be preserved
	if !bytes.Contains(result, []byte("frame1")) || !bytes.Contains(result, []byte("frame2")) {
		t.Error("Frame contents should be preserved")
	}

	// Should have 2 complete frames
	starts := bytes.Count(result, syncOutputStartSeq)
	ends := bytes.Count(result, syncOutputEndSeq)
	if starts != 2 || ends != 2 {
		t.Errorf("Expected 2 starts and 2 ends, got %d/%d", starts, ends)
	}
}

// TestFrameDetector_StripRedundantSequences_MixedContext tests sync and non-sync mixed
func TestFrameDetector_StripRedundantSequences_MixedContext(t *testing.T) {
	fd := NewFrameDetector()

	// Clear screen outside frame (should be kept)
	before := append(eraseScreenSeq, []byte("before frame")...)

	// Frame with clear screen (should be stripped)
	frameContent := append(eraseScreenSeq, []byte("inside frame")...)
	frame := buildSyncFrame(string(frameContent))

	// Clear screen outside frame (should be kept)
	after := append(eraseScreenSeq, []byte("after frame")...)

	data := append(append(before, frame...), after...)
	result := fd.StripRedundantSequencesInFrames(data)

	// Count ESC[2J - should be 2 (before and after, not inside)
	count := bytes.Count(result, eraseScreenSeq)
	if count != 2 {
		t.Errorf("Expected 2 ESC[2J sequences (outside frames), got %d", count)
	}

	// Verify content
	if !bytes.Contains(result, []byte("before frame")) {
		t.Error("Before frame content should be preserved")
	}
	if !bytes.Contains(result, []byte("inside frame")) {
		t.Error("Inside frame content should be preserved")
	}
	if !bytes.Contains(result, []byte("after frame")) {
		t.Error("After frame content should be preserved")
	}
}

// TestFrameDetector_StripRedundantSequences_IncompleteFrame tests incomplete frame handling
func TestFrameDetector_StripRedundantSequences_IncompleteFrame(t *testing.T) {
	fd := NewFrameDetector()

	// Incomplete frame (no end sequence)
	incomplete := append(syncOutputStartSeq, eraseScreenSeq...)
	incomplete = append(incomplete, []byte("incomplete content")...)

	result := fd.StripRedundantSequencesInFrames(incomplete)

	// ESC[2J should be stripped even in incomplete frame
	if bytes.Contains(result, eraseScreenSeq) {
		t.Error("ESC[2J should be stripped from incomplete frame")
	}

	// Content should be preserved
	if !bytes.Contains(result, []byte("incomplete content")) {
		t.Error("Content should be preserved")
	}
}

// TestFrameDetector_StripRedundantSequences_CursorHomeVariant tests ESC[;H variant
func TestFrameDetector_StripRedundantSequences_CursorHomeVariant(t *testing.T) {
	fd := NewFrameDetector()

	// Frame with ESC[;H variant
	frameContent := append(cursorHomeSeq2, []byte("content")...)
	frame := buildSyncFrame(string(frameContent))

	result := fd.StripRedundantSequencesInFrames(frame)

	// ESC[;H should be stripped
	if bytes.Contains(result, cursorHomeSeq2) {
		t.Error("ESC[;H should be stripped from sync frame")
	}
}

// TestFrameDetector_StripRedundantSequences_Empty tests empty data
func TestFrameDetector_StripRedundantSequences_Empty(t *testing.T) {
	fd := NewFrameDetector()

	result := fd.StripRedundantSequencesInFrames(nil)
	if result != nil {
		t.Error("nil input should return nil")
	}

	result = fd.StripRedundantSequencesInFrames([]byte{})
	if len(result) != 0 {
		t.Error("empty input should return empty")
	}
}

// TestFrameDetector_StripRedundantSequences_NoRedundantSeqs tests frame without redundant seqs
func TestFrameDetector_StripRedundantSequences_NoRedundantSeqs(t *testing.T) {
	fd := NewFrameDetector()

	// Frame without any clear screen or cursor home
	frame := buildSyncFrame("just normal content")
	original := make([]byte, len(frame))
	copy(original, frame)

	result := fd.StripRedundantSequencesInFrames(frame)

	// Should return the same data (no modification needed)
	if !bytes.Equal(result, original) {
		t.Error("Frame without redundant sequences should be unchanged")
	}
}

// TestFrameDetector_StripRedundantSequences_RealWorldResize simulates post-resize behavior
func TestFrameDetector_StripRedundantSequences_RealWorldResize(t *testing.T) {
	fd := NewFrameDetector()

	// Simulate Claude Code post-resize: every frame starts with ESC[2J ESC[H
	var data []byte
	for i := 0; i < 5; i++ {
		frameContent := append(eraseScreenSeq, cursorHomeSeq...)
		frameContent = append(frameContent, []byte("frame content with lots of text...")...)
		frame := buildSyncFrame(string(frameContent))
		data = append(data, frame...)
	}

	originalLen := len(data)
	result := fd.StripRedundantSequencesInFrames(data)

	// Should be significantly smaller (stripped 5 x (4 + 3) = 35 bytes)
	expectedStripped := 5 * (len(eraseScreenSeq) + len(cursorHomeSeq))
	actualStripped := originalLen - len(result)

	if actualStripped != expectedStripped {
		t.Errorf("Expected to strip %d bytes, stripped %d", expectedStripped, actualStripped)
	}

	// Should have no ESC[2J or ESC[H
	if bytes.Contains(result, eraseScreenSeq) || bytes.Contains(result, cursorHomeSeq) {
		t.Error("All redundant sequences should be stripped")
	}

	// Should have 5 complete frames
	if bytes.Count(result, syncOutputStartSeq) != 5 || bytes.Count(result, syncOutputEndSeq) != 5 {
		t.Error("All 5 frames should be preserved")
	}
}
