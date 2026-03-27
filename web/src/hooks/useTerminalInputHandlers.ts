import { Terminal as XTerm, IDisposable } from "@xterm/xterm";
import { MutableRefObject } from "react";
import { isTouchPrimaryInput } from "@/lib/platform";
import { uploadImage } from "@/lib/api/file";
import { toast } from "sonner";
import type { TerminalConnection } from "./useTerminalConnection";

/**
 * Tracks IME composition state on the xterm helper textarea.
 * Runs on all platforms — needed to prevent sending incomplete
 * IME input via term.onData.
 */
function setupCompositionTracking(
  textarea: HTMLTextAreaElement,
  disposables: IDisposable[],
): { isComposing: { current: boolean } } {
  const isComposing = { current: false };

  const handleCompositionStart = () => { isComposing.current = true; };
  const handleCompositionEnd = () => { isComposing.current = false; };

  textarea.addEventListener('compositionstart', handleCompositionStart);
  textarea.addEventListener('compositionend', handleCompositionEnd);

  disposables.push({
    dispose: () => {
      textarea.removeEventListener('compositionstart', handleCompositionStart);
      textarea.removeEventListener('compositionend', handleCompositionEnd);
    },
  });

  return { isComposing };
}

/**
 * Syncs the xterm helper textarea position to follow the terminal cursor.
 * Touch-device only — on desktop, xterm.js internally positions the textarea
 * for IME via its CompositionHelper.
 *
 * Only binds to onCursorMove (not onWriteParsed) to avoid output→input
 * coupling that causes IME candidate box flickering on Windows.
 */
function setupMobileTextareaSync(
  textarea: HTMLTextAreaElement,
  term: XTerm,
  disposables: IDisposable[],
): void {
  const syncTextareaPosition = () => {
    const cursorX = term.buffer.active.cursorX;
    const cursorY = term.buffer.active.cursorY - term.buffer.active.viewportY;
    const cellWidth = term.options.fontSize! * 0.6;
    const cellHeight = term.options.fontSize! * (term.options.lineHeight || 1.2);
    textarea.style.left = `${Math.max(0, cursorX * cellWidth)}px`;
    textarea.style.top = `${Math.max(0, cursorY * cellHeight)}px`;
  };

  const cursorDisposable = term.onCursorMove(syncTextareaPosition);
  const initialSyncRafId = requestAnimationFrame(syncTextareaPosition);

  disposables.push(
    cursorDisposable,
    { dispose: () => cancelAnimationFrame(initialSyncRafId) },
  );
}

/**
 * Sets up IME composition tracking and, on touch devices, textarea
 * position sync for correct IME candidate box placement.
 */
export function setupIME(
  container: HTMLDivElement,
  term: XTerm,
  disposables: IDisposable[],
): { isComposing: { current: boolean } } {
  const textarea = container.querySelector('.xterm-helper-textarea') as HTMLTextAreaElement;
  if (!textarea) return { isComposing: { current: false } };

  const result = setupCompositionTracking(textarea, disposables);

  // On touch devices, manually sync textarea position to follow cursor.
  // On desktop, xterm.js handles this internally — manual override would
  // conflict and cause IME candidate box flickering (e.g. Windows CJK IME).
  if (isTouchPrimaryInput()) {
    setupMobileTextareaSync(textarea, term, disposables);
  }

  return result;
}

/**
 * Intercepts paste events containing images — uploads them and sends
 * the resulting URL to the terminal connection.
 */
export function setupImagePaste(
  container: HTMLDivElement,
  connectionRef: MutableRefObject<TerminalConnection | null>,
  disposables: IDisposable[],
): void {
  const handlePaste = (e: ClipboardEvent) => {
    const items = e.clipboardData?.items;
    if (!items) return;

    for (let i = 0; i < items.length; i++) {
      const item = items[i];
      if (item.type.startsWith('image/')) {
        e.preventDefault();
        e.stopPropagation();
        const blob = item.getAsFile();
        if (!blob) continue;

        // Check connection before starting upload to fail fast
        if (!connectionRef.current) {
          toast.error('Terminal not connected');
          return;
        }

        const toastId = toast.loading('Uploading image...');
        uploadImage(blob)
          .then((url) => {
            // Re-check connection — it may have dropped during upload
            if (!connectionRef.current) {
              toast.error('Terminal disconnected during upload', { id: toastId });
              return;
            }
            connectionRef.current.send(url);
            toast.success('Image uploaded', { id: toastId });
          })
          .catch((err) => {
            toast.error(err instanceof Error ? err.message : 'Failed to upload image', { id: toastId });
          });
        return;
      }
    }
  };

  container.addEventListener('paste', handlePaste, true);
  disposables.push({ dispose: () => container.removeEventListener('paste', handlePaste, true) });
}
