/**
 * Terminal size estimation utilities
 *
 * Used to estimate terminal dimensions (cols/rows) before xterm.js is initialized.
 * This helps provide better initial PTY dimensions when creating pods.
 *
 * Note: These are estimates only. The actual terminal size will be determined
 * by xterm.js FitAddon after the terminal is mounted and will be sent to PTY
 * immediately via forceResize.
 */

// Font Configuration (should match useTerminal.ts)

const DEFAULT_FONT_SIZE = 14;

const DEFAULT_LINE_HEIGHT = 1.2;

const MONOSPACE_WIDTH_RATIO = 0.6;

const MIN_COLS = 20;

const MIN_ROWS = 5;

/** Maximum columns - reasonable upper bound to prevent extreme values */
const MAX_COLS = 300;

/** Maximum rows - reasonable upper bound to prevent extreme values */
const MAX_ROWS = 100;

/** Conservative fallback dimensions (standard VT100) */
const FALLBACK_COLS = 80;
const FALLBACK_ROWS = 24;

/** Mobile breakpoint in pixels (matches useBreakpoint.ts) */
const MOBILE_BREAKPOINT = 768;

// Desktop layout dimensions (matches IDEShell.tsx)
const DESKTOP_ACTIVITY_BAR_WIDTH = 48;

const DESKTOP_SIDEBAR_WIDTH = 240;

const DESKTOP_HORIZONTAL_PADDING = 32;

const DESKTOP_HEIGHT_RATIO = 0.65;

// Mobile layout dimensions (matches MobileShell.tsx)
const MOBILE_HORIZONTAL_PADDING = 16;

const MOBILE_HEIGHT_RATIO = 0.6;

const MOBILE_CONTROLS_HEIGHT = 40;

export function estimateTerminalSize(
  containerWidth: number,
  containerHeight: number,
  fontSize: number = DEFAULT_FONT_SIZE
): { cols: number; rows: number } {
  const charWidth = fontSize * MONOSPACE_WIDTH_RATIO;
  const lineHeight = fontSize * DEFAULT_LINE_HEIGHT;

  const cols = Math.min(
    MAX_COLS,
    Math.max(MIN_COLS, Math.floor(containerWidth / charWidth))
  );
  const rows = Math.min(
    MAX_ROWS,
    Math.max(MIN_ROWS, Math.floor(containerHeight / lineHeight))
  );

  return { cols, rows };
}

export function estimateWorkspaceTerminalSize(
  fontSize: number = DEFAULT_FONT_SIZE
): { cols: number; rows: number } {
  // SSR fallback - use conservative defaults
  if (typeof window === "undefined") {
    return { cols: FALLBACK_COLS, rows: FALLBACK_ROWS };
  }

  const isMobile = window.innerWidth < MOBILE_BREAKPOINT;

  if (isMobile) {
    const terminalWidth = window.innerWidth - MOBILE_HORIZONTAL_PADDING;
    const terminalHeight =
      window.innerHeight * MOBILE_HEIGHT_RATIO - MOBILE_CONTROLS_HEIGHT;
    return estimateTerminalSize(terminalWidth, terminalHeight, fontSize);
  }

  const terminalWidth =
    window.innerWidth -
    DESKTOP_ACTIVITY_BAR_WIDTH -
    DESKTOP_SIDEBAR_WIDTH -
    DESKTOP_HORIZONTAL_PADDING;
  const terminalHeight = window.innerHeight * DESKTOP_HEIGHT_RATIO;

  return estimateTerminalSize(terminalWidth, terminalHeight, fontSize);
}

/**
 * Get conservative default terminal size (VT100 standard)
 * Used as fallback when window dimensions are unavailable
 */
export function getDefaultTerminalSize(): { cols: number; rows: number } {
  return { cols: FALLBACK_COLS, rows: FALLBACK_ROWS };
}
