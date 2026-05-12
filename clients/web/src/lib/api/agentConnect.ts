// Connect-RPC adapter for proto.agent.v1.AgentService + UserAgentConfigService.
//
// Encodes requests via @bufbuild/protobuf .toBinary(), passes the Uint8Array
// to the wasm bridge (binary in / binary out per conventions §2.5), decodes
// responses via .fromBinary(). No JSON intermediate.
//
// Returns the existing snake_case web shapes (AgentData, ConfigSchema,
// UserAgentConfigData) so call sites don't have to flip off camelCase +
// BigInt — same dual-track pattern as podConnect.ts and repositoryConnect.ts.
//
// Data-mapping helpers live inline (under 200 LoC budget per CLAUDE.md).

import {
  AgentSchema,
  AgentListResponseSchema,
  ConfigSchemaSchema,
  CreateCustomAgentRequestSchema,
  DeleteCustomAgentRequestSchema,
  DeleteCustomAgentResponseSchema,
  DeleteUserAgentConfigRequestSchema,
  DeleteUserAgentConfigResponseSchema,
  GetAgentRequestSchema,
  GetAgentConfigSchemaRequestSchema,
  GetUserAgentConfigRequestSchema,
  ListAgentsRequestSchema,
  SetUserAgentConfigRequestSchema,
  UpdateCustomAgentRequestSchema,
  UserAgentConfigSchema,
  UserAgentConfigListResponseSchema,
  type Agent as ProtoAgent,
  type ConfigSchema as ProtoConfigSchema,
  type UserAgentConfig as ProtoUserAgentConfig,
} from "@proto/agent/v1/agent_pb";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { getAgentService } from "@/lib/wasm-core";
import type {
  AgentData,
  ConfigField,
  ConfigSchema,
  CredentialField,
  UserAgentConfigData,
} from "@/lib/api/agentTypes";

// ============== Proto → web shape mappers ==============

export function fromProtoAgent(p: ProtoAgent): AgentData {
  return {
    slug: p.slug,
    name: p.name,
    description: p.description,
    launch_command: p.launchCommand,
    is_builtin: p.isBuiltin,
    is_active: p.isActive,
    supported_modes: p.supportedModes,
  };
}

function fromProtoConfigSchema(p: ProtoConfigSchema): ConfigSchema {
  const fields: ConfigField[] = p.fields.map((f) => {
    const field: ConfigField = {
      name: f.name,
      // Cast is safe: the .proto string maps to ConfigField['type'] union.
      type: f.type as ConfigField["type"],
    };
    if (f.defaultJson) {
      try {
        field.default = JSON.parse(f.defaultJson);
      } catch {
        // Malformed JSON — drop the default, keep the field.
      }
    }
    if (f.options.length > 0) {
      field.options = f.options.map((o) => ({ value: o.value }));
    }
    return field;
  });
  const credential_fields: CredentialField[] = p.credentialFields.map((c) => ({
    name: c.name,
    type: c.type as CredentialField["type"],
    optional: c.optional,
  }));
  return { fields, credential_fields };
}

function fromProtoUserAgentConfig(p: ProtoUserAgentConfig): UserAgentConfigData {
  let config_values: Record<string, unknown> = {};
  if (p.configValuesJson) {
    try {
      config_values = JSON.parse(p.configValuesJson);
    } catch {
      // Malformed JSON — return empty object, matches legacy "no saved config".
    }
  }
  return {
    id: Number(p.id),
    user_id: Number(p.userId),
    agent_slug: p.agentSlug,
    agent_name: p.agentName,
    config_values,
    created_at: p.createdAt,
    updated_at: p.updatedAt,
  };
}

// ============== AgentService — org-scoped ==============

// AgentListResponse is the §9 multi-field exception — preserves the
// {builtin_agents, custom_agents, agents} split because the UI renders them
// with different visibility semantics (builtin = platform, custom = org).
export async function listAgents(orgSlug: string): Promise<{
  builtin_agents: AgentData[];
  custom_agents: AgentData[];
  agents: AgentData[];
}> {
  const req = create(ListAgentsRequestSchema, { orgSlug });
  const bytes = toBinary(ListAgentsRequestSchema, req);
  const respBytes = await getAgentService().list_agents_connect(bytes);
  const resp = fromBinary(AgentListResponseSchema, new Uint8Array(respBytes));
  return {
    builtin_agents: resp.builtinAgents.map(fromProtoAgent),
    custom_agents: resp.customAgents.map(fromProtoAgent),
    agents: resp.agents.map(fromProtoAgent),
  };
}

