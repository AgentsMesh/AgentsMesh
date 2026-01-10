import { useQuery, useMutation } from "@apollo/client/react";

// Apollo Client v4 compatible option types - simplified interface for common use cases
interface QueryOptions {
  fetchPolicy?: "cache-first" | "network-only" | "cache-only" | "no-cache" | "standby" | "cache-and-network";
  pollInterval?: number;
  notifyOnNetworkStatusChange?: boolean;
  context?: Record<string, unknown>;
  skip?: boolean;
  onCompleted?: (data: unknown) => void;
  onError?: (error: Error) => void;
}

interface MutationOptions {
  variables?: Record<string, unknown>;
  refetchQueries?: string[];
  awaitRefetchQueries?: boolean;
  onCompleted?: (data: unknown) => void;
  onError?: (error: Error) => void;
}

// Query hooks
import {
  GET_ME,
  GET_ORGANIZATION,
  GET_ORGANIZATION_MEMBERS,
  GET_RUNNERS,
  GET_RUNNER,
  GET_AVAILABLE_RUNNERS,
  GET_PODS,
  GET_POD,
  GET_CHANNELS,
  GET_CHANNEL,
  GET_CHANNEL_MESSAGES,
  GET_TICKETS,
  GET_TICKET,
  GET_AGENT_TYPES,
  GET_GIT_PROVIDERS,
  GET_REPOSITORIES,
  GET_LABELS,
} from "./queries";

// Mutation imports
import {
  LOGIN,
  REGISTER,
  REFRESH_TOKEN,
  CREATE_ORGANIZATION,
  UPDATE_ORGANIZATION,
  INVITE_MEMBER,
  REMOVE_MEMBER,
  CREATE_REGISTRATION_TOKEN,
  DELETE_RUNNER,
  UPDATE_RUNNER_STATUS,
  CREATE_POD,
  TERMINATE_POD,
  SEND_POD_INPUT,
  RESIZE_POD_TERMINAL,
  CREATE_CHANNEL,
  UPDATE_CHANNEL,
  ARCHIVE_CHANNEL,
  UNARCHIVE_CHANNEL,
  JOIN_CHANNEL,
  LEAVE_CHANNEL,
  SEND_CHANNEL_MESSAGE,
  CREATE_TICKET,
  UPDATE_TICKET,
  DELETE_TICKET,
  UPDATE_TICKET_STATUS,
  ASSIGN_TICKET,
  CREATE_LABEL,
  DELETE_LABEL,
  CREATE_GIT_PROVIDER,
  UPDATE_GIT_PROVIDER,
  DELETE_GIT_PROVIDER,
  CREATE_REPOSITORY,
  UPDATE_REPOSITORY,
  DELETE_REPOSITORY,
  SET_AGENT_CREDENTIALS,
  DELETE_AGENT_CREDENTIALS,
} from "./mutations";

// ============ Query Hooks ============

export const useMe = (options?: QueryOptions) => {
  return useQuery(GET_ME, options);
};

export const useOrganization = (id?: string, slug?: string, options?: QueryOptions) => {
  return useQuery(GET_ORGANIZATION, {
    variables: { id, slug },
    skip: !id && !slug,
    ...(options ?? {}),
  });
};

export const useOrganizationMembers = (organizationId: string, options?: QueryOptions) => {
  return useQuery(GET_ORGANIZATION_MEMBERS, {
    variables: { organizationId },
    skip: !organizationId,
    ...(options ?? {}),
  });
};

export const useRunners = (status?: string, options?: QueryOptions) => {
  return useQuery(GET_RUNNERS, {
    variables: { status },
    ...(options ?? {}),
  });
};

export const useRunner = (id: string, options?: QueryOptions) => {
  return useQuery(GET_RUNNER, {
    variables: { id },
    skip: !id,
    ...(options ?? {}),
  });
};

export const useAvailableRunners = (options?: QueryOptions) => {
  return useQuery(GET_AVAILABLE_RUNNERS, options);
};

interface PodFilter {
  status?: string;
  runnerId?: string;
  limit?: number;
  offset?: number;
}

export const usePods = (filter?: PodFilter, options?: QueryOptions) => {
  return useQuery(GET_PODS, {
    variables: { filter },
    ...(options ?? {}),
  });
};

export const usePod = (podKey: string, options?: QueryOptions) => {
  return useQuery(GET_POD, {
    variables: { podKey },
    skip: !podKey,
    ...(options ?? {}),
  });
};

interface ChannelFilter {
  includeArchived?: boolean;
  limit?: number;
  offset?: number;
}

export const useChannels = (filter?: ChannelFilter, options?: QueryOptions) => {
  return useQuery(GET_CHANNELS, {
    variables: { filter },
    ...(options ?? {}),
  });
};

export const useChannel = (id: string, options?: QueryOptions) => {
  return useQuery(GET_CHANNEL, {
    variables: { id },
    skip: !id,
    ...(options ?? {}),
  });
};

export const useChannelMessages = (
  channelId: string,
  limit = 50,
  offset = 0,
  options?: QueryOptions
) => {
  return useQuery(GET_CHANNEL_MESSAGES, {
    variables: { channelId, limit, offset },
    skip: !channelId,
    ...(options ?? {}),
  });
};

interface TicketFilter {
  status?: string;
  priority?: string;
  type?: string;
  assigneeId?: string;
  repositoryId?: string;
  search?: string;
  limit?: number;
  offset?: number;
}

