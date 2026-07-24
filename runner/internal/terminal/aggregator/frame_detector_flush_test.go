package aggregator

import "testing"

func TestFrameDetectorFindFlushBoundary(t *testing.T) {
	complete := buildSyncFrame("complete")
	incomplete := append(append([]byte(nil), syncOutputStartSeq...), []byte("partial")...)

	tests := []struct {
		name string
		data []byte
		want int
	}{
		{name: "plain", data: []byte("plain text"), want: len("plain text")},
		{name: "complete", data: complete, want: len(complete)},
		{name: "complete then partial", data: append(append([]byte(nil), complete...), incomplete...), want: len(complete)},
		{name: "partial", data: incomplete, want: 0},
		{name: "empty", data: nil, want: 0},
	}

	detector := NewFrameDetector()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			flushEnd, keepFrom := detector.FindFlushBoundary(test.data)
			if flushEnd != test.want || keepFrom != test.want {
				t.Fatalf("boundary = (%d,%d), want (%d,%d)", flushEnd, keepFrom, test.want, test.want)
			}
		})
	}
}
