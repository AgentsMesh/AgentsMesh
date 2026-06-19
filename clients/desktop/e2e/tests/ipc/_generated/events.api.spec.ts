// AUTO-GENERATED — do not edit by hand. Regenerate: pnpm --filter desktop e2e:gen
import { test } from "../../../fixtures/electron-shared.fixture";
import { invokeIpcContract } from "../../../helpers/ipc-contract";

test.describe.configure({ mode: "serial" });

test.describe("IPC · events", () => {
  test("eventsConnect", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "eventsConnect", returnType: "void" });
  });

  test("eventsDisconnect", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "eventsDisconnect", returnType: "void" });
  });

  test("eventsGetConnectionState", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "eventsGetConnectionState", returnType: "string" });
  });

  test("eventsGetTick", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "eventsGetTick", returnType: "number" });
  });

  test("eventsNudge", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "eventsNudge", returnType: "void" });
  });

  test("eventsUnsubscribe", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "eventsUnsubscribe", returnType: "void" }, 0);
  });
});
