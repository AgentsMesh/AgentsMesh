package vt

import (
	"github.com/clipperhouse/uax29/v2/graphemes"
	"github.com/mattn/go-runewidth"
)

var runeWidthCond = func() *runewidth.Condition {
	condition := runewidth.NewCondition()
	condition.EastAsianWidth = false
	return condition
}()

func standaloneGraphemeWidth(ch rune) int {
	if ch == '\ufe0f' {
		return 2
	}
	width := runeWidthCond.RuneWidth(ch)
	return min(max(width, 1), 2)
}

func graphemeClusterWidth(cell Cell, ch rune) int {
	width := int(cell.Width)
	if runeWidth := runeWidthCond.RuneWidth(ch); runeWidth > width {
		width = runeWidth
	}
	if ch == '\ufe0f' || regionalIndicatorCount(cell.Text()+string(ch)) == 2 {
		width = 2
	}
	return min(max(width, 1), 2)
}

func regionalIndicatorCount(value string) int {
	count := 0
	for _, ch := range value {
		if ch >= 0x1f1e6 && ch <= 0x1f1ff {
			count++
		}
	}
	return count
}

func graphemeJoins(value string, ch rune) bool {
	if value == "" {
		return false
	}
	candidate := value + string(ch)
	iterator := graphemes.FromString(candidate)
	return iterator.Next() && iterator.End() == len(candidate)
}
