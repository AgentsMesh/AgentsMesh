export const THEMES = {
  LIGHT: "light",
  DARK: "dark",
  SOLARIZED_LIGHT: "solarized-light",
  SOLARIZED_DARK: "solarized-dark",
  SYSTEM: "system",
} as const;

export type Theme = (typeof THEMES)[keyof typeof THEMES];

export interface ThemeConfig {
  id: Theme;
  nameKey: string;
  icon: "sun" | "moon" | "monitor" | "palette";
  isDark: boolean;
}

export const themeConfigs: ThemeConfig[] = [
  { id: "light", nameKey: "theme_light", icon: "sun", isDark: false },
  { id: "dark", nameKey: "theme_dark", icon: "moon", isDark: true },
  { id: "solarized-light", nameKey: "theme_solarized_light", icon: "palette", isDark: false },
  { id: "solarized-dark", nameKey: "theme_solarized_dark", icon: "palette", isDark: true },
  { id: "system", nameKey: "theme_system", icon: "monitor", isDark: false },
];

export const themeColors: Record<string, string> = {
  light: "#ffffff",
  dark: "#0a0a0a",
  "solarized-light": "#fdf6e3",
  "solarized-dark": "#002b36",
};
