package file

import (
	"time"
)

// File represents a stored file's metadata
type File struct {
	ID             int64     `gorm:"primaryKey" json:"id"`
	OrganizationID int64     `gorm:"not null;index" json:"organization_id"`
	UploaderID     int64     `gorm:"not null;index" json:"uploader_id"`
	OriginalName   string    `gorm:"size:255;not null" json:"original_name"`
	StorageKey     string    `gorm:"size:500;not null;uniqueIndex" json:"storage_key"`
	MimeType       string    `gorm:"size:100;not null" json:"mime_type"`
	Size           int64     `gorm:"not null" json:"size"`
	CreatedAt      time.Time `gorm:"not null;default:now()" json:"created_at"`
}

func (File) TableName() string {
	return "files"
}

// IsImage returns true if the file is an image
func (f *File) IsImage() bool {
	switch f.MimeType {
	case "image/jpeg", "image/png", "image/gif", "image/webp", "image/svg+xml":
		return true
	}
	return false
}

// IsPDF returns true if the file is a PDF
func (f *File) IsPDF() bool {
	return f.MimeType == "application/pdf"
}
