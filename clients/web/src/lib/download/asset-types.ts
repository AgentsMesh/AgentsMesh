export type Platform = "macos" | "windows" | "linux";
export type Arch = "x64" | "arm64" | "universal";
export type DesktopKind = "dmg" | "exe" | "appimage" | "deb" | "zip";
export type RunnerKind = "tarball" | "zip";

export interface ReleaseAsset {
  name: string;
  url: string;
  size: number;
}

export interface DesktopAsset extends ReleaseAsset {
  platform: Platform;
  arch: Arch;
  kind: DesktopKind;
}

export interface RunnerAsset extends ReleaseAsset {
  platform: Platform;
  arch: Arch;
  kind: RunnerKind;
}

export interface ReleaseSummary {
  version: string;
  tag: string;
  publishedAt: string;
  htmlUrl: string;
  desktop: DesktopAsset[];
  runner: RunnerAsset[];
  checksumsUrl?: string;
}

export const DESKTOP_KIND_LABEL: Record<DesktopKind, string> = {
  dmg: "DMG",
  exe: "Installer",
  appimage: "AppImage",
  deb: ".deb",
  zip: "ZIP",
};

export const RUNNER_KIND_EXT: Record<RunnerKind, string> = {
  tarball: ".tar.gz",
  zip: ".zip",
};

export const PLATFORM_LABEL: Record<Platform, string> = {
  macos: "macOS",
  windows: "Windows",
  linux: "Linux",
};

export const ARCH_LABEL: Record<Arch, string> = {
  arm64: "Apple Silicon / ARM64",
  x64: "Intel / x64",
  universal: "Universal",
};

export const platformLabel = (p: Platform): string => PLATFORM_LABEL[p];
export const archLabel = (a: Arch): string => ARCH_LABEL[a];
