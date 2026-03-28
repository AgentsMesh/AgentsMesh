/**
 * Utilities for generating PodFile Layer source from form fields.
 * A PodFile Layer is a DSL fragment that configures a Pod's environment.
 */

/**
 * Escape and quote a string value for PodFile syntax.
 */
function formatPodfileValue(value: unknown): string {
  if (typeof value === "string")
    return `"${value.replace(/\\/g, "\\\\").replace(/"/g, '\\"')}"`;
  if (typeof value === "boolean") return value ? "true" : "false";
  if (typeof value === "number") return String(value);
  return `"${String(value)}"`;
}

/**
 * Build a PodFile Layer source string from structured form parameters.
 * Each non-empty field is emitted as a PodFile declaration line.
 */
export function buildPodfileLayer(params: {
  configValues: Record<string, unknown>;
  repositoryUrl?: string;
  branchName?: string;
  credentialType?: string;
}): string {
  const lines: string[] = [];

  // CONFIG declarations
  for (const [key, value] of Object.entries(params.configValues)) {
    if (value !== undefined && value !== null && value !== "") {
      lines.push(`CONFIG ${key} = ${formatPodfileValue(value)}`);
    }
  }

  // Repository / branch / credential
  if (params.repositoryUrl) {
    lines.push(`REPO "${params.repositoryUrl}"`);
  }
  if (params.branchName) {
    lines.push(`BRANCH "${params.branchName}"`);
  }
  if (params.credentialType) {
    lines.push(`GIT_CREDENTIAL ${params.credentialType}`);
  }

  return lines.join("\n");
}
