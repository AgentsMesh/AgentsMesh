/**
 * Centralized test data constants and factory functions.
 * Generates unique names to avoid collisions in parallel/repeated runs.
 */

let counter = 0;

/** Generate a unique suffix for test entities. */
export function uniqueSuffix(): string {
  return `${Date.now()}-${++counter}`;
}

/** Generate a unique test email address. */
export function uniqueEmail(prefix = "e2e"): string {
  return `${prefix}-${uniqueSuffix()}@test.local`;
}

/** Common cleanup SQL fragments. */
export const CLEANUP = {
  /** Delete a user and their org memberships by email. */
  userByEmail: (email: string) => `
    DELETE FROM organization_members WHERE user_id IN (SELECT id FROM users WHERE email = '${email}');
    DELETE FROM users WHERE email = '${email}';
  `.trim(),

  /** Delete a user, their orgs (via membership), and memberships by email. */
  userAndOrgsByEmail: (email: string) => `
    DELETE FROM organizations WHERE id IN (
      SELECT om.organization_id FROM organization_members om
      JOIN users u ON om.user_id = u.id WHERE u.email = '${email}'
    );
    DELETE FROM users WHERE email = '${email}';
  `.trim(),
} as const;

/** Password hash for 'password123' (bcrypt, used in test user inserts). */
export const HASH_PASSWORD123 =
  "$2a$10$N9qo8uLOickgx2ZMRZoMye.LrFO1VD3cWjvdCMEBzO4Y6bO7zE6.2";

/** Password hash for 'devpass123' (bcrypt, matches seed data). */
export const HASH_DEVPASS123 =
  "$2a$10$/95Zk1f1HFGXACwCb.bOw.d3vTjclw5NdGwQuK1Eaji6cDq0PuXp2";

/** Build a structured MessageContent object for sending channel messages. */
export function textContent(text: string) {
  return {
    kind: "text",
    blocks: [{ type: "paragraph", elements: [{ type: "text", text }] }],
  };
}
