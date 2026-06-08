export { AppProviders } from "./AppProviders";
export { ThemeProvider, useTheme } from "./ThemeProvider";
export { DesktopIntlProvider, useLocale } from "./IntlProvider";
export { UpdaterProvider, useUpdater } from "./UpdaterProvider";
// RealtimeProvider lives in clients/web/src/providers — desktop mounts it
// inside DashboardShell + PopoutTerminalPage via the `@/providers` alias.
// The duplicate desktop copy was deleted (drift risk + double subscription).
export { RealtimeProvider, useRealtime } from "@/providers/RealtimeProvider";
