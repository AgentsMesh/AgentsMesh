import { describe, expect, it } from "vitest";

import { buildPageQuery } from "./pageQuery";

describe("buildPageQuery", () => {
  it("sets the page param when selecting a sub-page", () => {
    expect(buildPageQuery(new URLSearchParams(""), "p1", "root")).toBe("?page=p1");
  });

  it("clears the page param when selecting the root", () => {
    expect(buildPageQuery(new URLSearchParams("page=p1"), "root", "root")).toBe("?");
  });

  it("preserves unrelated params", () => {
    expect(buildPageQuery(new URLSearchParams("ws=w1"), "p2", "root")).toBe("?ws=w1&page=p2");
  });

  it("replaces an existing page param", () => {
    expect(buildPageQuery(new URLSearchParams("page=p1"), "p2", "root")).toBe("?page=p2");
  });
});
