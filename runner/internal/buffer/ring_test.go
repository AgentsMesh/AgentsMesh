package buffer

import (
	"bytes"
	"sync"
	"testing"
)

func TestNewRing(t *testing.T) {
	rb := NewRing(100)

	if rb == nil {
		t.Fatal("NewRing returned nil")
	}

	if rb.Cap() != 100 {
		t.Errorf("Cap: got %v, want 100", rb.Cap())
	}

	if rb.Len() != 0 {
		t.Errorf("Len: got %v, want 0", rb.Len())
	}
}

func TestRingWrite(t *testing.T) {
	rb := NewRing(10)

	n, err := rb.Write([]byte("hello"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if n != 5 {
		t.Errorf("n: got %v, want 5", n)
	}

	if rb.Len() != 5 {
		t.Errorf("Len: got %v, want 5", rb.Len())
	}
}

func TestRingBytes(t *testing.T) {
	rb := NewRing(10)

	rb.Write([]byte("hello"))

	data := rb.Bytes()
	if !bytes.Equal(data, []byte("hello")) {
		t.Errorf("Bytes: got %v, want hello", string(data))
	}
}

func TestRingOverflow(t *testing.T) {
	rb := NewRing(5)

	rb.Write([]byte("abcdefgh"))

	data := rb.Bytes()
	// Should only have last 5 bytes: "defgh"
	if !bytes.Equal(data, []byte("defgh")) {
		t.Errorf("Bytes: got %v, want defgh", string(data))
	}

	if rb.Len() != 5 {
		t.Errorf("Len: got %v, want 5", rb.Len())
	}
}

func TestRingReset(t *testing.T) {
	rb := NewRing(10)

	rb.Write([]byte("hello"))
	rb.Reset()

	if rb.Len() != 0 {
		t.Errorf("Len after reset: got %v, want 0", rb.Len())
	}

	data := rb.Bytes()
	if data != nil {
		t.Errorf("Bytes after reset: got %v, want nil", data)
	}
}

func TestRingEmptyBytes(t *testing.T) {
	rb := NewRing(10)

	data := rb.Bytes()
	if data != nil {
		t.Errorf("Bytes on empty buffer: got %v, want nil", data)
	}
}

func TestRingCap(t *testing.T) {
	rb := NewRing(50)

	if rb.Cap() != 50 {
		t.Errorf("Cap: got %v, want 50", rb.Cap())
	}
}

func TestRingWrapAround(t *testing.T) {
	rb := NewRing(5)

	// Write more than capacity to ensure wrap-around
	rb.Write([]byte("12345"))
	if rb.Len() != 5 {
		t.Errorf("Len after fill: got %v, want 5", rb.Len())
	}

	// Write more to trigger wrap
	rb.Write([]byte("67"))

	data := rb.Bytes()
	if !bytes.Equal(data, []byte("34567")) {
		t.Errorf("Bytes after wrap: got %v, want 34567", string(data))
	}
}

func TestRingMultipleWrites(t *testing.T) {
	rb := NewRing(20)

	rb.Write([]byte("hello"))
	rb.Write([]byte(" "))
	rb.Write([]byte("world"))

	data := rb.Bytes()
	if !bytes.Equal(data, []byte("hello world")) {
		t.Errorf("Bytes: got %v, want 'hello world'", string(data))
	}
}

func TestRingFullBuffer(t *testing.T) {
	rb := NewRing(5)

	rb.Write([]byte("12345"))

	if rb.Len() != 5 {
		t.Errorf("Len: got %v, want 5", rb.Len())
	}

	data := rb.Bytes()
	if !bytes.Equal(data, []byte("12345")) {
		t.Errorf("Bytes: got %v, want 12345", string(data))
	}
}

func TestRingConcurrentWrite(t *testing.T) {
	rb := NewRing(100)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				rb.Write([]byte("x"))
			}
		}(i)
	}

	wg.Wait()

	// Buffer should be full or have some data
	if rb.Len() == 0 {
		t.Error("buffer should not be empty after concurrent writes")
	}
}

func TestRingConcurrentReadWrite(t *testing.T) {
	rb := NewRing(50)

	var wg sync.WaitGroup

	// Writer
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			rb.Write([]byte("data"))
		}
	}()

	// Reader
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = rb.Bytes()
			_ = rb.Len()
		}
	}()

	wg.Wait()
}

