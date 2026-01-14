-- =============================================================================
-- AgentMesh Development Seed Data
-- =============================================================================
--
-- 此脚本创建开发环境所需的初始数据：
-- 1. 测试用户（已激活，可直接登录）
-- 2. 组织和成员关系
-- 3. Runner 注册令牌和预注册的 Runner
-- 4. 示例 Ticket
--
-- 密码: devpass123 (bcrypt hash)
-- Runner Token: dev-runner-token (用于 docker-compose 中的 runner 服务)
-- =============================================================================

-- 幂等性保护：仅在数据不存在时插入
DO $$
DECLARE
    v_user_id BIGINT;
    v_org_id BIGINT;
    v_token_id BIGINT;
    v_runner_id BIGINT;
BEGIN
    -- =========================================================================
    -- 1. 创建测试用户
    -- =========================================================================
    -- 密码: devpass123
    -- bcrypt hash (cost=10)

    INSERT INTO users (email, username, name, password_hash, is_active, is_email_verified)
    SELECT 'dev@agentmesh.local', 'devuser', 'Dev User',
           '$2a$10$/95Zk1f1HFGXACwCb.bOw.d3vTjclw5NdGwQuK1Eaji6cDq0PuXp2',
           TRUE, TRUE
    WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = 'dev@agentmesh.local')
    RETURNING id INTO v_user_id;

    -- 如果用户已存在，获取其 ID
    IF v_user_id IS NULL THEN
        SELECT id INTO v_user_id FROM users WHERE email = 'dev@agentmesh.local';
    END IF;

    RAISE NOTICE 'User ID: %', v_user_id;

    -- =========================================================================
    -- 2. 创建组织
    -- =========================================================================

    INSERT INTO organizations (name, slug, subscription_plan, subscription_status)
    SELECT 'Dev Organization', 'dev-org', 'pro', 'active'
    WHERE NOT EXISTS (SELECT 1 FROM organizations WHERE slug = 'dev-org')
    RETURNING id INTO v_org_id;

    -- 如果组织已存在，获取其 ID
    IF v_org_id IS NULL THEN
        SELECT id INTO v_org_id FROM organizations WHERE slug = 'dev-org';
    END IF;

    RAISE NOTICE 'Organization ID: %', v_org_id;

    -- =========================================================================
    -- 3. 添加用户为组织所有者
    -- =========================================================================

    INSERT INTO organization_members (organization_id, user_id, role)
    SELECT v_org_id, v_user_id, 'owner'
    WHERE NOT EXISTS (
        SELECT 1 FROM organization_members
        WHERE organization_id = v_org_id AND user_id = v_user_id
    );

    -- =========================================================================
    -- 4. 创建 Runner 注册令牌
    -- =========================================================================
    -- Token: dev-runner-token
    -- bcrypt hash (cost=10)

    INSERT INTO runner_registration_tokens (
        organization_id, token_hash, description, created_by_id, is_active, max_uses
    )
    SELECT v_org_id,
           '$2a$10$Q7dK5K91JqD8ZhTqXyQYj.cRmlKn9crzuMkYb6gvUdEP3zu/RkzE2',
           'Development Runner Token',
           v_user_id,
           TRUE,
           NULL  -- Unlimited uses
    WHERE NOT EXISTS (
        SELECT 1 FROM runner_registration_tokens
        WHERE organization_id = v_org_id
        AND description = 'Development Runner Token'
    )
    RETURNING id INTO v_token_id;

    -- =========================================================================
    -- 5. 预注册 Runner
    -- =========================================================================
    -- Auth Token: dev-runner-auth-token
    -- bcrypt hash (cost=10): $2a$10$YourHashHere...
    --
    -- 注意: Runner 会使用 RUNNER_TOKEN 环境变量连接，
    -- 这里预注册的 runner 使用固定的 node_id: dev-runner

    INSERT INTO runners (
        organization_id, node_id, description, auth_token_hash,
        status, max_concurrent_pods
    )
    SELECT v_org_id,
           'dev-runner',
           'Development Docker Runner',
           -- Auth token: dev-runner-auth-token (bcrypt hash)
           '$2a$10$XdoTNeuXKRVuNf4QUxeBSOhLkCe5A2AdcLItEXbtGRjg4B8wGLuHG',
           'offline',
           10
    WHERE NOT EXISTS (
        SELECT 1 FROM runners
        WHERE organization_id = v_org_id AND node_id = 'dev-runner'
    )
    RETURNING id INTO v_runner_id;

    IF v_runner_id IS NULL THEN
        SELECT id INTO v_runner_id FROM runners
        WHERE organization_id = v_org_id AND node_id = 'dev-runner';
    END IF;

    RAISE NOTICE 'Runner ID: %', v_runner_id;

    -- =========================================================================
    -- 6. 创建示例 Ticket
    -- =========================================================================
    -- identifier 格式: DEV-{number}
    -- number 是组织内自增的

    INSERT INTO tickets (
        organization_id, number, identifier, title, description,
        status, type, priority, reporter_id
    )
    SELECT v_org_id,
           1,
           'DEV-1',
           '示例任务：实现用户认证功能',
           E'## 目标\n实现基于 JWT 的用户认证系统。\n\n## 任务清单\n- [ ] 登录接口\n- [ ] 注册接口\n- [ ] Token 刷新',
           'backlog',
           'feature',
           'medium',
           v_user_id
    WHERE NOT EXISTS (
        SELECT 1 FROM tickets
        WHERE identifier = 'DEV-1'
    );

    INSERT INTO tickets (
        organization_id, number, identifier, title, description,
        status, type, priority, reporter_id
    )
    SELECT v_org_id,
           2,
           'DEV-2',
           '示例 Bug：修复页面加载缓慢问题',
           E'## 问题描述\n首页加载时间超过 3 秒。\n\n## 复现步骤\n1. 打开首页\n2. 观察加载时间',
           'backlog',
           'bug',
           'high',
           v_user_id
    WHERE NOT EXISTS (
        SELECT 1 FROM tickets
        WHERE identifier = 'DEV-2'
    );

    RAISE NOTICE 'Seed data created successfully!';
    RAISE NOTICE '  - User: dev@agentmesh.local / devpass123';
    RAISE NOTICE '  - Organization: dev-org';
    RAISE NOTICE '  - Runner: dev-runner (node_id)';

END $$;
