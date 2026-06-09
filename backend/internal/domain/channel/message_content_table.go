package channel

type TableCell struct {
	Elements []InlineElement `json:"elements,omitempty"`
	Align    string          `json:"align,omitempty"`
}

type TableRow struct {
	Header bool        `json:"header,omitempty"`
	Cells  []TableCell `json:"cells,omitempty"`
}
