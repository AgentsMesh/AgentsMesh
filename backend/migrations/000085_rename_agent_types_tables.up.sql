-- Rename agent_types → agents, custom_agent_types → custom_agents
-- "AgentType" concept is redundant; slug IS the agent identity.

ALTER TABLE agent_types RENAME TO agents;
ALTER TABLE custom_agent_types RENAME TO custom_agents;
