"use client";

import { buttonVariants } from "@/components/ui/button";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import type { SSOConfig } from "@/lib/api/sso";

interface SSODeleteDialogProps {
  config: SSOConfig | null;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}

export function SSODeleteDialog({ config, onOpenChange, onConfirm }: SSODeleteDialogProps) {
  return (
    <AlertDialog open={!!config} onOpenChange={(open) => !open && onOpenChange(false)}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Delete SSO Config</AlertDialogTitle>
          <AlertDialogDescription>
            Are you sure you want to delete SSO config &quot;{config?.name}&quot; ({config?.domain})?
            This action cannot be undone.
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction
            onClick={onConfirm}
            className={buttonVariants({ variant: "destructive" })}
          >
            Delete
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}
