import { UnicodeGraphemesAddon } from "@xterm/addon-unicode-graphemes";
import type {
  ITerminalAddon,
  IUnicodeHandling,
  IUnicodeVersionProvider,
  Terminal,
} from "@xterm/xterm";
import {
  DOUBLE_WIDTH_RANGES,
  RUNEWIDTH_UNICODE_VERSION,
  RUNEWIDTH_VERSION,
  ZERO_WIDTH_RANGES,
} from "./runewidthUnicode17Ranges";

const UPSTREAM_GRAPHEME_VERSION = "15-graphemes";
const WIDTH_MASK = 0b110;

export const RUNNER_UNICODE_VERSION = `${UPSTREAM_GRAPHEME_VERSION}+runewidth-${RUNEWIDTH_UNICODE_VERSION}`;
export { RUNEWIDTH_UNICODE_VERSION, RUNEWIDTH_VERSION };

function contains(ranges: Uint32Array, codepoint: number): boolean {
  let low = 0;
  let high = ranges.length / 2 - 1;
  while (low <= high) {
    const middle = (low + high) >> 1;
    const start = ranges[middle * 2];
    const end = ranges[middle * 2 + 1];
    if (codepoint < start) high = middle - 1;
    else if (codepoint > end) low = middle + 1;
    else return true;
  }
  return false;
}

export function runnerRuneWidth(codepoint: number): 0 | 1 | 2 {
  if (!Number.isInteger(codepoint) || codepoint < 0 || codepoint > 0x10FFFF) return 0;
  if (contains(ZERO_WIDTH_RANGES, codepoint)) return 0;
  if (contains(DOUBLE_WIDTH_RANGES, codepoint)) return 2;
  return 1;
}

function standaloneRunnerWidth(codepoint: number): 1 | 2 {
  if (codepoint === 0xFE0F) return 2;
  return runnerRuneWidth(codepoint) === 2 ? 2 : 1;
}

function encodedWidth(properties: number): number {
  return (properties >> 1) & 0b11;
}

export class RunnerUnicodeGraphemeProvider implements IUnicodeVersionProvider {
  readonly version = RUNNER_UNICODE_VERSION;

  constructor(private readonly graphemeProvider: IUnicodeVersionProvider) {}

  wcwidth(codepoint: number): 0 | 1 | 2 {
    return runnerRuneWidth(codepoint);
  }

  charProperties(codepoint: number, preceding: number): number {
    const properties = this.graphemeProvider.charProperties(codepoint, preceding);
    const shouldJoin = (properties & 1) !== 0;
    const width = shouldJoin
      ? Math.max(encodedWidth(preceding), encodedWidth(properties), runnerRuneWidth(codepoint))
      : standaloneRunnerWidth(codepoint);
    return (properties & ~WIDTH_MASK) | (width << 1);
  }
}

function createUnicodeFacade(
  unicode: IUnicodeHandling,
  isActivating: () => boolean,
): IUnicodeHandling {
  const mapVersion = (version: string) =>
    isActivating() && version === UPSTREAM_GRAPHEME_VERSION
      ? RUNNER_UNICODE_VERSION
      : version;

  return {
    get versions() {
      return unicode.versions;
    },
    get activeVersion() {
      return unicode.activeVersion;
    },
    set activeVersion(version: string) {
      unicode.activeVersion = mapVersion(version);
    },
    register(provider: IUnicodeVersionProvider) {
      unicode.register(
        provider.version === UPSTREAM_GRAPHEME_VERSION
          ? new RunnerUnicodeGraphemeProvider(provider)
          : provider,
      );
    },
  };
}

// The pinned upstream addon owns grapheme segmentation. The facade decorates
// the provider it registers so xterm and Runner share go-runewidth's cell
// widths without copying the addon's UAX29 state machine or using its private
// fields.
export class RunnerUnicodeGraphemesAddon implements ITerminalAddon {
  private readonly graphemeAddon = new UnicodeGraphemesAddon();
  private activating = false;

  activate(terminal: Terminal): void {
    const unicode = createUnicodeFacade(terminal.unicode, () => this.activating);
    const terminalFacade = new Proxy(terminal, {
      get(target, property, receiver) {
        if (property === "unicode") return unicode;
        const value = Reflect.get(target, property, receiver) as unknown;
        return typeof value === "function" ? value.bind(target) : value;
      },
    });
    this.activating = true;
    try {
      this.graphemeAddon.activate(terminalFacade);
    } finally {
      this.activating = false;
    }
  }

  dispose(): void {
    this.graphemeAddon.dispose();
  }
}
