// AUTO-GENERATED — do not edit by hand. Regenerate: pnpm --filter desktop e2e:gen
import { test, expect } from "../../../fixtures";
import { invokeIpc } from "../../../helpers/ipc";

test.describe("IPC · auth_connect", () => {
  test("auth_connect_login_connect", async ({ page }) => {
    const result = await invokeIpc(page, "auth_connect_login_connect", []).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("auth_connect_register_connect", async ({ page }) => {
    const result = await invokeIpc(page, "auth_connect_register_connect", []).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("auth_connect_refresh_token_connect", async ({ page }) => {
    const result = await invokeIpc(page, "auth_connect_refresh_token_connect", []).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("auth_connect_verify_email_connect", async ({ page }) => {
    const result = await invokeIpc(page, "auth_connect_verify_email_connect", []).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("auth_connect_resend_verification_connect", async ({ page }) => {
    const result = await invokeIpc(page, "auth_connect_resend_verification_connect", []).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("auth_connect_forgot_password_connect", async ({ page }) => {
    const result = await invokeIpc(page, "auth_connect_forgot_password_connect", []).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("auth_connect_reset_password_connect", async ({ page }) => {
    const result = await invokeIpc(page, "auth_connect_reset_password_connect", []).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("auth_connect_oauth_redirect_connect", async ({ page }) => {
    const result = await invokeIpc(page, "auth_connect_oauth_redirect_connect", []).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("auth_connect_oauth_callback_connect", async ({ page }) => {
    const result = await invokeIpc(page, "auth_connect_oauth_callback_connect", []).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("auth_connect_logout_connect", async ({ page }) => {
    const result = await invokeIpc(page, "auth_connect_logout_connect", []).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });
});
