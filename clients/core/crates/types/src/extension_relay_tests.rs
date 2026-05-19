use super::{
    MarketMcpServer, MarketMcpServerListResponse, MarketSkill, MarketSkillListResponse,
    RepoMcpServerInstall, RepoMcpServerInstallListResponse, RepoSkillInstall,
    RepoSkillInstallListResponse, SkillRegistry, SkillRegistryListResponse,
};

// ── SkillRegistry (existing coverage, preserved verbatim from PR #341/#349/#368) ──

const REGISTRY_PAYLOAD: &str = r#"{
    "id": 7,
    "organization_id": 42,
    "repository_url": "https://github.com/agentsmesh/skills-bundle",
    "branch": "main",
    "source_type": "auto",
    "detected_type": "collection",
    "compatible_agents": ["claude-code", "codex"],
    "auth_type": "github_pat",
    "last_synced_at": "2026-05-09T00:00:00Z",
    "last_commit_sha": "abc123def456",
    "sync_status": "success",
    "skill_count": 12,
    "is_active": true,
    "created_at": "2026-05-08T00:00:00Z",
    "updated_at": "2026-05-09T00:00:00Z"
}"#;

#[test]
fn skill_registry_decodes_backend_payload() {
    let r: SkillRegistry = serde_json::from_str(REGISTRY_PAYLOAD).unwrap();
    assert_eq!(r.id, 7);
    assert_eq!(r.sync_status.as_deref(), Some("success"));
    assert_eq!(r.skill_count, Some(12));
    assert_eq!(r.auth_type.as_deref(), Some("github_pat"));
    assert_eq!(r.is_active, Some(true));
}

#[test]
fn skill_registry_wasm_relay_preserves_ui_fields() {
    let typed: SkillRegistry = serde_json::from_str(REGISTRY_PAYLOAD).unwrap();
    let relayed = serde_json::to_string(&typed).unwrap();
    let parsed: serde_json::Value = serde_json::from_str(&relayed).unwrap();
    for key in [
        "sync_status",
        "auth_type",
        "skill_count",
        "is_active",
        "last_commit_sha",
        "detected_type",
    ] {
        assert!(!parsed[key].is_null(), "field `{key}` dropped by relay");
    }
}

// Regression for #341: backend wire wrapper is `skill_registries`, but PR #349
// used `#[serde(alias)]` on a Rust field named `registries`. Serde's `alias`
// only affects deserialization; re-serializing for the wasm relay emitted
// `{"registries": ...}` instead. This test pins the round-trip key.
#[test]
fn skill_registry_list_response_relay_key_matches_backend_wire() {
    let backend_wire = format!(r#"{{"skill_registries":[{REGISTRY_PAYLOAD}]}}"#);
    let typed: SkillRegistryListResponse = serde_json::from_str(&backend_wire).unwrap();
    assert_eq!(typed.skill_registries.len(), 1);

    let relayed = serde_json::to_string(&typed).unwrap();
    let parsed: serde_json::Value = serde_json::from_str(&relayed).unwrap();
    assert!(
        parsed.get("skill_registries").is_some(),
        "wasm relay must emit `skill_registries`, got: {relayed}"
    );
    assert!(
        parsed.get("registries").is_none(),
        "legacy `registries` key must not leak through: {relayed}"
    );
}

// ── MarketSkill (PR #398 — TS reads item.registry?.repository_url) ──

const MARKET_SKILL_PAYLOAD: &str = r#"{
    "id": 1,
    "registry_id": 10,
    "slug": "git-commit",
    "display_name": "Git Commit",
    "description": "Create polished commits",
    "license": "MIT",
    "category": "dev",
    "content_sha": "abc123",
    "version": 2,
    "is_active": true,
    "registry": {
        "id": 10,
        "organization_id": 42,
        "repository_url": "https://github.com/acme/skills",
        "branch": "main",
        "sync_status": "success"
    }
}"#;

