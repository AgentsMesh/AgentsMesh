import type { LoopData } from "@/lib/api/loop";

// Special value for RunnerHost credential
export const RUNNER_HOST_PROFILE_ID = 0;

export interface LoopCreateDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (createdLoop?: LoopData) => void;
  editLoop?: LoopData;
}
