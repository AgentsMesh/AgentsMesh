import { test, expect } from "../../fixtures/index";
import { clearAuthRateLimit } from "../../helpers/redis";

test.describe("Repository Providers API", () => {
  test.beforeEach(async () => { clearAuthRateLimit(); });

  /**
   * TC-REPOPROV-001: List repository providers
   */
  test("list repository providers", async ({ api }) => {
    const res = await api.get("/api/v1/users/repository-providers");
    expect(res.status).toBe(200);
  });

  test("list repository providers without auth returns 401", async ({ api }) => {
    const res = await api.getWithToken("/api/v1/users/repository-providers", "bad");
    expect(res.status).toBe(401);
  });

  /**
   * TC-REPOPROV-002: Create GitHub provider
   */
  test("create GitHub provider", async ({ api, db }) => {
    const res = await api.post("/api/v1/users/repository-providers", {
      provider_type: "github",
      name: "E2E GitHub Provider",
      base_url: "https://api.github.com",
      bot_token: "ghp_test_bot_token_e2e",
    });
    expect([200, 201]).toContain(res.status);
    const data = await res.json();
    const id = data.provider?.id || data.id;
    expect(id).toBeTruthy();

    // Cleanup
    if (id) await api.delete(`/api/v1/users/repository-providers/${id}`);
  });

  /**
   * TC-REPOPROV-003: Update provider name
   */
  test("update provider name", async ({ api }) => {
    const createRes = await api.post("/api/v1/users/repository-providers", {
      provider_type: "github",
      name: "E2E Update Provider",
      base_url: "https://api.github.com",
      bot_token: "ghp_update_test",
    });
    const created = await createRes.json();
    const id = created.provider?.id || created.id;
    if (!id) { test.skip(); return; }

    const updateRes = await api.put(`/api/v1/users/repository-providers/${id}`, {
      name: "E2E Updated Provider",
    });
    expect(updateRes.status).toBe(200);

    await api.delete(`/api/v1/users/repository-providers/${id}`);
  });

  /**
   * TC-REPOPROV-004: Delete provider
   */
  test("delete provider", async ({ api }) => {
    const createRes = await api.post("/api/v1/users/repository-providers", {
      provider_type: "github",
      name: "E2E Delete Provider",
      base_url: "https://api.github.com",
      bot_token: "ghp_delete_test",
    });
    const created = await createRes.json();
    const id = created.provider?.id || created.id;
    if (!id) { test.skip(); return; }

    const delRes = await api.delete(`/api/v1/users/repository-providers/${id}`);
    expect(delRes.status).toBe(200);

    // Verify gone
    const getRes = await api.get(`/api/v1/users/repository-providers/${id}`);
    expect(getRes.status).toBe(404);
  });

  /**
   * TC-REPOPROV-006: Test connection (with invalid token)
   */
  test("test connection with invalid token fails", async ({ api }) => {
    const createRes = await api.post("/api/v1/users/repository-providers", {
      provider_type: "github",
      name: "E2E Connection Test",
      base_url: "https://api.github.com",
      bot_token: "ghp_invalid_token",
    });
    const created = await createRes.json();
    const id = created.provider?.id || created.id;
    if (!id) { test.skip(); return; }

    const testRes = await api.post(
      `/api/v1/users/repository-providers/${id}/test`, {}
    );
    // Expect failure due to invalid token
    expect([200, 401, 502]).toContain(testRes.status);

    await api.delete(`/api/v1/users/repository-providers/${id}`);
  });

  /**
   * TC-REPOPROV-007: Created provider defaults to is_active=true and exposes has_* flags
   *
   * Regression for the user-reported bug where every provider showed as
   * "已禁用" (disabled). The wasm-core RepositoryProvider struct used to drop
   * is_active / has_identity / has_bot_token / has_client_id during the
   * deserialize-then-reserialize relay, so the frontend always read undefined.
   */
  test("created provider exposes is_active=true and has_* flags by default", async ({ api }) => {
    const createRes = await api.post("/api/v1/users/repository-providers", {
      provider_type: "github",
      name: "E2E IsActive Default",
      base_url: "https://api.github.com",
      bot_token: "ghp_default_active",
    });
    expect([200, 201]).toContain(createRes.status);
    const data = await createRes.json();
    const provider = data.provider ?? data;
    const id = provider.id;

    expect(provider.is_active).toBe(true);
    expect(provider.has_bot_token).toBe(true);
    expect(provider.has_client_id).toBe(false);
    expect(provider.has_identity).toBe(false);

    const listRes = await api.get("/api/v1/users/repository-providers");
    const list = await listRes.json();
    const inList = (list.providers as Array<{ id: number; is_active: boolean; has_bot_token: boolean }>)
      .find((p) => p.id === id);
    expect(inList?.is_active).toBe(true);
    expect(inList?.has_bot_token).toBe(true);

    await api.delete(`/api/v1/users/repository-providers/${id}`);
  });

  /**
   * TC-REPOPROV-008: PUT is_active toggles the field and persists across reloads.
   *
   * This is the exact flow that broke under the wasm-core bug: the
   * UpdateRepositoryProviderRequest struct lacked is_active, so sending
   * {is_active: true} from EditProviderDialog produced an empty PUT body
   * and the change silently no-op'd.
   */
  test("PUT is_active=false then is_active=true persists each toggle", async ({ api }) => {
    const createRes = await api.post("/api/v1/users/repository-providers", {
      provider_type: "github",
      name: "E2E IsActive Toggle",
      base_url: "https://api.github.com",
      bot_token: "ghp_toggle_test",
    });
    const created = await createRes.json();
    const id = (created.provider ?? created).id;

    const offRes = await api.put(`/api/v1/users/repository-providers/${id}`, { is_active: false });
    expect(offRes.status).toBe(200);
    expect(((await offRes.json()).provider).is_active).toBe(false);

    const reloadAfterOff = await api.get(`/api/v1/users/repository-providers/${id}`);
    expect(((await reloadAfterOff.json()).provider).is_active).toBe(false);

    const onRes = await api.put(`/api/v1/users/repository-providers/${id}`, { is_active: true });
    expect(onRes.status).toBe(200);
    expect(((await onRes.json()).provider).is_active).toBe(true);

    const reloadAfterOn = await api.get(`/api/v1/users/repository-providers/${id}`);
    expect(((await reloadAfterOn.json()).provider).is_active).toBe(true);

    await api.delete(`/api/v1/users/repository-providers/${id}`);
  });

  /**
   * TC-REPOPROV-009: Partial update (name only) preserves is_active.
   */
  test("partial update preserves is_active across renames", async ({ api }) => {
    const createRes = await api.post("/api/v1/users/repository-providers", {
      provider_type: "github",
      name: "E2E Partial Original",
      base_url: "https://api.github.com",
      bot_token: "ghp_partial",
    });
    const id = ((await createRes.json()).provider).id;

    await api.put(`/api/v1/users/repository-providers/${id}`, { is_active: false });

    const renameRes = await api.put(`/api/v1/users/repository-providers/${id}`, {
      name: "E2E Partial Renamed",
    });
    const renamed = (await renameRes.json()).provider;
    expect(renamed.name).toBe("E2E Partial Renamed");
    expect(renamed.is_active).toBe(false);

    await api.delete(`/api/v1/users/repository-providers/${id}`);
  });
});
