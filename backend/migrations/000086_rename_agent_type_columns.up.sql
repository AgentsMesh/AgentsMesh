-- Rename legacy agent_type_* columns to match new naming convention

ALTER TABLE token_usages RENAME COLUMN agent_type_slug TO agent_slug;

ALTER TABLE autopilot_controllers RENAME COLUMN control_agent_type TO control_agent_slug;

-- Extension marketplace tables
ALTER TABLE mcp_market_items RENAME COLUMN agent_type_filter TO agent_filter;
ALTER TABLE skill_market_items RENAME COLUMN agent_type_filter TO agent_filter;
