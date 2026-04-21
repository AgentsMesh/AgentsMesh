import { test, expect } from "../../fixtures";
import { ChannelsPage } from "../../pages/channels.page";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { gotoHash } from "../../helpers/nav";

// End-to-end: create a channel via IPC, navigate to it, type a message,
// send, assert the message appears in the list. Regression: desktop had
// no spec exercising the send path, so the archived-channel 400 + any
// future send regressions (schema drift, token loss, etc.) would only
// surface at runtime.
test("Channels · send message appears in list", async ({ page }) => {
  const name = `e2e-send-${Date.now()}`;

  // Create a fresh (non-archived) channel via IPC.
  const createResult = await page.evaluate(async (chName) => {
    const api = (window as unknown as {
      electronAPI: { invoke: (ch: string, ...a: unknown[]) => Promise<unknown> };
    }).electronAPI;
    const reqJson = JSON.stringify({ name: chName, visibility: "public" });
    const json = await api.invoke("channelCreateChannel", reqJson) as string;
    return JSON.parse(json) as { id: number };
  }, name);

  const channelId = createResult.id;
  expect(channelId).toBeGreaterThan(0);

  await gotoHash(page, `/${TEST_ORG_SLUG}/channels/${channelId}`);

  const channels = new ChannelsPage(page);
  await expect(channels.messageInput).toBeVisible({ timeout: 10_000 });

  const text = `hello from e2e ${Date.now()}`;
  await channels.messageInput.fill(text);
  await channels.sendButton.click();

  // Message list should eventually contain the text.
  await expect(page.getByText(text).first()).toBeVisible({ timeout: 10_000 });
});
