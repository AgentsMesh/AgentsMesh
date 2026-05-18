import type { Block } from "@/lib/api/blockstoreTypes";

// Fallback to top-level text when data.text is missing (issue #366: MCP
// agents often populate only the search-summary field).
export function readBlockText(block: Block): string {
  const dataText = block.data?.text;
  if (typeof dataText === "string") return dataText;
  if (typeof block.text === "string") return block.text;
  return "";
}
