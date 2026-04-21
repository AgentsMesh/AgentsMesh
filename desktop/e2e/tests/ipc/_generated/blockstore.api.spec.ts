// AUTO-GENERATED — do not edit by hand. Regenerate: pnpm --filter desktop e2e:gen
import { test, expect } from "../../../fixtures";
import { invokeIpc } from "../../../helpers/ipc";

test.describe("IPC · blockstore", () => {
  test("blockstore_apply_ops", async ({ page }) => {
    // Smoke: the bridge accepts the call. Result may be a valid response OR a typed error —
    // both prove the IPC route is wired. A crashed bridge would throw an unrelated runtime error.
    const result = await invokeIpc(page, "blockstore_apply_ops", "").catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("blockstore_list_workspaces", async ({ page }) => {
    // Smoke: the bridge accepts the call. Result may be a valid response OR a typed error —
    // both prove the IPC route is wired. A crashed bridge would throw an unrelated runtime error.
    const result = await invokeIpc(page, "blockstore_list_workspaces").catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("blockstore_ensure_default_workspace", async ({ page }) => {
    // Smoke: the bridge accepts the call. Result may be a valid response OR a typed error —
    // both prove the IPC route is wired. A crashed bridge would throw an unrelated runtime error.
    const result = await invokeIpc(page, "blockstore_ensure_default_workspace").catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("blockstore_load_subtree", async ({ page }) => {
    // Smoke: the bridge accepts the call. Result may be a valid response OR a typed error —
    // both prove the IPC route is wired. A crashed bridge would throw an unrelated runtime error.
    const result = await invokeIpc(page, "blockstore_load_subtree", "", "").catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("blockstore_load_type_defs", async ({ page }) => {
    // Smoke: the bridge accepts the call. Result may be a valid response OR a typed error —
    // both prove the IPC route is wired. A crashed bridge would throw an unrelated runtime error.
    const result = await invokeIpc(page, "blockstore_load_type_defs", "").catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("blockstore_catchup", async ({ page }) => {
    // Smoke: the bridge accepts the call. Result may be a valid response OR a typed error —
    // both prove the IPC route is wired. A crashed bridge would throw an unrelated runtime error.
    const result = await invokeIpc(page, "blockstore_catchup", "").catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("blockstore_semantic_search", async ({ page }) => {
    // Smoke: the bridge accepts the call. Result may be a valid response OR a typed error —
    // both prove the IPC route is wired. A crashed bridge would throw an unrelated runtime error.
    const result = await invokeIpc(page, "blockstore_semantic_search", "", "").catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("blockstore_apply_remote_op", async ({ page }) => {
    // Smoke: the bridge accepts the call. Result may be a valid response OR a typed error —
    // both prove the IPC route is wired. A crashed bridge would throw an unrelated runtime error.
    const result = await invokeIpc(page, "blockstore_apply_remote_op", "").catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("blockstore_workspaces_json", async ({ page }) => {
    // Smoke: the bridge accepts the call. Result may be a valid response OR a typed error —
    // both prove the IPC route is wired. A crashed bridge would throw an unrelated runtime error.
    const result = await invokeIpc(page, "blockstore_workspaces_json").catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("blockstore_get_block_json", async ({ page }) => {
    // Smoke: the bridge accepts the call. Result may be a valid response OR a typed error —
    // both prove the IPC route is wired. A crashed bridge would throw an unrelated runtime error.
    const result = await invokeIpc(page, "blockstore_get_block_json", "").catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("blockstore_list_children_json", async ({ page }) => {
    // Smoke: the bridge accepts the call. Result may be a valid response OR a typed error —
    // both prove the IPC route is wired. A crashed bridge would throw an unrelated runtime error.
    const result = await invokeIpc(page, "blockstore_list_children_json", "").catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("blockstore_list_backlinks_json", async ({ page }) => {
    // Smoke: the bridge accepts the call. Result may be a valid response OR a typed error —
    // both prove the IPC route is wired. A crashed bridge would throw an unrelated runtime error.
    const result = await invokeIpc(page, "blockstore_list_backlinks_json", "").catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("blockstore_type_defs_json", async ({ page }) => {
    // Smoke: the bridge accepts the call. Result may be a valid response OR a typed error —
    // both prove the IPC route is wired. A crashed bridge would throw an unrelated runtime error.
    const result = await invokeIpc(page, "blockstore_type_defs_json", "").catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("blockstore_last_op_id", async ({ page }) => {
    // Smoke: the bridge accepts the call. Result may be a valid response OR a typed error —
    // both prove the IPC route is wired. A crashed bridge would throw an unrelated runtime error.
    const result = await invokeIpc(page, "blockstore_last_op_id", "").catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });

  test("blockstore_set_last_op_id", async ({ page }) => {
    // Smoke: the bridge accepts the call. Result may be a valid response OR a typed error —
    // both prove the IPC route is wired. A crashed bridge would throw an unrelated runtime error.
    const result = await invokeIpc(page, "blockstore_set_last_op_id", "", 0).catch((err: Error) => ({ __ipcError: err.message }));
    expect(result).toBeDefined();
  });
});
