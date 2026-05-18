export const INSTALL_COMMANDS = {
  unix: "curl -fsSL https://agentsmesh.ai/install.sh | sh",
  windows: "irm https://agentsmesh.ai/install.ps1 | iex",
} as const;
