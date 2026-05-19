const DEFAULT_FONT_SIZE = 14;

const DEFAULT_LINE_HEIGHT = 1.2;

const MONOSPACE_WIDTH_RATIO = 0.6;

const MIN_COLS = 20;

const MIN_ROWS = 5;

const MAX_COLS = 300;

const MAX_ROWS = 100;

const FALLBACK_COLS = 80;
const FALLBACK_ROWS = 24;

const MOBILE_BREAKPOINT = 768;

const DESKTOP_ACTIVITY_BAR_WIDTH = 48;

const DESKTOP_SIDEBAR_WIDTH = 240;

const DESKTOP_HORIZONTAL_PADDING = 32;

const DESKTOP_HEIGHT_RATIO = 0.65;

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

export function getDefaultTerminalSize(): { cols: number; rows: number } {
  return { cols: FALLBACK_COLS, rows: FALLBACK_ROWS };
}
