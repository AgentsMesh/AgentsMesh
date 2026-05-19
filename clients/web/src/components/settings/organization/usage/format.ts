export function formatTokenCount(value: number): string {
  if (!Number.isFinite(value)) return "0";
  if (value >= 1_000_000_000_000) {
    return `${(value / 1_000_000_000_000).toFixed(2)}T`;
  }
  if (value >= 1_000_000_000) {
    return `${(value / 1_000_000_000).toFixed(2)}B`;
  }
  if (value >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(2)}M`;
  }
  if (value >= 1_000) {
    // Avoid "1000.0K" — promote to M when rounding crosses the boundary.
    const k = value / 1_000;
    if (k >= 999.95) {
      return `${(value / 1_000_000).toFixed(2)}M`;
    }
    return `${k.toFixed(1)}K`;
  }
  return String(value);
}

export function formatNumber(value: number): string {
  if (!Number.isFinite(value)) return "0";
  return value.toLocaleString();
}
