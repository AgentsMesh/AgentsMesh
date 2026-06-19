// Channel member + pod ops — membership management for users and pods.
// Split out of channelConnect.ts for SRP (file size limit).

import {
  ListChannelMembersRequestSchema,
  ListChannelMembersResponseSchema,
  JoinChannelRequestSchema,
  JoinChannelResponseSchema,
  LeaveChannelRequestSchema,
  LeaveChannelResponseSchema,
  InviteChannelMembersRequestSchema,
  InviteChannelMembersResponseSchema,
  RemoveChannelMemberRequestSchema,
  RemoveChannelMemberResponseSchema,
  ListChannelPodsRequestSchema,
  ListChannelPodsResponseSchema,
  JoinChannelPodRequestSchema,
  JoinChannelPodResponseSchema,
  LeaveChannelPodRequestSchema,
  LeaveChannelPodResponseSchema,
  type ChannelMember as ProtoChannelMember,
  type ChannelPod as ProtoChannelPod,
} from "@proto/channel/v1/channel_pb";
import { create, toBinary, fromBinary } from "@bufbuild/protobuf";
import { getChannelService } from "@/lib/wasm-core";

export interface ChannelMemberData {
  channel_id: number;
  user_id: number;
  role: string;
  is_muted: boolean;
  joined_at: string;
}

function memberFromProto(m: ProtoChannelMember): ChannelMemberData {
  return {
    channel_id: Number(m.channelId),
    user_id: Number(m.userId),
    role: m.role,
    is_muted: m.isMuted,
    joined_at: m.joinedAt,
  };
}

export interface ChannelPodSummary {
  id: number;
  pod_key: string;
  alias?: string;
  status: string;
  agent_status: string;
}

function podFromProto(p: ProtoChannelPod): ChannelPodSummary {
  return {
    id: Number(p.id),
    pod_key: p.podKey,
    alias: p.alias,
    status: p.status,
    agent_status: p.agentStatus,
  };
}

// ---------------- Members ----------------

export async function listChannelMembers(
  orgSlug: string, id: number, opts: { limit?: number; offset?: number } = {},
): Promise<{ members: ChannelMemberData[]; total: number }> {
  const req = create(ListChannelMembersRequestSchema, {
    orgSlug, id: BigInt(id), limit: opts.limit, offset: opts.offset,
  });
  const bytes = toBinary(ListChannelMembersRequestSchema, req);
  const respBytes = await getChannelService().listChannelMembersConnect(bytes);
  const resp = fromBinary(ListChannelMembersResponseSchema, new Uint8Array(respBytes));
  return { members: resp.items.map(memberFromProto), total: Number(resp.total) };
}

// Raw wire bytes for the fetch→state path (Rust apply_fetched_members).
export async function listChannelMembersRaw(orgSlug: string, id: number): Promise<Uint8Array> {
  const req = create(ListChannelMembersRequestSchema, { orgSlug, id: BigInt(id) });
  return new Uint8Array(
    await getChannelService().listChannelMembersConnect(toBinary(ListChannelMembersRequestSchema, req)),
  );
}

export async function joinChannel(orgSlug: string, id: number): Promise<void> {
  const req = create(JoinChannelRequestSchema, { orgSlug, id: BigInt(id) });
  const bytes = toBinary(JoinChannelRequestSchema, req);
  const respBytes = await getChannelService().joinChannelConnect(bytes);
  fromBinary(JoinChannelResponseSchema, new Uint8Array(respBytes));
}

export async function leaveChannel(orgSlug: string, id: number): Promise<void> {
  const req = create(LeaveChannelRequestSchema, { orgSlug, id: BigInt(id) });
  const bytes = toBinary(LeaveChannelRequestSchema, req);
  const respBytes = await getChannelService().leaveChannelConnect(bytes);
  fromBinary(LeaveChannelResponseSchema, new Uint8Array(respBytes));
}

export async function inviteChannelMembers(
  orgSlug: string, id: number, userIds: number[],
): Promise<void> {
  const req = create(InviteChannelMembersRequestSchema, {
    orgSlug, id: BigInt(id), userIds: userIds.map((n) => BigInt(n)),
  });
  const bytes = toBinary(InviteChannelMembersRequestSchema, req);
  const respBytes = await getChannelService().inviteChannelMembersConnect(bytes);
  fromBinary(InviteChannelMembersResponseSchema, new Uint8Array(respBytes));
}

export async function removeChannelMember(
  orgSlug: string, id: number, userId: number,
): Promise<void> {
  const req = create(RemoveChannelMemberRequestSchema, {
    orgSlug, id: BigInt(id), userId: BigInt(userId),
  });
  const bytes = toBinary(RemoveChannelMemberRequestSchema, req);
  const respBytes = await getChannelService().removeChannelMemberConnect(bytes);
  fromBinary(RemoveChannelMemberResponseSchema, new Uint8Array(respBytes));
}

// ---------------- Channel pods ----------------

export async function listChannelPods(
  orgSlug: string, id: number,
): Promise<{ pods: ChannelPodSummary[]; total: number }> {
  const req = create(ListChannelPodsRequestSchema, { orgSlug, id: BigInt(id) });
  const bytes = toBinary(ListChannelPodsRequestSchema, req);
  const respBytes = await getChannelService().listChannelPodsConnect(bytes);
  const resp = fromBinary(ListChannelPodsResponseSchema, new Uint8Array(respBytes));
  return { pods: resp.items.map(podFromProto), total: Number(resp.total) };
}

// Raw wire bytes for the fetch→state path (Rust apply_fetched_pods).
export async function listChannelPodsRaw(orgSlug: string, id: number): Promise<Uint8Array> {
  const req = create(ListChannelPodsRequestSchema, { orgSlug, id: BigInt(id) });
  return new Uint8Array(
    await getChannelService().listChannelPodsConnect(toBinary(ListChannelPodsRequestSchema, req)),
  );
}

export async function joinChannelPod(
  orgSlug: string, id: number, podKey: string,
): Promise<void> {
  const req = create(JoinChannelPodRequestSchema, { orgSlug, id: BigInt(id), podKey });
  const bytes = toBinary(JoinChannelPodRequestSchema, req);
  const respBytes = await getChannelService().joinChannelPodConnect(bytes);
  fromBinary(JoinChannelPodResponseSchema, new Uint8Array(respBytes));
}

export async function leaveChannelPod(
  orgSlug: string, id: number, podKey: string,
): Promise<void> {
  const req = create(LeaveChannelPodRequestSchema, { orgSlug, id: BigInt(id), podKey });
  const bytes = toBinary(LeaveChannelPodRequestSchema, req);
  const respBytes = await getChannelService().leaveChannelPodConnect(bytes);
  fromBinary(LeaveChannelPodResponseSchema, new Uint8Array(respBytes));
}
