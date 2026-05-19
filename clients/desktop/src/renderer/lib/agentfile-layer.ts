
import { POD_MODE_PTY } from "@/lib/pod-modes";

// MUST align with backend FormatStringLiteral (agentfile/format.go).
function escapeAgentfileString(s: string): string {
  return s
    .replace(/\\/g, "\\\\")
    .replace(/"/g, '\\"')
    .replace(/\n/g, "\\n")
    .replace(/\t/g, "\\t");
}

function formatAgentfileValue(value: unknown): string {
  if (typeof value === "string") return `"${escapeAgentfileString(value)}"`;
  if (typeof value === "boolean") return value ? "true" : "false";
  if (typeof value === "number") return String(value);
  return `"${escapeAgentfileString(String(value))}"`;
}

export function buildAgentfileLayer(params: {
  configValues: Record<string, unknown>;
  repositorySlug?: string;
  branchName?: string;
  interactionMode?: string;
  credentialProfileName?: string;
  prompt?: string;
}): string {
  const lines: string[] = [];

  if (params.interactionMode && params.interactionMode !== POD_MODE_PTY) {
    lines.push(`MODE ${params.interactionMode}`);
  }

  if (params.credentialProfileName) {
    lines.push(`CREDENTIAL "${escapeAgentfileString(params.credentialProfileName)}"`);
  }

  if (params.prompt) {
    lines.push(`PROMPT "${escapeAgentfileString(params.prompt)}"`);
  }

  for (const [key, value] of Object.entries(params.configValues)) {
    if (value !== undefined && value !== null && value !== "") {
      lines.push(`CONFIG ${key} = ${formatAgentfileValue(value)}`);
    }
  }

  if (params.repositorySlug) {
    lines.push(`REPO "${params.repositorySlug}"`);
  }
  if (params.branchName) {
    lines.push(`BRANCH "${params.branchName}"`);
  }

  return lines.join("\n");
}
