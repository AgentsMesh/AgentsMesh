import type {
  DesktopAsset,
  ReleaseSummary,
  RunnerAsset,
} from "./asset-types";
import { classifyDesktop, classifyRunner } from "./asset-classifier";

const GITHUB_API = "https://api.github.com/repos/AgentsMesh/AgentsMesh/releases/latest";
export const RELEASES_PAGE_URL = "https://github.com/AgentsMesh/AgentsMesh/releases";

interface GithubAsset {
  name: string;
  browser_download_url: string;
  size: number;
}

interface GithubRelease {
  tag_name: string;
  html_url: string;
  published_at: string;
  assets: GithubAsset[];
}

const PLATFORM_ORDER = ["macos", "windows", "linux"] as const;
const ARCH_ORDER = ["arm64", "x64", "universal"] as const;
const rankIn = <T extends readonly string[]>(order: T) => (v: string) => {
  const i = order.indexOf(v as T[number]);
  return i === -1 ? order.length : i;
};
const platformRank = rankIn(PLATFORM_ORDER);
const archRank = rankIn(ARCH_ORDER);

const byPlatformArch = <T extends { platform: string; arch: string }>(a: T, b: T) =>
  platformRank(a.platform) - platformRank(b.platform) ||
  archRank(a.arch) - archRank(b.arch);

export async function fetchLatestRelease(): Promise<ReleaseSummary | null> {
  try {
    // Cache strategy lives in the route segment (`export const revalidate`),
    // not here — keep this function a pure fetch + parse so callers control
    // freshness policy.
    const res = await fetch(GITHUB_API, {
      headers: { Accept: "application/vnd.github+json" },
    });
    if (!res.ok) return null;
    const data: GithubRelease = await res.json();
    const desktop: DesktopAsset[] = [];
    const runner: RunnerAsset[] = [];
    let checksumsUrl: string | undefined;

    for (const a of data.assets) {
      const raw = { name: a.name, url: a.browser_download_url, size: a.size };
      if (a.name === "checksums.txt") {
        checksumsUrl = a.browser_download_url;
        continue;
      }
      const desktopAsset = classifyDesktop(raw);
      if (desktopAsset) {
        desktop.push(desktopAsset);
        continue;
      }
      const runnerAsset = classifyRunner(raw);
      if (runnerAsset) runner.push(runnerAsset);
    }

    desktop.sort(byPlatformArch);
    runner.sort(byPlatformArch);

    return {
      version: data.tag_name.replace(/^v/, ""),
      tag: data.tag_name,
      htmlUrl: data.html_url,
      publishedAt: data.published_at,
      desktop,
      runner,
      checksumsUrl,
    };
  } catch {
    return null;
  }
}
