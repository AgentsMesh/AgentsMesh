CREATE TYPE sso_protocol AS ENUM ('oidc', 'saml', 'ldap');

CREATE TABLE sso_configs (
    id              BIGSERIAL PRIMARY KEY,
    domain          VARCHAR(255) NOT NULL,
    name            VARCHAR(100) NOT NULL,
    protocol        sso_protocol NOT NULL,
    is_enabled      BOOLEAN NOT NULL DEFAULT false,
    enforce_sso     BOOLEAN NOT NULL DEFAULT false,

    -- OIDC fields
    oidc_issuer_url              TEXT,
    oidc_client_id               VARCHAR(255),
    oidc_client_secret_encrypted TEXT,
    oidc_scopes                  TEXT,

    -- SAML fields
    saml_idp_metadata_url    TEXT,
    saml_idp_metadata_xml    TEXT,
    saml_idp_sso_url         TEXT,
    saml_idp_cert_encrypted  TEXT,
    saml_sp_entity_id        TEXT,
    saml_name_id_format      VARCHAR(100) DEFAULT 'urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress',

    -- LDAP fields
    ldap_host                    VARCHAR(255),
    ldap_port                    INT DEFAULT 389,
    ldap_use_tls                 BOOLEAN DEFAULT false,
    ldap_bind_dn                 TEXT,
    ldap_bind_password_encrypted TEXT,
    ldap_base_dn                 TEXT,
    ldap_user_filter             TEXT DEFAULT '(uid={{username}})',
    ldap_email_attr              VARCHAR(100) DEFAULT 'mail',
    ldap_name_attr               VARCHAR(100) DEFAULT 'cn',
    ldap_username_attr           VARCHAR(100) DEFAULT 'uid',

    -- Audit
    created_by      BIGINT REFERENCES users(id) ON DELETE SET NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_sso_configs_domain_protocol ON sso_configs(domain, protocol);
CREATE INDEX idx_sso_configs_domain_enabled ON sso_configs(domain, is_enabled);
