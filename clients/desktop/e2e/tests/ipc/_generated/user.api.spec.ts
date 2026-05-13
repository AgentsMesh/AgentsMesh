// AUTO-GENERATED — do not edit by hand. Regenerate: pnpm --filter desktop e2e:gen
import { test, expect } from "../../../fixtures";
import { invokeIpc } from "../../../helpers/ipc";

test.describe("IPC · user", () => {
  test("user_get_me_connect", async ({ page }) => {
    const result = await invokeIpc(page, "user_get_me_connect", []).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("user_update_me_connect", async ({ page }) => {
    const result = await invokeIpc(page, "user_update_me_connect", []).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("user_change_password_connect", async ({ page }) => {
    const result = await invokeIpc(page, "user_change_password_connect", []).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("user_list_identities_connect", async ({ page }) => {
    const result = await invokeIpc(page, "user_list_identities_connect", []).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("user_delete_identity_connect", async ({ page }) => {
    const result = await invokeIpc(page, "user_delete_identity_connect", []).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("user_search_users_connect", async ({ page }) => {
    const result = await invokeIpc(page, "user_search_users_connect", []).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });
});
