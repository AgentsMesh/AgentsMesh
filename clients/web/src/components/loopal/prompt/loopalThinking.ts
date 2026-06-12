// Thinking effort levels — i18n key ↔ serialized ThinkingConfig (loopal wire).
// Sending: key → config string (loopal.thinking payload). Display: parse a raw
// config back to a key the caller translates via t("thinking." + key).
export interface ThinkingOption {
  key: string;
  config: string;
}

export const THINKING_OPTIONS: ThinkingOption[] = [
  { key: "auto", config: JSON.stringify({ type: "auto" }) },
  { key: "off", config: JSON.stringify({ type: "disabled" }) },
  { key: "low", config: JSON.stringify({ type: "effort", level: "low" }) },
  { key: "med", config: JSON.stringify({ type: "effort", level: "medium" }) },
  { key: "high", config: JSON.stringify({ type: "effort", level: "high" }) },
  { key: "max", config: JSON.stringify({ type: "effort", level: "max" }) },
];

const LEVEL_KEY: Record<string, string> = {
  low: "low",
  medium: "med",
  high: "high",
  max: "max",
};

// Parse a raw ThinkingConfig JSON string into a short i18n key.
export function thinkingKey(raw: string | null): string | null {
  if (!raw) return null;
  try {
    const c = JSON.parse(raw) as { type?: string; level?: string };
    if (c.type === "auto") return "auto";
    if (c.type === "disabled") return "off";
    if (c.type === "effort" && c.level) return LEVEL_KEY[c.level] ?? c.level;
    return null;
  } catch {
    return null;
  }
}
