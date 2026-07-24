package aggregator

import "testing"

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
