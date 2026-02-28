-- Loop definitions table
--
-- API triggering uses the unified API Key system (loops:read, loops:write scopes),
-- not per-loop trigger tokens.
--
-- No FK constraints by design: data integrity is enforced at the application layer.
-- Rationale:
--   - FK couples tables at the DB level, causing cascading locks and blocking deletes
--   - ON DELETE CASCADE is dangerous in production (silent mass deletion)
--   - Tenant isolation is enforced by middleware (organization_id in every query)
--   - Reference validity (agent_type_id, runner_id, etc.) is validated at create/update time
--   - Dangling references are handled gracefully (null checks, fallback behavior)
CREATE TABLE loops (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    description TEXT,

    -- Agent configuration
    agent_type_id BIGINT,
    custom_agent_type_id BIGINT,
    permission_mode VARCHAR(50) NOT NULL DEFAULT 'bypassPermissions',

    -- Prompt
    prompt_template TEXT NOT NULL,
    prompt_variables JSONB DEFAULT '{}',

    -- Resource bindings (application-level references, no FK)
    repository_id BIGINT,
    runner_id BIGINT,
    branch_name VARCHAR(255),
    ticket_id BIGINT,
    credential_profile_id BIGINT,
    config_overrides JSONB DEFAULT '{}',

    -- Execution configuration
    execution_mode VARCHAR(20) NOT NULL DEFAULT 'autopilot',
    -- Cron is optional; all loops support API triggering via the unified API Key system
    cron_expression VARCHAR(100),

    -- Autopilot config (only used when execution_mode=autopilot)
    autopilot_config JSONB NOT NULL DEFAULT '{}',

    -- Webhook callback URL (POST run result when completed)
    callback_url VARCHAR(500),

    -- Status and policies
    status VARCHAR(20) NOT NULL DEFAULT 'enabled',
    -- persistent = reuse sandbox + agent session; fresh = clean slate each run
    sandbox_strategy VARCHAR(20) NOT NULL DEFAULT 'persistent',
    -- Whether to keep agent session (conversation history) across runs
    session_persistence BOOLEAN NOT NULL DEFAULT true,
    concurrency_policy VARCHAR(20) NOT NULL DEFAULT 'skip',
    max_concurrent_runs INT NOT NULL DEFAULT 1,
    timeout_minutes INT NOT NULL DEFAULT 60,

    -- Runtime state
    sandbox_path VARCHAR(500),
    last_pod_key VARCHAR(100),

    -- Ownership (application-level reference, no FK)
    created_by_id BIGINT NOT NULL,

    -- Statistics (denormalized)
    total_runs INT NOT NULL DEFAULT 0,
    successful_runs INT NOT NULL DEFAULT 0,
    failed_runs INT NOT NULL DEFAULT 0,

    -- Timing
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Unique slug within organization
CREATE UNIQUE INDEX idx_loops_org_slug ON loops(organization_id, slug);

-- Index for cron scheduler: find enabled loops with cron that are due
CREATE INDEX idx_loops_cron_due ON loops(next_run_at)
    WHERE status = 'enabled' AND cron_expression IS NOT NULL;

-- Index for listing by organization
CREATE INDEX idx_loops_org_status ON loops(organization_id, status);

-- Loop execution records table
--
-- No FK constraints — same rationale as loops table.
-- Parent-child integrity (loop_id → loops) is enforced by:
--   - TriggerRunAtomic: reads loop within FOR UPDATE transaction before creating run
--   - Delete: atomic subquery checks active runs before allowing loop deletion
--   - Orphan cleanup: periodic job marks orphan pending runs as failed
CREATE TABLE loop_runs (
    id BIGSERIAL PRIMARY KEY,
    organization_id BIGINT NOT NULL,
    loop_id BIGINT NOT NULL,
    run_number INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    pod_key VARCHAR(100),
    autopilot_controller_key VARCHAR(100),
    -- How this run was triggered: 'cron', 'api', 'manual'
    trigger_type VARCHAR(20) NOT NULL,
    trigger_source VARCHAR(255),
    trigger_params JSONB DEFAULT '{}',
    resolved_prompt TEXT,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    duration_sec INT,
    exit_summary TEXT,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Index for listing runs by loop (most recent first)
CREATE INDEX idx_loop_runs_loop_id ON loop_runs(loop_id, created_at DESC);

-- Index for finding active runs (concurrency checks)
CREATE INDEX idx_loop_runs_active ON loop_runs(loop_id, status) WHERE status IN ('pending', 'running');

-- Unique run number within a loop
CREATE UNIQUE INDEX idx_loop_runs_loop_number ON loop_runs(loop_id, run_number);

-- Index for looking up runs by pod_key (used by event handlers)
CREATE INDEX idx_loop_runs_pod_key ON loop_runs(pod_key) WHERE pod_key IS NOT NULL;

-- Index for looking up runs by autopilot_controller_key (used by event handlers)
CREATE INDEX idx_loop_runs_autopilot_key ON loop_runs(autopilot_controller_key) WHERE autopilot_controller_key IS NOT NULL;
