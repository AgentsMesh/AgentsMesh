-- Runner diagnostic log upload records
CREATE TABLE runner_logs (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    runner_id BIGINT NOT NULL REFERENCES runners(id) ON DELETE CASCADE,
    request_id VARCHAR(36) NOT NULL UNIQUE,
    storage_key VARCHAR(500),
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'collecting', 'uploading', 'completed', 'failed')),
    size_bytes BIGINT DEFAULT 0,
    error_message TEXT,
    requested_by_id BIGINT NOT NULL REFERENCES users(id),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP
);

CREATE INDEX idx_runner_logs_org_runner ON runner_logs(organization_id, runner_id, created_at DESC);
