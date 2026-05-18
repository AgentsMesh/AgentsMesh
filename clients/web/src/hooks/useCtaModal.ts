"use client";

import { useCallback, useState } from "react";

/**
 * Modal that wraps a "CTA → commit / cancel" interaction.
 *
 * - `open()` shows the modal
 * - `close()` hides without side-effect (cancel path)
 * - `commit()` hides AND fires the optional `onCommit` (typically a refetch)
 *
 * Designed for the empty-state pattern: `<Button onClick={modal.open}>` +
 * `<XxxModal open={modal.isOpen} onClose={modal.close} onCreated={modal.commit} />`.
 */
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
