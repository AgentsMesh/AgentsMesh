-- Block Store Phase 1: Block + Ref two-primitive foundation
-- See Analytics/Blockstore/docs/02_architecture_analysis.md

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Workspace: a named namespace inside an organization
CREATE TABLE block_workspaces (
    id              UUID PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    slug            VARCHAR(64) NOT NULL,
    name            VARCHAR(200) NOT NULL,
    root_block_id   UUID,
    created_by      BIGINT NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (organization_id, slug)
);
CREATE INDEX idx_block_workspaces_org ON block_workspaces (organization_id);

-- Block: minimal addressable data unit; no relationships on this table
CREATE TABLE blocks (
    id              UUID PRIMARY KEY,
    workspace_id    UUID NOT NULL REFERENCES block_workspaces(id) ON DELETE CASCADE,
    type            VARCHAR(64) NOT NULL,
    data            JSONB NOT NULL DEFAULT '{}'::jsonb,
    text            TEXT,
    meta            JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by      BIGINT NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    tsv             TSVECTOR GENERATED ALWAYS AS (to_tsvector('simple', COALESCE(text, ''))) STORED
);
CREATE INDEX idx_blocks_workspace_type ON blocks (workspace_id, type) WHERE deleted_at IS NULL;
CREATE INDEX idx_blocks_workspace_updated ON blocks (workspace_id, updated_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_blocks_tsv ON blocks USING GIN (tsv);
CREATE INDEX idx_blocks_data ON blocks USING GIN (data);

-- Ref: the only relationship primitive; rel differentiates nest / mention / embed / depends_on / ...
CREATE TABLE block_refs (
    id              BIGSERIAL PRIMARY KEY,
    workspace_id    UUID NOT NULL REFERENCES block_workspaces(id) ON DELETE CASCADE,
    from_id         UUID NOT NULL REFERENCES blocks(id) ON DELETE CASCADE,
    to_id           UUID NOT NULL REFERENCES blocks(id),
    rel             VARCHAR(64) NOT NULL,
    order_key       TEXT,
    anchor          TEXT,
    meta            JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by      BIGINT NOT NULL REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
-- UNIQUE with an expression must be a UNIQUE INDEX, not a table constraint.
CREATE UNIQUE INDEX idx_block_refs_unique_edge
    ON block_refs (from_id, to_id, rel, COALESCE(anchor, ''));
CREATE INDEX idx_block_refs_children ON block_refs (from_id, rel, order_key);
CREATE INDEX idx_block_refs_backlinks ON block_refs (to_id, rel);
CREATE UNIQUE INDEX idx_block_refs_single_nest_parent ON block_refs (to_id) WHERE rel = 'nest';

-- Op log: the single source of truth for collaboration / audit / undo
CREATE TABLE block_ops (
    id              BIGSERIAL PRIMARY KEY,
    workspace_id    UUID NOT NULL REFERENCES block_workspaces(id) ON DELETE CASCADE,
    idempotency_key VARCHAR(128) UNIQUE,
    actor_type      VARCHAR(16) NOT NULL,
    actor_id        BIGINT NOT NULL,
    op              VARCHAR(32) NOT NULL,
    target_block    UUID,
    target_ref      BIGINT,
    payload         JSONB NOT NULL,
    forward         JSONB NOT NULL,
    inverse         JSONB NOT NULL,
    parent_op_id    BIGINT,
    applied_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX idx_block_ops_stream ON block_ops (workspace_id, id);
CREATE INDEX idx_block_ops_actor ON block_ops (actor_type, actor_id);

ALTER TABLE block_workspaces
    ADD CONSTRAINT fk_block_workspaces_root
    FOREIGN KEY (root_block_id) REFERENCES blocks(id) ON DELETE SET NULL;
