import type { Arch, Platform } from "./asset-types";

export interface DetectedPlatform {
  platform: Platform;
  arch: Arch;
}

function detectPlatform(): DetectedPlatform | null {
  if (typeof navigator === "undefined") return null;
  const ua = navigator.userAgent;
  const platformStr = (navigator.platform || "").toLowerCase();

  if (/Mac|iPhone|iPad/i.test(ua) || platformStr.includes("mac")) {
    const arch: Arch = /Intel/i.test(ua) && !/ARM/i.test(ua) ? "x64" : "arm64";
    return { platform: "macos", arch };
  }
  if (/Win/i.test(ua) || platformStr.includes("win")) {
    return { platform: "windows", arch: "universal" };
  }
  if (/Linux/i.test(ua) || platformStr.includes("linux")) {
    const arch: Arch = /aarch64|arm64/i.test(ua) ? "arm64" : "x64";
    return { platform: "linux", arch };
  }
  return null;
}

let cachedDetected: DetectedPlatform | null | undefined;

export function getDetectedPlatform(): DetectedPlatform | null {
  if (cachedDetected !== undefined) return cachedDetected;
  cachedDetected = detectPlatform();
  return cachedDetected;
}

export function formatBytes(bytes: number): string {
  if (!bytes) return "";
  const mb = bytes / (1024 * 1024);
  return mb >= 100 ? `${mb.toFixed(0)} MB` : `${mb.toFixed(1)} MB`;
}