func TestRingIOWriter(t *testing.T) {
	rb := NewRing(100)

	// Use as io.Writer
	_, err := rb.Write([]byte("test"))
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}

	// Bytes should match
	data := rb.Bytes()
	if !bytes.Equal(data, []byte("test")) {
		t.Errorf("Bytes: got %v, want test", string(data))
	}
}

func TestRingLenDuringOverflow(t *testing.T) {
	rb := NewRing(5)

	rb.Write([]byte("123"))
	if rb.Len() != 3 {
		t.Errorf("Len after partial fill: got %v, want 3", rb.Len())
	}

	rb.Write([]byte("4567"))
	// Buffer should be full now
	if rb.Len() != 5 {
		t.Errorf("Len after overflow: got %v, want 5", rb.Len())
	}
}

func BenchmarkRingWrite(b *testing.B) {
	rb := NewRing(1024)
	data := []byte("benchmark data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rb.Write(data)
	}
}

func BenchmarkRingBytes(b *testing.B) {
	rb := NewRing(1024)
	rb.Write([]byte("some initial data"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.Bytes()
	}
}

func BenchmarkRingLen(b *testing.B) {
	rb := NewRing(1024)
	rb.Write([]byte("some initial data"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rb.Len()
	}
}

// UTF-8 boundary tests

func TestRingUTF8NoCorruption(t *testing.T) {
	rb := NewRing(20)

	// Write Chinese characters (3 bytes each)
	rb.Write([]byte("你好世界"))
	data := rb.Bytes()
	if string(data) != "你好世界" {
		t.Errorf("UTF-8 corruption: got %q, want '你好世界'", string(data))
	}
}

func TestRingUTF8BoundaryTrim(t *testing.T) {
	// Buffer size 10, Chinese chars are 3 bytes each
	// "你好世界" = 12 bytes, so wrap will occur
	rb := NewRing(10)

	// Write "你好世界" (12 bytes) to a 10-byte buffer
	// After wrap, we should have last 10 bytes, but trimmed to valid UTF-8 start
	rb.Write([]byte("你好世界"))

	data := rb.Bytes()
	// The result should be valid UTF-8 (no leading continuation bytes)
	if len(data) > 0 && (data[0]&0xC0) == 0x80 {
		t.Errorf("UTF-8 boundary not properly trimmed: starts with continuation byte 0x%02x", data[0])
	}

	// Result should be valid UTF-8 string
	s := string(data)
	for i, r := range s {
		if r == '\ufffd' {
			t.Errorf("UTF-8 replacement character at position %d", i)
		}
	}
}

func TestRingUTF8BoxDrawingOverflow(t *testing.T) {
	// Box drawing chars are 3 bytes each
	// "─" = E2 94 80
	rb := NewRing(8)

	// Write "────" (12 bytes) to 8-byte buffer
	rb.Write([]byte("────"))

	data := rb.Bytes()
	// Should not start with a continuation byte
	if len(data) > 0 && (data[0]&0xC0) == 0x80 {
		t.Errorf("UTF-8 boundary not properly trimmed: first byte is 0x%02x", data[0])
	}
}

func TestRingUTF8EmojiOverflow(t *testing.T) {
	// Emoji are 4 bytes each
	// "🚀" = F0 9F 9A 80
	rb := NewRing(6)

	// Write "🚀🚀" (8 bytes) to 6-byte buffer
	rb.Write([]byte("🚀🚀"))

	data := rb.Bytes()
	// Should not start with a continuation byte
	if len(data) > 0 && (data[0]&0xC0) == 0x80 {
		t.Errorf("UTF-8 boundary not properly trimmed: first byte is 0x%02x", data[0])
	}
}

func TestTrimToValidUTF8Start(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "empty",
			input:    []byte{},
			expected: []byte{},
		},
		{
			name:     "ascii",
			input:    []byte("hello"),
			expected: []byte("hello"),
		},
		{
			name:     "valid utf8",
			input:    []byte("你好"),
			expected: []byte("你好"),
		},
		{
			name:     "starts with continuation byte",
			input:    []byte{0x80, 0x41, 0x42}, // continuation, A, B
			expected: []byte{0x41, 0x42},       // A, B
		},
		{
			name:     "two continuation bytes",
			input:    []byte{0x80, 0x80, 0x41}, // two continuations, A
			expected: []byte{0x41},             // A
		},
		{
			name:     "partial Chinese char then valid",
			input:    append([]byte{0x94, 0x80}, []byte("好")...), // continuation bytes of "你" + "好"
			expected: []byte("好"),
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
