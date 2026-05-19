"use client";

import { useCallback, useState } from "react";

export function useCtaModal(onCommit?: () => void | Promise<void>) {
  const [isOpen, setIsOpen] = useState(false);
  const open = useCallback(() => setIsOpen(true), []);
  const close = useCallback(() => setIsOpen(false), []);
  const commit = useCallback(async () => {
    setIsOpen(false);
    await onCommit?.();
  }, [onCommit]);
  return { isOpen, open, close, commit };
}
