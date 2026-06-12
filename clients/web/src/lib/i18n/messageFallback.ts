type Messages = Record<string, unknown>;

function isPlainObject(v: unknown): v is Messages {
  return typeof v === "object" && v !== null && !Array.isArray(v);
}

// Deep-merge locale messages over an English base so any key a locale omits
// falls back to en instead of rendering a raw key-path. base = en, override =
// active locale; locale values win, en fills the gaps recursively.
export function deepMergeMessages(base: Messages, override: Messages): Messages {
  const out: Messages = { ...base };
  for (const [k, v] of Object.entries(override)) {
    const bv = out[k];
    out[k] = isPlainObject(bv) && isPlainObject(v) ? deepMergeMessages(bv, v) : v;
  }
  return out;
}
