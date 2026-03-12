/**
 * Chart color palette with light/dark mode variants.
 *
 * Light mode uses deeper tones (600 level) for good contrast on white backgrounds.
 * Dark mode uses brighter tones (400 level) so areas/bars remain visible on dark backgrounds.
 */
export const chartColors = {
  input: { light: "#2563eb", dark: "#60a5fa" }, // blue-600 / blue-400
  output: { light: "#059669", dark: "#34d399" }, // emerald-600 / emerald-400
  cacheCreation: { light: "#9333ea", dark: "#c084fc" }, // purple-600 / purple-300
  cacheRead: { light: "#d97706", dark: "#fbbf24" }, // amber-600 / amber-400
} as const;

/** Returns resolved hex colors based on current theme. */
export function resolveChartColors(isDark: boolean) {
  const mode = isDark ? "dark" : "light";
  return {
    input: chartColors.input[mode],
    output: chartColors.output[mode],
    cacheCreation: chartColors.cacheCreation[mode],
    cacheRead: chartColors.cacheRead[mode],
  };
}
