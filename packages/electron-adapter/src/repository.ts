import { invoke } from "./invoke";
import { coerceConnectResponse } from "./connect-response";
import type { IRepositoryService, IRepoState } from "@agentsmesh/service-interface";
import { fromBinary } from "@bufbuild/protobuf";
import {
  ReplaceCachedRepositoriesRequestSchema,
  SetCurrentRepoRequestSchema,
  ReplaceBranchesRequestSchema,
  InsertRepositoryRequestSchema,
  PatchRepositoryRequestSchema,
} from "@agentsmesh/proto/repo_state/v1/repo_state_pb";
import type { Repository as ProtoRepository, Branch as ProtoBranch } from "@agentsmesh/proto/repository/v1/repository_pb";

// Web's wasm-side `WasmRepositoryService` exposes `<verb>Connect(bytes)`
// methods. RepositoryService in Rust core is a thin proxy (no cache, no
// business logic — see ADR 2026-05-24-service-binding-matrix.md "Decision
// 1: thin-proxy services don't require binding symmetry"). Desktop renderer
// reaches the backend through `connectCall` generic IPC proxy (main owns
// the auth header injection + URL prefix) — adding a dedicated napi binding
// would be form symmetry without functional gain, so we go directly to the
// backend Connect endpoint through main process.
async function connectCall(method: string, request: Uint8Array): Promise<Uint8Array> {
  const resp = await invoke<number[] | Uint8Array>(
    "connectCall",
    "proto.repository.v1.RepositoryService",
    method,
    Array.from(request),
  );
  return coerceConnectResponse(resp);
}

function repositoryToCache(r: ProtoRepository): Record<string, unknown> {
  const ZERO = BigInt(0);
  return {
    id: Number(r.id),
    organization_id: Number(r.organizationId),
    provider_type: r.providerType,
    provider_base_url: r.providerBaseUrl,
    http_clone_url: r.httpCloneUrl || undefined,
    ssh_clone_url: r.sshCloneUrl || undefined,
    external_id: r.externalId,
    name: r.name,
    slug: r.slug,
    default_branch: r.defaultBranch,
    ticket_prefix: r.ticketPrefix,
    visibility: r.visibility,
    imported_by_user_id:
      r.importedByUserId !== undefined && r.importedByUserId !== ZERO
        ? Number(r.importedByUserId) : undefined,
    is_active: r.isActive,
    webhook_config: r.webhookConfig
      ? {
          id: r.webhookConfig.id,
          url: r.webhookConfig.url,
          events: r.webhookConfig.events ?? [],
          is_active: r.webhookConfig.isActive,
          needs_manual_setup: r.webhookConfig.needsManualSetup,
          last_error: r.webhookConfig.lastError,
          created_at: r.webhookConfig.createdAt,
        }
      : undefined,
    created_at: r.createdAt,
    updated_at: r.updatedAt,
  };
}

function branchToCache(b: ProtoBranch): Record<string, unknown> {
  return { name: b.name };
}

// Service-is-State-superset 模式（对齐 ElectronPodService）：内部 cache 由 Service 持有，
// provider 让 repoState 别名同一实例，renderer 端 useRepositories() 读到的就是这里的 cache。
export class ElectronRepositoryService implements IRepositoryService, IRepoState {
  private _repositoriesCache = "[]";
  private _currentRepoCache: string | null = null;
  private _branchesCache = "[]";

  // ===== IRepoState =====

  repositories_json(): string { return this._repositoriesCache; }
  current_repo_json(): unknown { return this._currentRepoCache; }
  branches_json(): string { return this._branchesCache; }

  remove_repository(id: string): void {
    const repos = JSON.parse(this._repositoriesCache) as { id: number }[];
    this._repositoriesCache = JSON.stringify(repos.filter(r => String(r.id) !== id));
  }

  // Proto-bytes mutators — decode locally + update JS cache synchronously,
  // then fire-and-forget NAPI sync. Mirrors ElectronRunnerService.

  replace_cached_repositories(reqBytes: Uint8Array): void {
    const req = fromBinary(ReplaceCachedRepositoriesRequestSchema, reqBytes);
    this._repositoriesCache = JSON.stringify(req.repositories.map(repositoryToCache));
    void invoke<void>("repoReplaceCachedRepositories", Array.from(reqBytes)).catch(() => undefined);
  }

  set_current_repo_proto(reqBytes: Uint8Array): void {
    const req = fromBinary(SetCurrentRepoRequestSchema, reqBytes);
    this._currentRepoCache = req.repository ? JSON.stringify(repositoryToCache(req.repository)) : null;
    void invoke<void>("repoSetCurrentRepoProto", Array.from(reqBytes)).catch(() => undefined);
  }

  replace_branches(reqBytes: Uint8Array): void {
    const req = fromBinary(ReplaceBranchesRequestSchema, reqBytes);
    this._branchesCache = JSON.stringify(req.branches.map(branchToCache));
    void invoke<void>("repoReplaceBranches", Array.from(reqBytes)).catch(() => undefined);
  }

