-- =============================================================================
-- AgentsMesh Self-Hosted Seed Data
-- =============================================================================
--
-- Creates initial data:
-- 1. Admin user (admin@localhost.local / Admin@123)
-- 2. Default organization
-- 3. Runner registration token
--
-- =============================================================================

DO $$
DECLARE
    v_admin_id BIGINT;
    v_org_id BIGINT;
    v_token_id BIGINT;
BEGIN
    -- =========================================================================
    -- 1. Create Admin User
    -- =========================================================================
    -- Email: admin@localhost.local
    -- Password: Admin@123 (bcrypt hash, cost=10)

    INSERT INTO users (email, username, name, password_hash, is_active, is_email_verified, is_system_admin)
    SELECT 'admin@localhost.local', 'admin', 'System Administrator',
           '$2a$10$/sqBNKm8PFodPmVg8PjQMOMWzl03gjRpPWMRy31UbNzAGW/25r12C',
           TRUE, TRUE, TRUE
    WHERE NOT EXISTS (SELECT 1 FROM users WHERE email = 'admin@localhost.local')
    RETURNING id INTO v_admin_id;

    IF v_admin_id IS NULL THEN
        SELECT id INTO v_admin_id FROM users WHERE email = 'admin@localhost.local';
    END IF;

    RAISE NOTICE 'Admin User ID: %', v_admin_id;

    -- =========================================================================
    -- 2. Create Default Organization
    -- =========================================================================

    INSERT INTO organizations (name, slug, subscription_plan, subscription_status)
    SELECT 'Default Organization', 'default', 'selfhost', 'active'
    WHERE NOT EXISTS (SELECT 1 FROM organizations WHERE slug = 'default')
    RETURNING id INTO v_org_id;

    IF v_org_id IS NULL THEN
        SELECT id INTO v_org_id FROM organizations WHERE slug = 'default';
    END IF;

    RAISE NOTICE 'Organization ID: %', v_org_id;

    -- =========================================================================
    -- 3. Add Admin as Organization Owner
    -- =========================================================================

    INSERT INTO organization_members (organization_id, user_id, role)
    SELECT v_org_id, v_admin_id, 'owner'
    WHERE NOT EXISTS (
        SELECT 1 FROM organization_members
        WHERE organization_id = v_org_id AND user_id = v_admin_id
    );

    -- =========================================================================
    -- 4. Create Subscription
    -- =========================================================================

    INSERT INTO subscriptions (
        organization_id, plan_id, status, billing_cycle,
        current_period_start, current_period_end,
        auto_renew, seat_count
    )
    SELECT v_org_id, 4, 'active', 'yearly',
           NOW(), NOW() + INTERVAL '365 days',
           TRUE, 9999
    WHERE NOT EXISTS (
        SELECT 1 FROM subscriptions WHERE organization_id = v_org_id
    );

    -- =========================================================================
    -- 5. Create Runner Registration Token
    -- =========================================================================
    -- Token: selfhost-runner-token (token_hash is its sha256 hex digest)
    INSERT INTO runner_grpc_registration_tokens (
        organization_id, token_hash, description, created_by_id, is_active, max_uses, expires_at
    )
    SELECT v_org_id,
           'af525cf5146bd40521e23661dd84b6e611022ef5ef351d0e828a5278214720c4',
           'Self-Hosted Runner Registration Token',
           v_admin_id,
           TRUE,
           1000000,
           NOW() + INTERVAL '10 years'
    WHERE NOT EXISTS (
        SELECT 1 FROM runner_grpc_registration_tokens
        WHERE organization_id = v_org_id
        AND description = 'Self-Hosted Runner Registration Token'
    )
    RETURNING id INTO v_token_id;

    RAISE NOTICE 'Runner Registration Token ID: %', v_token_id;

    RAISE NOTICE '=========================================================';
    RAISE NOTICE 'Seed data created successfully!';
    RAISE NOTICE '=========================================================';
    RAISE NOTICE 'Admin Account:';
    RAISE NOTICE '  Email: admin@localhost.local';
    RAISE NOTICE '  Password: Admin@123';
    RAISE NOTICE '';
    RAISE NOTICE 'Runner Registration:';
    RAISE NOTICE '  Token: selfhost-runner-token';
    RAISE NOTICE '=========================================================';

END $$;
