/**
 * Interaction mode constants — canonical source for frontend.
 * Mirrors podfile.ModePTY / podfile.ModeACP on the backend.
 */
export const POD_MODE_PTY = "pty" as const;
export const POD_MODE_ACP = "acp" as const;

export type PodMode = typeof POD_MODE_PTY | typeof POD_MODE_ACP;
