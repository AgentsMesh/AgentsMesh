import { test, expect } from "../../fixtures";
import { ChannelsPage } from "../../pages/channels.page";
import { TEST_ORG_SLUG } from "../../helpers/env";

async function createChannelViaApi(
  api: { login: () => Promise<unknown>; post: (p: string, b?: unknown) => Promise<Response> },
  name: string,
): Promise<number> {
  await api.login();
  const res = await api.post(`/api/v1/orgs/${TEST_ORG_SLUG}/channels`, {
    name,
    description: "Desktop toolbar e2e",
    visibility: "public",
  });
  if (!res.ok) throw new Error(`create channel failed: ${res.status} ${await res.text()}`);
  const data = (await res.json()) as { channel: { id: number } };
  return data.channel.id;
}

test.describe("Channel toolbar (desktop)", () => {
  let channelId: number;
  let channelName: string;

  test.beforeEach(async ({ api, page }) => {
    const consoleErrors: string[] = [];
    const pageErrors: string[] = [];
    page.on("console", (msg) => {
      if (msg.type() === "error") consoleErrors.push(msg.text());
    });
    page.on("pageerror", (err) => pageErrors.push(`${err.name}: ${err.message}`));

    channelName = `desk-tb-${Date.now()}`;
    channelId = await createChannelViaApi(api, channelName);
    const channels = new ChannelsPage(page);
    await channels.goto();
    await page.waitForTimeout(800);
    await page.getByText(channelName, { exact: true }).first().click({ timeout: 10_000 });
    await page.waitForTimeout(800);

    if (pageErrors.length) console.log("[pageerror]", pageErrors.join("\n---\n"));
    if (consoleErrors.length) console.log("[console.error]", consoleErrors.join("\n---\n"));
  });

  test("i18n: empty channel shows localized text", async ({ page }) => {
    await expect(page.getByText("channels.messages.noMessages")).toHaveCount(0);
    await expect(page.getByText(/暂无消息|No messages yet/i)).toBeVisible({ timeout: 5000 });
  });

  test("header renders three action buttons and no close X", async ({ page }) => {
    await expect(page.getByTestId("channel-header-search")).toBeVisible();
    await expect(page.getByTestId("channel-header-more")).toBeVisible();
    // Old duplicated "member" pill must be gone from the header.
    await expect(page.getByTestId("channel-header-members")).toHaveCount(1); // only inside rail
    await expect(page.locator('button[aria-label="close" i]')).toHaveCount(0);
  });

  test("toolbar shows the three compose actions", async ({ page }) => {
    await expect(page.getByTestId("message-input-textarea")).toBeVisible();
    await expect(page.getByTestId("format-toolbar-trigger")).toBeVisible();
    await expect(page.getByTestId("toolbar-mention")).toBeVisible();
    await expect(page.getByTestId("toolbar-attach")).toBeVisible();
  });

  test("search modal opens from header", async ({ page }) => {
    await page.getByTestId("channel-header-search").click();
    await expect(page.getByTestId("message-search-input")).toBeVisible();
  });

  test("more button toggles the right drawer", async ({ page }) => {
    const rail = page.getByTestId("channel-right-rail");
    const more = page.getByTestId("channel-header-more");

    // Drawer is open by default.
    await expect(rail).toBeVisible();
    // `aria-pressed` reflects the open state for a11y.
    await expect(more).toHaveAttribute("aria-pressed", "true");

    // First click collapses the drawer.
    await more.click();
    await expect(rail).toHaveCount(0);
    await expect(more).toHaveAttribute("aria-pressed", "false");

    // Second click re-opens.
    await more.click();
    await expect(rail).toBeVisible();
    await expect(more).toHaveAttribute("aria-pressed", "true");
  });

  test("settings modal opens from the rail with prefilled name", async ({ page }) => {
    // Ensure the drawer is open, then click the ⚙ icon inside the DOCUMENT
    // section header.
    await page.getByTestId("channel-rail-settings").click();
    const nameInput = page.getByTestId("channel-settings-name");
    await expect(nameInput).toBeVisible();
    await expect(nameInput).toHaveValue(channelName, { timeout: 20_000 });
    await page.getByRole("button", { name: /^归档$|^Archive$/i }).first().click();
    await expect(page.getByTestId("channel-settings-archive-toggle")).toBeVisible();
  });

  test("members popover opens and lists an empty state cleanly", async ({ page }) => {
    await page.getByTestId("channel-header-members").first().click();
    await expect(
      page.getByRole("heading", { name: /频道成员|Channel Members/i }).first(),
    ).toBeVisible({ timeout: 5000 });
  });

  test("B toolbar opens format menu and wraps selection in bold", async ({ page }) => {
    const textarea = page.getByTestId("message-input-textarea");
    await textarea.fill("hello world");
    await textarea.evaluate((el: HTMLTextAreaElement) => el.setSelectionRange(0, 5));

    await page.getByTestId("format-toolbar-trigger").click();
    await expect(page.getByTestId("format-toolbar-menu")).toBeVisible();
    await page.getByTestId("format-bold").click();

    await expect(textarea).toHaveValue("**hello** world");
  });

  test("@ toolbar inserts @ at cursor and opens the mention dropdown", async ({ page }) => {
    const textarea = page.getByTestId("message-input-textarea");
    await textarea.fill("hi ");
    await textarea.evaluate((el: HTMLTextAreaElement) =>
      el.setSelectionRange(el.value.length, el.value.length),
    );
    await page.getByTestId("toolbar-mention").click();
    await expect(textarea).toHaveValue("hi @");
    // Dropdown appears when at least one candidate (org member) is loaded.
    await expect(page.getByTestId("mention-dropdown")).toBeVisible({ timeout: 10_000 });
  });

  test("attachment upload does not throw JSON parse error", async ({ page }) => {
    const parseErrors: string[] = [];
    page.on("pageerror", (err) => {
      if (/is not valid JSON/i.test(err.message)) parseErrors.push(err.message);
    });
    page.on("console", (msg) => {
      if (msg.type() === "error" && /is not valid JSON/i.test(msg.text())) {
        parseErrors.push(msg.text());
      }
    });

    const fs = await import("node:fs/promises");
    const path = await import("node:path");
    const os = await import("node:os");
    const tmp = path.join(os.tmpdir(), `desk-tb-upload-${Date.now()}.png`);
    const pngBytes = Buffer.from(
      "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNgAAIAAAUAAeImBZsAAAAASUVORK5CYII=",
      "base64",
    );
    await fs.writeFile(tmp, pngBytes);

    const input = page.getByTestId("message-attachment-input");
    await input.setInputFiles(tmp);

    // Give the upload pipeline a moment to resolve; the assertion is that no
    // `JSON.parse` error landed on the page.
    await page.waitForTimeout(1500);
    await fs.unlink(tmp).catch(() => undefined);
    expect(parseErrors).toEqual([]);
  });

  test("B wrap produces styled bold when rendered", async ({ page }) => {
    // Build + send a bold message using the format toolbar; it must render
    // as <strong> in the message list (not literal **text**).
    const textarea = page.getByTestId("message-input-textarea");
    await textarea.fill("hello");
    await textarea.evaluate((el: HTMLTextAreaElement) => el.setSelectionRange(0, 5));
    await page.getByTestId("format-toolbar-trigger").click();
    await page.getByTestId("format-bold").click();
    await expect(textarea).toHaveValue("**hello**");
    await textarea.press("Enter");
    await expect(page.locator("strong", { hasText: "hello" })).toBeVisible({ timeout: 10_000 });
  });

  test("attachment hidden file input is wired", async ({ page }) => {
    await expect(page.getByTestId("toolbar-attach")).toBeVisible();
    const input = page.getByTestId("message-attachment-input");
    await expect(input).toBeAttached();
    expect(await input.getAttribute("type")).toBe("file");
  });
});