  insert_repository(reqBytes: Uint8Array): void {
    const req = fromBinary(InsertRepositoryRequestSchema, reqBytes);
    if (req.repository) {
      const repos = JSON.parse(this._repositoriesCache) as Record<string, unknown>[];
      repos.push(repositoryToCache(req.repository));
      this._repositoriesCache = JSON.stringify(repos);
    }
    void invoke<void>("repoInsertRepository", Array.from(reqBytes)).catch(() => undefined);
  }

  patch_repository(reqBytes: Uint8Array): void {
    const req = fromBinary(PatchRepositoryRequestSchema, reqBytes);
    if (req.repository) {
      const patched = repositoryToCache(req.repository);
      const repos = JSON.parse(this._repositoriesCache) as { id: number }[];
      const idx = repos.findIndex(r => String(r.id) === req.id);
      if (idx >= 0) repos[idx] = patched as { id: number };
      this._repositoriesCache = JSON.stringify(repos);
    }
    void invoke<void>("repoPatchRepository", Array.from(reqBytes)).catch(() => undefined);
  }

  // ===== IRepositoryService =====

  async list(): Promise<string> {
    return invoke<string>("repositoryList");
  }

  async get(id: bigint): Promise<string> {
    return invoke<string>("repositoryGet", Number(id));
  }

  async create(json: string): Promise<string> {
    return invoke<string>("repositoryCreate", json);
  }

  async update(id: bigint, json: string): Promise<string> {
    return invoke<string>("repositoryUpdate", Number(id), json);
  }

  async delete(id: bigint): Promise<void> {
    await invoke<void>("repositoryDelete", Number(id));
  }

  async list_branches(id: bigint): Promise<string> {
    return invoke<string>("repositoryListBranches", Number(id));
  }

  async sync_branches(id: bigint, json: string): Promise<string> {
    return invoke<string>("repositorySyncBranches", Number(id), json);
  }

  async list_merge_requests(id: bigint, branch?: string | null, state?: string | null): Promise<string> {
    return invoke<string>("repositoryListMergeRequests", Number(id), branch, state);
  }

  async register_webhook(id: bigint): Promise<void> {
    await invoke<void>("repositoryRegisterWebhook", Number(id));
  }

  async delete_webhook(id: bigint): Promise<void> {
    await invoke<void>("repositoryDeleteWebhook", Number(id));
  }

  async get_webhook_secret(id: bigint): Promise<string> {
    return invoke<string>("repositoryGetWebhookSecret", Number(id));
  }

  async get_webhook_status(id: bigint): Promise<string> {
    return invoke<string>("repositoryGetWebhookStatus", Number(id));
  }

  async mark_webhook_configured(id: bigint): Promise<void> {
    await invoke<void>("repositoryMarkWebhookConfigured", Number(id));
  }

  // ── Connect-RPC binary surface (mirrors WasmRepositoryService) ──

  listRepositoriesConnect(request: Uint8Array): Promise<Uint8Array> {
    return connectCall("ListRepositories", request);
  }
  getRepositoryConnect(request: Uint8Array): Promise<Uint8Array> {
    return connectCall("GetRepository", request);
  }
  createRepositoryConnect(request: Uint8Array): Promise<Uint8Array> {
    return connectCall("CreateRepository", request);
  }
  updateRepositoryConnect(request: Uint8Array): Promise<Uint8Array> {
    return connectCall("UpdateRepository", request);
  }
  deleteRepositoryConnect(request: Uint8Array): Promise<Uint8Array> {
    return connectCall("DeleteRepository", request);
  }
  listRepositoryBranchesConnect(request: Uint8Array): Promise<Uint8Array> {
    return connectCall("ListRepositoryBranches", request);
  }
  syncRepositoryBranchesConnect(request: Uint8Array): Promise<Uint8Array> {
    return connectCall("SyncRepositoryBranches", request);
  }
  listRepositoryMergeRequestsConnect(request: Uint8Array): Promise<Uint8Array> {
    return connectCall("ListRepositoryMergeRequests", request);
  }
  registerRepositoryWebhookConnect(request: Uint8Array): Promise<Uint8Array> {
    return connectCall("RegisterRepositoryWebhook", request);
  }
  deleteRepositoryWebhookConnect(request: Uint8Array): Promise<Uint8Array> {
    return connectCall("DeleteRepositoryWebhook", request);
  }
  getRepositoryWebhookStatusConnect(request: Uint8Array): Promise<Uint8Array> {
    return connectCall("GetRepositoryWebhookStatus", request);
  }
  getRepositoryWebhookSecretConnect(request: Uint8Array): Promise<Uint8Array> {
    return connectCall("GetRepositoryWebhookSecret", request);
  }
  markRepositoryWebhookConfiguredConnect(request: Uint8Array): Promise<Uint8Array> {
    return connectCall("MarkRepositoryWebhookConfigured", request);
  }
}
