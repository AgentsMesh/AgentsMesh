import { describe, it, expect } from "vitest";
import {
  buildCredentialsPayload,
  emptyFormState,
  initFormStateFromProfile,
} from "../formState";
import { getCredentialFormSpec } from "../index";

describe("buildCredentialsPayload — XOR sibling deletion", () => {
  it("sends the selected option and empties deselected siblings", () => {
    const spec = getCredentialFormSpec("claude-code");
    const state = emptyFormState(spec);
    const oneof = spec.fields.find((f) => f.kind === "oneof");
    if (oneof?.kind !== "oneof") throw new Error("expected oneof auth group");

    state.selectedOneOf[oneof.group] = "ANTHROPIC_API_KEY";
    state.values["ANTHROPIC_API_KEY"] = "sk-key";

    const payload = buildCredentialsPayload(spec, state);

    expect(payload["ANTHROPIC_API_KEY"]).toBe("sk-key");
    // The deselected Auth Token sibling is sent empty — the wire signal telling
    // the backend to drop any stored token, so switching API Key ↔ Auth Token
    // never leaves both set.
    expect(payload["ANTHROPIC_AUTH_TOKEN"]).toBe("");
  });

  it("does not emit a spurious empty for an unfilled declared text field", () => {
    const spec = getCredentialFormSpec("claude-code");
    const state = emptyFormState(spec);
    const oneof = spec.fields.find((f) => f.kind === "oneof");
    if (oneof?.kind === "oneof") {
      state.selectedOneOf[oneof.group] = "ANTHROPIC_API_KEY";
    }
    state.values["ANTHROPIC_API_KEY"] = "sk-key";

    const payload = buildCredentialsPayload(spec, state);
    // Base URL left blank → omitted (a text field is cleared by blanking; an
    // unfilled-and-unconfigured one carries no signal). The delete signal is
    // reserved for deselected oneof siblings (only once the new option has a
    // value), explicit secret removals, and removed custom-env rows.
    expect(payload).not.toHaveProperty("ANTHROPIC_BASE_URL");
  });

  it("leaves stored auth untouched when the toggled-to option is still blank", () => {
    const spec = getCredentialFormSpec("claude-code");
    const state = emptyFormState(spec);
    // User toggled the radio to API Key but typed nothing (e.g. they only meant
    // to rename the profile). The stored Auth Token must NOT be deleted — a
    // half-finished switch should leave the working credential intact.
    state.selectedOneOf["anthropic_auth"] = "ANTHROPIC_API_KEY";
    const payload = buildCredentialsPayload(spec, state, ["ANTHROPIC_AUTH_TOKEN"]);
    expect(payload).not.toHaveProperty("ANTHROPIC_API_KEY");
    expect(payload).not.toHaveProperty("ANTHROPIC_AUTH_TOKEN");
  });
});

describe("buildCredentialsPayload — standalone secret deletion (explicit remove)", () => {
  it("sends an explicitly removed standalone secret as empty (delete signal)", () => {
    const spec = getCredentialFormSpec("loopal");
    const state = emptyFormState(spec);
    state.removedKeys = ["OPENAI_API_KEY"];
    const payload = buildCredentialsPayload(spec, state, [
      "ANTHROPIC_API_KEY",
      "OPENAI_API_KEY",
    ]);
    // Explicit remove → "" delete signal; the untouched sibling is omitted (keep).
    expect(payload["OPENAI_API_KEY"]).toBe("");
    expect(payload).not.toHaveProperty("ANTHROPIC_API_KEY");
  });

  it("omits a blank, not-removed standalone secret (keep current)", () => {
    const spec = getCredentialFormSpec("loopal");
    const state = emptyFormState(spec);
    const payload = buildCredentialsPayload(spec, state, ["ANTHROPIC_API_KEY"]);
    expect(payload).not.toHaveProperty("ANTHROPIC_API_KEY");
  });

  it("sends a newly typed standalone secret as its value", () => {
    const spec = getCredentialFormSpec("loopal");
    const state = emptyFormState(spec);
    state.values["OPENAI_API_KEY"] = "sk-new";
    const payload = buildCredentialsPayload(spec, state);
    expect(payload["OPENAI_API_KEY"]).toBe("sk-new");
  });
});

describe("buildCredentialsPayload — custom-env row deletion", () => {
  it("sends a previously-configured custom key as empty when its row was removed", () => {
    const spec = getCredentialFormSpec("codex-cli");
    const state = emptyFormState(spec); // no custom rows → the row was removed
    const payload = buildCredentialsPayload(spec, state, [
      "OPENAI_API_KEY",
      "HTTP_PROXY",
    ]);
    // The declared secret follows the "blank = keep" rule (omitted); the removed
    // custom HTTP_PROXY row becomes a delete signal.
    expect(payload["HTTP_PROXY"]).toBe("");
    expect(payload).not.toHaveProperty("OPENAI_API_KEY");
  });

  it("keeps a present-but-blank custom row (value masked, not re-typed)", () => {
    const spec = getCredentialFormSpec("codex-cli");
    const state = emptyFormState(spec);
    state.customEnv = [{ id: "1", key: "HTTP_PROXY", value: "" }];
    const payload = buildCredentialsPayload(spec, state, ["HTTP_PROXY"]);
    expect(payload).not.toHaveProperty("HTTP_PROXY");
  });

  it("sends a custom row carrying a new value", () => {
    const spec = getCredentialFormSpec("codex-cli");
    const state = emptyFormState(spec);
    state.customEnv = [{ id: "1", key: "HTTP_PROXY", value: "http://p" }];
    const payload = buildCredentialsPayload(spec, state, ["HTTP_PROXY"]);
    expect(payload["HTTP_PROXY"]).toBe("http://p");
  });
});

describe("initFormStateFromProfile — custom-env rebuild unions both slots", () => {
  it("rebuilds a custom row for a non-secret key arriving via configured_values", () => {
    const spec = getCredentialFormSpec("codex-cli"); // declares OPENAI_API_KEY
    const state = initFormStateFromProfile(spec, {
      id: 1,
      agent_slug: "codex-cli",
      name: "x",
      is_default: false,
      is_active: true,
      created_at: "",
      updated_at: "",
      configured_fields: ["OPENAI_API_KEY"],
      configured_values: { CUSTOM_BASE: "https://x" },
    });
    const row = state.customEnv.find((e) => e.key === "CUSTOM_BASE");
    expect(row?.value).toBe("https://x");
    // A declared key never becomes a custom row.
    expect(state.customEnv.find((e) => e.key === "OPENAI_API_KEY")).toBeUndefined();
  });
});

describe("buildCredentialsPayload — prototype-named keys (no `in` swallow)", () => {
  it("emits a delete signal for a key named like an Object.prototype member", () => {
    const spec = getCredentialFormSpec("codex-cli");
    const state = emptyFormState(spec);
    // Stored custom keys named like Object.prototype members, all removed (rows
    // gone). On a plain {} `"toString" in values` is true and `out["__proto__"]=`
    // hits the proto setter, both of which would swallow the delete signal.
    const payload = buildCredentialsPayload(spec, state, [
      "toString",
      "constructor",
      "__proto__",
    ]);
    expect(payload["toString"]).toBe("");
    expect(payload["constructor"]).toBe("");
    // __proto__ must be an OWN "" property, not swallowed by the prototype setter.
    expect(Object.prototype.hasOwnProperty.call(payload, "__proto__")).toBe(true);
    expect(payload["__proto__"]).toBe("");
  });
});
