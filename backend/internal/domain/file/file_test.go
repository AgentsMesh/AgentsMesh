package file

import (
	"testing"
)

func TestFileTableName(t *testing.T) {
	f := File{}
	if f.TableName() != "files" {
		t.Errorf("expected 'files', got %s", f.TableName())
	}
}

func TestFileIsImage(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		expected bool
	}{
		{"jpeg", "image/jpeg", true},
		{"png", "image/png", true},
		{"gif", "image/gif", true},
		{"webp", "image/webp", true},
		{"svg", "image/svg+xml", true},
		{"pdf", "application/pdf", false},
		{"text", "text/plain", false},
		{"json", "application/json", false},
		{"octet-stream", "application/octet-stream", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &File{MimeType: tt.mimeType}
			if f.IsImage() != tt.expected {
				t.Errorf("mimeType %s: expected IsImage() = %v, got %v", tt.mimeType, tt.expected, f.IsImage())
			}
		})
	}
}

func TestFileIsPDF(t *testing.T) {
	tests := []struct {
		name     string
		mimeType string
		expected bool
	}{
		{"pdf", "application/pdf", true},
		{"jpeg", "image/jpeg", false},
		{"png", "image/png", false},
		{"text", "text/plain", false},
		{"json", "application/json", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &File{MimeType: tt.mimeType}
			if f.IsPDF() != tt.expected {
				t.Errorf("mimeType %s: expected IsPDF() = %v, got %v", tt.mimeType, tt.expected, f.IsPDF())
			}
		})
	}
}

func TestFileStruct(t *testing.T) {
	f := File{
		ID:             1,
		OrganizationID: 100,
		UploaderID:     50,
		OriginalName:   "test-image.png",
		StorageKey:     "orgs/100/files/2024/01/abc123.png",
		MimeType:       "image/png",
		Size:           102400,
	}

	if f.ID != 1 {
		t.Errorf("expected ID 1, got %d", f.ID)
	}
	if f.OrganizationID != 100 {
		t.Errorf("expected OrganizationID 100, got %d", f.OrganizationID)
	}
	if f.UploaderID != 50 {
		t.Errorf("expected UploaderID 50, got %d", f.UploaderID)
	}
	if f.OriginalName != "test-image.png" {
		t.Errorf("expected OriginalName 'test-image.png', got %s", f.OriginalName)
	}
	if f.StorageKey != "orgs/100/files/2024/01/abc123.png" {
		t.Errorf("expected StorageKey 'orgs/100/files/2024/01/abc123.png', got %s", f.StorageKey)
	}
	if f.MimeType != "image/png" {
		t.Errorf("expected MimeType 'image/png', got %s", f.MimeType)
	}
	if f.Size != 102400 {
		t.Errorf("expected Size 102400, got %d", f.Size)
	}
}

// Benchmark tests
func BenchmarkFileIsImage(b *testing.B) {
	f := &File{MimeType: "image/png"}
	for i := 0; i < b.N; i++ {
		f.IsImage()
	}
}

func BenchmarkFileIsPDF(b *testing.B) {
	f := &File{MimeType: "application/pdf"}
	for i := 0; i < b.N; i++ {
		f.IsPDF()
	}
}
