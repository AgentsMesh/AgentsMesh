-- Revert column renames

ALTER TABLE token_usages RENAME COLUMN agent_slug TO agent_type_slug;

ALTER TABLE autopilot_controllers RENAME COLUMN control_agent_slug TO control_agent_type;

ALTER TABLE mcp_market_items RENAME COLUMN agent_filter TO agent_type_filter;
ALTER TABLE skill_market_items RENAME COLUMN agent_filter TO agent_type_filter;
