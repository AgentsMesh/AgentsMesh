// AUTO-GENERATED — do not edit by hand. Regenerate: pnpm --filter desktop e2e:gen
import { test } from "../../../fixtures/electron-shared.fixture";
import { invokeIpcContract } from "../../../helpers/ipc-contract";

test.describe.configure({ mode: "serial" });

test.describe("IPC · uncategorized", () => {
  test("appAutopilotAppendIteration", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appAutopilotAppendIteration", returnType: "void" }, []);
  });

  test("appAutopilotApplyFetchedControllers", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appAutopilotApplyFetchedControllers", returnType: "void" }, []);
  });

  test("appAutopilotApplyFetchedCurrentController", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appAutopilotApplyFetchedCurrentController", returnType: "void" }, []);
  });

  test("appAutopilotApplyFetchedIterations", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appAutopilotApplyFetchedIterations", returnType: "void" }, "", []);
  });

  test("appAutopilotControllersJson", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appAutopilotControllersJson", returnType: "string" });
  });

  test("appAutopilotControllersProto", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appAutopilotControllersProto", returnType: "Array<number>" });
  });

  test("appAutopilotInsertController", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appAutopilotInsertController", returnType: "void" }, []);
  });

  test("appAutopilotIterationsJson", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appAutopilotIterationsJson", returnType: "string" }, "");
  });

  test("appAutopilotIterationsProto", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appAutopilotIterationsProto", returnType: "Array<number>" }, "");
  });

  test("appAutopilotPatchController", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appAutopilotPatchController", returnType: "void" }, []);
  });

  test("appAutopilotRemoveControllerProto", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appAutopilotRemoveControllerProto", returnType: "void" }, []);
  });

  test("appAutopilotSetCurrentControllerProto", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appAutopilotSetCurrentControllerProto", returnType: "void" }, []);
  });

  test("appAutopilotThinkingHistoryJson", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appAutopilotThinkingHistoryJson", returnType: "string" }, "");
  });

  test("appAutopilotThinkingJson", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appAutopilotThinkingJson", returnType: "string" }, "");
  });

  test("appAutopilotUpdateThinkingProto", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appAutopilotUpdateThinkingProto", returnType: "void" }, []);
  });

  test("appAvailableRunnersJson", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appAvailableRunnersJson", returnType: "string" });
  });

  test("appAvailableRunnersProto", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appAvailableRunnersProto", returnType: "Array<number>" });
  });

  test("appChannelApplyFetchedChannel", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appChannelApplyFetchedChannel", returnType: "void" }, []);
  });

  test("appChannelApplyFetchedChannels", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appChannelApplyFetchedChannels", returnType: "void" }, []);
  });

  test("appChannelApplyFetchedMembers", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appChannelApplyFetchedMembers", returnType: "void" }, 0, []);
  });

  test("appChannelApplyFetchedMessages", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appChannelApplyFetchedMessages", returnType: "void" }, 0, []);
  });

  test("appChannelApplyFetchedMessagesPrepend", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appChannelApplyFetchedMessagesPrepend", returnType: "void" }, 0, []);
  });

  test("appChannelApplyFetchedPods", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appChannelApplyFetchedPods", returnType: "void" }, 0, []);
  });

  test("appChannelApplyMessageEdited", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appChannelApplyMessageEdited", returnType: "void" }, []);
  });

  test("appChannelClearUnread", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appChannelClearUnread", returnType: "void" }, 0);
  });

  test("appChannelInsertChannel", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appChannelInsertChannel", returnType: "void" }, []);
  });

  test("appChannelInsertMessage", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appChannelInsertMessage", returnType: "void" }, []);
  });

  test("appChannelMentionCountsJson", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appChannelMentionCountsJson", returnType: "string" });
  });

  test("appChannelMessagesJson", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appChannelMessagesJson", returnType: "string" }, 0);
  });

  test("appChannelPatchMemberCount", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appChannelPatchMemberCount", returnType: "void" }, []);
  });

  test("appChannelPodsJson", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appChannelPodsJson", returnType: "string" }, 0);
  });

  test("appChannelRemoveMember", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appChannelRemoveMember", returnType: "void" }, []);
  });

  test("appChannelRemoveMessage", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appChannelRemoveMessage", returnType: "void" }, 0, 0);
  });

  test("appChannelReplaceMembers", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appChannelReplaceMembers", returnType: "void" }, []);
  });

  test("appChannelReplacePods", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appChannelReplacePods", returnType: "void" }, []);
  });

  test("appChannelReplaceUnreadCounts", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appChannelReplaceUnreadCounts", returnType: "void" }, []);
  });

  test("appChannelsJson", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appChannelsJson", returnType: "string" });
  });

  test("appChannelUnreadCountsJson", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appChannelUnreadCountsJson", returnType: "string" });
  });

  test("appCurrentRunnerJson", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appCurrentRunnerJson", returnType: "string" });
  });

  test("appCurrentRunnerProto", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appCurrentRunnerProto", returnType: "Array<number>" });
  });

  test("appGetMeshNodeJson", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appGetMeshNodeJson", returnType: "string" }, "");
  });

  test("appGetPodJson", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appGetPodJson", returnType: "string" }, "");
  });

  test("appGetPodProto", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appGetPodProto", returnType: "Array<number>" }, "");
  });

  test("appMeshReplaceTopology", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appMeshReplaceTopology", returnType: "void" }, []);
  });

  test("appPodApplyAppendedPods", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appPodApplyAppendedPods", returnType: "void" }, []);
  });

  test("appPodApplyFetchedPods", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appPodApplyFetchedPods", returnType: "void" }, []);
  });

  test("appPodInsertCreated", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appPodInsertCreated", returnType: "void" }, []);
  });

  test("appPodMarkTerminated", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appPodMarkTerminated", returnType: "void" }, []);
  });

  test("appPodPatchPerpetual", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appPodPatchPerpetual", returnType: "void" }, []);
  });

  test("appPodRemove", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appPodRemove", returnType: "void" }, "");
  });

  test("appPodsJson", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appPodsJson", returnType: "string" });
  });

  test("appRunnerApplyFetched", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appRunnerApplyFetched", returnType: "void" }, []);
  });

  test("appRunnerApplyFetchedAvailable", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appRunnerApplyFetchedAvailable", returnType: "void" }, []);
  });

  test("appRunnerApplyFetchedCurrent", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appRunnerApplyFetchedCurrent", returnType: "void" }, []);
  });

  test("appRunnerPatch", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appRunnerPatch", returnType: "void" }, []);
  });

  test("appRunnerRemove", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appRunnerRemove", returnType: "void" }, []);
  });

  test("appRunnerSetCurrent", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appRunnerSetCurrent", returnType: "void" }, []);
  });

  test("appRunnersJson", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appRunnersJson", returnType: "string" });
  });

  test("appRunnersProto", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appRunnersProto", returnType: "Array<number>" });
  });

  test("appSelectChannel", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appSelectChannel", returnType: "void" }, 0);
  });

  test("appSetCurrentChannel", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appSetCurrentChannel", returnType: "void" }, 0);
  });

  test("appSetCurrentUser", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "appSetCurrentUser", returnType: "void" }, 0);
  });

  test("relayDisconnect", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "relayDisconnect", returnType: "void" }, "");
  });

  test("relayDisconnectAll", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "relayDisconnectAll", returnType: "void" });
  });

  test("relayForceResize", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "relayForceResize", returnType: "void" }, "", 0, 0);
  });

  test("relayGetPodSize", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "relayGetPodSize", returnType: "Array<number>" }, "");
  });

  test("relayGetStatus", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "relayGetStatus", returnType: "string" }, "");
  });

  test("relayIsRunnerDisconnected", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "relayIsRunnerDisconnected", returnType: "boolean" }, "");
  });

  test("relaySend", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "relaySend", returnType: "void" }, "", "");
  });

  test("relaySendAcpCommand", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "relaySendAcpCommand", returnType: "void" }, "", "");
  });

  test("relaySendResize", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "relaySendResize", returnType: "void" }, "", 0, 0);
  });

  test("relayUnsubscribe", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "relayUnsubscribe", returnType: "void" }, "", "");
  });
});
