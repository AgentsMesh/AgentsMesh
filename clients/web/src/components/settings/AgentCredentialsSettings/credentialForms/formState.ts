import type { CredentialProfileViewModel } from "../../_shared/credentialViewModel";
import { getConfiguredKeys } from "../../_shared/credentialViewModel";
import type {
  CredentialFormSpec,
  CustomEnvEntry,
  OneOfCredentialField,
} from "./types";
import { getEnvKeysFromSpec } from "./index";

export interface CredentialFormState {
  values: Record<string, string>;
  selectedOneOf: Record<string, string>;
  customEnv: CustomEnvEntry[];
  // declared secrets the user explicitly removed via the trash button. Their
  // stored value never round-trips, so a blank input reads as "keep" — only an
  // explicit remove can delete one. buildCredentialsPayload sends these as ""
  // (the backend's delete signal). custom-env rows signal deletion by row
  // absence instead (see previousKeys).
  removedKeys: string[];
}

function newId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`;
}

function defaultOneOfSelection(field: OneOfCredentialField): string {
  return field.options[0]?.envKey ?? "";
}

export function emptyFormState(spec: CredentialFormSpec): CredentialFormState {
  const selectedOneOf: Record<string, string> = {};
  for (const field of spec.fields) {
    if (field.kind === "oneof") {
      selectedOneOf[field.group] = defaultOneOfSelection(field);
    }
  }
  return { values: {}, selectedOneOf, customEnv: [], removedKeys: [] };
}

export function initFormStateFromProfile(
  spec: CredentialFormSpec,
  profile: CredentialProfileViewModel | null
): CredentialFormState {
  if (!profile) return emptyFormState(spec);

  const declaredKeys = getEnvKeysFromSpec(spec);
  // Unify on getConfiguredKeys (configured_fields ∪ configured_values keys): a
  // non-secret key (one IsNonSecretKey allows) round-trips in configured_values,
  // not configured_fields. Both the oneof preselect AND the custom-env rebuild
  // must see it, else the edit form drops it and a later save clears it.
  const configured = new Set(getConfiguredKeys(profile));
  const configuredValues = profile.configured_values ?? {};
  const values: Record<string, string> = {};
  const selectedOneOf: Record<string, string> = {};

  for (const field of spec.fields) {
    if (field.kind === "oneof") {
      const presentOption = field.options.find((o) => configured.has(o.envKey));
      const target = presentOption?.envKey ?? defaultOneOfSelection(field);
      selectedOneOf[field.group] = target;
      if (presentOption?.kind === "text" && configuredValues[target]) {
        values[target] = configuredValues[target];
      }
    } else if (field.kind === "text" && configuredValues[field.envKey]) {
      values[field.envKey] = configuredValues[field.envKey];
    }
  }

  const customEnv: CustomEnvEntry[] = [];
  for (const key of configured) {
    if (declaredKeys.has(key)) continue;
    customEnv.push({ id: newId(), key, value: configuredValues[key] ?? "" });
  }

  return { values, selectedOneOf, customEnv, removedKeys: [] };
}

// Collect the form state into the wire payload `Record<envKey, value>`. A blank
// value is the backend's "delete this key" signal; an omitted key means "keep".
// Rather than scatter `out[k] = ""` across branches, gather a single `deleted`
// set from the three sources that can request a deletion, gather surviving
// `values` separately, then merge — a real value always wins over a stale
// delete, and the empty-string signal is emitted in exactly one place.
//
// Deletion sources:
//   - oneof: committing a switch (the selected option has a value) clears every
//     sibling. A blank selection commits nothing, so a half-finished toggle
//     keeps the stored auth instead of wiping a working credential.
//   - declared secret: a blank input means "keep" (the value never round-trips),
//     so only an explicit remove (state.removedKeys) deletes it.
//   - custom-env: a previously-configured key whose row is gone is deleted; an
//     emptied-but-present row reads as "keep" (its masked value never echoes).
export function buildCredentialsPayload(
  spec: CredentialFormSpec,
  state: CredentialFormState,
  previousKeys: string[] = []
): Record<string, string> {
  const declaredKeys = getEnvKeysFromSpec(spec);
  const removed = new Set(state.removedKeys);
  // Null-prototype map so an ENV key named like an Object.prototype member
  // (toString, constructor, __proto__, …) is written and deleted as an OWN
  // property. On a plain {} `key in values` walks the prototype (true for
  // "toString") and `values["__proto__"]=` hits the proto setter, either of
  // which would silently swallow that key's delete signal — a failure-to-revoke.
  const values: Record<string, string> = Object.create(null);
  const deleted = new Set<string>();

  for (const field of spec.fields) {
    if (field.kind === "oneof") {
      const sel = state.selectedOneOf[field.group];
      const v = state.values[sel];
      if (v) {
        values[sel] = v;
        for (const opt of field.options) {
          if (opt.envKey !== sel) deleted.add(opt.envKey);
        }
      }
    } else {
      const v = state.values[field.envKey];
      if (v) values[field.envKey] = v;
      else if (removed.has(field.envKey)) deleted.add(field.envKey);
    }
  }

  const currentCustomKeys = new Set<string>();
  for (const entry of state.customEnv) {
    const key = entry.key.trim();
    if (!key) continue;
    currentCustomKeys.add(key);
    if (entry.value) values[key] = entry.value;
  }
  for (const key of previousKeys) {
    if (!declaredKeys.has(key) && !currentCustomKeys.has(key)) deleted.add(key);
  }

  // Fold the delete signals straight into the (already null-proto) values map:
  // a deleted key not already carrying a value becomes "". A real value wins —
  // the `in` guard keeps a resubmitted key from being overwritten with "".
  for (const key of deleted) {
    if (!(key in values)) values[key] = "";
  }
  return values;
}
