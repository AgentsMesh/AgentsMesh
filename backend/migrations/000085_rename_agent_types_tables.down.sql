-- Revert: agents → agent_types, custom_agents → custom_agent_types

ALTER TABLE agents RENAME TO agent_types;
ALTER TABLE custom_agents RENAME TO custom_agent_types;
