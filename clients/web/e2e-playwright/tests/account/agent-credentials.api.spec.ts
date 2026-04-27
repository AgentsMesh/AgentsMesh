import { test, expect } from "../../fixtures/index";
import { clearAuthRateLimit } from "../../helpers/redis";

test.describe("Agent Credentials API", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });

  /**
   * TC-AGENTCRED-001: List agent credentials
   */
  test("list agent credentials", async ({ api }) => {
    const res = await api.get("/api/v1/users/agent-credentials");
    expect(res.status).toBe(200);
  });

  test("list agent credentials without auth returns 401", async ({ api }) => {
    const res = await api.getWithToken("/api/v1/users/agent-credentials", "bad");
    expect(res.status).toBe(401);
  });

  /**
   * TC-AGENTCRED-002: Create agent credential profile
   */
  test("create agent credential profile", async ({ api, db }) => {
    db.cleanup(
      `DELETE FROM user_agent_credential_profiles WHERE name = 'E2E Agent Cred Test'`
    );
    // Use the correct route: /agents/:slug instead of /types/:slug
    const res = await api.post("/api/v1/users/agent-credentials/agents/claude-code", {
      name: "E2E Agent Cred Test",
      description: "E2E test credential",
      is_runner_host: false,
      credentials: { ANTHROPIC_API_KEY: "sk-ant-test-e2e" },
    });
    expect([200, 201]).toContain(res.status);

    db.cleanup(
      `DELETE FROM user_agent_credential_profiles WHERE name = 'E2E Agent Cred Test'`
    );
  });

  /**
   * TC-AGENTCRED-004: Delete agent credential
   */
  test("delete non-existent agent credential returns 404", async ({ api }) => {
    const res = await api.delete("/api/v1/users/agent-credentials/profiles/999999");
    expect(res.status).toBe(404);
  });
});
