
function normalizeVersion(version: string): string {
  return version.trim().replace(/^v/, "");
}

function parseSemver(version: string): [number, number, number] | null {
  const match = normalizeVersion(version).match(/^(\d+)\.(\d+)\.(\d+)/);
  if (!match) return null;
  return [parseInt(match[1], 10), parseInt(match[2], 10), parseInt(match[3], 10)];
}

export function isVersionOutdated(
  current?: string | null,
  latest?: string | null
): boolean {
  if (!current || !latest) return false;

  if (current === "dev" || latest === "dev") return false;

  const currentParts = parseSemver(current);
  const latestParts = parseSemver(latest);

  if (!currentParts || !latestParts) return false;

  for (let i = 0; i < 3; i++) {
    if (currentParts[i] < latestParts[i]) return true;
    if (currentParts[i] > latestParts[i]) return false;
  }

  return false;
}
