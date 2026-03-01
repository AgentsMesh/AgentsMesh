-- =============================================================================
-- AgentsMesh Development Seed Data
-- =============================================================================
--
-- 此脚本创建开发环境所需的初始数据：
-- 1. 测试用户（已激活，可直接登录）
-- 2. 管理员用户（系统管理员，可访问 Admin Console）
-- 3. 组织和成员关系
-- 4. Runner 注册令牌和预注册的 Runner
-- 5. 示例 Ticket
--
-- 普通用户密码: devpass123 (bcrypt hash)
-- 管理员密码: adminpass123 (bcrypt hash)
-- Runner Token: dev-runner-token (用于 docker-compose 中的 runner 服务)
-- =============================================================================

-- 幂等性保护：仅在数据不存在时插入
DO $$
DECLARE
    v_user_id BIGINT;
    v_admin_id BIGINT;
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
    SELECT 'dev@agentsmesh.local', 'devuser', 'Dev User',
           '$2a$10$/95Zk1f1HFGXACwCb.bOw.d3vTjclw5NdGwQuK1Eaji6cDq0PuXp2',
           TRUE, TRUE
    WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = 'dev@agentsmesh.local')
    RETURNING id INTO v_user_id;

    -- 如果用户已存在，获取其 ID
    IF v_user_id IS NULL THEN
        SELECT id INTO v_user_id FROM users WHERE email = 'dev@agentsmesh.local';
    END IF;

    RAISE NOTICE 'User ID: %', v_user_id;

    -- =========================================================================
    -- 1.1 创建管理员用户
    -- =========================================================================
    -- 密码: adminpass123
    -- bcrypt hash (cost=10)
    -- 使用 is_system_admin = TRUE 标记为系统管理员

    INSERT INTO users (email, username, name, password_hash, is_active, is_email_verified, is_system_admin)
    SELECT 'admin@agentsmesh.local', 'admin', 'System Admin',
           '$2a$10$Juf5W26ZmMZUuGNPs2D8beEO9SKY9T1PbeX5ASTNb7E/5wY6oabX6',
           TRUE, TRUE, TRUE
    WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = 'admin@agentsmesh.local')
    RETURNING id INTO v_admin_id;

    -- 如果管理员用户已存在，获取其 ID
    IF v_admin_id IS NULL THEN
        SELECT id INTO v_admin_id FROM users WHERE email = 'admin@agentsmesh.local';
    END IF;

    RAISE NOTICE 'Admin User ID: %', v_admin_id;

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
    -- 3.1 创建 Pro 订阅 (plan_id = 2)
    -- =========================================================================
    -- Pro 计划：10 concurrent pods, 10 runners, 10 users

    INSERT INTO subscriptions (
        organization_id, plan_id, status, billing_cycle,
        current_period_start, current_period_end,
        auto_renew, seat_count
    )
    SELECT v_org_id, 2, 'active', 'monthly',
           NOW(), NOW() + INTERVAL '30 days',
           TRUE, 10
    WHERE NOT EXISTS (
        SELECT 1 FROM subscriptions WHERE organization_id = v_org_id
    );

    -- =========================================================================
    -- 4. 创建 Runner 注册令牌 (gRPC/mTLS)
    -- =========================================================================
    -- Token: dev-runner-token
    -- bcrypt hash (cost=10)

    INSERT INTO runner_grpc_registration_tokens (
        organization_id, token_hash, description, created_by_id, is_active, max_uses
    )
    SELECT v_org_id,
           '$2a$10$Q7dK5K91JqD8ZhTqXyQYj.cRmlKn9crzuMkYb6gvUdEP3zu/RkzE2',
           'Development Runner Token',
           v_user_id,
           TRUE,
           NULL  -- Unlimited uses
    WHERE NOT EXISTS (
        SELECT 1 FROM runner_grpc_registration_tokens
        WHERE organization_id = v_org_id
        AND description = 'Development Runner Token'
    )
    RETURNING id INTO v_token_id;

    -- =========================================================================
    -- 5. 预注册 Runner (使用证书认证)
    -- =========================================================================
    -- Runner 使用 mTLS 证书认证，不再使用 auth_token_hash
    -- 证书在 dev.sh 中生成并挂载到 runner 容器
    -- cert_serial_number 在 runner 首次连接时由 backend 自动填充

    INSERT INTO runners (
        organization_id, node_id, description,
        status, max_concurrent_pods
    )
    SELECT v_org_id,
           'dev-runner',
           'Development Docker Runner',
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
    -- slug 格式: DEV-{number}
    -- number 是组织内自增的

    INSERT INTO tickets (
        organization_id, number, slug, title, content,
        status, priority, reporter_id
    )
    SELECT v_org_id,
           1,
           'DEV-1',
           'Implement JWT-based user authentication',
           E'## Objective\nBuild a secure JWT-based authentication system for the platform.\n\n## Tasks\n- [ ] Login API endpoint\n- [ ] Registration API endpoint\n- [ ] Token refresh mechanism\n- [ ] Password reset flow',
           'backlog',
           'medium',
           v_user_id
    WHERE NOT EXISTS (
        SELECT 1 FROM tickets
        WHERE slug = 'DEV-1'
    );

    INSERT INTO tickets (
        organization_id, number, slug, title, content,
        status, priority, reporter_id
    )
    SELECT v_org_id,
           2,
           'DEV-2',
           'Fix slow page load time on dashboard',
           E'## Problem\nThe dashboard page takes over 3 seconds to load, causing poor user experience.\n\n## Steps to Reproduce\n1. Navigate to the dashboard\n2. Observe the loading time with DevTools Network tab\n\n## Expected Behavior\nPage should load within 1 second.',
           'backlog',
           'high',
           v_user_id
    WHERE NOT EXISTS (
        SELECT 1 FROM tickets
        WHERE slug = 'DEV-2'
    );

    -- =========================================================================
    -- 7. 创建 User Repository Provider (Local Gitea)
    -- =========================================================================
    -- 为 dev user 创建本地 Gitea Provider
    -- Runner 使用 runner_local 模式 (容器内 ~/.ssh/ 的 deploy key)
    -- 不需要 Bot Token (Gitea API 由 init 脚本管理)

    INSERT INTO user_repository_providers (
        user_id, provider_type, name, base_url,
        is_default, is_active
    )
    SELECT v_user_id,
           'gitlab',
           'Local Gitea',
           'http://gitea:3000',
           TRUE,
           TRUE
    WHERE NOT EXISTS (
        SELECT 1 FROM user_repository_providers
        WHERE user_id = v_user_id
          AND name = 'Local Gitea'
    );

    -- =========================================================================
    -- 8. 创建示例 Repositories (Local Gitea)
    -- =========================================================================
    -- 使用本地 Gitea Git 服务器上的测试仓库
    -- SSH Deploy Key 配置在 deploy/dev/runner-ssh/ 目录
    -- 并由 gitea/init-gitea.sh 自动注册到 Gitea

    -- 8.1 Demo WebApp (静态 Web 应用)
    INSERT INTO repositories (
        organization_id, provider_type, provider_base_url,
        external_id, name, full_path, clone_url,
        default_branch, ticket_prefix, visibility, imported_by_user_id,
        is_active
    )
    SELECT v_org_id,
           'gitlab',
           'http://gitea:3000',
           '1',
           'Demo WebApp',
           'dev-org/demo-webapp',
           'git@gitea:dev-org/demo-webapp.git',
           'main',
           'WEB',
           'organization',
           v_user_id,
           TRUE
    WHERE NOT EXISTS (
        SELECT 1 FROM repositories
        WHERE organization_id = v_org_id AND full_path = 'dev-org/demo-webapp'
    );

    -- 8.2 Demo API (Go API 项目)
    INSERT INTO repositories (
        organization_id, provider_type, provider_base_url,
        external_id, name, full_path, clone_url,
        default_branch, ticket_prefix, visibility, imported_by_user_id,
        is_active
    )
    SELECT v_org_id,
           'gitlab',
           'http://gitea:3000',
           '2',
           'Demo API',
           'dev-org/demo-api',
           'git@gitea:dev-org/demo-api.git',
           'main',
           'API',
           'organization',
           v_user_id,
           TRUE
    WHERE NOT EXISTS (
        SELECT 1 FROM repositories
        WHERE organization_id = v_org_id AND full_path = 'dev-org/demo-api'
    );

    RAISE NOTICE 'Seed data created successfully!';
    RAISE NOTICE '  - User: dev@agentsmesh.local / devpass123';
    RAISE NOTICE '  - Admin: admin@agentsmesh.local / adminpass123';
    RAISE NOTICE '  - Organization: dev-org';
    RAISE NOTICE '  - Runner: dev-runner (node_id)';
    RAISE NOTICE '  - Git Provider: Local Gitea (http://gitea:3000)';
    RAISE NOTICE '  - Repository: dev-org/demo-webapp (Gitea)';
    RAISE NOTICE '  - Repository: dev-org/demo-api (Gitea)';

END $$;
