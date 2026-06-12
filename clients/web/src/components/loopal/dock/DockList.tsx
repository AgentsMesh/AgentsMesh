import type { ReactNode } from "react";
import type { UseConfirmDialogOptions } from "@/components/ui/use-confirm-dialog";

// Confirm-dialog signature LoopalBottomDock hands down to the dock sections that
// issue destructive controls (kill / delete / disconnect). Single source so the
// sections don't each redeclare it.
export type Confirm = (o: UseConfirmDialogOptions) => Promise<boolean>;

// Vertical row-list container shared by the dock's data sections. Centralizes
// the section spacing so a layout change lands on every panel at once; rows
// stay section-specific (their content/actions differ too much to generalize).
export function DockList({ children }: { children: ReactNode }) {
  return <div className="flex flex-col gap-1.5 p-2">{children}</div>;
}
