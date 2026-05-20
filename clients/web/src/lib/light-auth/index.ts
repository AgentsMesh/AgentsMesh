// Single-import surface for the (auth) route group. Pages import named
// functions from "@/lib/light-auth" rather than reaching into the per-file
// submodules — keeps refactors painless.
// Error handling convention:
//   - Discovery / probe / best-effort reads (lightDiscoverSSO,
//     fetchFirstOrgSlug, lightFetchMe) swallow exceptions and return
//     empty / null. They must not break the page if the backend is
//     unhappy.
//   - State-changing or critical-path calls (lightLogin, lightRegister,
//     lightVerifyEmail, lightForgotPassword, lightResetPassword,
//     lightCreateOrganization, lightAcceptInvitation, lightAuthorizeRunner,
//     lightLdapAuth) throw ApiError on 4xx/5xx. Callers display the
//     server message / map known error codes.
// persistLoginResponse / persistOAuthTokens / OAuthCallbackTokens are
// intentionally NOT re-exported here — they're internal helpers consumed
// by the business functions in this module, not by route pages.

export { lightFetch, type LightFetchOptions } from "./api-fetch";
export { type AuthLoginResponse } from "./persist";
export {
  fetchFirstOrgSlug,
  resolvePostLoginUrlLight,
} from "./post-login-redirect";
export { lightLogin, type LightLoginInput } from "./login";
export { lightRegister, type LightRegisterInput } from "./register";
export { lightVerifyEmail, lightResendVerification } from "./verify-email";
export {
  lightForgotPassword,
  lightResetPassword,
  type LightResetPasswordInput,
} from "./password-reset";
export {
  consumeOAuthCallbackParams,
  type OAuthCallbackResult,
} from "./oauth-callback";
export {
  lightDiscoverSSO,
  lightLdapAuth,
  type LightLdapAuthInput,
} from "./sso-discover";
export {
  lightListOrganizations,
  lightCreateOrganization,
  lightCreatePersonalOrganization,
  type LightOrganization,
  type LightCreateOrgInput,
} from "./organizations";
export {
  lightFetchInvitation,
  lightAcceptInvitation,
} from "./invitations";
export {
  lightGetRunnerAuthStatus,
  lightAuthorizeRunner,
  lightCreateRunnerToken,
  lightListRunners,
  type LightAuthorizeRunnerInput,
} from "./runners";
export { lightFetchMe, type LightUser } from "./me";
