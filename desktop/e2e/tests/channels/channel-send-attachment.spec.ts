import { test, expect } from "../../fixtures";
import { ChannelsPage } from "../../pages/channels.page";
import { TEST_ORG_SLUG } from "../../helpers/env";
import { gotoHash } from "../../helpers/nav";
import { writeFileSync, mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

// Regression guard: sending a message with ONLY an attachment (no typed text)
// used to 400 because the frontend built an empty `elements: []` block and
// backend `extractBody` rejected it as empty. Fix: MessageInput falls back
// to `📎 ${filename}` when body would otherwise be empty.
test("Channels · attachment-only message sends without 400", async ({ page }) => {
  // Prepare a 1x1 PNG on disk so <input type="file"> can consume it.
  const dir = mkdtempSync(join(tmpdir(), "e2e-attach-"));
  const filePath = join(dir, "pixel.png");
  const pngBytes = Buffer.from(
    "89504e470d0a1a0a0000000d49484452000000010000000108060000001f15c4" +
    "8900000001735247420aaeced0000000097048597300000ec300000ec301c76f" +
    "a864000000184944415478da63fcffff3f03020403030303030303030303030303" +
    "dddddddd5a7c8e00000000049454e44ae426082",
    "hex",
  );
  writeFileSync(filePath, pngBytes);

  const name = `e2e-attach-${Date.now()}`;
  const createResult = await page.evaluate(async (chName) => {
    const api = (window as unknown as {
      electronAPI: { invoke: (ch: string, ...a: unknown[]) => Promise<unknown> };
    }).electronAPI;
    const json = await api.invoke(
      "channelCreateChannel",
      JSON.stringify({ name: chName, visibility: "public" }),
    ) as string;
    return JSON.parse(json) as { id: number };
  }, name);

  const channelId = createResult.id;
  await gotoHash(page, `/${TEST_ORG_SLUG}/channels/${channelId}`);

  const channels = new ChannelsPage(page);
  await expect(channels.messageInput).toBeVisible({ timeout: 10_000 });

  // Upload via the hidden file input inside MessageInputToolbar.
  const fileInput = page.locator('input[type="file"]').first();
  await fileInput.setInputFiles(filePath);

  // Wait until the attachment preview chip shows up (clear sign upload done).
  await expect(page.getByRole("button", { name: /remove/i })).toBeVisible({ timeout: 10_000 });

  // Send without typing any text.
  await channels.sendButton.click();

  // Assert no "HTTP 400" error got logged to console (the old failure mode).
  const consoleErrors: string[] = [];
  page.on("console", (msg) => {
    if (msg.type() === "error") consoleErrors.push(msg.text());
  });
  await page.waitForTimeout(1500);

  expect(
    consoleErrors.some((e) => /HTTP 400|VALIDATION_FAILED/i.test(e)),
    `attachment-only send triggered 400: ${consoleErrors.join(" | ")}`,
  ).toBe(false);

  // MessageList should render the AttachmentCard (thumbnail <img> pointing
  // at the uploaded file) — body is intentionally empty for pure-attachment
  // messages, so we don't look for filename text.
  await expect(
    page.locator('[data-testid="message-attachment"] img, [data-testid="message-attachment"] a').first(),
  ).toBeVisible({ timeout: 10_000 });
});
