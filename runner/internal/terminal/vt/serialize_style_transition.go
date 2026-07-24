package vt

// diffStyle generates SGR parameters for style transition
func (h *StringSerializeHandler) diffStyle(cell, oldCell Cell) []string {
	var sgrSeq []string

	fgChanged := !cell.Fg.Equals(oldCell.Fg)
	bgChanged := !cell.Bg.Equals(oldCell.Bg)
	flagsChanged := !equalFlags(cell, oldCell)

	if !fgChanged && !bgChanged && !flagsChanged {
		return nil
	}

	if cell.IsAttributeDefault() {
		if !oldCell.IsAttributeDefault() {
			sgrSeq = append(sgrSeq, "0")
		}
	} else {
		if fgChanged {
			sgrSeq = append(sgrSeq, buildFgColorSGR(cell.Fg)...)
		}
		if bgChanged {
			sgrSeq = append(sgrSeq, buildBgColorSGR(cell.Bg)...)
		}
		if flagsChanged {
			sgrSeq = append(sgrSeq, buildFlagsSGR(cell, oldCell)...)
		}
	}

	return sgrSeq
}