#[test]
fn market_skill_wasm_relay_preserves_ui_fields() {
    let typed: MarketSkill = serde_json::from_str(MARKET_SKILL_PAYLOAD).unwrap();
    assert_eq!(typed.display_name.as_deref(), Some("Git Commit"));
    assert!(typed.registry.is_some());

    let relayed = serde_json::to_string(&typed).unwrap();
    let parsed: serde_json::Value = serde_json::from_str(&relayed).unwrap();
    for key in [
        "slug",
        "display_name",
        "description",
        "category",
        "registry",
        "registry_id",
        "is_active",
        "content_sha",
    ] {
        assert!(!parsed[key].is_null(), "field `{key}` dropped by relay");
    }
    assert!(
        !parsed["registry"]["repository_url"].is_null(),
        "nested registry.repository_url dropped — TS AddSkillDialog reads this"
    );
}

#[test]
fn market_skill_list_response_relay_key_matches_backend_wire() {
    let backend_wire = format!(r#"{{"skills":[{MARKET_SKILL_PAYLOAD}]}}"#);
    let typed: MarketSkillListResponse = serde_json::from_str(&backend_wire).unwrap();
    assert_eq!(typed.skills.len(), 1);

    let relayed = serde_json::to_string(&typed).unwrap();
    let parsed: serde_json::Value = serde_json::from_str(&relayed).unwrap();
    assert!(parsed.get("skills").is_some(), "must emit `skills`: {relayed}");
}

// ── MarketMcpServer (TS uses command / default_args / env_var_schema / etc.) ──

const MARKET_MCP_PAYLOAD: &str = r#"{
    "id": 1,
    "name": "GitHub MCP",
    "slug": "github",
    "description": "Access GitHub repos",
    "icon": "github",
    "transport_type": "http",
    "command": "npx",
    "default_args": ["mcp-github", "--token"],
    "default_http_url": "https://api.github.com",
    "default_http_headers": [
        {"name": "Authorization", "required": true, "sensitive": true}
    ],
    "env_var_schema": [
        {"name": "GH_TOKEN", "label": "GitHub Token", "required": true, "sensitive": true}
    ],
    "category": "dev",
    "source": "registry",
    "registry_name": "MCP Registry",
    "version": "1.2.0",
    "repository_url": "https://github.com/example/mcp-github"
}"#;

#[test]
fn market_mcp_server_wasm_relay_preserves_ui_fields() {
    let typed: MarketMcpServer = serde_json::from_str(MARKET_MCP_PAYLOAD).unwrap();
    assert_eq!(typed.command.as_deref(), Some("npx"));
    assert_eq!(typed.default_args.as_ref().map(|v| v.len()), Some(2));
    assert_eq!(typed.env_var_schema.as_ref().map(|v| v.len()), Some(1));

    let relayed = serde_json::to_string(&typed).unwrap();
    let parsed: serde_json::Value = serde_json::from_str(&relayed).unwrap();
    for key in [
        "command",
        "default_args",
        "default_http_url",
        "default_http_headers",
        "env_var_schema",
        "source",
        "version",
        "repository_url",
        "transport_type",
        "icon",
    ] {
        assert!(!parsed[key].is_null(), "field `{key}` dropped by relay");
    }
}

#[test]
fn market_mcp_server_list_response_relay_key_matches_backend_wire() {
    let backend_wire = format!(r#"{{"mcp_servers":[{MARKET_MCP_PAYLOAD}]}}"#);
    let typed: MarketMcpServerListResponse = serde_json::from_str(&backend_wire).unwrap();
    assert_eq!(typed.mcp_servers.len(), 1);

    let relayed = serde_json::to_string(&typed).unwrap();
    let parsed: serde_json::Value = serde_json::from_str(&relayed).unwrap();
    assert!(
        parsed.get("mcp_servers").is_some(),
        "must emit `mcp_servers`: {relayed}"
    );
    assert!(
        parsed.get("total").is_none(),
        "phantom `total` must not leak: {relayed}"
    );
}

// ── RepoSkillInstall (rename skill_slug→slug, source→install_source) ──

const REPO_SKILL_INSTALL_PAYLOAD: &str = r#"{
    "id": 5,
    "organization_id": 42,
    "market_item_id": 1,
    "installed_by": 7,
    "slug": "git-commit",
    "scope": "org",
    "install_source": "market",
    "source_url": "https://github.com/acme/skills/git-commit",
    "content_sha": "abc123",
    "package_size": 1024,
    "is_enabled": true
}"#;

