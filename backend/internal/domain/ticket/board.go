package ticket

// BoardColumn represents a kanban board column
type BoardColumn struct {
	Status  string   `json:"status"`
	Count   int      `json:"count"`
	Tickets []Ticket `json:"tickets"`
}

// Board represents a kanban board view
type Board struct {
	Columns []BoardColumn `json:"columns"`
}
