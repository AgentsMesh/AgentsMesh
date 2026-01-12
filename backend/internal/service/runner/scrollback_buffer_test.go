package runner

import (
	"bytes"
	"testing"
	"unicode/utf8"
)

func TestNewScrollbackBuffer(t *testing.T) {
	buffer := NewScrollbackBuffer(1024)
	if buffer == nil {
		t.Fatal("NewScrollbackBuffer returned nil")
	}
	if buffer.maxSize != 1024 {
		t.Errorf("maxSize = %d, want 1024", buffer.maxSize)
	}
	if len(buffer.data) != 0 {
		t.Errorf("data length = %d, want 0", len(buffer.data))
	}
}

func TestScrollbackBufferWrite(t *testing.T) {
	buffer := NewScrollbackBuffer(100)

	buffer.Write([]byte("hello "))
	if string(buffer.data) != "hello " {
		t.Errorf("data = %q, want %q", buffer.data, "hello ")
	}

	buffer.Write([]byte("world"))
	if string(buffer.data) != "hello world" {
		t.Errorf("data = %q, want %q", buffer.data, "hello world")
	}
}

func TestScrollbackBufferWriteOverflow(t *testing.T) {
	buffer := NewScrollbackBuffer(10)

	buffer.Write([]byte("1234567890"))
	if len(buffer.data) != 10 {
		t.Errorf("data length = %d, want 10", len(buffer.data))
	}

	// Writing more should trim from the beginning
	buffer.Write([]byte("ABCDE"))
	if len(buffer.data) != 10 {
		t.Errorf("data length = %d, want 10", len(buffer.data))
	}
	// Should have last 10 bytes: "67890ABCDE"
	if string(buffer.data) != "67890ABCDE" {
		t.Errorf("data = %q, want %q", buffer.data, "67890ABCDE")
	}
}

func TestScrollbackBufferGetData(t *testing.T) {
	buffer := NewScrollbackBuffer(100)
	buffer.Write([]byte("test data"))

	data := buffer.GetData()
	if string(data) != "test data" {
		t.Errorf("GetData() = %q, want %q", data, "test data")
	}

	// Verify it's a copy
	data[0] = 'X'
	if string(buffer.data) == "Xest data" {
		t.Error("GetData() should return a copy, not the original")
	}
}

func TestScrollbackBufferGetRecentLines(t *testing.T) {
	buffer := NewScrollbackBuffer(1000)

	t.Run("empty buffer", func(t *testing.T) {
		lines := buffer.GetRecentLines(5)
		if lines != nil {
			t.Errorf("expected nil, got %q", lines)
		}
	})

	t.Run("less lines than requested", func(t *testing.T) {
		buffer.data = []byte("line1\nline2\nline3")
		lines := buffer.GetRecentLines(10)
		if string(lines) != "line1\nline2\nline3" {
			t.Errorf("got %q, want all lines", lines)
		}
	})

	t.Run("more lines than requested", func(t *testing.T) {
		buffer.data = []byte("line1\nline2\nline3\nline4\nline5")
		lines := buffer.GetRecentLines(2)
		// Should get last 2 lines
		if !bytes.Contains(lines, []byte("line5")) {
			t.Errorf("got %q, expected to contain line5", lines)
		}
	})
}

func TestScrollbackBufferClear(t *testing.T) {
	buffer := NewScrollbackBuffer(100)
	buffer.Write([]byte("test data"))

	buffer.Clear()
	if len(buffer.data) != 0 {
		t.Errorf("data length after clear = %d, want 0", len(buffer.data))
	}
}

func TestScrollbackBufferConcurrency(t *testing.T) {
	buffer := NewScrollbackBuffer(10000)
	done := make(chan bool, 4)

	// Writer 1
	go func() {
		for i := 0; i < 100; i++ {
			buffer.Write([]byte("writer1 data\n"))
		}
		done <- true
	}()

	// Writer 2
	go func() {
		for i := 0; i < 100; i++ {
			buffer.Write([]byte("writer2 data\n"))
		}
		done <- true
	}()

	// Reader 1
	go func() {
		for i := 0; i < 100; i++ {
			_ = buffer.GetData()
		}
		done <- true
	}()

	// Reader 2
	go func() {
		for i := 0; i < 100; i++ {
			_ = buffer.GetRecentLines(10)
		}
		done <- true
	}()

	for i := 0; i < 4; i++ {
		<-done
	}
}

