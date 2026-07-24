package vt

// Color represents a terminal color.
// It can be a default color, a 256-color palette index, or a true RGB color.
type Color struct {
	// ColorType: 0=default, 1=palette (256 colors), 2=RGB (true color)
	colorType uint8
	// For palette colors: the color index (0-255)
	// For RGB: packed as index (unused, use r/g/b fields)
	index uint8
	// RGB components (only used when colorType == 2)
	r, g, b uint8
}

// DefaultColor returns the default terminal color
func DefaultColor() Color {
	return Color{colorType: 0}
}

// PaletteColor creates a color from a 256-color palette index
func PaletteColor(index uint8) Color {
	return Color{colorType: 1, index: index}
}

// RGBColor creates a true color from RGB components
func RGBColor(r, g, b uint8) Color {
	return Color{colorType: 2, r: r, g: g, b: b}
}

// IsDefault returns true if this is the default color
func (c Color) IsDefault() bool {
	return c.colorType == 0
}

// IsPalette returns true if this is a palette color
func (c Color) IsPalette() bool {
	return c.colorType == 1
}

// IsRGB returns true if this is a true color
func (c Color) IsRGB() bool {
	return c.colorType == 2
}

// Index returns the palette index (only valid for palette colors)
func (c Color) Index() uint8 {
	return c.index
}

// RGB returns the RGB components (only valid for RGB colors)
func (c Color) RGB() (r, g, b uint8) {
	return c.r, c.g, c.b
}

// Equals compares two colors for equality
func (c Color) Equals(other Color) bool {
	if c.colorType != other.colorType {
		return false
	}
	switch c.colorType {
	case 0:
		return true
	case 1:
		return c.index == other.index
	case 2:
		return c.r == other.r && c.g == other.g && c.b == other.b
	default:
		return false
	}
}
