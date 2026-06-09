package channel

import (
	"encoding/json"
	"fmt"
	"net/url"
)

const (
	MaxBlockCount       = 100
	MaxElementsPerBlock = 1000
	MaxTextLength       = 10000
	MaxCodeBlockLength  = 50000
	MaxContentSize      = 100 * 1024 // 100KB
	MaxNestingDepth     = 3
	MaxTableRows        = 100
	MaxTableColumns     = 32
)

var validBlockTypes = map[string]bool{
	"paragraph": true, "heading": true, "code_block": true, "quote": true, "list": true, "table": true,
}

var validElementTypes = map[string]bool{
	InlineText: true, InlineMention: true, InlineLink: true, InlineLinebreak: true,
}

var validEntityTypes = map[string]bool{
	EntityPod: true, EntityUser: true, EntityTicket: true, EntityChannel: true,
}

var allowedURLSchemes = map[string]bool{
	"http": true, "https": true, "mailto": true,
}

func (mc *MessageContent) Validate() error {
	if mc.Kind != "text" {
		return fmt.Errorf("invalid content kind: %q", mc.Kind)
	}
	if len(mc.Blocks) > MaxBlockCount {
		return fmt.Errorf("too many blocks: %d (max %d)", len(mc.Blocks), MaxBlockCount)
	}
	data, err := json.Marshal(mc)
	if err != nil {
		return fmt.Errorf("content serialization failed: %w", err)
	}
	if len(data) > MaxContentSize {
		return fmt.Errorf("content too large: %d bytes (max %d)", len(data), MaxContentSize)
	}
	for i := range mc.Blocks {
		if err := mc.Blocks[i].validate(0); err != nil {
			return fmt.Errorf("block[%d]: %w", i, err)
		}
	}
	return nil
}

func (b *Block) validate(depth int) error {
	if depth > MaxNestingDepth {
		return fmt.Errorf("nesting too deep (max %d)", MaxNestingDepth)
	}
	if !validBlockTypes[b.Type] {
		return fmt.Errorf("invalid block type: %q", b.Type)
	}
	if len(b.Elements) > MaxElementsPerBlock {
		return fmt.Errorf("too many elements: %d (max %d)", len(b.Elements), MaxElementsPerBlock)
	}
	if len(b.Items) > MaxElementsPerBlock {
		return fmt.Errorf("too many list items: %d (max %d)", len(b.Items), MaxElementsPerBlock)
	}
	if b.Type == "code_block" && len(b.Text) > MaxCodeBlockLength {
		return fmt.Errorf("code block too long: %d chars (max %d)", len(b.Text), MaxCodeBlockLength)
	}
	for i := range b.Elements {
		if err := b.Elements[i].validate(); err != nil {
			return fmt.Errorf("element[%d]: %w", i, err)
		}
	}
	for i := range b.Items {
		for j := range b.Items[i] {
			if err := b.Items[i][j].validate(depth + 1); err != nil {
				return fmt.Errorf("item[%d][%d]: %w", i, j, err)
			}
		}
	}
	for i := range b.Children {
		if err := b.Children[i].validate(depth + 1); err != nil {
			return fmt.Errorf("child[%d]: %w", i, err)
		}
	}
	if b.Type == "table" {
		return validateTableRows(b.Rows)
	}
	if len(b.Rows) > 0 {
		return fmt.Errorf("rows not allowed on %q block", b.Type)
	}
	return nil
}

func (el *InlineElement) validate() error {
	if !validElementTypes[el.Type] {
		return fmt.Errorf("invalid element type: %q", el.Type)
	}
	if el.Type == InlineText && len(el.Text) > MaxTextLength {
		return fmt.Errorf("text too long: %d chars (max %d)", len(el.Text), MaxTextLength)
	}
	if el.Type == InlineMention && el.EntityType != "" && !validEntityTypes[el.EntityType] {
		return fmt.Errorf("invalid entity type: %q", el.EntityType)
	}
	if el.Type == InlineLink && el.URL != "" {
		if !isAllowedURLScheme(el.URL) {
			return fmt.Errorf("disallowed URL scheme in %q", el.URL)
		}
	}
	return nil
}

func isAllowedURLScheme(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return allowedURLSchemes[u.Scheme]
}

var validTableAligns = map[string]bool{"": true, "left": true, "center": true, "right": true}

func validateTableRows(rows []TableRow) error {
	if len(rows) > MaxTableRows {
		return fmt.Errorf("too many table rows: %d (max %d)", len(rows), MaxTableRows)
	}
	sawBody := false
	cols := -1
	for i := range rows {
		if rows[i].Header {
			if sawBody {
				return fmt.Errorf("header row[%d] after body row", i)
			}
		} else {
			sawBody = true
		}
		if len(rows[i].Cells) > MaxTableColumns {
			return fmt.Errorf("too many table columns: %d (max %d)", len(rows[i].Cells), MaxTableColumns)
		}
		if cols < 0 {
			cols = len(rows[i].Cells)
		} else if len(rows[i].Cells) != cols {
			return fmt.Errorf("row[%d] has %d cells, want %d", i, len(rows[i].Cells), cols)
		}
		for j := range rows[i].Cells {
			if err := validateTableCell(rows[i].Cells[j], i, j); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateTableCell(cell TableCell, i, j int) error {
	if !validTableAligns[cell.Align] {
		return fmt.Errorf("row[%d].cell[%d]: invalid align %q", i, j, cell.Align)
	}
	if len(cell.Elements) > MaxElementsPerBlock {
		return fmt.Errorf("row[%d].cell[%d]: too many elements: %d (max %d)", i, j, len(cell.Elements), MaxElementsPerBlock)
	}
	for k := range cell.Elements {
		if err := cell.Elements[k].validate(); err != nil {
			return fmt.Errorf("row[%d].cell[%d].element[%d]: %w", i, j, k, err)
		}
	}
	return nil
}
