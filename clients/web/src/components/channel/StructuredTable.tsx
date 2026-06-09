"use client";

import type { Block, TableRow, TableCell } from "@/lib/viewModels/channelMessage";
import { RenderInline } from "./StructuredInline";

const alignStyle = (align?: TableCell["align"]): React.CSSProperties | undefined =>
  align ? { textAlign: align } : undefined;

export function RenderTable({ block }: { block: Block }) {
  if (!block.rows?.length) return null;
  const headerRows: TableRow[] = [];
  const bodyRows: TableRow[] = [];
  for (const row of block.rows) (row.header ? headerRows : bodyRows).push(row);

  return (
    <div className="overflow-x-auto">
      <table className="min-w-max">
        {headerRows.length > 0 && (
          <thead>
            {headerRows.map((row, i) => (
              <TableRowCells key={i} row={row} head />
            ))}
          </thead>
        )}
        <tbody>
          {bodyRows.map((row, i) => (
            <TableRowCells key={i} row={row} />
          ))}
        </tbody>
      </table>
    </div>
  );
}

function TableRowCells({ row, head }: { row: TableRow; head?: boolean }) {
  const Cell = head ? "th" : "td";
  return (
    <tr>
      {row.cells?.map((cell, j) => (
        <Cell key={j} style={alignStyle(cell.align)}>
          {cell.elements?.map((el, k) => (
            <RenderInline key={k} element={el} />
          ))}
        </Cell>
      ))}
    </tr>
  );
}
