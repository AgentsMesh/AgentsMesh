"use client";

import { useCallback, useRef, useState } from "react";
import { uploadImage } from "@/lib/api/file";

export interface AttachmentState {
  key: string | null;
  name: string | null;
  pending: boolean;
  error: string | null;
}

export interface UseFileAttachmentResult extends AttachmentState {
  inputRef: React.RefObject<HTMLInputElement | null>;
  pick: () => void;
  clear: () => void;
  handleChange: React.ChangeEventHandler<HTMLInputElement>;
}

/**
 * Wraps a hidden `<input type="file">` plus upload state. The consumer embeds
 * the ref'd input into the DOM and calls `pick()` from a button.
 */
export function useFileAttachment(): UseFileAttachmentResult {
  const inputRef = useRef<HTMLInputElement>(null);
  const [state, setState] = useState<AttachmentState>({
    key: null,
    name: null,
    pending: false,
    error: null,
  });

  const pick = useCallback(() => {
    inputRef.current?.click();
  }, []);

  const clear = useCallback(() => {
    setState({ key: null, name: null, pending: false, error: null });
    if (inputRef.current) inputRef.current.value = "";
  }, []);

  const handleChange = useCallback<React.ChangeEventHandler<HTMLInputElement>>(
    async (event) => {
      const file = event.target.files?.[0];
      if (!file) return;
      setState({ key: null, name: file.name, pending: true, error: null });
      try {
        const url = await uploadImage(file);
        setState({ key: url, name: file.name, pending: false, error: null });
      } catch (err) {
        const msg = err instanceof Error ? err.message : String(err);
        setState({ key: null, name: file.name, pending: false, error: msg });
      }
    },
    [],
  );

  return { ...state, inputRef, pick, clear, handleChange };
}
