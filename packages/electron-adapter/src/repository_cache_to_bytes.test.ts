import { describe, it, expect, beforeEach } from "vitest";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import {
  RepositorySchema, ListRepositoriesResponseSchema,
} from "@agentsmesh/proto/repository/v1/repository_pb";
import { ReplaceCachedRepositoriesRequestSchema } from "@agentsmesh/proto/repo_state/v1/repo_state_pb";
import { repositoryToCache } from "./projections/repository";
import { repositoriesBytes } from "./repository_cache_to_bytes";
import { ElectronRepositoryService } from "./repository";

const wireRepo = (id: bigint, slug: string) => create(RepositorySchema, {
  id, organizationId: 1n, providerType: "github", providerBaseUrl: "https://github.com",
  httpCloneUrl: "https://github.com/x.git", sshCloneUrl: "git@github.com:x.git",
  externalId: "ext-1", name: "x", slug, defaultBranch: "main", ticketPrefix: "X",
  visibility: "private", importedByUserId: 9n, isActive: true,
  createdAt: "2026-01-01", updatedAt: "2026-01-02",
});

// cache→bytes must round-trip every field repositoryToCache reads, or desktop
// diverges from web (which decodes the same bytes through repositoryToCache).
describe("repository cache→bytes round-trip", () => {
  it("preserves repository fields through cache → bytes → state", () => {
    const cache = repositoryToCache(wireRepo(1n, "repo-a"));
    const decoded = fromBinary(ReplaceCachedRepositoriesRequestSchema, repositoriesBytes(JSON.stringify([cache])));
    expect(repositoryToCache(decoded.repositories[0])).toEqual(cache);
  });
});

describe("ElectronRepositoryService fetch→state", () => {
  let invokes: string[];
  beforeEach(() => {
    invokes = [];
    (globalThis as { window?: unknown }).window = {
      electronAPI: { invoke: async (ch: string) => { invokes.push(ch); return undefined; } },
    };
  });

  it("apply_fetched_repositories caches + fans to main + reads back via bytes", () => {
    const svc = new ElectronRepositoryService();
    const bytes = toBinary(ListRepositoriesResponseSchema, create(ListRepositoriesResponseSchema, {
      items: [wireRepo(1n, "repo-a"), wireRepo(2n, "repo-b")],
    }));
    svc.apply_fetched_repositories(bytes);
    expect(invokes).toContain("repoReplaceCachedRepositories");
    const decoded = fromBinary(ReplaceCachedRepositoriesRequestSchema, svc.repositories_bytes());
    expect(decoded.repositories.map((r) => r.slug)).toEqual(["repo-a", "repo-b"]);
  });
});
