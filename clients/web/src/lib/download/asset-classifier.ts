import type {
  DesktopAsset,
  ReleaseAsset,
  RunnerAsset,
} from "./asset-types";

const DESKTOP_DMG = /^AgentsMesh-[\d.]+(?:-(arm64))?\.dmg$/i;
const DESKTOP_NSIS = /^AgentsMesh\.Setup\.[\d.]+\.exe$/i;
const DESKTOP_APPIMAGE = /^AgentsMesh-[\d.]+(?:-(arm64))?\.AppImage$/i;
const DESKTOP_DEB = /^agentsmesh_[\d.]+_(amd64|arm64)\.deb$/i;
const DESKTOP_ZIP_MAC = /^AgentsMesh-[\d.]+(?:-(arm64))?-mac\.zip$/i;
const RUNNER_ARCHIVE = /^agentsmesh-runner_[\d.]+_(linux|darwin|windows)_(amd64|arm64)\.(tar\.gz|zip)$/i;

export function classifyDesktop(asset: ReleaseAsset): DesktopAsset | null {
  let m: RegExpMatchArray | null;
  if ((m = asset.name.match(DESKTOP_DMG))) {
    return { ...asset, platform: "macos", arch: m[1] ? "arm64" : "x64", kind: "dmg" };
  }
  if (DESKTOP_NSIS.test(asset.name)) {
    return { ...asset, platform: "windows", arch: "universal", kind: "exe" };
  }
  if ((m = asset.name.match(DESKTOP_APPIMAGE))) {
    return { ...asset, platform: "linux", arch: m[1] ? "arm64" : "x64", kind: "appimage" };
  }
  if ((m = asset.name.match(DESKTOP_DEB))) {
    return { ...asset, platform: "linux", arch: m[1].toLowerCase() === "arm64" ? "arm64" : "x64", kind: "deb" };
  }
  if ((m = asset.name.match(DESKTOP_ZIP_MAC))) {
    return { ...asset, platform: "macos", arch: m[1] ? "arm64" : "x64", kind: "zip" };
  }
  return null;
}

export function classifyRunner(asset: ReleaseAsset): RunnerAsset | null {
  const m = asset.name.match(RUNNER_ARCHIVE);
  if (!m) return null;
  const platform = m[1].toLowerCase() === "darwin" ? "macos" : (m[1].toLowerCase() as "linux" | "windows");
  const arch = m[2].toLowerCase() === "amd64" ? "x64" : "arm64";
  const kind = m[3].toLowerCase() === "zip" ? "zip" : "tarball";
  return { ...asset, platform, arch, kind };
}