func BenchmarkScrollbackBufferWrite(b *testing.B) {
	buffer := NewScrollbackBuffer(DefaultScrollbackSize)
	data := []byte("benchmark test data line\n")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buffer.Write(data)
	}
}

func BenchmarkScrollbackBufferGetData(b *testing.B) {
	buffer := NewScrollbackBuffer(DefaultScrollbackSize)
	buffer.Write(make([]byte, DefaultScrollbackSize/2))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buffer.GetData()
	}
}

func TestTrimToValidUTF8Start(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "empty slice",
			input:    []byte{},
			expected: []byte{},
		},
		{
			name:     "valid ASCII",
			input:    []byte("hello"),
			expected: []byte("hello"),
		},
		{
			name:     "valid UTF-8 Chinese",
			input:    []byte("你好"),
			expected: []byte("你好"),
		},
		{
			name:     "valid UTF-8 mixed",
			input:    []byte("hello你好world"),
			expected: []byte("hello你好world"),
		},
		{
			name:     "starts with continuation byte (1 byte)",
			input:    append([]byte{0x80}, []byte("hello")...), // 10xxxxxx continuation byte
			expected: []byte("hello"),
		},
		{
			name:     "starts with 2 continuation bytes",
			input:    append([]byte{0x80, 0x80}, []byte("test")...), // Two continuation bytes
			expected: []byte("test"),
		},
		{
			name:     "starts with 3 continuation bytes",
			input:    append([]byte{0x80, 0x80, 0x80}, []byte("abc")...), // Three continuation bytes
			expected: []byte("abc"),
		},
		{
			name:     "truncated multi-byte sequence at start",
			input:    append([]byte{0xE4, 0xBD}, []byte("hello")...), // Truncated 3-byte UTF-8 (missing last byte of 你)
			expected: []byte("hello"),
		},
		{
			name:     "valid 2-byte UTF-8 at start",
			input:    []byte{0xC3, 0xA9, 'h', 'i'}, // é (2-byte UTF-8) + "hi"
			expected: []byte{0xC3, 0xA9, 'h', 'i'},
		},
		{
			name:     "valid 3-byte UTF-8 at start",
			input:    []byte{0xE4, 0xBD, 0xA0, 'h', 'i'}, // 你 (3-byte UTF-8) + "hi"
			expected: []byte{0xE4, 0xBD, 0xA0, 'h', 'i'},
		},
		{
			name:     "valid 4-byte UTF-8 emoji at start",
			input:    []byte{0xF0, 0x9F, 0x98, 0x80, 'h', 'i'}, // 😀 (4-byte UTF-8) + "hi"
			expected: []byte{0xF0, 0x9F, 0x98, 0x80, 'h', 'i'},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trimToValidUTF8Start(tt.input)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("trimToValidUTF8Start(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestScrollbackBufferWriteUTF8Boundary(t *testing.T) {
	// Test that buffer correctly handles UTF-8 boundary when trimming
	buffer := NewScrollbackBuffer(10)

	// Write UTF-8 content that will overflow
	// 你好 = 6 bytes (3 bytes each)
	// Adding more will force trim
	buffer.Write([]byte("你好")) // 6 bytes
	buffer.Write([]byte("世界")) // 6 more bytes = 12 total, need to trim to 10

	data := buffer.GetData()
	// After trimming, should have valid UTF-8
	if len(data) > 10 {
		t.Errorf("buffer exceeded maxSize: got %d bytes", len(data))
	}

	// Check it's valid UTF-8
	for i := 0; i < len(data); {
		r, size := utf8.DecodeRune(data[i:])
		if r == utf8.RuneError && size == 1 {
			t.Errorf("invalid UTF-8 at position %d", i)
			break
		}
		i += size
	}
}
