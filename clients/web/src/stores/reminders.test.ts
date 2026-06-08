import { describe, it, expect, beforeEach } from "vitest";
import { useReminderStore } from "./reminders";

const get = () => useReminderStore.getState();
const visibleIds = () =>
  Object.values(get().reminders)
    .filter((r) => !(r.id in get().dismissed))
    .map((r) => r.id);

describe("reminder store", () => {
  beforeEach(() => useReminderStore.setState({ reminders: {}, signatures: {}, dismissed: {} }));

  it("set then clear", () => {
    get().setReminder({ id: "a", tone: "info", message: "m" }, "s1");
    expect(visibleIds()).toEqual(["a"]);
    get().clearReminder("a");
    expect(visibleIds()).toEqual([]);
  });

  it("dismiss hides until the signature changes", () => {
    get().setReminder({ id: "u", tone: "success", message: "v1" }, "1.0");
    get().dismiss("u");
    expect(visibleIds()).toEqual([]);
    get().setReminder({ id: "u", tone: "success", message: "v1" }, "1.0"); // same sig
    expect(visibleIds()).toEqual([]);
    get().setReminder({ id: "u", tone: "success", message: "v2" }, "1.1"); // new sig
    expect(visibleIds()).toEqual(["u"]);
  });

  it("clear resets dismiss so a fresh episode resurfaces", () => {
    get().setReminder({ id: "e", tone: "warning", message: "x" }, "recovering");
    get().dismiss("e");
    expect(visibleIds()).toEqual([]);
    get().clearReminder("e"); // reconnected
    get().setReminder({ id: "e", tone: "warning", message: "x" }, "recovering"); // disconnect again
    expect(visibleIds()).toEqual(["e"]);
  });
});
