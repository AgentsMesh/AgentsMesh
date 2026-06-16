import type { Block } from "@/lib/viewModels/blockstore";

export function pageDisplayMeta(
  block: Block | null | undefined,
): { title: string; icon?: string } {
  if (!block) return { title: "Untitled" };
  const fromTitle = typeof block.data?.title === "string" ? block.data.title.trim() : "";
  const fromText = block.text?.trim() ?? "";
  const icon = typeof block.data?.icon === "string" ? block.data.icon : undefined;
  return { title: fromTitle || fromText || "Untitled", icon };
}
