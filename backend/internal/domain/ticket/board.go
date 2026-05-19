package ticket

type BoardColumn struct {
	Status  string   `json:"status"`
	Count   int      `json:"count"`
	Tickets []Ticket `json:"tickets"`
}

type Board struct {
	Columns        []BoardColumn    `json:"columns"`
	PriorityCounts map[string]int64 `json:"priority_counts"`
}
