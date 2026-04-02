/**
 * Utilities for generating PodFile Layer source from form fields.
 * A PodFile Layer is a DSL fragment that configures a Pod's environment.
 */

import { POD_MODE_PTY } from "@/lib/pod-modes";

/**
 * Escape a string for use in a PodFile quoted value.
 * Must align with backend FormatStringLiteral (podfile/format.go).
 */
function escapePodfileString(s: string): string {
  return s
    .replace(/\\/g, "\\\\")
    .replace(/"/g, '\\"')
    .replace(/\n/g, "\\n")
    .replace(/\t/g, "\\t");
}

/**
 * Escape and quote a string value for PodFile syntax.
 * Must align with backend FormatStringLiteral (podfile/format.go).
 */
function formatPodfileValue(value: unknown): string {
  if (typeof value === "string") return `"${escapePodfileString(value)}"`;
  if (typeof value === "boolean") return value ? "true" : "false";
  if (typeof value === "number") return String(value);
  return `"${escapePodfileString(String(value))}"`;
}

/**
 * Build a PodFile Layer source string from structured form parameters.
 * Each non-empty field is emitted as a PodFile declaration line.
 */
export function buildPodfileLayer(params: {
  configValues: Record<string, unknown>;
  repositorySlug?: string;
  branchName?: string;
  interactionMode?: string;
  credentialProfileName?: string;
  prompt?: string;
}): string {
  const lines: string[] = [];

  // MODE declaration (if not default PTY)
  if (params.interactionMode && params.interactionMode !== POD_MODE_PTY) {
    lines.push(`MODE ${params.interactionMode}`);
  }

  // CREDENTIAL declaration (profile name; omit for runner_host default)
  if (params.credentialProfileName) {
    lines.push(`CREDENTIAL "${escapePodfileString(params.credentialProfileName)}"`);
  }

  // PROMPT declaration (initial prompt content)
  if (params.prompt) {
    lines.push(`PROMPT "${escapePodfileString(params.prompt)}"`);
  }

  // CONFIG declarations
  for (const [key, value] of Object.entries(params.configValues)) {
    if (value !== undefined && value !== null && value !== "") {
      lines.push(`CONFIG ${key} = ${formatPodfileValue(value)}`);
    }
  }

  // Repository slug / branch
  if (params.repositorySlug) {
    lines.push(`REPO "${params.repositorySlug}"`);
  }
  if (params.branchName) {
    lines.push(`BRANCH "${params.branchName}"`);
  }

  return lines.join("\n");
}
