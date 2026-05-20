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
    v_user2_id BIGINT;
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
    -- 2.1 创建第二个测试用户（同组织成员，用于多用户测试）
    -- =========================================================================
    -- 密码: devpass123 (与主测试用户相同)

    INSERT INTO users (email, username, name, password_hash, is_active, is_email_verified)
    SELECT 'dev2@agentsmesh.local', 'devuser2', 'Dev User 2',
           '$2a$10$/95Zk1f1HFGXACwCb.bOw.d3vTjclw5NdGwQuK1Eaji6cDq0PuXp2',
           TRUE, TRUE
    WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = 'dev2@agentsmesh.local')
    RETURNING id INTO v_user2_id;

    IF v_user2_id IS NULL THEN
        SELECT id INTO v_user2_id FROM users WHERE email = 'dev2@agentsmesh.local';
    END IF;

    RAISE NOTICE 'User 2 ID: %', v_user2_id;

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
    -- 3.1 添加第二用户为组织成员
    -- =========================================================================

    INSERT INTO organization_members (organization_id, user_id, role)
    SELECT v_org_id, v_user2_id, 'member'
    WHERE NOT EXISTS (
        SELECT 1 FROM organization_members
        WHERE organization_id = v_org_id AND user_id = v_user2_id
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
    -- SHA256 hash of "dev-runner-token"
    -- echo -n 'dev-runner-token' | shasum -a 256

    INSERT INTO runner_grpc_registration_tokens (
        organization_id, token_hash, description, created_by_id, is_active, max_uses
    )
    SELECT v_org_id,
           'cee9d12fb9fefdfafe98d97f5c8a247e071a0e6778089dee7cf2be571ee606d2',
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

    -- Second runner row so dev-runner-2 (started by docker-compose for
    -- mcp-e2e cross-runner specs) can register. The container's mTLS cert
    -- carries CN=dev-runner-2; backend matches that against this row to
    -- attach the gRPC stream.
    INSERT INTO runners (
        organization_id, node_id, description,
        status, max_concurrent_pods
    )
    SELECT v_org_id,
           'dev-runner-2',
           'Development Docker Runner (cross-runner e2e)',
           'offline',
           10
    WHERE NOT EXISTS (
        SELECT 1 FROM runners
        WHERE organization_id = v_org_id AND node_id = 'dev-runner-2'
    );

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
        external_id, name, slug, http_clone_url,
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
        WHERE organization_id = v_org_id AND slug = 'dev-org/demo-webapp'
    );

    -- 8.2 Demo API (Go API 项目)
    INSERT INTO repositories (
        organization_id, provider_type, provider_base_url,
        external_id, name, slug, http_clone_url,
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
        WHERE organization_id = v_org_id AND slug = 'dev-org/demo-api'
    );

    RAISE NOTICE 'Seed data created successfully!';
    RAISE NOTICE '  - User: dev@agentsmesh.local / devpass123';
    RAISE NOTICE '  - User 2: dev2@agentsmesh.local / devpass123';
    RAISE NOTICE '  - Admin: admin@agentsmesh.local / adminpass123';
    RAISE NOTICE '  - Organization: dev-org (dev + dev2)';
    RAISE NOTICE '  - Runner: dev-runner (node_id)';
    RAISE NOTICE '  - Git Provider: Local Gitea (http://gitea:3000)';
    RAISE NOTICE '  - Repository: dev-org/demo-webapp (Gitea)';
    RAISE NOTICE '  - Repository: dev-org/demo-api (Gitea)';

    -- =========================================================================
    -- 9. 创建示例 Channel 和 Messages (各种消息结构)
    -- =========================================================================
    -- 展示新的结构化消息模型: body + content JSONB + mentions JSONB

    DECLARE
        v_ch_general_id BIGINT;
        v_ch_dev_id BIGINT;
    BEGIN

    -- 9.1 General 频道
    -- slug 必须显式提供（Phase 4 收尾 migration 000143 让 channels.slug NOT NULL）
    INSERT INTO channels (organization_id, name, slug, description, visibility, created_by_user_id)
    SELECT v_org_id, 'general', 'general', 'General discussion channel', 'public', v_user_id
    WHERE NOT EXISTS (SELECT 1 FROM channels WHERE organization_id = v_org_id AND name = 'general')
    RETURNING id INTO v_ch_general_id;

    IF v_ch_general_id IS NULL THEN
        SELECT id INTO v_ch_general_id FROM channels WHERE organization_id = v_org_id AND name = 'general';
    END IF;

    -- 9.2 Dev Discussion 频道
    INSERT INTO channels (organization_id, name, slug, description, visibility, created_by_user_id)
    SELECT v_org_id, 'dev-discussion', 'dev-discussion', 'Development team discussion', 'public', v_user_id
    WHERE NOT EXISTS (SELECT 1 FROM channels WHERE organization_id = v_org_id AND name = 'dev-discussion')
    RETURNING id INTO v_ch_dev_id;

    IF v_ch_dev_id IS NULL THEN
        SELECT id INTO v_ch_dev_id FROM channels WHERE organization_id = v_org_id AND name = 'dev-discussion';
    END IF;

    -- 频道成员
    INSERT INTO channel_members (channel_id, user_id, role) VALUES (v_ch_general_id, v_user_id, 'creator') ON CONFLICT DO NOTHING;
    INSERT INTO channel_members (channel_id, user_id, role) VALUES (v_ch_general_id, v_user2_id, 'member') ON CONFLICT DO NOTHING;
    INSERT INTO channel_members (channel_id, user_id, role) VALUES (v_ch_general_id, v_admin_id, 'member') ON CONFLICT DO NOTHING;
    INSERT INTO channel_members (channel_id, user_id, role) VALUES (v_ch_dev_id, v_user_id, 'creator') ON CONFLICT DO NOTHING;
    INSERT INTO channel_members (channel_id, user_id, role) VALUES (v_ch_dev_id, v_user2_id, 'member') ON CONFLICT DO NOTHING;

    -- 仅在频道消息为空时插入示例消息
    IF NOT EXISTS (SELECT 1 FROM channel_messages WHERE channel_id = v_ch_general_id LIMIT 1) THEN

    -- ── 纯文本消息 ──
    INSERT INTO channel_messages (channel_id, sender_user_id, message_type, body, content, mentions) VALUES
    (v_ch_general_id, v_user_id, 'text',
     'Welcome to the general channel! This is where we discuss everything.',
     '{"kind":"text","blocks":[{"type":"paragraph","elements":[{"type":"text","text":"Welcome to the general channel! This is where we discuss everything."}]}]}',
     '{}');

    -- ── 多段落消息 ──
    INSERT INTO channel_messages (channel_id, sender_user_id, message_type, body, content, mentions) VALUES
    (v_ch_general_id, v_user2_id, 'text',
     E'I''ve been looking into the auth module.\nThere are a few issues we need to address.\nLet''s discuss in the next standup.',
     '{"kind":"text","blocks":[{"type":"paragraph","elements":[{"type":"text","text":"I''ve been looking into the auth module."}]},{"type":"paragraph","elements":[{"type":"text","text":"There are a few issues we need to address."}]},{"type":"paragraph","elements":[{"type":"text","text":"Let''s discuss in the next standup."}]}]}',
     '{}');

    -- ── 富文本格式（bold, italic, code）──
    INSERT INTO channel_messages (channel_id, sender_user_id, message_type, body, content, mentions) VALUES
    (v_ch_general_id, v_user_id, 'text',
     'The fix is in the handleAuth function — we need to validate the token before proceeding.',
     '{"kind":"text","blocks":[{"type":"paragraph","elements":[{"type":"text","text":"The fix is in the "},{"type":"text","text":"handleAuth","bold":true},{"type":"text","text":" function — we need to "},{"type":"text","text":"validate the token","italic":true},{"type":"text","text":" before proceeding."}]}]}',
     '{}');

    -- ── 用户 @mention 消息 ──
    INSERT INTO channel_messages (channel_id, sender_user_id, message_type, body, content, mentions) VALUES
    (v_ch_general_id, v_user_id, 'text',
     E'@devuser2 can you review the PR for the login flow?',
     ('{"kind":"text","blocks":[{"type":"paragraph","elements":[{"type":"mention","entity_type":"user","entity_key":"' || v_user2_id || '","display":"devuser2"},{"type":"text","text":" can you review the PR for the login flow?"}]}]}')::jsonb,
     ('{"users":[' || v_user2_id || ']}')::jsonb);

    -- ── 链接消息 ──
    INSERT INTO channel_messages (channel_id, sender_user_id, message_type, body, content, mentions) VALUES
    (v_ch_general_id, v_user2_id, 'text',
     'Sure! Here is the PR link: Review PR #42',
     '{"kind":"text","blocks":[{"type":"paragraph","elements":[{"type":"text","text":"Sure! Here is the PR link: "},{"type":"link","text":"Review PR #42","url":"https://github.com/example/repo/pull/42"}]}]}',
     '{}');

    -- ── inline code 消息 ──
    INSERT INTO channel_messages (channel_id, sender_user_id, message_type, body, content, mentions) VALUES
    (v_ch_general_id, v_user_id, 'text',
     'Try running npm install and then npm test to reproduce.',
     '{"kind":"text","blocks":[{"type":"paragraph","elements":[{"type":"text","text":"Try running "},{"type":"text","text":"npm install","code":true},{"type":"text","text":" and then "},{"type":"text","text":"npm test","code":true},{"type":"text","text":" to reproduce."}]}]}',
     '{}');

    -- ── strikethrough 消息 ──
    INSERT INTO channel_messages (channel_id, sender_user_id, message_type, body, content, mentions) VALUES
    (v_ch_general_id, v_user2_id, 'text',
     'The deadline is Friday Monday — we got an extension.',
     '{"kind":"text","blocks":[{"type":"paragraph","elements":[{"type":"text","text":"The deadline is "},{"type":"text","text":"Friday","strike":true},{"type":"text","text":" "},{"type":"text","text":"Monday","bold":true},{"type":"text","text":" — we got an extension."}]}]}',
     '{}');

    -- ── 系统消息 ──
    INSERT INTO channel_messages (channel_id, message_type, body, content, mentions) VALUES
    (v_ch_general_id, 'system',
     'Dev User 2 joined the channel',
     NULL,
     '{}');

    END IF;

    -- dev-discussion 频道消息
    IF NOT EXISTS (SELECT 1 FROM channel_messages WHERE channel_id = v_ch_dev_id LIMIT 1) THEN

    -- ── 混合格式 + mention ──
    INSERT INTO channel_messages (channel_id, sender_user_id, message_type, body, content, mentions) VALUES
    (v_ch_dev_id, v_user_id, 'text',
     E'@devuser2 I pushed a fix for the API timeout issue. The root cause was in the middleware — see the docs for details.',
     ('{"kind":"text","blocks":[{"type":"paragraph","elements":[{"type":"mention","entity_type":"user","entity_key":"' || v_user2_id || '","display":"devuser2"},{"type":"text","text":" I pushed a fix for the "},{"type":"text","text":"API timeout","bold":true},{"type":"text","text":" issue."}]},{"type":"paragraph","elements":[{"type":"text","text":"The root cause was in the "},{"type":"text","text":"middleware","code":true},{"type":"text","text":" — see the "},{"type":"link","text":"docs","url":"https://docs.example.com/middleware"},{"type":"text","text":" for details."}]}]}')::jsonb,
     ('{"users":[' || v_user2_id || ']}')::jsonb);

    -- ── 回复消息 ──
    INSERT INTO channel_messages (channel_id, sender_user_id, message_type, body, content, mentions) VALUES
    (v_ch_dev_id, v_user2_id, 'text',
     'Thanks! The fix looks clean. Approved.',
     '{"kind":"text","blocks":[{"type":"paragraph","elements":[{"type":"text","text":"Thanks! The fix looks "},{"type":"text","text":"clean","italic":true},{"type":"text","text":". "},{"type":"text","text":"Approved.","bold":true}]}]}',
     '{}');

    -- ── 带 linebreak 的消息 ──
    INSERT INTO channel_messages (channel_id, sender_user_id, message_type, body, content, mentions) VALUES
    (v_ch_dev_id, v_user_id, 'text',
     E'Deployment checklist:\n1. Run migrations\n2. Verify health endpoints\n3. Monitor error rates',
     '{"kind":"text","blocks":[{"type":"paragraph","elements":[{"type":"text","text":"Deployment checklist:"},{"type":"linebreak"},{"type":"text","text":"1. Run migrations"},{"type":"linebreak"},{"type":"text","text":"2. Verify health endpoints"},{"type":"linebreak"},{"type":"text","text":"3. Monitor error rates"}]}]}',
     '{}');

    END IF;

    RAISE NOTICE '  - Channel: general (with sample messages)';
    RAISE NOTICE '  - Channel: dev-discussion (with sample messages)';

    END; -- inner DECLARE block

END $$;
