// Decorative avatar background hues (GitHub Primer palette, matching the design
// tokens in tokens-preview.pastel). Not semantic — deterministic per seed so a
// given user/pod always keeps the same color.
export const AVATAR_PALETTE = [
  "bg-[#0969DA]", "bg-[#1A7F37]", "bg-[#9A6700]", "bg-[#8250DF]",
  "bg-[#0550AE]", "bg-[#BC4C00]", "bg-[#1F883D]", "bg-[#6639BA]",
];

export function paletteFor(seed: string | number): string {
  const s = String(seed);
  let hash = 0;
  for (let i = 0; i < s.length; i++) hash = (hash * 31 + s.charCodeAt(i)) >>> 0;
  return AVATAR_PALETTE[hash % AVATAR_PALETTE.length];
}
