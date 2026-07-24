// AUTO-GENERATED — do not edit by hand. Regenerate: pnpm --filter desktop e2e:gen
import { test } from "../../../fixtures/electron-shared.fixture";
import { invokeIpcContract } from "../../../helpers/ipc-contract";

test.describe.configure({ mode: "serial" });

test.describe("IPC · channel", () => {
  test("channelArchiveChannelConnect", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "channelArchiveChannelConnect", returnType: "Array<number>" }, []);
  });

  test("channelCreateChannelConnect", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "channelCreateChannelConnect", returnType: "Array<number>" }, []);
  });

  test("channelDeleteChannelMessageConnect", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "channelDeleteChannelMessageConnect", returnType: "Array<number>" }, []);
  });

  test("channelEditChannelMessageConnect", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "channelEditChannelMessageConnect", returnType: "Array<number>" }, []);
  });

  test("channelGetChannelConnect", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "channelGetChannelConnect", returnType: "Array<number>" }, []);
  });

  test("channelGetChannelUnreadCountsConnect", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "channelGetChannelUnreadCountsConnect", returnType: "Array<number>" }, []);
  });

  test("channelGetMessageReadByConnect", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "channelGetMessageReadByConnect", returnType: "Array<number>" }, []);
  });

  test("channelListChannelMembersConnect", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "channelListChannelMembersConnect", returnType: "Array<number>" }, []);
  });

  test("channelListChannelMessagesConnect", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "channelListChannelMessagesConnect", returnType: "Array<number>" }, []);
  });

  test("channelListChannelsConnect", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "channelListChannelsConnect", returnType: "Array<number>" }, []);
  });

  test("channelMarkChannelReadConnect", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "channelMarkChannelReadConnect", returnType: "Array<number>" }, []);
  });

  test("channelMarkChannelUnreadConnect", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "channelMarkChannelUnreadConnect", returnType: "Array<number>" }, []);
  });

  test("channelMuteChannelConnect", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "channelMuteChannelConnect", returnType: "Array<number>" }, []);
  });

  test("channelPinChannelConnect", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "channelPinChannelConnect", returnType: "Array<number>" }, []);
  });

  test("channelSendChannelMessageConnect", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "channelSendChannelMessageConnect", returnType: "Array<number>" }, []);
  });

  test("channelUnarchiveChannelConnect", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "channelUnarchiveChannelConnect", returnType: "Array<number>" }, []);
  });

  test("channelUpdateChannelConnect", async ({ sharedPage }) => {
    await invokeIpcContract(sharedPage, { method: "channelUpdateChannelConnect", returnType: "Array<number>" }, []);
  });
});
