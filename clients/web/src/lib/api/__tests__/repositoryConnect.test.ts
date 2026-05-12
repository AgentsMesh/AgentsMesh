// Unit tests for clients/web/src/lib/api/repositoryConnectShapes.ts.
//
// Covers the proto → snake_case web shape mappings — keeps PR #329's lineage
// fields (preparation_script, imported_by_user_id, webhook_config) honest
// when the proto SSOT changes.
//
// Note: this test file imports @proto/repository/v1/repository_pb which
// resolves via vitest.config.ts. Under `bazel test //clients/web:unit`
// the `proto/gen` tree is in .bazelignore so the import fails — the
// test runs under `pnpm test:run` (or once ts_proto_library lands;
// see runbook "TS proto codegen toolchain").

import { describe, it, expect } from "vitest";
import { create } from "@bufbuild/protobuf";
import {
  RepositorySchema,
  WebhookStatusSchema,
  WebhookSecretSchema,
  WebhookResultSchema,
  MergeRequestSchema,
} from "@proto/repository/v1/repository_pb";
import {
  fromProtoRepository,
  fromProtoWebhookStatus,
  fromProtoWebhookSecret,
  fromProtoWebhookResult,
  fromProtoMergeRequest,
} from "../repositoryConnectShapes";

describe("fromProtoRepository", () => {
  it("converts the full 19-field Repository proto to RepositoryData", () => {
    const proto = create(RepositorySchema, {
      id: BigInt(1),
      organizationId: BigInt(42),
      providerType: "github",
      providerBaseUrl: "https://github.com",
      httpCloneUrl: "https://github.com/org/repo.git",
      sshCloneUrl: "git@github.com:org/repo.git",
      externalId: "ext-123",
      name: "repo",
      slug: "org/repo",
      defaultBranch: "main",
      ticketPrefix: "PROJ",
      visibility: "organization",
      importedByUserId: BigInt(7),
      preparationScript: "echo hi",
      preparationTimeout: 600,
      isActive: true,
      createdAt: "2025-01-01T00:00:00Z",
      updatedAt: "2025-01-02T00:00:00Z",
    });
    const out = fromProtoRepository(proto);
    expect(out.id).toBe(1);
    expect(out.organization_id).toBe(42);
    expect(out.provider_type).toBe("github");
    expect(out.http_clone_url).toBe("https://github.com/org/repo.git");
    expect(out.ssh_clone_url).toBe("git@github.com:org/repo.git");
    expect(out.ticket_prefix).toBe("PROJ");
    expect(out.imported_by_user_id).toBe(7);
    expect(out.is_active).toBe(true);
    expect(out.created_at).toBe("2025-01-01T00:00:00Z");
    expect(out.updated_at).toBe("2025-01-02T00:00:00Z");
    expect(out.webhook_config).toBeUndefined();
  });

  it("maps webhook_config when present (preserves PR #329's nested config)", () => {
    const proto = create(RepositorySchema, {
      id: BigInt(1),
      organizationId: BigInt(1),
      providerType: "gitlab",
      providerBaseUrl: "https://gitlab.com",
      externalId: "ext",
      name: "n",
      slug: "s",
      defaultBranch: "main",
      visibility: "organization",
      isActive: true,
      createdAt: "now",
      updatedAt: "now",
      webhookConfig: {
        id: "wh-1",
        url: "https://api.example/webhooks/1",
        events: ["merge_request", "pipeline"],
        isActive: true,
        needsManualSetup: false,
      },
    });
    const out = fromProtoRepository(proto);
    expect(out.webhook_config).toBeDefined();
    expect(out.webhook_config?.id).toBe("wh-1");
    expect(out.webhook_config?.events).toEqual(["merge_request", "pipeline"]);
    expect(out.webhook_config?.is_active).toBe(true);
    expect(out.webhook_config?.needs_manual_setup).toBe(false);
  });

  it("treats empty http_clone_url / ssh_clone_url as undefined", () => {
    const proto = create(RepositorySchema, {
      id: BigInt(1),
      organizationId: BigInt(1),
      providerType: "github",
      providerBaseUrl: "https://github.com",
      externalId: "ext",
      name: "n",
      slug: "s",
      defaultBranch: "main",
      visibility: "organization",
      isActive: true,
      createdAt: "now",
      updatedAt: "now",
    });
    const out = fromProtoRepository(proto);
    expect(out.http_clone_url).toBeUndefined();
    expect(out.ssh_clone_url).toBeUndefined();
  });
});

describe("fromProtoWebhookStatus", () => {
  it("flattens the WebhookStatus proto into the web shape", () => {
    const proto = create(WebhookStatusSchema, {
      registered: true,
      webhookId: "wh-7",
      webhookUrl: "https://api.example/webhooks/7",
      events: ["push", "merge_request"],
      isActive: true,
      needsManualSetup: false,
      registeredAt: "2025-01-01T00:00:00Z",
    });
    const out = fromProtoWebhookStatus(proto);
    expect(out.registered).toBe(true);
    expect(out.webhook_id).toBe("wh-7");
    expect(out.webhook_url).toBe("https://api.example/webhooks/7");
    expect(out.events).toEqual(["push", "merge_request"]);
    expect(out.is_active).toBe(true);
    expect(out.needs_manual_setup).toBe(false);
    expect(out.registered_at).toBe("2025-01-01T00:00:00Z");
  });
});

describe("fromProtoWebhookSecret", () => {
  it("maps the webhook_secret field (PR #345 raw_key bug-class pin)", () => {
    const proto = create(WebhookSecretSchema, {
      webhookUrl: "https://api.example/webhooks/9",
      webhookSecret: "super-secret-value",
      events: ["push"],
    });
    const out = fromProtoWebhookSecret(proto);
    expect(out.webhook_url).toBe("https://api.example/webhooks/9");
    expect(out.webhook_secret).toBe("super-secret-value");
    expect(out.events).toEqual(["push"]);
  });
});

describe("fromProtoWebhookResult", () => {
  it("maps error_message field to error (RepositoryService.RegisterRepositoryWebhook)", () => {
    const proto = create(WebhookResultSchema, {
      repoId: BigInt(42),
      registered: false,
      needsManualSetup: true,
      manualWebhookUrl: "https://example/manual",
      manualWebhookSecret: "manual-secret",
      errorMessage: "OAuth not available",
    });
    const out = fromProtoWebhookResult(proto);
    expect(out.repo_id).toBe(42);
    expect(out.needs_manual_setup).toBe(true);
    expect(out.manual_webhook_url).toBe("https://example/manual");
    expect(out.manual_webhook_secret).toBe("manual-secret");
    expect(out.error).toBe("OAuth not available");
  });
});

describe("fromProtoMergeRequest", () => {
  it("maps MergeRequest proto fields to MergeRequestInfo", () => {
    const proto = create(MergeRequestSchema, {
      id: BigInt(99),
      mrIid: 7,
      title: "Add feature",
      state: "opened",
      mrUrl: "https://gitlab/mr/7",
      sourceBranch: "feature/x",
      targetBranch: "main",
      pipelineStatus: "running",
      pipelineUrl: "https://gitlab/pipelines/123",
    });
    const out = fromProtoMergeRequest(proto);
    expect(out.id).toBe(99);
    expect(out.mr_iid).toBe(7);
    expect(out.state).toBe("opened");
    expect(out.source_branch).toBe("feature/x");
    expect(out.target_branch).toBe("main");
    expect(out.pipeline_status).toBe("running");
    expect(out.pipeline_url).toBe("https://gitlab/pipelines/123");
  });
});
