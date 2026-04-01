package ticket

// Label represents a label that can be applied to tickets
type Label struct {
	ID             int64  `gorm:"primaryKey" json:"id"`
	OrganizationID int64  `gorm:"not null;index" json:"organization_id"`
	RepositoryID   *int64 `gorm:"index" json:"repository_id,omitempty"` // nil = organization-level

	Name  string `gorm:"size:100;not null" json:"name"`
	Color string `gorm:"size:7;not null;default:'#6B7280'" json:"color"` // Hex color
}

func (Label) TableName() string {
	return "labels"
}

// TicketLabel represents the many-to-many relationship between tickets and labels
type TicketLabel struct {
	TicketID int64 `gorm:"primaryKey" json:"ticket_id"`
	LabelID  int64 `gorm:"primaryKey" json:"label_id"`
}

func (TicketLabel) TableName() string {
	return "ticket_labels"
}
