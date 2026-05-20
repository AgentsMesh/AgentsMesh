-- Phase 4 finale (api_keys): symmetric to channels — see 000143 for context.
ALTER TABLE api_keys VALIDATE CONSTRAINT api_keys_slug_format;
ALTER TABLE api_keys ALTER COLUMN slug SET NOT NULL;
ALTER TABLE api_keys ADD CONSTRAINT api_keys_org_slug_unique UNIQUE (organization_id, slug);
