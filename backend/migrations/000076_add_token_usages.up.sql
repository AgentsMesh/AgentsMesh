-- Token usage records: one row per model per pod session
CREATE TABLE token_usages (
    id                    BIGSERIAL PRIMARY KEY,
    organization_id       BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    pod_id                BIGINT REFERENCES pods(id) ON DELETE SET NULL,
    pod_key               VARCHAR(100) NOT NULL,
    user_id               BIGINT REFERENCES users(id) ON DELETE SET NULL,
    runner_id             BIGINT REFERENCES runners(id) ON DELETE SET NULL,
    agent_type_slug       VARCHAR(50) NOT NULL,
    model                 VARCHAR(100),
    input_tokens          BIGINT NOT NULL DEFAULT 0,
    output_tokens         BIGINT NOT NULL DEFAULT 0,
    cache_creation_tokens BIGINT NOT NULL DEFAULT 0,
    cache_read_tokens     BIGINT NOT NULL DEFAULT 0,
    session_started_at    TIMESTAMPTZ,
    session_ended_at      TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_token_usages_org_created ON token_usages(organization_id, created_at);
CREATE INDEX idx_token_usages_org_agent   ON token_usages(organization_id, agent_type_slug, created_at);
CREATE INDEX idx_token_usages_org_user    ON token_usages(organization_id, user_id, created_at);
CREATE INDEX idx_token_usages_pod_key     ON token_usages(pod_key);
CREATE INDEX idx_token_usages_org_model   ON token_usages(organization_id, model, created_at);
