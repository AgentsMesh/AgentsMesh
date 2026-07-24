package aggregator

import (
	"testing"
)

// Helper to build sync output frame (incremental - no clear screen)
func buildSyncFrame(content string) []byte {
	return append(append(syncOutputStartSeq, []byte(content)...), syncOutputEndSeq...)
}

// Helper to build a synchronized full-redraw frame for throttling tests.
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
