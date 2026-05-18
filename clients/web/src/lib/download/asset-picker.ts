import type { DesktopAsset } from "./asset-types";
import type { DetectedPlatform } from "./platform-detect";

// Choose the best desktop asset for a user based on detected platform.
// Priority: exact platform match → exact arch (or universal) → first platform asset → first overall.
export function pickPrimaryDesktop(
  desktop: DesktopAsset[],
  detected: DetectedPlatform | null,
): DesktopAsset | null {
  if (desktop.length === 0) return null;
  if (!detected) return desktop[0];
  const samePlatform = desktop.filter((a) => a.platform === detected.platform);
  if (samePlatform.length === 0) return desktop[0];
  const exactArch = samePlatform.find(
    (a) => a.arch === detected.arch || a.arch === "universal",
  );
  return exactArch ?? samePlatform[0];
}
