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
)

var validBlockTypes = map[string]bool{
	"paragraph": true, "heading": true, "code_block": true, "quote": true, "list": true,
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
			if err := b.Items[i][j].validate(); err != nil {
				return fmt.Errorf("item[%d][%d]: %w", i, j, err)
			}
		}
	}
	for i := range b.Children {
		if err := b.Children[i].validate(depth + 1); err != nil {
			return fmt.Errorf("child[%d]: %w", i, err)
		}
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
