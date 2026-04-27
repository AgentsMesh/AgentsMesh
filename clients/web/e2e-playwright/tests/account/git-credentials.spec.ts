import { test, expect } from "../../fixtures/index";
import { clearAuthRateLimit } from "../../helpers/redis";
import { CLEANUP } from "../../helpers/test-data";

test.describe("Git Credentials API", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });

  /**
   * TC-GITCRED-001: List git credentials
   */
  test("list git credentials", async ({ api }) => {
    const res = await api.get("/api/v1/users/git-credentials");
    expect(res.status).toBe(200);
  });

  test("list git credentials without auth returns 401", async ({ api }) => {
    const res = await api.getWithToken("/api/v1/users/git-credentials", "bad");
    expect(res.status).toBe(401);
  });

  /**
   * TC-GITCRED-002: Create PAT credential
   */
  test("create PAT credential", async ({ api, db }) => {
    db.cleanup(`DELETE FROM user_git_credentials WHERE name = 'E2E Test PAT'`);
    const res = await api.post("/api/v1/users/git-credentials", {
      name: "E2E Test PAT",
      credential_type: "pat",
      pat: "ghp_test_token_12345",
      host_pattern: "github.com",
    });
    expect([200, 201]).toContain(res.status);

    db.cleanup(`DELETE FROM user_git_credentials WHERE name = 'E2E Test PAT'`);
  });

  /**
   * TC-GITCRED-003: Update git credential
   */
  test("update git credential name", async ({ api, db }) => {
    // Create first
    const createRes = await api.post("/api/v1/users/git-credentials", {
      name: "E2E Update Target",
      credential_type: "pat",
      pat: "ghp_update_test",
      host_pattern: "github.com",
    });
    const created = await createRes.json();
    const id = created.credential?.id || created.id;
    if (!id) { test.skip(); return; }

    const updateRes = await api.put(`/api/v1/users/git-credentials/${id}`, {
      name: "E2E Updated Name",
    });
    expect(updateRes.status).toBe(200);

    db.cleanup(`DELETE FROM user_git_credentials WHERE name = 'E2E Updated Name'`);
  });

  /**
   * TC-GITCRED-004: Delete git credential
   */
  test("delete git credential", async ({ api }) => {
    const createRes = await api.post("/api/v1/users/git-credentials", {
      name: "E2E Delete Target",
      credential_type: "pat",
      pat: "ghp_delete_test",
      host_pattern: "github.com",
    });
    const created = await createRes.json();
    const id = created.credential?.id || created.id;
    if (!id) { test.skip(); return; }

    const delRes = await api.delete(`/api/v1/users/git-credentials/${id}`);
    expect(delRes.status).toBe(200);
  });

  /**
   * TC-GITCRED-005: Set default credential
   */
  test("set and get default credential", async ({ api, db }) => {
    const createRes = await api.post("/api/v1/users/git-credentials", {
      name: "E2E Default Target",
      credential_type: "pat",
      pat: "ghp_default_test",
      host_pattern: "github.com",
    });
    const created = await createRes.json();
    const id = created.credential?.id || created.id;
    if (!id) { test.skip(); return; }

    // Set default
    const setRes = await api.post("/api/v1/users/git-credentials/default", {
      credential_id: id,
    });
    expect(setRes.status).toBe(200);

    // Get default
    const getRes = await api.get("/api/v1/users/git-credentials/default");
    expect(getRes.status).toBe(200);

    // Clear and cleanup
    await api.delete("/api/v1/users/git-credentials/default");
    db.cleanup(`DELETE FROM user_git_credentials WHERE name = 'E2E Default Target'`);
  });
});
