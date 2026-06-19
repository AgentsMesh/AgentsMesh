import { describe, it, expect } from "vitest";
import { create } from "@bufbuild/protobuf";
import { RepositorySchema, BranchSchema } from "@agentsmesh/proto/repository/v1/repository_pb";
import { repositoryToCache, branchToCache } from "./repository";

// Guards the shared repository projection against proto-schema drift.

describe("repositoryToCache", () => {
  it("maps every scalar field + nested webhook_config", () => {
    const c = repositoryToCache(create(RepositorySchema, {
      id: 9n, organizationId: 2n, providerType: "github",
      providerBaseUrl: "https://github.com", httpCloneUrl: "https://github.com/o/r.git",
      externalId: "ext-1", name: "demo", slug: "org/demo", defaultBranch: "main",
      ticketPrefix: "DEV", visibility: "private", importedByUserId: 7n, isActive: true,
      webhookConfig: { id: "wh1", url: "https://hook", events: ["push"], isActive: true, needsManualSetup: false },
      createdAt: "c0", updatedAt: "u0",
    }));
    expect(c.id).toBe(9);
    expect(c.organization_id).toBe(2);
    expect(c.provider_type).toBe("github");
    expect(c.http_clone_url).toBe("https://github.com/o/r.git");
    expect(c.external_id).toBe("ext-1");
    expect(c.slug).toBe("org/demo");
    expect(c.default_branch).toBe("main");
    expect(c.imported_by_user_id).toBe(7);
    expect(c.is_active).toBe(true);
    expect(c.webhook_config?.id).toBe("wh1");
    expect(c.webhook_config?.events).toEqual(["push"]);
  });

  it("normalizes empty clone urls + absent webhook to undefined", () => {
    const c = repositoryToCache(create(RepositorySchema, {
      id: 1n, organizationId: 1n, name: "r", slug: "o/r", externalId: "e",
      providerType: "github", providerBaseUrl: "u", defaultBranch: "main",
      visibility: "private", isActive: false, createdAt: "c", updatedAt: "u",
    }));
    expect(c.http_clone_url).toBeUndefined();
    expect(c.ssh_clone_url).toBeUndefined();
    expect(c.webhook_config).toBeUndefined();
  });
});

describe("branchToCache", () => {
  it("maps the branch name", () => {
    expect(branchToCache(create(BranchSchema, { name: "feature/x" }))).toEqual({ name: "feature/x" });
  });
});