export const useTickets = (filter?: TicketFilter, options?: QueryOptions) => {
  return useQuery(GET_TICKETS, {
    variables: { filter },
    ...(options ?? {}),
  });
};

export const useTicket = (identifier: string, options?: QueryOptions) => {
  return useQuery(GET_TICKET, {
    variables: { identifier },
    skip: !identifier,
    ...(options ?? {}),
  });
};

export const useAgentTypes = (options?: QueryOptions) => {
  return useQuery(GET_AGENT_TYPES, options);
};

export const useGitProviders = (options?: QueryOptions) => {
  return useQuery(GET_GIT_PROVIDERS, options);
};

export const useRepositories = (gitProviderId?: string, options?: QueryOptions) => {
  return useQuery(GET_REPOSITORIES, {
    variables: { gitProviderId },
    ...(options ?? {}),
  });
};

export const useLabels = (repositoryId?: string, options?: QueryOptions) => {
  return useQuery(GET_LABELS, {
    variables: { repositoryId },
    ...(options ?? {}),
  });
};

// ============ Mutation Hooks ============

export const useLogin = (options?: MutationOptions) => {
  return useMutation(LOGIN, options);
};

export const useRegister = (options?: MutationOptions) => {
  return useMutation(REGISTER, options);
};

export const useRefreshToken = (options?: MutationOptions) => {
  return useMutation(REFRESH_TOKEN, options);
};

export const useCreateOrganization = (options?: MutationOptions) => {
  return useMutation(CREATE_ORGANIZATION, options);
};

export const useUpdateOrganization = (options?: MutationOptions) => {
  return useMutation(UPDATE_ORGANIZATION, options);
};

export const useInviteMember = (options?: MutationOptions) => {
  return useMutation(INVITE_MEMBER, options);
};

export const useRemoveMember = (options?: MutationOptions) => {
  return useMutation(REMOVE_MEMBER, options);
};

export const useCreateRegistrationToken = (options?: MutationOptions) => {
  return useMutation(CREATE_REGISTRATION_TOKEN, options);
};

export const useDeleteRunner = (options?: MutationOptions) => {
  return useMutation(DELETE_RUNNER, options);
};

export const useUpdateRunnerStatus = (options?: MutationOptions) => {
  return useMutation(UPDATE_RUNNER_STATUS, options);
};

export const useCreatePod = (options?: MutationOptions) => {
  return useMutation(CREATE_POD, options);
};

export const useTerminatePod = (options?: MutationOptions) => {
  return useMutation(TERMINATE_POD, options);
};

export const useSendPodInput = (options?: MutationOptions) => {
  return useMutation(SEND_POD_INPUT, options);
};

export const useResizePodTerminal = (options?: MutationOptions) => {
  return useMutation(RESIZE_POD_TERMINAL, options);
};

export const useCreateChannel = (options?: MutationOptions) => {
  return useMutation(CREATE_CHANNEL, options);
};

export const useUpdateChannel = (options?: MutationOptions) => {
  return useMutation(UPDATE_CHANNEL, options);
};

export const useArchiveChannel = (options?: MutationOptions) => {
  return useMutation(ARCHIVE_CHANNEL, options);
};

export const useUnarchiveChannel = (options?: MutationOptions) => {
  return useMutation(UNARCHIVE_CHANNEL, options);
};

export const useJoinChannel = (options?: MutationOptions) => {
  return useMutation(JOIN_CHANNEL, options);
};

export const useLeaveChannel = (options?: MutationOptions) => {
  return useMutation(LEAVE_CHANNEL, options);
};

export const useSendChannelMessage = (options?: MutationOptions) => {
  return useMutation(SEND_CHANNEL_MESSAGE, options);
};

export const useCreateTicket = (options?: MutationOptions) => {
  return useMutation(CREATE_TICKET, options);
};

export const useUpdateTicket = (options?: MutationOptions) => {
  return useMutation(UPDATE_TICKET, options);
};

export const useDeleteTicket = (options?: MutationOptions) => {
  return useMutation(DELETE_TICKET, options);
};

export const useUpdateTicketStatus = (options?: MutationOptions) => {
  return useMutation(UPDATE_TICKET_STATUS, options);
};

export const useAssignTicket = (options?: MutationOptions) => {
  return useMutation(ASSIGN_TICKET, options);
};

export const useCreateLabel = (options?: MutationOptions) => {
  return useMutation(CREATE_LABEL, options);
};

export const useDeleteLabel = (options?: MutationOptions) => {
  return useMutation(DELETE_LABEL, options);
};

export const useCreateGitProvider = (options?: MutationOptions) => {
  return useMutation(CREATE_GIT_PROVIDER, options);
};

export const useUpdateGitProvider = (options?: MutationOptions) => {
  return useMutation(UPDATE_GIT_PROVIDER, options);
};

export const useDeleteGitProvider = (options?: MutationOptions) => {
  return useMutation(DELETE_GIT_PROVIDER, options);
};

export const useCreateRepository = (options?: MutationOptions) => {
  return useMutation(CREATE_REPOSITORY, options);
};

export const useUpdateRepository = (options?: MutationOptions) => {
  return useMutation(UPDATE_REPOSITORY, options);
};

export const useDeleteRepository = (options?: MutationOptions) => {
  return useMutation(DELETE_REPOSITORY, options);
};

export const useSetAgentCredentials = (options?: MutationOptions) => {
  return useMutation(SET_AGENT_CREDENTIALS, options);
};

export const useDeleteAgentCredentials = (options?: MutationOptions) => {
  return useMutation(DELETE_AGENT_CREDENTIALS, options);
};
