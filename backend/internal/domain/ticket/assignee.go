package ticket

type AssigneeUser struct {
	ID        int64   `gorm:"primaryKey" json:"id"`
	Username  string  `json:"username"`
	Name      *string `json:"name,omitempty"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

func (AssigneeUser) TableName() string { return "users" }

type Assignee struct {
	TicketID int64         `gorm:"primaryKey" json:"ticket_id"`
	UserID   int64         `gorm:"primaryKey" json:"user_id"`
	User     *AssigneeUser `gorm:"foreignKey:UserID;references:ID" json:"user,omitempty"`
}

func (Assignee) TableName() string {
	return "ticket_assignees"
}
