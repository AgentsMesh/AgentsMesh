package vt

// CellAttrs represents text attributes (bold, italic, etc.)
type CellAttrs uint16

const (
	AttrNone          CellAttrs = 0
	AttrBold          CellAttrs = 1 << 0
	AttrDim           CellAttrs = 1 << 1
	AttrItalic        CellAttrs = 1 << 2
	AttrUnderline     CellAttrs = 1 << 3
	AttrBlink         CellAttrs = 1 << 4
	AttrInverse       CellAttrs = 1 << 5
	AttrInvisible     CellAttrs = 1 << 6
	AttrStrikethrough CellAttrs = 1 << 7
	AttrOverline      CellAttrs = 1 << 8
)

// UnderlineStyle represents different underline styles
// Matches xterm.js UnderlineStyle enum
type UnderlineStyle uint8

const (
	UnderlineNone   UnderlineStyle = 0
	UnderlineSingle UnderlineStyle = 1
	UnderlineDouble UnderlineStyle = 2
	UnderlineCurly  UnderlineStyle = 3
	UnderlineDotted UnderlineStyle = 4
	UnderlineDashed UnderlineStyle = 5
)

// Has checks if the attribute is set
func (a CellAttrs) Has(attr CellAttrs) bool {
	return a&attr != 0
}

// Cell represents one terminal buffer slot with its style. A grapheme cluster
// is owned by one content cell and may be followed by a width-zero placeholder.
// HasContent distinguishes a printed space from a null/erased cell, matching
// xterm's HAS_CONTENT bit.
type Cell struct {
	Char           rune
	Combining      string
	HasContent     bool
	Fg             Color
	Bg             Color
	Attrs          CellAttrs
	Width          uint8          // 0 = placeholder after CJK, 1 = normal, 2 = CJK wide char
	UnderlineStyle UnderlineStyle // Underline style (single, double, curly, etc.)
	UnderlineColor Color          // Underline color (for colored underlines)
}

// NewCell creates a new cell with default styling
func NewCell(ch rune) Cell {
	return Cell{
		Char:           ch,
		HasContent:     ch != 0 && ch != ' ',
		Fg:             DefaultColor(),
		Bg:             DefaultColor(),
		Attrs:          AttrNone,
		Width:          1,
		UnderlineStyle: UnderlineNone,
		UnderlineColor: DefaultColor(),
	}
}

// NewStyledCell creates a new cell with specified styling
func NewStyledCell(ch rune, fg, bg Color, attrs CellAttrs) Cell {
	return Cell{
		Char:           ch,
		HasContent:     ch != 0 && ch != ' ',
		Fg:             fg,
		Bg:             bg,
		Attrs:          attrs,
		Width:          1,
		UnderlineStyle: UnderlineNone,
		UnderlineColor: DefaultColor(),
	}
}

// NewFullStyledCell creates a new cell with all style options
func NewFullStyledCell(ch rune, fg, bg Color, attrs CellAttrs, width uint8, ulStyle UnderlineStyle, ulColor Color) Cell {
	return Cell{
		Char:           ch,
		HasContent:     ch != 0 && ch != ' ',
		Fg:             fg,
		Bg:             bg,
		Attrs:          attrs,
		Width:          width,
		UnderlineStyle: ulStyle,
		UnderlineColor: ulColor,
	}
}

// IsEmpty returns true if the cell is empty (space with default styling)
func (c Cell) IsEmpty() bool {
	return !c.HasContent && c.Char == ' ' && c.Combining == "" &&
		c.Fg.IsDefault() && c.Bg.IsDefault() && c.Attrs == AttrNone
}

// Text returns the complete code-point sequence owned by this cell.
func (c Cell) Text() string {
	if !c.HasContent || c.Char == 0 {
		return ""
	}
	if c.Combining == "" {
		return string(c.Char)
	}
	return string(c.Char) + c.Combining
}

// IsPlaceholder distinguishes a wide-character continuation from a real
// zero-width code point stored in its own terminal slot.
func (c Cell) IsPlaceholder() bool {
	return c.Width == 0 && !c.HasContent && c.Char == 0
}

// StyleEquals compares only the style (not the character)
func (c Cell) StyleEquals(other Cell) bool {
	return c.Fg.Equals(other.Fg) && c.Bg.Equals(other.Bg) && c.Attrs == other.Attrs &&
		c.UnderlineStyle == other.UnderlineStyle && c.UnderlineColor.Equals(other.UnderlineColor)
}

// IsAttributeDefault returns true if all attributes are at default values
func (c Cell) IsAttributeDefault() bool {
	return c.Fg.IsDefault() && c.Bg.IsDefault() && c.Attrs == AttrNone &&
		c.UnderlineStyle == UnderlineNone && c.UnderlineColor.IsDefault()
}

// GetWidth returns the cell width (0 for placeholder, 1 for normal, 2 for CJK)
func (c Cell) GetWidth() uint8 {
	if c.Width == 0 {
		return 0
	}
	return c.Width
}
