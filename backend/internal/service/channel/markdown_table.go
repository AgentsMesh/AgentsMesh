package channel

import (
	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
)

func (w *mdWalker) tableRows(t *east.Table) []channel.TableRow {
	var rows []channel.TableRow
	for n := t.FirstChild(); n != nil; n = n.NextSibling() {
		switch r := n.(type) {
		case *east.TableHeader:
			rows = append(rows, w.tableRow(r, true))
		case *east.TableRow:
			rows = append(rows, w.tableRow(r, false))
		}
	}
	return rows
}

func (w *mdWalker) tableRow(row ast.Node, header bool) channel.TableRow {
	var cells []channel.TableCell
	for c := row.FirstChild(); c != nil; c = c.NextSibling() {
		cell, ok := c.(*east.TableCell)
		if !ok {
			continue
		}
		cells = append(cells, channel.TableCell{
			Elements: w.inline(cell),
			Align:    cellAlign(cell.Alignment),
		})
	}
	return channel.TableRow{Header: header, Cells: cells}
}

func cellAlign(a east.Alignment) string {
	switch a {
	case east.AlignLeft:
		return "left"
	case east.AlignCenter:
		return "center"
	case east.AlignRight:
		return "right"
	}
	return ""
}
