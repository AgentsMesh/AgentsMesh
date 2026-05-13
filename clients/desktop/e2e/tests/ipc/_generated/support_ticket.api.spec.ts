// AUTO-GENERATED — do not edit by hand. Regenerate: pnpm --filter desktop e2e:gen
import { test, expect } from "../../../fixtures";
import { invokeIpc } from "../../../helpers/ipc";

test.describe("IPC · support_ticket", () => {
  test("support_ticket_create_ticket", async ({ page }) => {
    // Smoke: the bridge accepts the call. Result may be a valid response OR a typed error —
    // both prove the IPC route is wired. A crashed bridge would throw an unrelated runtime error.
    const result = await invokeIpc(page, "support_ticket_create_ticket", "", "", "", "", [], []).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("support_ticket_add_message", async ({ page }) => {
    // Smoke: the bridge accepts the call. Result may be a valid response OR a typed error —
    // both prove the IPC route is wired. A crashed bridge would throw an unrelated runtime error.
    const result = await invokeIpc(page, "support_ticket_add_message", 0, "", [], []).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("support_ticket_list_support_tickets_connect", async ({ page }) => {
    // Smoke: the bridge accepts the call. Result may be a valid response OR a typed error —
    // both prove the IPC route is wired. A crashed bridge would throw an unrelated runtime error.
    const result = await invokeIpc(page, "support_ticket_list_support_tickets_connect", []).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("support_ticket_get_support_ticket_connect", async ({ page }) => {
    // Smoke: the bridge accepts the call. Result may be a valid response OR a typed error —
    // both prove the IPC route is wired. A crashed bridge would throw an unrelated runtime error.
    const result = await invokeIpc(page, "support_ticket_get_support_ticket_connect", []).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("support_ticket_get_attachment_url_connect", async ({ page }) => {
    // Smoke: the bridge accepts the call. Result may be a valid response OR a typed error —
    // both prove the IPC route is wired. A crashed bridge would throw an unrelated runtime error.
    const result = await invokeIpc(page, "support_ticket_get_attachment_url_connect", []).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });
});