export async function getAgent(orgSlug: string, agentSlug: string): Promise<AgentData> {
  const req = create(GetAgentRequestSchema, { orgSlug, agentSlug });
  const bytes = toBinary(GetAgentRequestSchema, req);
  const respBytes = await getAgentService().get_agent_connect(bytes);
  return fromProtoAgent(fromBinary(AgentSchema, new Uint8Array(respBytes)));
}

export async function getAgentConfigSchema(
  orgSlug: string,
  agentSlug: string,
): Promise<ConfigSchema> {
  const req = create(GetAgentConfigSchemaRequestSchema, { orgSlug, agentSlug });
  const bytes = toBinary(GetAgentConfigSchemaRequestSchema, req);
  const respBytes = await getAgentService().get_agent_config_schema_connect(bytes);
  return fromProtoConfigSchema(fromBinary(ConfigSchemaSchema, new Uint8Array(respBytes)));
}

export interface CreateCustomAgentInput {
  slug: string;
  name: string;
  description?: string;
  agentfile_source?: string;
  launch_command?: string;
  default_args?: string;
}

export async function createCustomAgent(
  orgSlug: string,
  input: CreateCustomAgentInput,
): Promise<AgentData> {
  const req = create(CreateCustomAgentRequestSchema, {
    orgSlug,
    slug: input.slug,
    name: input.name,
    description: input.description,
    agentfileSource: input.agentfile_source,
    launchCommand: input.launch_command,
    defaultArgs: input.default_args,
  });
  const bytes = toBinary(CreateCustomAgentRequestSchema, req);
  const respBytes = await getAgentService().create_custom_agent_connect(bytes);
  return fromProtoAgent(fromBinary(AgentSchema, new Uint8Array(respBytes)));
}

export async function updateCustomAgent(
  orgSlug: string,
  agentSlug: string,
  updates: Record<string, unknown>,
): Promise<AgentData> {
  const req = create(UpdateCustomAgentRequestSchema, {
    orgSlug,
    agentSlug,
    updatesJson: JSON.stringify(updates),
  });
  const bytes = toBinary(UpdateCustomAgentRequestSchema, req);
  const respBytes = await getAgentService().update_custom_agent_connect(bytes);
  return fromProtoAgent(fromBinary(AgentSchema, new Uint8Array(respBytes)));
}

export async function deleteCustomAgent(orgSlug: string, agentSlug: string): Promise<void> {
  const req = create(DeleteCustomAgentRequestSchema, { orgSlug, agentSlug });
  const bytes = toBinary(DeleteCustomAgentRequestSchema, req);
  const respBytes = await getAgentService().delete_custom_agent_connect(bytes);
  fromBinary(DeleteCustomAgentResponseSchema, new Uint8Array(respBytes));
}

// ============== UserAgentConfigService — user-scoped ==============

// UserAgentConfigListResponse is the §9 sub-resource exception — preserves
// the `configs` field name (not `items`) because the REST shape carries it.
export async function listUserAgentConfigs(): Promise<UserAgentConfigData[]> {
  const respBytes = await getAgentService().list_user_agent_configs_connect();
  const resp = fromBinary(UserAgentConfigListResponseSchema, new Uint8Array(respBytes));
  return resp.configs.map(fromProtoUserAgentConfig);
}

export async function getUserAgentConfig(agentSlug: string): Promise<UserAgentConfigData> {
  const req = create(GetUserAgentConfigRequestSchema, { agentSlug });
  const bytes = toBinary(GetUserAgentConfigRequestSchema, req);
  const respBytes = await getAgentService().get_user_agent_config_connect(bytes);
  return fromProtoUserAgentConfig(fromBinary(UserAgentConfigSchema, new Uint8Array(respBytes)));
}

export async function setUserAgentConfig(
  agentSlug: string,
  configValues: Record<string, unknown>,
): Promise<UserAgentConfigData> {
  const req = create(SetUserAgentConfigRequestSchema, {
    agentSlug,
    configValuesJson: JSON.stringify(configValues),
  });
  const bytes = toBinary(SetUserAgentConfigRequestSchema, req);
  const respBytes = await getAgentService().set_user_agent_config_connect(bytes);
  return fromProtoUserAgentConfig(fromBinary(UserAgentConfigSchema, new Uint8Array(respBytes)));
}

export async function deleteUserAgentConfig(agentSlug: string): Promise<void> {
  const req = create(DeleteUserAgentConfigRequestSchema, { agentSlug });
  const bytes = toBinary(DeleteUserAgentConfigRequestSchema, req);
  const respBytes = await getAgentService().delete_user_agent_config_connect(bytes);
  fromBinary(DeleteUserAgentConfigResponseSchema, new Uint8Array(respBytes));
}
