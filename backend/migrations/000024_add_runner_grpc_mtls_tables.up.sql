-- Migration: Add gRPC + mTLS related tables for Runner authentication
-- This migration adds tables for:
-- 1. runner_certificates - Track issued Runner certificates
-- 2. runner_pending_auths - Pending interactive (Tailscale-style) registrations
-- 3. runner_registration_tokens - Pre-generated registration tokens

-- ==================== Runner Certificates ====================
-- Tracks all certificates issued to Runners for mTLS authentication

CREATE TABLE runner_certificates (
    id BIGSERIAL PRIMARY KEY,
    runner_id BIGINT REFERENCES runners(id) ON DELETE CASCADE,
    serial_number VARCHAR(64) UNIQUE NOT NULL,
    fingerprint VARCHAR(128) NOT NULL,
    issued_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    revoked_at TIMESTAMP,
    revocation_reason VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for looking up certificates by runner
CREATE INDEX idx_runner_certs_runner_id ON runner_certificates(runner_id);

-- Index for certificate revocation checks (by serial number)
CREATE INDEX idx_runner_certs_serial ON runner_certificates(serial_number);

-- Index for finding expired certificates
CREATE INDEX idx_runner_certs_expires ON runner_certificates(expires_at);

-- Index for finding revoked certificates
CREATE INDEX idx_runner_certs_revoked ON runner_certificates(revoked_at) WHERE revoked_at IS NOT NULL;

-- Add certificate fields to runners table
ALTER TABLE runners
    ADD COLUMN cert_serial_number VARCHAR(64),
    ADD COLUMN cert_expires_at TIMESTAMP;

-- ==================== Pending Auth (Tailscale-style Interactive Registration) ====================
-- Stores pending authorization requests for interactive Runner registration
-- Flow: Runner generates machine_key -> Backend returns auth URL -> User authorizes in browser

CREATE TABLE runner_pending_auths (
    id BIGSERIAL PRIMARY KEY,
    auth_key VARCHAR(64) UNIQUE NOT NULL,        -- Unique key for this auth request
    machine_key VARCHAR(128) NOT NULL,           -- Runner-generated machine identifier
    node_id VARCHAR(255),                        -- Optional user-specified node ID
    labels JSONB,                                -- Optional labels
    authorized BOOLEAN DEFAULT FALSE,            -- Whether user has authorized
    organization_id BIGINT REFERENCES organizations(id),  -- Which org the Runner is authorized for
    runner_id BIGINT REFERENCES runners(id),     -- Created runner ID (after authorization)
    expires_at TIMESTAMP NOT NULL,               -- Auth request expiration (e.g., 15 minutes)
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for looking up by auth_key (primary lookup method)
CREATE INDEX idx_runner_pending_auths_key ON runner_pending_auths(auth_key);

-- Index for cleaning up expired requests
CREATE INDEX idx_runner_pending_auths_expires ON runner_pending_auths(expires_at);

-- ==================== Registration Tokens (Pre-generated Token Registration) ====================
-- Stores pre-generated registration tokens for automated/scripted Runner registration
-- Created by org admins in Web UI, used by Runner CLI with --token flag

CREATE TABLE runner_registration_tokens (
    id BIGSERIAL PRIMARY KEY,
    token_hash VARCHAR(128) UNIQUE NOT NULL,     -- SHA-256 hash of token (never store plaintext)
    organization_id BIGINT REFERENCES organizations(id) NOT NULL,
    name VARCHAR(255),                           -- Optional descriptive name
    labels JSONB,                                -- Labels to apply to registered Runners
    single_use BOOLEAN DEFAULT TRUE,             -- Whether token is single-use
    max_uses INT DEFAULT 1,                      -- Maximum number of times token can be used
    used_count INT DEFAULT 0,                    -- How many times token has been used
    expires_at TIMESTAMP NOT NULL,               -- Token expiration
    created_by BIGINT REFERENCES users(id),      -- Who created this token
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for looking up by organization
CREATE INDEX idx_runner_reg_tokens_org ON runner_registration_tokens(organization_id);

-- Index for finding expired tokens
CREATE INDEX idx_runner_reg_tokens_expires ON runner_registration_tokens(expires_at);

-- ==================== Reactivation Tokens (For Expired Certificate Recovery) ====================
-- Stores one-time tokens for reactivating Runners with expired certificates
-- Generated via Web UI, valid for 10 minutes

CREATE TABLE runner_reactivation_tokens (
    id BIGSERIAL PRIMARY KEY,
    token_hash VARCHAR(128) UNIQUE NOT NULL,     -- SHA-256 hash of token
    runner_id BIGINT REFERENCES runners(id) ON DELETE CASCADE NOT NULL,
    expires_at TIMESTAMP NOT NULL,               -- Token expiration (e.g., 10 minutes)
    used_at TIMESTAMP,                           -- When token was used (NULL if unused)
    created_by BIGINT REFERENCES users(id),      -- Who created this token
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Index for looking up by runner
CREATE INDEX idx_runner_reactivation_runner ON runner_reactivation_tokens(runner_id);

-- Index for finding expired tokens
CREATE INDEX idx_runner_reactivation_expires ON runner_reactivation_tokens(expires_at);
