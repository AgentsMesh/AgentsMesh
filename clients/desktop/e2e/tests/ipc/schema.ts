// AUTO-GENERATED — regenerate: pnpm --filter desktop e2e:gen
//
// Source of truth: clients/core/crates/node-bridge/index.d.ts (the
// napi-rs-emitted TypeScript declaration of AppState). Desktop main
// reflects over the prototype to register one ipcMain handler per method,
// so this mirror is what the renderer can actually invoke at runtime.
export interface IpcMethodSchema {
  name: string;
  group: string;
  params: Array<{ name: string; type: string }>;
  returnType: string;
}

export const ipcSchema: IpcMethodSchema[] = [
  {
    "name": "apikeyCreateConnect",
    "group": "apikey",
    "params": [],
    "returnType": "any",
    "ipcExposable": true
  },
  {
    "name": "apikeyDeleteConnect",
    "group": "apikey",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "apikeyGetConnect",
    "group": "apikey",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "apikeyListConnect",
    "group": "apikey",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "apikeyRevokeConnect",
    "group": "apikey",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "apikeyUpdateConnect",
    "group": "apikey",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "authApplySessionProto",
    "group": "auth",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "authBootstrap",
    "group": "auth",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "authBootstrapProto",
    "group": "auth",
    "params": [],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "authClearSession",
    "group": "auth",
    "params": [],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "authFetchOrganizations",
    "group": "auth",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "authFetchOrganizationsProto",
    "group": "auth",
    "params": [],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "authGetCurrentOrgJson",
    "group": "auth",
    "params": [],
    "returnType": "string | undefined | null",
    "ipcExposable": true
  },
  {
    "name": "authGetCurrentUserJson",
    "group": "auth",
    "params": [],
    "returnType": "string | undefined | null",
    "ipcExposable": true
  },
  {
    "name": "authGetCurrentUserProto",
    "group": "auth",
    "params": [],
    "returnType": "Array<number> | undefined | null",
    "ipcExposable": true
  },
  {
    "name": "authGetExpiresAt",
    "group": "auth",
    "params": [],
    "returnType": "number | undefined | null",
    "ipcExposable": true
  },
  {
    "name": "authGetOrganizationsJson",
    "group": "auth",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "authGetToken",
    "group": "auth",
    "params": [],
    "returnType": "string | undefined | null",
    "ipcExposable": true
  },
  {
    "name": "authIsAuthenticated",
    "group": "auth",
    "params": [],
    "returnType": "boolean",
    "ipcExposable": true
  },
  {
    "name": "authLogin",
    "group": "auth",
    "params": [
      {
        "name": "email",
        "type": "string"
      },
      {
        "name": "password",
        "type": "string"
      }
    ],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "authLoginProto",
    "group": "auth",
    "params": [
      {
        "name": "email",
        "type": "string"
      },
      {
        "name": "password",
        "type": "string"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "authLogout",
    "group": "auth",
    "params": [],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "authRefreshToken",
    "group": "auth",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "authRefreshTokenProto",
    "group": "auth",
    "params": [],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "authSetCurrentOrgProto",
    "group": "auth",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "authSetOrganizationsProto",
    "group": "auth",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "authSwitchOrg",
    "group": "auth",
    "params": [
      {
        "name": "slug",
        "type": "string"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "authConnectForgotPasswordConnect",
    "group": "auth_connect",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "authConnectLoginConnect",
    "group": "auth_connect",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "authConnectLogoutConnect",
    "group": "auth_connect",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "authConnectOauthCallbackConnect",
    "group": "auth_connect",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "authConnectOauthRedirectConnect",
    "group": "auth_connect",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "authConnectRefreshTokenConnect",
    "group": "auth_connect",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "authConnectRegisterConnect",
    "group": "auth_connect",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "authConnectResendVerificationConnect",
    "group": "auth_connect",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "authConnectResetPasswordConnect",
    "group": "auth_connect",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "authConnectVerifyEmailConnect",
    "group": "auth_connect",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "bindingAcceptBindingConnect",
    "group": "binding",
    "params": [
      {
        "name": "request",
        "type": "Uint8Array"
      }
    ],
    "returnType": "Uint8Array",
    "ipcExposable": true
  },
  {
    "name": "bindingApproveScopesConnect",
    "group": "binding",
    "params": [
      {
        "name": "request",
        "type": "Uint8Array"
      }
    ],
    "returnType": "Uint8Array",
    "ipcExposable": true
  },
  {
    "name": "bindingCheckBindingConnect",
    "group": "binding",
    "params": [
      {
        "name": "request",
        "type": "Uint8Array"
      }
    ],
    "returnType": "Uint8Array",
    "ipcExposable": true
  },
  {
    "name": "bindingGetBoundPodsConnect",
    "group": "binding",
    "params": [
      {
        "name": "request",
        "type": "Uint8Array"
      }
    ],
    "returnType": "Uint8Array",
    "ipcExposable": true
  },
  {
    "name": "bindingGetPendingBindingsConnect",
    "group": "binding",
    "params": [
      {
        "name": "request",
        "type": "Uint8Array"
      }
    ],
    "returnType": "Uint8Array",
    "ipcExposable": true
  },
  {
    "name": "bindingListBindingsConnect",
    "group": "binding",
    "params": [
      {
        "name": "request",
        "type": "Uint8Array"
      }
    ],
    "returnType": "Uint8Array",
    "ipcExposable": true
  },
  {
    "name": "bindingRejectBindingConnect",
    "group": "binding",
    "params": [
      {
        "name": "request",
        "type": "Uint8Array"
      }
    ],
    "returnType": "Uint8Array",
    "ipcExposable": true
  },
  {
    "name": "bindingRequestBindingConnect",
    "group": "binding",
    "params": [
      {
        "name": "request",
        "type": "Uint8Array"
      }
    ],
    "returnType": "Uint8Array",
    "ipcExposable": true
  },
  {
    "name": "bindingRequestScopesConnect",
    "group": "binding",
    "params": [
      {
        "name": "request",
        "type": "Uint8Array"
      }
    ],
    "returnType": "Uint8Array",
    "ipcExposable": true
  },
  {
    "name": "bindingUnbindConnect",
    "group": "binding",
    "params": [
      {
        "name": "request",
        "type": "Uint8Array"
      }
    ],
    "returnType": "Uint8Array",
    "ipcExposable": true
  },
  {
    "name": "blockstoreApplyRemoteOp",
    "group": "blockstore",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "blockstoreBacklinksJson",
    "group": "blockstore",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "blockstoreBlocksJson",
    "group": "blockstore",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "blockstoreCatchup",
    "group": "blockstore",
    "params": [
      {
        "name": "workspaceId",
        "type": "string"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "blockstoreEnsureDefaultWorkspace",
    "group": "blockstore",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "blockstoreGetBlockJson",
    "group": "blockstore",
    "params": [
      {
        "name": "id",
        "type": "string"
      }
    ],
    "returnType": "string | undefined | null",
    "ipcExposable": true
  },
  {
    "name": "blockstoreLastOpId",
    "group": "blockstore",
    "params": [
      {
        "name": "workspaceId",
        "type": "string"
      }
    ],
    "returnType": "number",
    "ipcExposable": true
  },
  {
    "name": "blockstoreLastOpIdsJson",
    "group": "blockstore",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "blockstoreListBacklinksJson",
    "group": "blockstore",
    "params": [
      {
        "name": "targetId",
        "type": "string"
      }
    ],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "blockstoreListChildrenJson",
    "group": "blockstore",
    "params": [
      {
        "name": "parentId",
        "type": "string"
      }
    ],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "blockstoreListWorkspaces",
    "group": "blockstore",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "blockstoreLoadSubtree",
    "group": "blockstore",
    "params": [
      {
        "name": "workspaceId",
        "type": "string"
      },
      {
        "name": "rootId",
        "type": "string"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "blockstoreLoadTypeDefs",
    "group": "blockstore",
    "params": [
      {
        "name": "workspaceId",
        "type": "string"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "blockstoreNestChildrenJson",
    "group": "blockstore",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "blockstoreProjectLocalOps",
    "group": "blockstore",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "blockstoreRefsJson",
    "group": "blockstore",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "blockstoreReplaceWorkspaces",
    "group": "blockstore",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "blockstoreSetLastOpId",
    "group": "blockstore",
    "params": [
      {
        "name": "workspaceId",
        "type": "string"
      },
      {
        "name": "id",
        "type": "number"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "blockstoreTypeDefsJson",
    "group": "blockstore",
    "params": [
      {
        "name": "workspaceId",
        "type": "string"
      }
    ],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "blockstoreUpsertBlocks",
    "group": "blockstore",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "blockstoreUpsertRefs",
    "group": "blockstore",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "blockstoreUpsertWorkspace",
    "group": "blockstore",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "blockstoreWorkspacesJson",
    "group": "blockstore",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "channelArchiveChannelConnect",
    "group": "channel",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "channelCreateChannelConnect",
    "group": "channel",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "channelDeleteChannelMessageConnect",
    "group": "channel",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "channelEditChannelMessageConnect",
    "group": "channel",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "channelGetChannelConnect",
    "group": "channel",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "channelGetChannelUnreadCountsConnect",
    "group": "channel",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "channelListChannelMembersConnect",
    "group": "channel",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "channelListChannelMessagesConnect",
    "group": "channel",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "channelListChannelsConnect",
    "group": "channel",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "channelMarkChannelReadConnect",
    "group": "channel",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "channelSendChannelMessageConnect",
    "group": "channel",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "channelUnarchiveChannelConnect",
    "group": "channel",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "channelUpdateChannelConnect",
    "group": "channel",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "envBundleCreateEnvBundleConnect",
    "group": "env_bundle",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "envBundleDeleteEnvBundleConnect",
    "group": "env_bundle",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "envBundleGetEnvBundleConnect",
    "group": "env_bundle",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "envBundleListEnvBundlesConnect",
    "group": "env_bundle",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "envBundleSetPrimaryEnvBundleConnect",
    "group": "env_bundle",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "envBundleUpdateEnvBundleConnect",
    "group": "env_bundle",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "eventsConnect",
    "group": "events",
    "params": [],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "eventsDisconnect",
    "group": "events",
    "params": [],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "eventsGetConnectionState",
    "group": "events",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "eventsGetTick",
    "group": "events",
    "params": [],
    "returnType": "number",
    "ipcExposable": true
  },
  {
    "name": "eventsNudge",
    "group": "events",
    "params": [],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "eventsOnConnectionStateChange",
    "group": "events",
    "params": [
      {
        "name": "callback",
        "type": "(err: unknown, arg: string) => void"
      }
    ],
    "returnType": "number",
    "ipcExposable": false
  },
  {
    "name": "eventsSubscribeAll",
    "group": "events",
    "params": [
      {
        "name": "callback",
        "type": "(err: unknown, arg: string) => void"
      }
    ],
    "returnType": "number",
    "ipcExposable": false
  },
  {
    "name": "eventsUnsubscribe",
    "group": "events",
    "params": [
      {
        "name": "id",
        "type": "number"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "extensionCreateSkillRegistryConnect",
    "group": "extension",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "extensionDeleteSkillRegistryConnect",
    "group": "extension",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "extensionInstallCustomMcpServerConnect",
    "group": "extension",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "extensionInstallMcpFromMarketConnect",
    "group": "extension",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "extensionInstallSkillFromGithubConnect",
    "group": "extension",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "extensionInstallSkillFromMarketConnect",
    "group": "extension",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "extensionInstallSkillFromUploadedFileConnect",
    "group": "extension",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "extensionListMarketMcpServersConnect",
    "group": "extension",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "extensionListMarketSkillsConnect",
    "group": "extension",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "extensionListRepoMcpServersConnect",
    "group": "extension",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "extensionListRepoSkillsConnect",
    "group": "extension",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "extensionListSkillRegistriesConnect",
    "group": "extension",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "extensionListSkillRegistryOverridesConnect",
    "group": "extension",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "extensionPresignSkillUploadConnect",
    "group": "extension",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "extensionSyncSkillRegistryConnect",
    "group": "extension",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "extensionTogglePlatformRegistryConnect",
    "group": "extension",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "extensionUninstallMcpServerConnect",
    "group": "extension",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "extensionUninstallSkillConnect",
    "group": "extension",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "extensionUpdateMcpServerConnect",
    "group": "extension",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "extensionUpdateSkillConnect",
    "group": "extension",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "fileUploadFile",
    "group": "file",
    "params": [
      {
        "name": "fileData",
        "type": "Array<number>"
      },
      {
        "name": "filename",
        "type": "string"
      },
      {
        "name": "contentType",
        "type": "string"
      }
    ],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "invitationAcceptInvitationConnect",
    "group": "invitation",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "invitationCreateInvitationConnect",
    "group": "invitation",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "invitationGetInvitationByTokenConnect",
    "group": "invitation",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "invitationListInvitationsConnect",
    "group": "invitation",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "invitationListPendingInvitationsConnect",
    "group": "invitation",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "invitationResendInvitationConnect",
    "group": "invitation",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "invitationRevokeInvitationConnect",
    "group": "invitation",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "localRunnerBinaryPath",
    "group": "local_runner",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "localRunnerFallbackVersion",
    "group": "local_runner",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "localRunnerHostTarget",
    "group": "local_runner",
    "params": [],
    "returnType": "string | undefined | null",
    "ipcExposable": true
  },
  {
    "name": "localRunnerInstallBinary",
    "group": "local_runner",
    "params": [
      {
        "name": "releaseUrl",
        "type": "string"
      },
      {
        "name": "expectedSha256",
        "type": "string | undefined | null"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "localRunnerInstalledVersion",
    "group": "local_runner",
    "params": [],
    "returnType": "string | undefined | null",
    "ipcExposable": true
  },
  {
    "name": "localRunnerIsInstalled",
    "group": "local_runner",
    "params": [],
    "returnType": "boolean",
    "ipcExposable": true
  },
  {
    "name": "localRunnerIsRegistered",
    "group": "local_runner",
    "params": [],
    "returnType": "boolean",
    "ipcExposable": true
  },
  {
    "name": "localRunnerLocalNodeId",
    "group": "local_runner",
    "params": [],
    "returnType": "string | undefined | null",
    "ipcExposable": true
  },
  {
    "name": "localRunnerRegister",
    "group": "local_runner",
    "params": [
      {
        "name": "token",
        "type": "string"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "localRunnerServiceInstall",
    "group": "local_runner",
    "params": [],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "localRunnerServiceStart",
    "group": "local_runner",
    "params": [],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "localRunnerServiceStatus",
    "group": "local_runner",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "localRunnerServiceStop",
    "group": "local_runner",
    "params": [],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "localRunnerServiceUninstall",
    "group": "local_runner",
    "params": [],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "meshBatchGetTicketPodsConnect",
    "group": "mesh",
    "params": [
      {
        "name": "request",
        "type": "Uint8Array"
      }
    ],
    "returnType": "Uint8Array",
    "ipcExposable": true
  },
  {
    "name": "meshCreatePodForTicketConnect",
    "group": "mesh",
    "params": [
      {
        "name": "request",
        "type": "Uint8Array"
      }
    ],
    "returnType": "Uint8Array",
    "ipcExposable": true
  },
  {
    "name": "meshFetchTopology",
    "group": "mesh",
    "params": [],
    "returnType": "Uint8Array",
    "ipcExposable": true
  },
  {
    "name": "meshGetMeshTopologyConnect",
    "group": "mesh",
    "params": [
      {
        "name": "request",
        "type": "Uint8Array"
      }
    ],
    "returnType": "Uint8Array",
    "ipcExposable": true
  },
  {
    "name": "meshGetTicketPodsConnect",
    "group": "mesh",
    "params": [
      {
        "name": "request",
        "type": "Uint8Array"
      }
    ],
    "returnType": "Uint8Array",
    "ipcExposable": true
  },
  {
    "name": "promocodeGetRedemptionHistoryConnect",
    "group": "promocode",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "promocodeRedeemPromoCodeConnect",
    "group": "promocode",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "promocodeValidatePromoCodeConnect",
    "group": "promocode",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "runnerAuthorizeRunner",
    "group": "runner",
    "params": [
      {
        "name": "requestBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "runnerGetAuthStatus",
    "group": "runner",
    "params": [
      {
        "name": "requestBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ssoDiscoverConnect",
    "group": "sso",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ssoLdapAuthConnect",
    "group": "sso",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "supportTicketAddSupportTicketMessageConnect",
    "group": "support_ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "supportTicketAssociateAttachmentsConnect",
    "group": "support_ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "supportTicketCreateSupportTicketConnect",
    "group": "support_ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "supportTicketGetAttachmentUrlConnect",
    "group": "support_ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "supportTicketGetSupportTicketConnect",
    "group": "support_ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "supportTicketListSupportTicketsConnect",
    "group": "support_ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "supportTicketPresignAttachmentUploadConnect",
    "group": "support_ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketAddAssigneeConnect",
    "group": "ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketAddLabelConnect",
    "group": "ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketCreateLabelConnect",
    "group": "ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketCreateTicketConnect",
    "group": "ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketDeleteLabelConnect",
    "group": "ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketDeleteTicketConnect",
    "group": "ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketGetActiveTicketsConnect",
    "group": "ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketGetBoardConnect",
    "group": "ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketGetSubTicketsConnect",
    "group": "ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketGetTicketConnect",
    "group": "ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketGetTicketPods",
    "group": "ticket",
    "params": [
      {
        "name": "slug",
        "type": "string"
      },
      {
        "name": "activeOnly",
        "type": "boolean | undefined | null"
      }
    ],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "ticketListLabelsConnect",
    "group": "ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketListTicketsConnect",
    "group": "ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketRemoveAssigneeConnect",
    "group": "ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketRemoveLabelConnect",
    "group": "ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketUpdateLabelConnect",
    "group": "ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketUpdateTicketConnect",
    "group": "ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketUpdateTicketStatusConnect",
    "group": "ticket",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketRelationsCreateCommentConnect",
    "group": "ticket_relations",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketRelationsCreateRelationConnect",
    "group": "ticket_relations",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketRelationsDeleteCommentConnect",
    "group": "ticket_relations",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketRelationsDeleteRelationConnect",
    "group": "ticket_relations",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketRelationsLinkCommitConnect",
    "group": "ticket_relations",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketRelationsListCommentsConnect",
    "group": "ticket_relations",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketRelationsListCommitsConnect",
    "group": "ticket_relations",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketRelationsListMergeRequestsConnect",
    "group": "ticket_relations",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketRelationsListRelationsConnect",
    "group": "ticket_relations",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketRelationsUnlinkCommitConnect",
    "group": "ticket_relations",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "ticketRelationsUpdateCommentConnect",
    "group": "ticket_relations",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "appAutopilotAppendIteration",
    "group": "uncategorized",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appAutopilotApplyFetchedControllers",
    "group": "uncategorized",
    "params": [
      {
        "name": "respBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appAutopilotApplyFetchedCurrentController",
    "group": "uncategorized",
    "params": [
      {
        "name": "respBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appAutopilotApplyFetchedIterations",
    "group": "uncategorized",
    "params": [
      {
        "name": "key",
        "type": "string"
      },
      {
        "name": "respBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appAutopilotControllersJson",
    "group": "uncategorized",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "appAutopilotControllersProto",
    "group": "uncategorized",
    "params": [],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "appAutopilotInsertController",
    "group": "uncategorized",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appAutopilotIterationsJson",
    "group": "uncategorized",
    "params": [
      {
        "name": "key",
        "type": "string"
      }
    ],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "appAutopilotIterationsProto",
    "group": "uncategorized",
    "params": [
      {
        "name": "key",
        "type": "string"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "appAutopilotPatchController",
    "group": "uncategorized",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appAutopilotRemoveControllerProto",
    "group": "uncategorized",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appAutopilotSetCurrentControllerProto",
    "group": "uncategorized",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appAutopilotThinkingHistoryJson",
    "group": "uncategorized",
    "params": [
      {
        "name": "key",
        "type": "string"
      }
    ],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "appAutopilotThinkingJson",
    "group": "uncategorized",
    "params": [
      {
        "name": "key",
        "type": "string"
      }
    ],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "appAutopilotUpdateThinkingProto",
    "group": "uncategorized",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appAvailableRunnersJson",
    "group": "uncategorized",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "appAvailableRunnersProto",
    "group": "uncategorized",
    "params": [],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "appChannelApplyFetchedChannel",
    "group": "uncategorized",
    "params": [
      {
        "name": "respBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appChannelApplyFetchedChannels",
    "group": "uncategorized",
    "params": [
      {
        "name": "respBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appChannelApplyFetchedMembers",
    "group": "uncategorized",
    "params": [
      {
        "name": "channelId",
        "type": "number"
      },
      {
        "name": "respBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appChannelApplyFetchedMessages",
    "group": "uncategorized",
    "params": [
      {
        "name": "channelId",
        "type": "number"
      },
      {
        "name": "respBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appChannelApplyFetchedMessagesPrepend",
    "group": "uncategorized",
    "params": [
      {
        "name": "channelId",
        "type": "number"
      },
      {
        "name": "respBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appChannelApplyFetchedPods",
    "group": "uncategorized",
    "params": [
      {
        "name": "channelId",
        "type": "number"
      },
      {
        "name": "respBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appChannelApplyMessageEdited",
    "group": "uncategorized",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appChannelClearUnread",
    "group": "uncategorized",
    "params": [
      {
        "name": "channelId",
        "type": "number"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appChannelInsertChannel",
    "group": "uncategorized",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appChannelInsertMessage",
    "group": "uncategorized",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appChannelMentionCountsJson",
    "group": "uncategorized",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "appChannelMessagesJson",
    "group": "uncategorized",
    "params": [
      {
        "name": "channelId",
        "type": "number"
      }
    ],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "appChannelPatchMemberCount",
    "group": "uncategorized",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appChannelPodsJson",
    "group": "uncategorized",
    "params": [
      {
        "name": "channelId",
        "type": "number"
      }
    ],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "appChannelRemoveMember",
    "group": "uncategorized",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appChannelRemoveMessage",
    "group": "uncategorized",
    "params": [
      {
        "name": "channelId",
        "type": "number"
      },
      {
        "name": "messageId",
        "type": "number"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appChannelReplaceMembers",
    "group": "uncategorized",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appChannelReplacePods",
    "group": "uncategorized",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appChannelReplaceUnreadCounts",
    "group": "uncategorized",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appChannelsJson",
    "group": "uncategorized",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "appChannelUnreadCountsJson",
    "group": "uncategorized",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "appCurrentRunnerJson",
    "group": "uncategorized",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "appCurrentRunnerProto",
    "group": "uncategorized",
    "params": [],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "appGetMeshNodeJson",
    "group": "uncategorized",
    "params": [
      {
        "name": "podKey",
        "type": "string"
      }
    ],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "appGetPodJson",
    "group": "uncategorized",
    "params": [
      {
        "name": "podKey",
        "type": "string"
      }
    ],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "appGetPodProto",
    "group": "uncategorized",
    "params": [
      {
        "name": "podKey",
        "type": "string"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "appMeshReplaceTopology",
    "group": "uncategorized",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appPodApplyAppendedPods",
    "group": "uncategorized",
    "params": [
      {
        "name": "respBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appPodApplyFetchedPods",
    "group": "uncategorized",
    "params": [
      {
        "name": "respBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appPodInsertCreated",
    "group": "uncategorized",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appPodMarkTerminated",
    "group": "uncategorized",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appPodPatchPerpetual",
    "group": "uncategorized",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appPodRemove",
    "group": "uncategorized",
    "params": [
      {
        "name": "podKey",
        "type": "string"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appPodsJson",
    "group": "uncategorized",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "appRunnerApplyFetched",
    "group": "uncategorized",
    "params": [
      {
        "name": "respBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appRunnerApplyFetchedAvailable",
    "group": "uncategorized",
    "params": [
      {
        "name": "respBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appRunnerApplyFetchedCurrent",
    "group": "uncategorized",
    "params": [
      {
        "name": "respBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appRunnerPatch",
    "group": "uncategorized",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appRunnerRemove",
    "group": "uncategorized",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appRunnerSetCurrent",
    "group": "uncategorized",
    "params": [
      {
        "name": "reqBytes",
        "type": "Array<number>"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appRunnersJson",
    "group": "uncategorized",
    "params": [],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "appRunnersProto",
    "group": "uncategorized",
    "params": [],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "appSelectChannel",
    "group": "uncategorized",
    "params": [
      {
        "name": "id",
        "type": "number | undefined | null"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appSetCurrentChannel",
    "group": "uncategorized",
    "params": [
      {
        "name": "id",
        "type": "number | undefined | null"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "appSetCurrentUser",
    "group": "uncategorized",
    "params": [
      {
        "name": "userId",
        "type": "number | undefined | null"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "relayDisconnect",
    "group": "uncategorized",
    "params": [
      {
        "name": "podKey",
        "type": "string"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "relayDisconnectAll",
    "group": "uncategorized",
    "params": [],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "relayForceResize",
    "group": "uncategorized",
    "params": [
      {
        "name": "podKey",
        "type": "string"
      },
      {
        "name": "cols",
        "type": "number"
      },
      {
        "name": "rows",
        "type": "number"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "relayGetPodSize",
    "group": "uncategorized",
    "params": [
      {
        "name": "podKey",
        "type": "string"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "relayGetStatus",
    "group": "uncategorized",
    "params": [
      {
        "name": "podKey",
        "type": "string"
      }
    ],
    "returnType": "string",
    "ipcExposable": true
  },
  {
    "name": "relayIsRunnerDisconnected",
    "group": "uncategorized",
    "params": [
      {
        "name": "podKey",
        "type": "string"
      }
    ],
    "returnType": "boolean",
    "ipcExposable": true
  },
  {
    "name": "relayOnAcpMessage",
    "group": "uncategorized",
    "params": [
      {
        "name": "podKey",
        "type": "string"
      },
      {
        "name": "onAcp",
        "type": "(err: unknown, arg: string) => void"
      }
    ],
    "returnType": "void",
    "ipcExposable": false
  },
  {
    "name": "relayOnPodDisconnected",
    "group": "uncategorized",
    "params": [
      {
        "name": "onDisconnect",
        "type": "(err: unknown, arg: string) => void"
      }
    ],
    "returnType": "void",
    "ipcExposable": false
  },
  {
    "name": "relayOnStatusChange",
    "group": "uncategorized",
    "params": [
      {
        "name": "podKey",
        "type": "string"
      },
      {
        "name": "onStatus",
        "type": "(err: unknown, arg: string) => void"
      }
    ],
    "returnType": "void",
    "ipcExposable": false
  },
  {
    "name": "relaySend",
    "group": "uncategorized",
    "params": [
      {
        "name": "podKey",
        "type": "string"
      },
      {
        "name": "data",
        "type": "string"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "relaySendAcpCommand",
    "group": "uncategorized",
    "params": [
      {
        "name": "podKey",
        "type": "string"
      },
      {
        "name": "command",
        "type": "string"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "relaySendResize",
    "group": "uncategorized",
    "params": [
      {
        "name": "podKey",
        "type": "string"
      },
      {
        "name": "cols",
        "type": "number"
      },
      {
        "name": "rows",
        "type": "number"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "relaySubscribe",
    "group": "uncategorized",
    "params": [
      {
        "name": "podKey",
        "type": "string"
      },
      {
        "name": "subscriptionId",
        "type": "string"
      },
      {
        "name": "relayUrl",
        "type": "string"
      },
      {
        "name": "token",
        "type": "string"
      },
      {
        "name": "onOutput",
        "type": "(err: unknown, arg: Array<number>) => void"
      }
    ],
    "returnType": "void",
    "ipcExposable": false
  },
  {
    "name": "relayUnsubscribe",
    "group": "uncategorized",
    "params": [
      {
        "name": "podKey",
        "type": "string"
      },
      {
        "name": "subscriptionId",
        "type": "string"
      }
    ],
    "returnType": "void",
    "ipcExposable": true
  },
  {
    "name": "userChangePasswordConnect",
    "group": "user",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "userDeleteIdentityConnect",
    "group": "user",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "userGetMeConnect",
    "group": "user",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "userListIdentitiesConnect",
    "group": "user",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "userSearchUsersConnect",
    "group": "user",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  },
  {
    "name": "userUpdateMeConnect",
    "group": "user",
    "params": [
      {
        "name": "request",
        "type": "Array<number>"
      }
    ],
    "returnType": "Array<number>",
    "ipcExposable": true
  }
];