#[test]
fn repo_skill_install_wasm_relay_preserves_ui_fields() {
    let typed: RepoSkillInstall = serde_json::from_str(REPO_SKILL_INSTALL_PAYLOAD).unwrap();
    assert_eq!(typed.slug.as_deref(), Some("git-commit"));
    assert_eq!(typed.install_source.as_deref(), Some("market"));

    let relayed = serde_json::to_string(&typed).unwrap();
    let parsed: serde_json::Value = serde_json::from_str(&relayed).unwrap();
    for key in [
        "slug",
        "install_source",
        "source_url",
        "is_enabled",
        "package_size",
        "content_sha",
        "market_item_id",
    ] {
        assert!(!parsed[key].is_null(), "field `{key}` dropped by relay");
    }
    assert!(
        parsed.get("skill_slug").is_none(),
        "legacy fabricated `skill_slug` must not leak: {relayed}"
    );
    assert!(
        parsed.get("source").is_none() || !parsed["source"].is_string(),
        "legacy fabricated `source` must not leak (must be `install_source`): {relayed}"
    );
}

#[test]
fn repo_skill_install_list_response_relay_key_matches_backend_wire() {
    let backend_wire = format!(r#"{{"skills":[{REPO_SKILL_INSTALL_PAYLOAD}]}}"#);
    let typed: RepoSkillInstallListResponse = serde_json::from_str(&backend_wire).unwrap();
    assert_eq!(typed.skills.len(), 1);

    let relayed = serde_json::to_string(&typed).unwrap();
    let parsed: serde_json::Value = serde_json::from_str(&relayed).unwrap();
    assert!(parsed.get("skills").is_some(), "must emit `skills`: {relayed}");
}

// ── RepoMcpServerInstall (TS uses command / args / http_url / env_vars) ──

const REPO_MCP_INSTALL_PAYLOAD: &str = r#"{
    "id": 7,
    "organization_id": 42,
    "market_item_id": 3,
    "installed_by": 7,
    "name": "GitHub MCP",
    "slug": "github",
    "transport_type": "http",
    "command": "npx",
    "args": ["mcp-github"],
    "http_url": "https://api.github.com",
    "http_headers": {"Authorization": "Bearer ***"},
    "env_vars": {"GH_TOKEN": "encrypted"},
    "scope": "org",
    "is_enabled": true
}"#;

#[test]
fn repo_mcp_server_install_wasm_relay_preserves_ui_fields() {
    let typed: RepoMcpServerInstall = serde_json::from_str(REPO_MCP_INSTALL_PAYLOAD).unwrap();
    assert_eq!(typed.command.as_deref(), Some("npx"));
    assert!(typed.env_vars.is_some());

    let relayed = serde_json::to_string(&typed).unwrap();
    let parsed: serde_json::Value = serde_json::from_str(&relayed).unwrap();
    for key in [
        "command",
        "args",
        "http_url",
        "http_headers",
        "env_vars",
        "transport_type",
        "slug",
        "is_enabled",
    ] {
        assert!(!parsed[key].is_null(), "field `{key}` dropped by relay");
    }
    assert!(
        parsed.get("repository_id").is_none(),
        "phantom `repository_id` must not leak: {relayed}"
    );
}

#[test]
fn repo_mcp_server_install_list_response_relay_key_matches_backend_wire() {
    let backend_wire = format!(r#"{{"mcp_servers":[{REPO_MCP_INSTALL_PAYLOAD}]}}"#);
    let typed: RepoMcpServerInstallListResponse = serde_json::from_str(&backend_wire).unwrap();
    assert_eq!(typed.mcp_servers.len(), 1);

    let relayed = serde_json::to_string(&typed).unwrap();
    let parsed: serde_json::Value = serde_json::from_str(&relayed).unwrap();
    assert!(
        parsed.get("mcp_servers").is_some(),
        "must emit `mcp_servers`: {relayed}"
    );
}
