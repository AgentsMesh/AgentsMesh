import { getRunnerService } from "@/lib/wasm-core";
import { readCurrentOrg } from "@/stores/auth";
import {
  listRunners as listRunnersConnect,
  listAvailableRunners as listAvailableRunnersConnect,
  getRunner as getRunnerConnect,
  updateRunner as updateRunnerConnect,
  deleteRunner as deleteRunnerConnect,
  createRunnerToken as createRunnerTokenConnect,
  listRunnerLogs as listRunnerLogsConnect,
  querySandboxes as querySandboxesConnect,
  requestLogUpload as requestLogUploadConnect,
  upgradeRunner as upgradeRunnerConnect,
} from "./runnerConnect";

export type { RunnerData, GRPCRegistrationToken, RunnerPodData, SandboxStatus, RelayConnectionInfo, RunnerLogData } from "./runnerTypes";

export interface RunnerAuthStatus {
  status: string;
  runner_id?: number;
  node_id?: string;
  organization_name?: string;
}

function orgSlug(): string {
  return readCurrentOrg()?.slug ?? "";
}

export const runnerApi = {
  list: async (status?: string) => {
    const { items, total, limit, offset, latest_runner_version } = await listRunnersConnect(orgSlug(), { status });
    return { runners: items, total, limit, offset, latest_runner_version };
  },
  get: async (id: number) => {
    return await getRunnerConnect(orgSlug(), id);
  },
  fetchRunners: async (status?: string) => {
    const { items, total, limit, offset, latest_runner_version } = await listRunnersConnect(orgSlug(), { status });
    return { runners: items, total, limit, offset, latest_runner_version };
  },
  fetchRunner: async (id: number) => {
    return await getRunnerConnect(orgSlug(), id);
  },
  fetchAvailableRunners: async () => {
    const { items, total } = await listAvailableRunnersConnect(orgSlug());
    return { runners: items, total };
  },
  update: async (id: number, data: Record<string, unknown>) => {
    return await updateRunnerConnect(orgSlug(), id, data);
  },
  delete: async (id: number) => {
    await deleteRunnerConnect(orgSlug(), id);
  },
  createToken: async (data?: Record<string, unknown>) => {
    return await createRunnerTokenConnect(orgSlug(), data);
  },
  listLogs: async (id: number) => {
    const { items, total, limit, offset } = await listRunnerLogsConnect(orgSlug(), id);
    return { logs: items, total, limit, offset };
  },
  // list_runner_pods isn't owned by proto.runner_api.v1 — it spans pod state
  // (mesh plane). Keep on legacy wasm surface until the mesh side migrates.
  listPods: async (id: number, filters?: { status?: string; limit?: number; offset?: number }) => {
    const json = await getRunnerService().list_runner_pods(
      BigInt(id), filters?.status ?? null, filters?.limit ?? null, filters?.offset ?? null,
    );
    return JSON.parse(json);
  },
  querySandboxes: async (id: number, podKeys: string[]) => {
    return await querySandboxesConnect(orgSlug(), id, podKeys);
  },
  requestLogUpload: async (id: number) => {
    await requestLogUploadConnect(orgSlug(), id);
  },
  upgrade: async (id: number, data?: Record<string, unknown>) => {
    const targetVersion = (data?.target_version as string) ?? "";
    return await upgradeRunnerConnect(orgSlug(), id, targetVersion);
  },
};

// get_auth_status / authorize_runner aren't part of proto.runner_api.v1 —
// they live on a separate auth-key flow. Keep the wasm surface here.
export const runnerAuthApi = {
  getAuthStatus: async (authKey: string): Promise<RunnerAuthStatus> => {
    const json = await getRunnerService().get_auth_status(authKey);
    return JSON.parse(json);
  },
  authorize: async (data: { auth_key: string }) => {
    await getRunnerService().authorize_runner(JSON.stringify(data));
  },
};
